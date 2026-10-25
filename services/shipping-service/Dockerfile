FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/shipping-service ./cmd/server

FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/shipping-service /usr/local/bin/shipping-service
USER app
EXPOSE 8012 9012
ENTRYPOINT ["/usr/local/bin/shipping-service"]
