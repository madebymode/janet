# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o janet-backfill ./cmd/janet-backfill

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S janet && \
    adduser -u 1001 -S janet -G janet

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/janet-backfill .

# Change ownership
RUN chown janet:janet /app/janet-backfill

# Switch to non-root user
USER janet

# Set default command - exit if no params provided
ENTRYPOINT ["./janet-backfill"]
CMD ["--help-and-exit"]
