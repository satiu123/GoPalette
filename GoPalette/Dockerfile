FROM golang:1.25 AS builder

# 定义构建参数，用来指定我们要编译哪个服务
# 例如：app/user/service
ARG APP_RELATIVE_PATH
ARG HTTP_PORT
ARG GRPC_PORT

WORKDIR /src

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOPROXY=https://goproxy.cn go build \
    -ldflags "-s -w" \
    -o /src/bin/server \
    ./${APP_RELATIVE_PATH}/cmd/server/...

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates  \
    netbase \
    && rm -rf /var/lib/apt/lists/ \
    && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/bin/server /app/server

WORKDIR /app

EXPOSE ${HTTP_PORT}
EXPOSE ${GRPC_PORT}
VOLUME /data/conf

# 启动命令
CMD ["./server", "-conf", "/data/conf"]