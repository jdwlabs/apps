# nginx-unprivileged rather than nginx: the upstream image runs as root and
# binds :80, neither of which survives runAsNonRoot with all capabilities
# dropped. Same Debian lineage and shell, so the entrypoint is unchanged.
FROM nginxinc/nginx-unprivileged:1.27 AS run

ENV JSFOLDER=/usr/share/nginx/html/*.js

USER root
COPY apps/frontend/usersui/start-nginx.sh /usr/bin/start-nginx.sh
COPY apps/frontend/usersui/default.conf /etc/nginx/conf.d/default.conf
COPY dist/apps/frontend/usersui/ /usr/share/nginx/html/
# The entrypoint rewrites the built bundles in place to inject runtime config,
# so the serving user has to own the document root, not just read it.
RUN chmod +x /usr/bin/start-nginx.sh \
    && chown -R 101:101 /usr/share/nginx/html

# Numeric, not the "nginx" name: runAsNonRoot can only verify a numeric uid.
USER 101

EXPOSE 8080

ENTRYPOINT [ "start-nginx.sh" ]
