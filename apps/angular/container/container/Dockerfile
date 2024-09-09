FROM docker.io/node:lts-alpine as builder
ENV JQ_VERSION=1.7
RUN wget --no-check-certificate https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64 -O /tmp/jq-linux64
RUN cp /tmp/jq-linux64 /usr/bin/jq
RUN chmod +x /usr/bin/jq
WORKDIR /app
COPY . .
RUN jq 'to_entries | map_values({ (.key) : ("$" + .key) }) | reduce .[] as $item ({}; . + $item)' apps/angular/container/container/src/config.json > apps/angular/container/container/src/config.tmp.json && mv apps/angular/container/container/src/config.tmp.json apps/angular/container/container/src/config.json
RUN npm install && npx nx build container

FROM docker.io/nginx:1.27 as run
ENV JSFOLDER=/usr/share/nginx/html/*.js
COPY apps/angular/container/container/start-nginx.sh /usr/bin/start-nginx.sh
RUN chmod +x /usr/bin/start-nginx.sh
COPY apps/angular/container/container/default.conf /etc/nginx/conf.d
COPY --from=builder /app/dist/apps/angular/container/container/ /usr/share/nginx/html/
ENTRYPOINT [ "start-nginx.sh" ]