package sandbox

import (
	"context"
	"testing"
	"time"
)

// TestNewContainerPool tests pool creation
func TestNewContainerPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker-dependent test in short mode")
	}

	ctx := context.Background()
	cfg := PoolConfig{
		Size:                5,
		MaxUsesPerContainer: 10,
		IdleTimeout:         5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		StartupContainers:   2,
		Image:               "alpine:latest",
		MemoryLimit:         "256m",
		CPULimit:            1,
		RepoRoot:            "/tmp",
	}

	pool, err := NewContainerPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	stats := pool.Stats()
	if stats["total_containers"].(int) != 2 {
		t.Errorf("Expected 2 startup containers, got %d", stats["total_containers"])
	}
}

// TestPoolGetReturn tests getting and returning containers
func TestPoolGetReturn(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker-dependent test in short mode")
	}

	ctx := context.Background()
	cfg := PoolConfig{
		Size:                3,
		MaxUsesPerContainer: 10,
		IdleTimeout:         5 * time.Minute,
		HealthCheckInterval: 0, // Disable health checks for test
		StartupContainers:   1,
		Image:               "alpine:latest",
		MemoryLimit:         "256m",
		CPULimit:            1,
		RepoRoot:            "/tmp",
	}

	pool, err := NewContainerPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Get a container
	container, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get container: %v", err)
	}

	if container.UsageCount != 1 {
		t.Errorf("Expected usage count 1, got %d", container.UsageCount)
	}

	if !container.InUse {
		t.Error("Container should be marked as in use")
	}

	// Return the container
	err = pool.Return(ctx, container)
	if err != nil {
		t.Fatalf("Failed to return container: %v", err)
	}

	if container.InUse {
		t.Error("Container should not be marked as in use after return")
	}

	// Get it again
	container2, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get container second time: %v", err)
	}

	if container2.ID != container.ID {
		t.Error("Expected to get same container back")
	}

	if container2.UsageCount != 2 {
		t.Errorf("Expected usage count 2, got %d", container2.UsageCount)
	}

	pool.Return(ctx, container2)
}

// TestPoolRecycling tests container recycling when max uses reached
func TestPoolRecycling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker-dependent test in short mode")
	}

	ctx := context.Background()
	cfg := PoolConfig{
		Size:                3,
		MaxUsesPerContainer: 3, // Low limit to trigger recycling
		IdleTimeout:         5 * time.Minute,
		HealthCheckInterval: 0,
		StartupContainers:   1,
		Image:               "alpine:latest",
		MemoryLimit:         "256m",
		CPULimit:            1,
		RepoRoot:            "/tmp",
	}

	pool, err := NewContainerPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Use container 3 times
	var containerID string
	for i := 0; i < 3; i++ {
		container, err := pool.Get(ctx)
		if err != nil {
			t.Fatalf("Failed to get container iteration %d: %v", i, err)
		}
		if i == 0 {
			containerID = container.ID
		}
		pool.Return(ctx, container)
	}

	// Fourth get should return a different container (recycled)
	container, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get container after recycling: %v", err)
	}

	if container.ID == containerID {
		t.Error("Expected different container after recycling")
	}

	if container.UsageCount != 1 {
		t.Errorf("New container should have usage count 1, got %d", container.UsageCount)
	}

	pool.Return(ctx, container)
}

// TestPoolConcurrency tests concurrent access to pool
func TestPoolConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker-dependent test in short mode")
	}

	ctx := context.Background()
	cfg := PoolConfig{
		Size:                5,
		MaxUsesPerContainer: 100,
		IdleTimeout:         5 * time.Minute,
		HealthCheckInterval: 0,
		StartupContainers:   3,
		Image:               "alpine:latest",
		MemoryLimit:         "256m",
		CPULimit:            1,
		RepoRoot:            "/tmp",
	}

	pool, err := NewContainerPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Run 10 concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			container, err := pool.Get(ctx)
			if err != nil {
				t.Errorf("Failed to get container: %v", err)
				done <- false
				return
			}

			// Simulate some work
			time.Sleep(10 * time.Millisecond)

			err = pool.Return(ctx, container)
			if err != nil {
				t.Errorf("Failed to return container: %v", err)
				done <- false
				return
			}

			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := pool.Stats()
	t.Logf("Pool stats after concurrent test: %+v", stats)
}

// TestPoolClose tests graceful shutdown
func TestPoolClose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker-dependent test in short mode")
	}

	ctx := context.Background()
	cfg := PoolConfig{
		Size:                3,
		MaxUsesPerContainer: 10,
		IdleTimeout:         5 * time.Minute,
		HealthCheckInterval: 0,
		StartupContainers:   2,
		Image:               "alpine:latest",
		MemoryLimit:         "256m",
		CPULimit:            1,
		RepoRoot:            "/tmp",
	}

	pool, err := NewContainerPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	stats := pool.Stats()
	containerCount := stats["total_containers"].(int)

	err = pool.Close()
	if err != nil {
		t.Fatalf("Failed to close pool: %v", err)
	}

	// Try to get container after close (should fail)
	_, err = pool.Get(ctx)
	if err == nil {
		t.Error("Expected error when getting container from closed pool")
	}

	// Verify stats show all containers destroyed
	stats = pool.Stats()
	if stats["containers_destroyed"].(int64) != int64(containerCount) {
		t.Errorf("Expected %d containers destroyed, got %d",
			containerCount, stats["containers_destroyed"])
	}
}

// TestPoolInvalidConfig tests pool creation with invalid configuration
func TestPoolInvalidConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		config PoolConfig
	}{
		{
			name: "zero size",
			config: PoolConfig{
				Size:                0,
				MaxUsesPerContainer: 10,
				Image:               "alpine:latest",
			},
		},
		{
			name: "zero max uses",
			config: PoolConfig{
				Size:                5,
				MaxUsesPerContainer: 0,
				Image:               "alpine:latest",
			},
		},
		{
			name: "empty image",
			config: PoolConfig{
				Size:                5,
				MaxUsesPerContainer: 10,
				Image:               "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewContainerPool(ctx, tt.config)
			if err == nil {
				t.Error("Expected error for invalid config")
			}
		})
	}
}

// BenchmarkPoolGet benchmarks getting containers from pool
func BenchmarkPoolGet(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping Docker-dependent benchmark in short mode")
	}

	ctx := context.Background()
	cfg := PoolConfig{
		Size:                10,
		MaxUsesPerContainer: 1000,
		IdleTimeout:         5 * time.Minute,
		HealthCheckInterval: 0,
		StartupContainers:   5,
		Image:               "alpine:latest",
		MemoryLimit:         "256m",
		CPULimit:            1,
		RepoRoot:            "/tmp",
	}

	pool, err := NewContainerPool(ctx, cfg)
	if err != nil {
		b.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		container, err := pool.Get(ctx)
		if err != nil {
			b.Fatalf("Failed to get container: %v", err)
		}
		pool.Return(ctx, container)
	}
}
