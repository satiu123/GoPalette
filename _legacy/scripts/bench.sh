#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080/api}"
CONCURRENCY="${CONCURRENCY:-20}"
REQUESTS="${REQUESTS:-400}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl 未安装，无法执行压测脚本"
  exit 1
fi

echo "压测开始"
echo "BASE_URL=${BASE_URL} CONCURRENCY=${CONCURRENCY} REQUESTS=${REQUESTS}"

echo "[1/3] 预热缓存（首页列表 + 热门搜索）"
curl -s "${BASE_URL}/articles?page=1&page_size=12" >/dev/null || true
curl -s "${BASE_URL}/search?q=go&page=1&page_size=12" >/dev/null || true

echo "[2/3] 列表接口压测"
if command -v hey >/dev/null 2>&1; then
  hey -n "${REQUESTS}" -c "${CONCURRENCY}" "${BASE_URL}/articles?page=1&page_size=12"
else
  echo "未检测到 hey，使用 curl 轻量循环代替（无详细百分位数据）"
  start=$(date +%s)
  for _ in $(seq 1 "${REQUESTS}"); do
    curl -s "${BASE_URL}/articles?page=1&page_size=12" >/dev/null
  done
  end=$(date +%s)
  echo "完成 ${REQUESTS} 次请求，耗时 $((end-start)) 秒"
fi

echo "[3/3] 搜索接口压测"
if command -v hey >/dev/null 2>&1; then
  hey -n "${REQUESTS}" -c "${CONCURRENCY}" "${BASE_URL}/search?q=go&page=1&page_size=12"
else
  start=$(date +%s)
  for _ in $(seq 1 "${REQUESTS}"); do
    curl -s "${BASE_URL}/search?q=go&page=1&page_size=12" >/dev/null
  done
  end=$(date +%s)
  echo "完成 ${REQUESTS} 次请求，耗时 $((end-start)) 秒"
fi

echo "压测结束。"
