# syntax=docker/dockerfile:1.7

FROM node:26-alpine AS frontend-build
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend-build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/seed ./cmd/seed

FROM alpine:3.23 AS application
RUN apk add --no-cache postgresql18 postgresql18-contrib \
    && addgroup -S calendar \
    && adduser -S -G calendar calendar
WORKDIR /app
COPY --from=backend-build /out/ ./
COPY --from=frontend-build /src/frontend/dist/ ./public/
COPY --chmod=755 scripts/container-entrypoint.sh ./entrypoint
USER calendar
ENV PATH="/usr/libexec/postgresql18:${PATH}"
ENV PORT=8080
ENV STATIC_DIR=/app/public
EXPOSE 8080
HEALTHCHECK --interval=2s --timeout=3s --retries=15 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${PORT}/api/health" || exit 1
ENTRYPOINT ["/app/entrypoint"]
