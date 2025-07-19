# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies (needed for CGO)
RUN apk add --no-cache build-base ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the bot binary with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o janet-bot ./cmd/janet-bot

# Final stage - minimal Alpine image
FROM alpine:latest

# Install ca-certificates for HTTPS connections and glibc compatibility
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user
RUN adduser -D -s /bin/sh janet

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/janet-bot .

# Change ownership of the binary
RUN chown janet:janet /app/janet-bot

# Switch to non-root user
USER janet

# Expose health check port (if needed)
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD pgrep janet-bot || exit 1

# Command to run
ENTRYPOINT ["./janet-bot"]
