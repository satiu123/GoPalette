FROM golang:1.25 AS builder

ARG APP_RELATIVE_PATH

WORKDIR /src

COPY go.work go.work.sum ./
COPY api/go.mod api/go.sum ./api/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY app/bff/service/go.mod app/bff/service/go.sum ./app/bff/service/
COPY app/comment/service/go.mod app/comment/service/go.sum ./app/comment/service/
COPY app/gateway/service/go.mod app/gateway/service/go.sum ./app/gateway/service/
COPY app/post/service/go.mod app/post/service/go.sum ./app/post/service/
COPY app/search/service/go.mod app/search/service/go.sum ./app/search/service/
COPY app/user/service/go.mod app/user/service/go.sum ./app/user/service/

RUN --mount=type=cache,id=gopalette-go-mod,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,id=gopalette-go-build,target=/root/.cache/go-build,sharing=locked \
    GOPROXY=https://goproxy.cn go mod download

COPY . .

#CGO_ENABLED=0 静态编译
RUN --mount=type=cache,id=gopalette-go-mod,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,id=gopalette-go-build,target=/root/.cache/go-build,sharing=locked \
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
