# =============================================================================
# SONG (Secure Open Network Gateway) - Multi-Stage Dockerfile
# =============================================================================
# Stage 1: Builder
# =============================================================================
FROM golang:1.21-alpine AS builder

# Install git for go modules
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG BUILD_DATE

# Build the application
# -ldflags for binary optimization and stripping debug symbols
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$${VERSION} -X main.buildDate=$${BUILD_DATE}" \
    -o /build/song-server \
    ./cmd/server

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM alpine:3.19 AS runtime

# Labels for image metadata
LABEL maintainer="AzzurroTech <support@azzurrotech.com"
LABEL org.opencontainers.image.title="SONG FaaS Server"
LABEL org.opencontainers.image.description="Secure Open Network Gateway - Functions as a Service"
LABEL org.opencontainers.image.vendor="AzzurroTech"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    dumb-init \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1000 song && \
    adduser -u 1000 -G song -s /bin/sh -D song

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/song-server /app/song-server

# Copy configuration template
COPY .env.example /app/.env.example

# Copy migrations directory if it exists
COPY --chmod=755 migrations /app/migrations

# Create necessary directories with proper permissions
RUN mkdir -p /app/static /app/templates /app/functions /app/data /app/core /app/bin && \
    chown -R song:song /app && \
    chmod 700 /app/core /app/bin

# Set environment variables
ENV SONG_ENV=production \
    SONG_HOST=0.0.0.0 \
    SONG_PORT=8080 \
    CORE_DIR=/app/core \
    STATIC_DIR=/app/static \
    TEMPLATE_DIR=/app/templates \
    FUNCTIONS_DIR=/app/functions \
    DATA_DIR=/app/data \
    DB_ENABLED=false

# Expose port
EXPOSE 8080

# Switch to non-root user
USER song

# Use dumb-init to handle signals properly
ENTRYPOINT ["dumb-init", "--"]

# Run the server
CMD ["/app/song-server"]

# =============================================================================
# Stage 3: Development (Optional)
# =============================================================================
FROM golang:1.21-alpine AS development

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata file

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Enable hot reload with air or just run with verbose output
CMD ["go", "run", "./cmd/server"]

# =============================================================================
# Health Check
# =============================================================================
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1