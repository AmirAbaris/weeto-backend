# builder: compile static linux binary
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/api

# api: minimal runtime (~15MB)
FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /api /api
EXPOSE 8080
USER nobody
ENTRYPOINT ["/api"]
