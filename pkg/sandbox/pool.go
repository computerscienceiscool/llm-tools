package sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

// PooledContainer represents a container in the pool
type PooledContainer struct {
	ID         string
	Image      string
	UsageCount int
	MaxUses    int
	CreatedAt  time.Time
	LastUsedAt time.Time
	InUse      bool
	Healthy    bool
	mu         sync.Mutex
}

// ContainerPool manages a pool of reusable Docker containers
type ContainerPool struct {
	client                   *client.Client
	containers               []*PooledContainer
	available                chan *PooledContainer
	config                   PoolConfig
	mu                       sync.RWMutex
	closed                   bool
	wg                       sync.WaitGroup
	healthCheckTicker        *time.Ticker
	statsContainersCreated   int64
	statsContainersDestroyed int64
	statsPoolHits            int64
	statsPoolMisses          int64
}

// PoolConfig holds pool configuration
type PoolConfig struct {
	Size                int
	MaxUsesPerContainer int
	IdleTimeout         time.Duration
	HealthCheckInterval time.Duration
	StartupContainers   int
	Image               string
	MemoryLimit         string
	CPULimit            int
	RepoRoot            string
}

// NewContainerPool creates a new container pool
func NewContainerPool(ctx context.Context, cfg PoolConfig) (*ContainerPool, error) {
	// Validate configuration
	if cfg.Size <= 0 {
		return nil, fmt.Errorf("pool size must be positive, got %d", cfg.Size)
	}
	if cfg.MaxUsesPerContainer <= 0 {
		return nil, fmt.Errorf("max uses per container must be positive, got %d", cfg.MaxUsesPerContainer)
	}
	if cfg.StartupContainers > cfg.Size {
		cfg.StartupContainers = cfg.Size
	}
	if cfg.Image == "" {
		return nil, fmt.Errorf("container image cannot be empty")
	}

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test Docker connection
	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("Docker daemon not available: %w", err)
	}

	// Pull image if needed
	if err := PullDockerImage(cfg.Image, false); err != nil {
		cli.Close()
		return nil, fmt.Errorf("failed to pull image %s: %w", cfg.Image, err)
	}

	pool := &ContainerPool{
		client:     cli,
		containers: make([]*PooledContainer, 0, cfg.Size),
		available:  make(chan *PooledContainer, cfg.Size),
		config:     cfg,
		closed:     false,
	}

	// Pre-create startup containers
	for i := 0; i < cfg.StartupContainers; i++ {
		container, err := pool.createContainer(ctx)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create startup container %d: %w", i, err)
		}
		pool.containers = append(pool.containers, container)
		pool.available <- container
	}

	// Start health check goroutine
	if cfg.HealthCheckInterval > 0 {
		pool.healthCheckTicker = time.NewTicker(cfg.HealthCheckInterval)
		pool.wg.Add(1)
		go pool.healthCheckLoop()
	}

	return pool, nil
}

// Get acquires a container from the pool
func (p *ContainerPool) Get(ctx context.Context) (*PooledContainer, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	select {
	case container := <-p.available:
		// Got container from pool
		container.mu.Lock()
		container.InUse = true
		container.UsageCount++
		container.LastUsedAt = time.Now()
		container.mu.Unlock()

		p.mu.Lock()
		p.statsPoolHits++
		p.mu.Unlock()

		return container, nil

	case <-time.After(30 * time.Second):
		// Timeout waiting for container - try to create new one if pool not full
		p.mu.Lock()
		if len(p.containers) < p.config.Size {
			p.mu.Unlock()
			container, err := p.createContainer(ctx)
			if err != nil {
				return nil, fmt.Errorf("pool exhausted and failed to create container: %w", err)
			}

			p.mu.Lock()
			p.containers = append(p.containers, container)
			p.statsPoolMisses++
			p.mu.Unlock()

			container.mu.Lock()
			container.InUse = true
			container.UsageCount++
			container.LastUsedAt = time.Now()
			container.mu.Unlock()

			return container, nil
		}
		p.mu.Unlock()
		return nil, fmt.Errorf("pool exhausted: timeout waiting for available container")

	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled while waiting for container")
	}
}

// Return releases a container back to the pool
func (p *ContainerPool) Return(ctx context.Context, container *PooledContainer) error {
	if container == nil {
		return fmt.Errorf("cannot return nil container")
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		// Pool is closed, destroy the container
		return p.destroyContainer(ctx, container)
	}
	p.mu.RUnlock()

	container.mu.Lock()
	container.InUse = false

	// Check if container should be recycled
	shouldRecycle := container.UsageCount >= container.MaxUses || !container.Healthy

	if shouldRecycle {
		container.mu.Unlock()
		// Remove from pool and destroy
		p.removeContainer(container)
		if err := p.destroyContainer(ctx, container); err != nil {
			return fmt.Errorf("failed to destroy container during recycle: %w", err)
		}

		// Create replacement container
		newContainer, err := p.createContainer(ctx)
		if err != nil {
			return fmt.Errorf("failed to create replacement container: %w", err)
		}

		p.mu.Lock()
		p.containers = append(p.containers, newContainer)
		p.mu.Unlock()

		p.available <- newContainer
		return nil
	}

	container.mu.Unlock()

	// Return to available pool
	select {
	case p.available <- container:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context canceled while returning container")
	default:
		// Channel full (shouldn't happen but handle it)
		return fmt.Errorf("failed to return container to pool: channel full")
	}
}

// createContainer creates a new container for the pool
func (p *ContainerPool) createContainer(ctx context.Context) (*PooledContainer, error) {
	// Create minimal container config - just keeps the container running
	containerConfig := &container.Config{
		Image: p.config.Image,
		Cmd:   []string{"sleep", "infinity"},
		Tty:   true,
		User:  "1000:1000",
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   p.config.RepoRoot,
				Target:   "/workspace",
				ReadOnly: false,
			},
		},
		NetworkMode: "none",
		Resources: container.Resources{
			Memory:   parseMemoryLimit(p.config.MemoryLimit),
			NanoCPUs: int64(p.config.CPULimit) * 1000000000,
		},
		CapDrop:     []string{"ALL"},
		SecurityOpt: []string{"no-new-privileges"},
	}

	resp, err := p.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	if err := p.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		p.client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	pooledContainer := &PooledContainer{
		ID:         resp.ID,
		Image:      p.config.Image,
		UsageCount: 0,
		MaxUses:    p.config.MaxUsesPerContainer,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
		InUse:      false,
		Healthy:    true,
	}

	p.mu.Lock()
	p.statsContainersCreated++
	p.mu.Unlock()

	return pooledContainer, nil
}

// destroyContainer removes a container
func (p *ContainerPool) destroyContainer(ctx context.Context, container *PooledContainer) error {
	if container == nil {
		return nil
	}

	err := p.client.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return fmt.Errorf("failed to remove container %s: %w", container.ID[:12], err)
	}

	p.mu.Lock()
	p.statsContainersDestroyed++
	p.mu.Unlock()

	return nil
}

// removeContainer removes a container from the pool's tracking
func (p *ContainerPool) removeContainer(target *PooledContainer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, c := range p.containers {
		if c.ID == target.ID {
			// Remove from slice
			p.containers = append(p.containers[:i], p.containers[i+1:]...)
			break
		}
	}
}

// healthCheck checks if a container is healthy
func (p *ContainerPool) healthCheck(ctx context.Context, container *PooledContainer) bool {
	inspect, err := p.client.ContainerInspect(ctx, container.ID)
	if err != nil {
		return false
	}

	return inspect.State.Running && !inspect.State.Restarting
}

// healthCheckLoop runs periodic health checks and idle cleanup
func (p *ContainerPool) healthCheckLoop() {
	defer p.wg.Done()

	ctx := context.Background()

	for {
		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		select {
		case <-p.healthCheckTicker.C:
			p.mu.RLock()
			containers := make([]*PooledContainer, len(p.containers))
			copy(containers, p.containers)
			p.mu.RUnlock()

			for _, c := range containers {
				c.mu.Lock()

				// Check if container is idle and should be removed
				if !c.InUse && p.config.IdleTimeout > 0 {
					idleTime := time.Since(c.LastUsedAt)
					if idleTime > p.config.IdleTimeout {
						// Container exceeded idle timeout
						c.Healthy = false
						c.mu.Unlock()

						// Remove and destroy idle container
						p.removeContainer(c)
						p.destroyContainer(ctx, c)
						continue
					}
				}

				// Check health for non-idle containers
				if !c.InUse {
					healthy := p.healthCheck(ctx, c)
					if !healthy {
						c.Healthy = false
					}
				}

				c.mu.Unlock()
			}
		}
	}
}

// Close shuts down the pool and cleans up all containers
func (p *ContainerPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Stop health check
	if p.healthCheckTicker != nil {
		p.healthCheckTicker.Stop()
	}

	// Wait for health check goroutine
	p.wg.Wait()

	// Close available channel
	close(p.available)

	// Destroy all containers
	ctx := context.Background()
	for _, container := range p.containers {
		p.destroyContainer(ctx, container)
	}

	// Close Docker client
	if p.client != nil {
		return p.client.Close()
	}

	return nil
}

// Stats returns pool statistics
func (p *ContainerPool) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	inUseCount := 0
	for _, c := range p.containers {
		c.mu.Lock()
		if c.InUse {
			inUseCount++
		}
		c.mu.Unlock()
	}

	return map[string]interface{}{
		"total_containers":     len(p.containers),
		"available_containers": len(p.available),
		"in_use_containers":    inUseCount,
		"containers_created":   p.statsContainersCreated,
		"containers_destroyed": p.statsContainersDestroyed,
		"pool_hits":            p.statsPoolHits,
		"pool_misses":          p.statsPoolMisses,
	}
}
