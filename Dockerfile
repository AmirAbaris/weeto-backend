# builder: compile static linux binary
FROM golang:1.26-alpine AS builder
RUN sed -i 's#https://dl-cdn.alpinelinux.org/alpine#https://mirrors.aliyun.com/alpine#g' /etc/apk/repositories \
    && apk add --no-cache git
RUN go env -w GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /worker ./cmd/worker

# api: minimal runtime (~15MB)
FROM alpine:3.21 AS api

RUN sed -i 's#https://dl-cdn.alpinelinux.org/alpine#https://mirrors.aliyun.com/alpine#g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates wget tzdata

COPY --from=builder /api /api
EXPOSE 8080
USER nobody
ENTRYPOINT ["/api"]

# worker: notification outbox processor
FROM alpine:3.21 AS worker

RUN sed -i 's#https://dl-cdn.alpinelinux.org/alpine#https://mirrors.aliyun.com/alpine#g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata

COPY --from=builder /worker /worker
USER nobody
ENTRYPOINT ["/worker"]