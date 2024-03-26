FROM docker.io/nginx:stable-alpine
COPY apps/angular/container/container/default.conf /etc/nginx/conf.d
COPY dist/apps/angular/container/container/ /usr/share/nginx/html/
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
