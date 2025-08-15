# Build stage
FROM golang:1.21-alpine AS builder

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o webhook ./cmd/webhook

# Final stage - scratch image
FROM scratch

# Copy ca-certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /app/webhook /webhook

# Create non-root user
USER 65534:65534

# Expose port
EXPOSE 8443

# Run webhook
ENTRYPOINT ["/webhook"]