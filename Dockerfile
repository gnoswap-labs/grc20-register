FROM golang:1.22 

COPY . /app

WORKDIR /app

RUN go build -o grc20-register ./cmd

ENTRYPOINT [ "/app/grc20-register" ]