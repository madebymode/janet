FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO
RUN apk add --no-cache build-base ca-certificates

WORKDIR /app

# Copy go modules first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with CGO enabled for SQLite support
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
RUN go build -ldflags="-w -s" -tags="sqlite_omit_load_extension" -o migrate ./cmd/migrate

# Final stage - minimal image
FROM alpine:latest

# Install ca-certificates
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S janet && \
    adduser -S janet -u 1001 -G janet

# Copy binary
COPY --from=builder /app/migrate /usr/local/bin/migrate

# Change ownership
RUN chown janet:janet /usr/local/bin/migrate

# Switch to non-root user
USER janet

CMD ["migrate", "/app"]
