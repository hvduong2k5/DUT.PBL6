FROM postgres:16-alpine

ENV POSTGRES_USER=postgres \
    POSTGRES_PASSWORD=postgrespassword

COPY init-databases.sql /docker-entrypoint-initdb.d/init-databases.sql

EXPOSE 5432
