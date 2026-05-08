FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o mkt-service .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/mkt-service .
EXPOSE 8085
CMD ["./mkt-service"]