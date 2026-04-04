#!/usr/bin/env bash
# scripts/import_articles.sh
set -e

BASE_URL="http://localhost:8080/api"
ARTICLES_FILE="data/articles.json"
ADMIN_USER="admin_import"
ADMIN_PASS="admin123456"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${YELLOW}[INFO]${NC} $1"; }
pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }

if [ ! -f "$ARTICLES_FILE" ]; then
    fail "找不到文件: $ARTICLES_FILE"
fi

# 1. 注册/登录管理员
info "登录管理员账户..."
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}")

ACCESS_TOKEN=$(echo "$LOGIN_RESP" | jq -r '.data.access_token // empty')

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" == "null" ]; then
    info "登录失败，尝试注册管理员账户..."
    REG_RESP=$(curl -s -X POST "$BASE_URL/register" \
      -H "Content-Type: application/json" \
      -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\",\"role\":\"admin\"}")
    
    if echo "$REG_RESP" | grep -q "注册成功"; then
        pass "注册成功"
        LOGIN_RESP=$(curl -s -X POST "$BASE_URL/login" \
          -H "Content-Type: application/json" \
          -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}")
        ACCESS_TOKEN=$(echo "$LOGIN_RESP" | jq -r '.data.access_token')
    else
        fail "无法登录或注册管理员: $REG_RESP"
    fi
fi
pass "登录成功"

# 2. 获取/创建分类 "技术栈"
info "准备分类 '技术栈'..."
CATS=$(curl -s "$BASE_URL/categories")
CAT_ID=$(echo "$CATS" | jq -r '.data[] | select(.name == "技术栈") | .id' | head -n 1)

if [ -z "$CAT_ID" ] || [ "$CAT_ID" == "null" ]; then
    CAT_RESP=$(curl -s -X POST "$BASE_URL/categories" \
      -H "Content-Type: application/json" \
      -H "Authorization: $ACCESS_TOKEN" \
      -d '{"name":"技术栈"}')
    CAT_ID=$(echo "$CAT_RESP" | jq -r '.data.id')
    pass "创建分类 '技术栈' ID: $CAT_ID"
else
    pass "使用现有分类 '技术栈' ID: $CAT_ID"
fi

# 3. 导入文章
count=$(jq '. | length' "$ARTICLES_FILE")
info "开始导入 $count 篇文章..."

for i in $(seq 0 $((count - 1))); do
    article_data=$(jq ".[$i]" "$ARTICLES_FILE")
    title=$(echo "$article_data" | jq -r ".title")
    tags=$(echo "$article_data" | jq -c ".tags")
    status=$(echo "$article_data" | jq -r ".status")

    info "正在处理: $title"

    # 处理标签
    tag_ids="[]"
    while IFS= read -r tag_name; do
        [ -z "$tag_name" ] && continue
        # 获取现有标签
        ALL_TAGS=$(curl -s "$BASE_URL/tags")
        tid=$(echo "$ALL_TAGS" | jq -r ".data[] | select(.name == \"$tag_name\") | .id" | head -n 1)
        
        if [ -z "$tid" ] || [ "$tid" == "null" ]; then
            # 创建新标签
            TAG_RESP=$(curl -s -X POST "$BASE_URL/tags" \
              -H "Content-Type: application/json" \
              -H "Authorization: $ACCESS_TOKEN" \
              -d "{\"name\":\"$tag_name\"}")
            tid=$(echo "$TAG_RESP" | jq -r '.data.id')
            info "  创建标签 '$tag_name' ID: $tid"
        else
            info "  使用标签 '$tag_name' ID: $tid"
        fi
        tag_ids=$(echo "$tag_ids" | jq -c ". += [$tid]")
    done < <(echo "$tags" | jq -r '.[]')

    # 准备发布文章的 JSON
    PAYLOAD=$(echo "$article_data" | jq -c --argjson cid "$CAT_ID" --argjson tids "$tag_ids" "{
        title: .title,
        summary: .summary,
        content: .content,
        category_id: \$cid,
        tag_ids: \$tids,
        status: .status
    }")

    # 发布文章
    ART_RESP=$(curl -s -X POST "$BASE_URL/articles" \
      -H "Content-Type: application/json" \
      -H "Authorization: $ACCESS_TOKEN" \
      -d "$PAYLOAD")
    
    ART_ID=$(echo "$ART_RESP" | jq -r '.data.id // empty')
    if [ -n "$ART_ID" ] && [ "$ART_ID" != "null" ]; then
        pass "  文章导入成功 ID: $ART_ID"
    else
        echo "Payload: $PAYLOAD"
        fail "  文章导入失败: $ART_RESP"
    fi
done

pass "所有文章导入完成！"
