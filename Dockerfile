FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server-agent ./cmd/agent


FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/server-agent /app/server-agent

CMD ["/app/server-agent"]