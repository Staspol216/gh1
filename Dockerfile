FROM golang:1.26.1 AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/api

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/app .

EXPOSE 9000

CMD ["./app"]