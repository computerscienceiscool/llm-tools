FROM golang:1.22.2-alpine

# Create non-root user
RUN addgroup -g 1000 llmuser && \
    adduser -D -u 1000 -G llmuser llmuser

# Install only essential tools
RUN apk add --no-cache coreutils

# Set working directory
WORKDIR /workspace

# Switch to non-root user
USER llmuser

# Default command (will be overridden)
CMD ["/bin/sh"]
