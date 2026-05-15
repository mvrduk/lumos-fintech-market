
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o anx-service .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/analytics-service .

EXPOSE 9091

CMD ["./analytics-service"]