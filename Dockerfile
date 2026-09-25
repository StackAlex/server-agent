FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN echo "=== APP ===" && ls -la /app
RUN echo "=== CONFIGS ===" && ls -la /app/configs
RUN echo "=== CONFIG ===" && cat /app/configs/config.yaml

RUN go build -o server-agent ./cmd/agent


FROM alpine:3.22

RUN apk add --no-cache bash

WORKDIR /app

COPY --from=builder /app/server-agent /app/server-agent
COPY --from=builder /app/configs/config.yaml /app/configs/config.yaml

CMD ["/app/server-agent"]