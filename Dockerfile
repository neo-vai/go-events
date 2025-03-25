FROM golang:1.26.1-alpine3.23 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app ./cmd/api

FROM alpine:3.23.3
WORKDIR /app
COPY --from=builder /app/app .

EXPOSE 8080
CMD ["./app"]