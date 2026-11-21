FROM golang:1.22.12-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/content-service ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/content-migrate ./cmd/migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app
COPY --from=builder /out/ /usr/local/bin/
# Applies pending migrations (unless RUN_MIGRATIONS=false), then starts the API.
RUN printf '%s\n' '#!/bin/sh' \
  'set -e' \
  'if [ "${RUN_MIGRATIONS:-true}" != "false" ]; then content-migrate up; fi' \
  'exec content-service' > /entrypoint.sh && chmod +x /entrypoint.sh
USER app
EXPOSE 8010
HEALTHCHECK --interval=10s --timeout=3s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8010/livez || exit 1
ENTRYPOINT ["/entrypoint.sh"]
