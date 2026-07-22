# Use a specific version of distroless based on SHA256 digest for reproducibility and security
FROM nginx:1.31 AS run

ENV JSFOLDER=/usr/share/nginx/html/*.js
COPY apps/frontend/rolesui/start-nginx.sh /usr/bin/start-nginx.sh
RUN chmod +x /usr/bin/start-nginx.sh
COPY apps/frontend/rolesui/default.conf /etc/nginx/conf.d
COPY dist/apps/frontend/rolesui/ /usr/share/nginx/html/

ENTRYPOINT [ "start-nginx.sh" ]