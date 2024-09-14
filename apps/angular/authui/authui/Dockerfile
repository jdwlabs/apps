FROM docker.io/node:lts-alpine AS builder
ENV JQ_VERSION=1.7
RUN case "$(uname -m)" in \
        x86_64) wget --no-check-certificate https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64 -O /usr/bin/jq ;; \
        aarch64) wget --no-check-certificate https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64-arm -O /usr/bin/jq ;; \
        *) echo "Unsupported architecture: $(uname -m)" && exit 1 ;; \
    esac
RUN chmod +x /usr/bin/jq
WORKDIR /app
COPY . .
RUN jq 'to_entries | map_values({ (.key) : ("$" + .key) }) | reduce .[] as $item ({}; . + $item)' apps/angular/authui/authui/src/config.json > apps/angular/authui/authui/src/config.tmp.json && mv apps/angular/authui/authui/src/config.tmp.json apps/angular/authui/authui/src/config.json
RUN npm install && npx nx build authui

FROM docker.io/nginx:1.27 AS run
ENV JSFOLDER=/usr/share/nginx/html/*.js
COPY apps/angular/authui/authui/start-nginx.sh /usr/bin/start-nginx.sh
RUN chmod +x /usr/bin/start-nginx.sh
COPY apps/angular/authui/authui/default.conf /etc/nginx/conf.d
COPY --from=builder /app/dist/apps/angular/authui/authui/ /usr/share/nginx/html/
ENTRYPOINT [ "start-nginx.sh" ]