FROM alpine:3.20
RUN apk add --no-cache ca-certificates ffmpeg && addgroup -S app && adduser -S -G app app
COPY --chmod=0555 build/fulfillment-server build/fulfillment-migrate /usr/local/bin/
USER app
EXPOSE 8002 9002
ENTRYPOINT ["fulfillment-server"]
