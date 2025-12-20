# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o calmnews ./cmd/calmnews

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/calmnews /app/calmnews

# Create data directory
RUN mkdir -p /app/data && chmod 755 /app/data

# Expose port
EXPOSE 8080

# Run as non-root user
RUN addgroup -g 1000 calmnews && \
    adduser -D -u 1000 -G calmnews calmnews && \
    chown -R calmnews:calmnews /app

USER calmnews

# Set data directory via environment variable
ENV CALMNEWS_DATA_DIR=/app/data

# Run the application
CMD ["/app/calmnews"]

