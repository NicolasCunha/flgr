# Production image, per docs/architecture/adr/0002-single-docker-image-with-nginx.md:
# the flgr-server binary and the flgr-web-client static build, served by
# Nginx, in a single image. Not used for local development — see
# docs/architecture/adr/0012-local-development-environment.md for that.

# ---- Backend build ----
FROM golang:1.26-bookworm AS backend-build
WORKDIR /src/flgr-server
COPY flgr-server/go.mod flgr-server/go.sum ./
RUN go mod download
COPY flgr-server/ ./
RUN CGO_ENABLED=0 go build -o /out/flgr-server ./cmd/server

# ---- Frontend build ----
FROM node:24-bookworm AS frontend-build
WORKDIR /src/flgr-web-client
COPY flgr-web-client/package.json flgr-web-client/package-lock.json ./
RUN npm ci
COPY flgr-web-client/ ./
RUN npm run build

# ---- Final image ----
FROM nginx:1.27-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends supervisor ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=backend-build /out/flgr-server /usr/local/bin/flgr-server
COPY --from=frontend-build /src/flgr-web-client/dist /usr/share/nginx/html
COPY flgr-server/migrations /app/migrations
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY deploy/supervisord.conf /etc/supervisor/conf.d/supervisord.conf

WORKDIR /app
EXPOSE 80

CMD ["supervisord", "-n", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
