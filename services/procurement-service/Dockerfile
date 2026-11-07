FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/procurement-service ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/procurement-migrate ./cmd/migrate

FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/procurement-service /app/procurement-service
COPY --from=build /out/procurement-migrate /app/procurement-migrate
USER app
EXPOSE 8008
ENTRYPOINT ["/app/procurement-service"]
