FROM pgsty/silo:latest

ENV MINIO_ROOT_USER=minioadmin \
    MINIO_ROOT_PASSWORD=minioadminpassword

EXPOSE 9000 9001

CMD ["server", "/data", "--console-address", ":9001"]
