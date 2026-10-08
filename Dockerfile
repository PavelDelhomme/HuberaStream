FROM nginx:1.27-alpine
RUN apk add --no-cache wget
COPY web/nginx.conf /etc/nginx/conf.d/default.conf
COPY web/ /usr/share/nginx/html/
HEALTHCHECK --interval=20s --timeout=3s --retries=3 \
  CMD wget -qO- http://127.0.0.1/health >/dev/null || exit 1
