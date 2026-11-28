# Real ClamAV engine; the test harness mounts its tiny deterministic signature DB.
# This image is for isolated acceptance tests, not production malware coverage.
FROM alpine:3.20
RUN apk add --no-cache clamav-daemon
ENTRYPOINT ["clamd"]
