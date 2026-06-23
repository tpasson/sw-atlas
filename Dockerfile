# syntax=docker/dockerfile:1

# ---- Stage 1: build the Vue frontend ----
FROM node:22-alpine AS web
WORKDIR /web
COPY package.json package-lock.json ./
RUN npm ci
COPY index.html vite.config.js ./
COPY src ./src
RUN npm run build

# ---- Stage 2: build the Go backend (static binary) ----
FROM golang:1.25-alpine AS api
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /atlas ./cmd/atlas

# ---- Stage 3: minimal runtime (API + built SPA) ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 atlas
WORKDIR /app
COPY --from=api /atlas /usr/local/bin/atlas
COPY --from=web /web/dist /app/dist
ENV ATLAS_STATIC_DIR=/app/dist \
    ATLAS_LISTEN_ADDR=:8080
USER atlas
EXPOSE 8080
# Serves the API and the SPA; migrations run automatically on startup.
ENTRYPOINT ["atlas"]
CMD ["serve"]
