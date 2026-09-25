FROM alpine:3.22

RUN apk add --no-cache bash util-linux

WORKDIR /app

COPY --from=builder /app/server-agent /app/server-agent
COPY --from=builder /app/configs/config.yaml /app/configs/config.yaml

CMD ["/app/server-agent"]