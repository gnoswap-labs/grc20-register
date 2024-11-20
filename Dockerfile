#===============
# Stage 1: Build
#===============

FROM golang:1.22 AS builder

COPY . /app

WORKDIR /app

RUN go build -o grc20-register ./cmd
RUN ls -l /app/grc20-register

#===============
# Stage 2: Run
#===============

FROM alpine

COPY --from=builder /app/grc20-register /usr/local/bin/grc20-register

RUN ls -l /usr/local/bin/grc20-register

ENTRYPOINT [ "/usr/local/bin/grc20-register" ]