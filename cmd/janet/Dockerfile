# Build stage
FROM golang:alpine as builder

# Install build dependencies
RUN apk add --no-cache build-base sqlite-dev

WORKDIR /app
COPY . .

# Set CGO_ENABLED and build the binary
ENV CGO_ENABLED=1
RUN go build -o ./cmd/janet/janet ./cmd/janet/main.go

# Final image stage
FROM alpine:latest

# Install SQLite3 runtime package and ca-certificates
RUN apk add --no-cache sqlite-libs ca-certificates

COPY --from=builder /app/cmd/janet/janet /janet
ENTRYPOINT ["/janet"]
