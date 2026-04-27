FROM golang:1.25 AS builder

ARG APP_RELATIVE_PATH

WORKDIR /src
COPY . .

#CGO_ENABLED=0 静态编译
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOPROXY=https://goproxy.cn go build \
    -ldflags "-s -w -extldflags '-static'" \
    -o /src/bin/server \
    ./${APP_RELATIVE_PATH}/cmd/server/...

# 使用 alpine 作为最终运行环境
FROM alpine:latest

ARG HTTP_PORT
ARG GRPC_PORT

# 安装 ca-certificates 和 tzdata（时区数据），并清理缓存
RUN sed -i 's|https://dl-cdn.alpinelinux.org/alpine|https://mirrors.tuna.tsinghua.edu.cn/alpine|g' /etc/apk/repositories \
    && apk --no-cache add ca-certificates tzdata

COPY --from=builder /src/bin/server /app/server

WORKDIR /app

EXPOSE ${HTTP_PORT}
EXPOSE ${GRPC_PORT}
VOLUME /data/conf

CMD ["./server", "-conf", "/data/conf"]
