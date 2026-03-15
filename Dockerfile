# Build aşaması
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /compute-service ./cmd/main.go

# Çalıştırma aşaması
FROM alpine:latest
WORKDIR /
COPY --from=builder /compute-service /compute-service
COPY config.yaml /config.yaml
EXPOSE 8080
CMD ["/compute-service"]
