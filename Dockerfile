# Multi-stage build for PayTrack application
FROM golang:1.23.11-alpine AS builder

# Install git and ca-certificates for dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mgmtngd ./cmd/mgmtngd

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 paytrack && \
    adduser -D -s /bin/sh -u 1001 -G paytrack paytrack

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/mgmtngd .

# Copy sample configuration (optional)
COPY --from=builder /app/sample ./sample

# Create directories for logs and config
RUN mkdir -p logs private && \
    chown -R paytrack:paytrack /app

# Switch to non-root user
USER paytrack

# Expose port
EXPOSE 6789

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:6789/health || exit 1

# Command to run the application
CMD ["./mgmtngd", "--config=./private/config.yaml"] 