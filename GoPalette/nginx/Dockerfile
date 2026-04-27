# syntax=docker/dockerfile:1.7

FROM oven/bun:1.2.15-alpine AS frontend-builder
WORKDIR /app/frontend

ENV NUXT_TELEMETRY_DISABLED=1

COPY frontend/package.json frontend/bun.lock* ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile

COPY frontend/ ./
RUN --mount=type=cache,target=/app/frontend/node_modules/.cache \
    bun x nuxi build

FROM node:22-alpine AS frontend-runtime
WORKDIR /app/frontend

ENV HOST=0.0.0.0
ENV PORT=3000
ENV NODE_ENV=production

COPY --from=frontend-builder /app/frontend/.output /app/frontend/.output

EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]

FROM nginx:1.29-alpine AS nginx
COPY GoPalette/nginx/default.conf /etc/nginx/conf.d/default.conf
