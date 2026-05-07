FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o card-service .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/card-service .
EXPOSE 8083
CMD ["./card-service"]