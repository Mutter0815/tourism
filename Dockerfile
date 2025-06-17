# Dockerfile для сборки backend-сервиса (API)
FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/api .
COPY migrations ./migrations
EXPOSE 8080
CMD ["./api"]
