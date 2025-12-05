FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO_ENABLED=0 for static binary
RUN CGO_ENABLED=0 go build -o webhook ./cmd/webhook

FROM scratch
WORKDIR /app
# Copy CA certificates for registry interaction
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/webhook .
ENTRYPOINT ["./webhook"]
