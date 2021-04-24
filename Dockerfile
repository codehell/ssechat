FROM golang:1.16

WORKDIR /app

RUN go get -u github.com/cosmtrek/air

ENTRYPOINT air

