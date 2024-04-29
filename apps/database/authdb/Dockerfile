FROM postgres:16.2
LABEL maintainer="Jake Willmsen"
LABEL description="PostgreSQL image with custom initialization scripts."
LABEL license="MIT"

ENV POSTGRES_USER=default_user
ENV POSTGRES_PASSWORD=default_password
ENV POSTGRES_DB=jdw

COPY src/ /docker-entrypoint-initdb.d/
