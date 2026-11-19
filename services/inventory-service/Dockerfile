# Inventory service — gRPC only. Build stage runs on the builder's native
# platform and cross-compiles, so the same Dockerfile produces the target
# (linux/arm64 on the Coolify VPS) binary quickly on any host.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/inventory-service ./cmd/server

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/inventory-service ./inventory-service
# SQL migrations ship with the image in case the runtime applies them from disk.
COPY --from=build /src/migrations ./migrations
USER app
ENV PORT=8001
EXPOSE 8001
ENTRYPOINT ["./inventory-service"]
