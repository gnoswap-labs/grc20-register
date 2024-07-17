#===============
# Stage 1: Build
#===============

FROM golang:1.22-alpine as builder

COPY . /app

WORKDIR /app

RUN go build -o grc20-register ./cmd

#===============
# Stage 2: Run
#===============

FROM alpine

COPY --from=builder /app/grc20-register /usr/local/bin/grc20-register

ENTRYPOINT [ "/usr/local/bin/grc20-register" ]