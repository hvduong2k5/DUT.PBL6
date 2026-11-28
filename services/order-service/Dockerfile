FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && addgroup -g 10001 app && adduser -D -u 10001 -G app app
COPY --chmod=0555 build/order-api build/order-simulator /usr/local/bin/
USER app
EXPOSE 8004 9004 8100
CMD ["order-api"]
