FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/order-api ./cmd/server && CGO_ENABLED=0 go build -o /bin/order-simulator ./cmd/simulator
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && addgroup -g 10001 app && adduser -D -u 10001 -G app app
COPY --from=builder /bin/order-api /bin/order-simulator /usr/local/bin/
USER app
EXPOSE 8004 9004 8100
CMD ["order-api"]
