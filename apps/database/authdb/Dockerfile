FROM postgres:16.4

ENV POSTGRES_USER=default_user
ENV POSTGRES_PASSWORD=default_password
ENV POSTGRES_DB=jdw

COPY apps/database/authdb/src/ /docker-entrypoint-initdb.d/

# postgres image entrypoint supports running as the postgres user directly,
# skipping the root chown/gosu path used for the default root startup
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=5 \
  CMD pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" || exit 1
USER postgres
