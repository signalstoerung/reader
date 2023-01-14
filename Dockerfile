# syntax=docker/dockerfile:1

FROM golang:1.19-alpine
RUN apk add build-base

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./
COPY www/*.html ./www/

RUN go build -o /reader

EXPOSE 8000

CMD [ "/reader" ]

