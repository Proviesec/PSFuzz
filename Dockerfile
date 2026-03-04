# PSFuzz Dockerfile
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy module and source (main + internal packages)
COPY go.* ./
COPY main.go ./
COPY internal/ ./internal/
COPY lists/ ./lists/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o psfuzz .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 psfuzz && \
    adduser -D -u 1000 -G psfuzz psfuzz

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/psfuzz /app/psfuzz
COPY --from=builder /build/lists /app/lists

# Copy configuration example (optional: mount your own with -v myconfig.json:/app/config.json -cf /app/config.json)
COPY config.example.json /app/config.example.json

# Create directories for wordlists and output
RUN mkdir -p /app/wordlists /app/output && \
    chown -R psfuzz:psfuzz /app

# Switch to non-root user
USER psfuzz

# Set entrypoint
ENTRYPOINT ["/app/psfuzz"]

# Default command (show help)
CMD ["-h"]

# Labels
LABEL maintainer="Proviesec"
LABEL version="1.0.0"
LABEL description="PSFuzz - Web path and file discovery tool"
LABEL org.opencontainers.image.source="https://github.com/Proviesec/PSFuzz"

