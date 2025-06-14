FROM golang:1.21-alpine

WORKDIR /app

RUN apk add --no-cache docker-cli

COPY . .

RUN go build -o runner

ENTRYPOINT ["./runner"]