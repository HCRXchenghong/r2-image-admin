# ---------- 前端构建 ----------
FROM node:22-alpine AS web
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---------- 后端构建（含 libvips） ----------
FROM golang:1.25.13-alpine AS api
RUN apk add --no-cache build-base pkgconf vips-dev
WORKDIR /app/backend
COPY backend/ ./
RUN go mod download
COPY --from=web /app/backend/web/dist ./web/dist
RUN CGO_ENABLED=1 go build -tags vips -o /out/r2-image-admin ./cmd/server

# ---------- 运行 ----------
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata vips \
    && addgroup -S r2admin \
    && adduser -S -D -H -G r2admin r2admin \
    && mkdir -p /app \
    && chown -R r2admin:r2admin /app
WORKDIR /app
COPY --chown=r2admin:r2admin --from=api /out/r2-image-admin /usr/local/bin/r2-image-admin
USER r2admin:r2admin
EXPOSE 8080
ENTRYPOINT ["r2-image-admin"]
