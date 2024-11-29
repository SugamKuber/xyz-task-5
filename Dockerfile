FROM golang:1.23.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN go mod tidy
COPY . .
RUN go build -a -installsuffix cgo -o slack-message-processor main.go
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/slack-message-processor .
COPY .env .
ENV GIN_MODE=release
CMD ["./slack-message-processor"]