# GoPalette k6 压力测试

## 隔离压测环境

为避免污染本地或生产样例数据，压测推荐使用仓库里的独立 Compose 文件：

```bash
cp .env.k6.example .env.k6
docker compose --env-file .env.k6 -p gopalette-k6 -f compose.k6.yaml up -d --build
```

这套环境使用独立数据卷和端口：

| 组件 | 压测端口 |
| --- | --- |
| Gateway | `18080` |
| Frontend | `13001` |
| Grafana | `13000` |
| Prometheus | `19091` |
| MariaDB | `13306` |
| Redis | `16379` |
| Meilisearch | `17700` |

启动后 `seed-k6-data` 会自动等待 `post` 服务完成表迁移，并预置一个 `K6 Load Test` 分类。写文章和写评论脚本默认会读取这个分类 ID；如果你想用别的分类，可以在运行 k6 时显式设置 `CATEGORY_ID`。
压测环境默认把 MariaDB `max_connections` 设为 `600`，并把每个 Go 服务的 MySQL 连接池限制为 `MYSQL_MAX_OPEN_CONNS=80`，避免阶梯加压时服务无限开连接把数据库提前打爆。

压测脚本请指向隔离网关：

```bash
-e GATEWAY_BASE=http://localhost:18080
```

如果需要把 k6 指标写入隔离 Prometheus：

```bash
K6_PROMETHEUS_RW_SERVER_URL=http://localhost:19091/api/v1/write \
K6_PROMETHEUS_RW_TREND_AS_NATIVE_HISTOGRAM=true \
k6 run -o experimental-prometheus-rw k6/post-write.js \
  -e GATEWAY_BASE=http://localhost:18080
```

如果需要导出HTML报告

```bash
env K6_WEB_DASHBOARD=true \
    K6_WEB_DASHBOARD_EXPORT=result/post-write-report.html \
    K6_PROMETHEUS_RW_SERVER_URL=http://localhost:19091/api/v1/write \
    K6_PROMETHEUS_RW_TREND_AS_NATIVE_HISTOGRAM=true \
    k6 run -o experimental-prometheus-rw \
      --summary-export result/post-write-summary.json \
      --tag testid=post-write-800-20260519 \
      k6/post-write.js
```

停止压测环境：

```bash
docker compose --env-file .env.k6 -p gopalette-k6 -f compose.k6.yaml down
```

连同压测数据一起删除：

```bash
docker compose --env-file .env.k6 -p gopalette-k6 -f compose.k6.yaml down -v
```

## 高并发写文章

`post-write.js` 只压 `POST /v1/posts`。脚本默认会在 `setup()` 中自动注册并登录压测账号，拿到 token 后再执行文章写入。
默认 `POST_STATUS=0`，也就是创建草稿，用来观察文章数据库写入本身；如果要同时压搜索索引链路，可设置 `POST_STATUS=1`。

```bash
k6 run k6/post-write.js \
  -e GATEWAY_BASE=http://localhost:18080 \
  -e ACCOUNT_COUNT=50 \
  -e RAMP_STAGES='30s:100,1m:200,1m:400,1m:800,30s:0' \
  -e PRE_ALLOCATED_VUS=200 \
  -e MAX_VUS=1000
```

## 固定 RPS 写文章

`post-write-fixed.js` 适合逐档基准测试。每次只测一个固定 RPS，测完清空压测数据库后再跑下一档。

```bash
K6_PROMETHEUS_RW_SERVER_URL=http://localhost:19091/api/v1/write \
K6_PROMETHEUS_RW_TREND_AS_NATIVE_HISTOGRAM=true \
k6 run -o experimental-prometheus-rw k6/post-write-fixed.js \
  -e GATEWAY_BASE=http://localhost:18080 \
  -e RATE=500 \
  -e DURATION=3m \
  -e ACCOUNT_COUNT=100 \
  -e PRE_ALLOCATED_VUS=300 \
  -e MAX_VUS=1500
```

五个档位只需要替换 `RATE`：

```bash
for rps in 500 800 1000 1500 2000; do
  K6_PROMETHEUS_RW_SERVER_URL=http://localhost:19091/api/v1/write \
  K6_PROMETHEUS_RW_TREND_AS_NATIVE_HISTOGRAM=true \
  k6 run -o experimental-prometheus-rw k6/post-write-fixed.js \
    -e GATEWAY_BASE=http://localhost:18080 \
    -e RATE=$rps \
    -e DURATION=3m \
    -e ACCOUNT_COUNT=100 \
    -e PRE_ALLOCATED_VUS=500 \
    -e MAX_VUS=2500

done
```

## 高并发写评论

`comment-write.js` 只压 `POST /v1/comments`。脚本同样会自动注册并登录压测账号；`POST_IDS` 未设置时，会先从文章列表取文章 ID，若没有可用文章则自动创建一篇发布状态的种子文章。

```bash
k6 run k6/comment-write.js \
  -e GATEWAY_BASE=http://localhost:18080 \
  -e ACCOUNT_COUNT=500 \
  -e POST_IDS='1,2,3' \
  -e RAMP_STAGES='30s:200,1m:500,1m:800,1m:1200,30s:0' \
  -e PRE_ALLOCATED_VUS=500 \
  -e MAX_VUS=2000
```

评论服务目前按用户限制每分钟 3 次创建。要压评论“成功写入吞吐”，需要把 `ACCOUNT_COUNT` 调大，或直接传入多个 `AUTH_TOKENS`；只用少量账号时，大部分请求会返回 `429`，更适合验证限流保护。

## 账号与认证参数

两个写入脚本都支持自动注册账号，也支持复用已有 token：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `AUTH_TOKENS` | 空 | 逗号分隔的用户 access token，设置后跳过自动注册 |
| `ACCOUNT_COUNT` | 文章 `20`，评论 `100` | 自动注册账号数量 |
| `ACCOUNT_PREFIX` | `k6-${Date.now()}` | 自动注册账号前缀；复用同一前缀时可重复登录已存在账号 |
| `ACCOUNT_PASSWORD` | `K6LoadTest123!` | 自动注册账号密码 |
| `ACCOUNT_EMAIL_DOMAIN` | `k6.gopalette.local` | 自动注册邮箱域名 |
| `GATEWAY_BASE` | `http://localhost:8080` | 网关地址 |
| `CATEGORY_ID` | 空 | 指定文章分类；为空时读取 `CATEGORY_NAME` 对应分类 |
| `CATEGORY_NAME` | `K6 Load Test` | 自动查找的压测分类名称 |
| `RATE` | 文章 `50`，评论 `200` | 未设置 `RAMP_STAGES` 时的基础阶梯速率 |
| `RAMP_STAGES` | 内置阶梯 | 阶梯式到达率，格式为 `duration:target`，多个阶段用逗号分隔 |
| `PRE_ALLOCATED_VUS` | 文章 `150`，评论 `300` | 预分配 VU 数 |
| `MAX_VUS` | 文章 `800`，评论 `1500` | 最大 VU 数 |

数据库连接相关参数在 `.env.k6` 中调整：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `MYSQL_MAX_OPEN_CONNS` | `80` | 每个 Go 服务的 MySQL 最大打开连接数 |
| `MYSQL_MAX_IDLE_CONNS` | `40` | 每个 Go 服务的 MySQL 最大空闲连接数 |
| `MARIADB_MAX_CONNECTIONS` | `600` | 压测 MariaDB 最大连接数 |
| `MARIADB_INNODB_BUFFER_POOL_SIZE` | `512M` | 压测 MariaDB InnoDB buffer pool |

## 高并发混合写入：文章 + 评论

`high-concurrency-write.js` 会同时压两类写接口：

- `POST /v1/posts`：创建文章，使用唯一 `slug`，适合观察文章写入、标签关联、搜索异步投递对系统的影响。
- `POST /v1/comments`：创建评论，默认接受 `200` 和 `429`，适合同时观察正常写入吞吐与评论限流。

```bash
k6 run k6/high-concurrency-write.js \
  -e GATEWAY_BASE=http://localhost:8080 \
  -e AUTH_TOKENS='tokenA,tokenB,tokenC' \
  -e POST_IDS='1,2,3' \
  -e POST_RATE=80 \
  -e COMMENT_RATE=500 \
  -e DURATION=5m \
  -e PRE_ALLOCATED_VUS=500 \
  -e MAX_VUS=2000
```

常用参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `GATEWAY_BASE` | `http://localhost:8080` | 网关地址 |
| `AUTH_TOKENS` | 空 | 逗号分隔的用户 access token，支持带或不带 `Bearer` 前缀 |
| `LOGIN_EMAIL` / `LOGIN_PASSWORD` | 空 | 未提供 token 时使用单账号登录 |
| `POST_IDS` | 空 | 评论写入目标文章池；为空时脚本会拉取文章列表，仍为空则创建种子文章 |
| `POST_RATE` | `50` | 每秒创建文章请求数 |
| `COMMENT_RATE` | `200` | 每秒创建评论请求数 |
| `DURATION` | `3m` | 压测持续时间 |
| `POST_STATUS` | `0` | 创建文章状态，`0` 草稿，`1` 发布 |
| `CATEGORY_ID` | `0` | 创建文章/种子文章的分类 ID |
| `MIN_COMMENT_SUCCESS_RATIO` | `0.01` | 评论成功率阈值；单 token 压限流时可调低 |
| `MAX_UNEXPECTED_STATUS_RATIO` | `0.01` | 非预期状态码比例阈值 |

评论服务目前按用户限制每分钟 3 次创建。要压评论“成功写入吞吐”，请传入多个用户 token；只传一个 token 时，大部分请求会返回 `429`，这更适合验证限流保护是否稳定。

## 评论限流专项

`comment-write-abuse-rate-limit.js` 专门验证单用户高频评论是否触发限流：

```bash
k6 run k6/comment-write-abuse-rate-limit.js \
  -e GATEWAY_BASE=http://localhost:8080 \
  -e AUTH_TOKEN='token' \
  -e POST_ID=1 \
  -e RATE=500 \
  -e DURATION=3m \
  -e MIN_429_RATIO=0.6
```

## 搜索读压测

`search-posts-ramp.js` 会从 `keywords.csv` 读取关键词并压测 `/v1/search/posts`：

```bash
k6 run k6/search-posts-ramp.js \
  -e GATEWAY_BASE=http://localhost:8080
```
