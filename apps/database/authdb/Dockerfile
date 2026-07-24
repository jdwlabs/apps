FROM postgres:18.4

ENV POSTGRES_USER=default_user
ENV POSTGRES_PASSWORD=default_password
ENV POSTGRES_DB=jdw

COPY apps/database/authdb/src/ /docker-entrypoint-initdb.d/

USER postgres

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" || exit 1
