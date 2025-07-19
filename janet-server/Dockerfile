# Build stage for React app
FROM node:20-alpine AS react_builder

WORKDIR /app/janet-server/web-react

# Install dependencies first (better caching)
COPY janet-server/web-react/package*.json ./
RUN npm ci

# Copy source and build
COPY janet-server/web-react/ .
RUN npm run build

# Build stage for Go application
FROM golang:1.24-alpine AS go_builder

# Install build dependencies
RUN apk add --no-cache build-base ca-certificates

WORKDIR /app

# Copy go modules files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built React app into the Go build context
COPY --from=react_builder /app/janet-server/web/static ./janet-server/web/static

# Set CGO_ENABLED and build the single binary with embedded web assets
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
RUN go build -mod=mod -ldflags="-w -s" -o janet-server-bin ./janet-server

# Final image stage
FROM alpine:latest

# Install ca-certificates and tzdata for timezone support
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1001 -S janet && \
    adduser -S janet -u 1001 -G janet

# Copy the single binary (web assets are embedded)
COPY --from=go_builder /app/janet-server-bin /janet-server

# Change ownership to non-root user
RUN chown janet:janet /janet-server && chmod +x /janet-server

# Switch to non-root user
USER janet

# Expose web interface port
EXPOSE 8080

# Set default environment variables
ENV JANET_WEB_LISTEN_ADDR=:8080
ENV JANET_DATABASE_URL=postgres://user:password@host:port/database?sslmode=disable
ENV JANET_WEB_PASSWORD=admin

ENTRYPOINT ["/janet-server"]