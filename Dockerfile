FROM golang:1.22-bullseye AS builder
WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 go build -o grc20_register ./cmd

FROM scratch
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder ["/app/grc20_register", "/app/.env", "/"]

ENTRYPOINT ["/grc20_register", "start"]
