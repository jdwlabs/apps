FROM docker.io/node:lts-alpine AS builder
ENV JQ_VERSION=1.7
RUN apk add --no-cache jq
WORKDIR /app
COPY . .
RUN jq 'to_entries | map_values({ (.key) : ("$" + .key) }) | reduce .[] as $item ({}; . + $item)' apps/angular/container/container/src/config.json > apps/angular/container/container/src/config.tmp.json && mv apps/angular/container/container/src/config.tmp.json apps/angular/container/container/src/config.json
RUN npm install && npx nx build container

FROM docker.io/nginx:1.27 AS run
ENV JSFOLDER=/usr/share/nginx/html/*.js
COPY apps/angular/container/container/start-nginx.sh /usr/bin/start-nginx.sh
RUN chmod +x /usr/bin/start-nginx.sh
COPY apps/angular/container/container/default.conf /etc/nginx/conf.d
COPY --from=builder /app/dist/apps/angular/container/container/ /usr/share/nginx/html/
ENTRYPOINT [ "start-nginx.sh" ]