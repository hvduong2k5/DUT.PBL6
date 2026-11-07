FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/fulfillment-server ./cmd/server && CGO_ENABLED=0 go build -trimpath -o /out/fulfillment-migrate ./cmd/migrate
FROM alpine:3.20
RUN apk add --no-cache ca-certificates ffmpeg && addgroup -S app && adduser -S -G app app
COPY --from=build /out/ /usr/local/bin/
USER app
EXPOSE 8002 9002
ENTRYPOINT ["fulfillment-server"]
