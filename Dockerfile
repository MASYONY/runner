FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o runner ./cmd/runner

FROM alpine:latest

RUN apk add --no-cache docker-cli

WORKDIR /app

COPY --from=builder /app/runner .

ENTRYPOINT ["./runner"]