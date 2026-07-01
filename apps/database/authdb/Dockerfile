FROM postgres:16.4

ENV POSTGRES_USER=default_user
ENV POSTGRES_PASSWORD=default_password
ENV POSTGRES_DB=jdw

COPY apps/database/authdb/src/ /docker-entrypoint-initdb.d/
