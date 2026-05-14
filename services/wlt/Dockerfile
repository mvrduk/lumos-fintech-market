FROM golang:1.24-alpine AS builder

# Нужно для confluent-kafka-go (CGO)
RUN apk add --no-cache gcc musl-dev librdkafka-dev

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

# Убираем CGO_ENABLED=0
RUN GOOS=linux go build -tags musl -o wlt-service .

FROM alpine:latest

# Рантайм зависимость librdkafka
RUN apk --no-cache add ca-certificates librdkafka

WORKDIR /root/

COPY --from=builder /app/wlt-service .

EXPOSE 8084

CMD ["./wlt-service"]