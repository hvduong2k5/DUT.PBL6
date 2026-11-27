FROM golang:1.25.0-alpine AS test
RUN apk add --no-cache gcc musl-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
CMD ["go", "test", "-race", "-count=1", "-p=2", "-coverpkg=./...", "./..."]

FROM alpine:3.22 AS local
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 traceability
COPY --chmod=0555 build/traceability /usr/local/bin/traceability
USER 10001
EXPOSE 8003 9003
ENTRYPOINT ["/usr/local/bin/traceability"]
