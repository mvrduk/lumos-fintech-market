FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
# Называем бинарник правильно - pay-service
RUN CGO_ENABLED=0 GOOS=linux go build -o pay-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/pay-service .
EXPOSE 8082
CMD ["./pay-service"]
