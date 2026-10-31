# =============================================================================
# STAGE 1: Build binary statically linked
# =============================================================================
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=1.0.0" \
    -o /app/bin/profile-service \
    ./cmd/server/main.go

# =============================================================================
# STAGE 2: Distroless minimal runtime image (< 25MB)
# =============================================================================
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

COPY --from=builder /app/bin/profile-service /app/profile-service

USER appuser:appgroup

EXPOSE 8080 50051

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
    CMD wget -q --spider http://localhost:8080/livez || exit 1

ENTRYPOINT ["/app/profile-service"]
