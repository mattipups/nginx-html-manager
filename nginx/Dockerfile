# syntax=docker/dockerfile:1
ARG NGINX_IMAGE=cgr.dev/chainguard/nginx:latest
FROM ${NGINX_IMAGE}
COPY nginx/nginx.conf /etc/nginx/nginx.conf
USER 65532:65532
EXPOSE 8080 8081
ENTRYPOINT ["/usr/sbin/nginx"]
CMD ["-g", "daemon off;"]
