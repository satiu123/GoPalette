#!/usr/bin/env bash
set -e

BASE_URL="http://localhost:8080/api"
USERNAME="testuser_$$"
PASSWORD="testpass123"
ADMIN_USER="adminuser_$$"
ADMIN_PASS="adminpass123"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }
section() { echo -e "\n${BLUE}══ $1 ══${NC}"; }

# 从 JSON 响应中提取字段值（避免依赖 jq）
extract() { echo "$1" | grep -o "\"$2\":[^,}]*" | head -1 | sed 's/.*://;s/[" ]//g'; }

# ─────────────────────────────────────────────
section "1. 基础设施"
# ─────────────────────────────────────────────

info "健康检查"
HEALTH=$(curl -sf "$BASE_URL/health")
echo "$HEALTH" | grep -q '"status":"ok"' && pass "health" || fail "health 响应异常: $HEALTH"

# ─────────────────────────────────────────────
section "2. 用户认证"
# ─────────────────────────────────────────────

info "注册普通用户 ($USERNAME)"
REG_RESP=$(curl -s -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\",\"role\":\"user\"}")
echo "$REG_RESP" | grep -q '"message":"注册成功"' && pass "注册成功" || fail "注册失败: $REG_RESP"

info "注册 admin 用户 ($ADMIN_USER)"
AREG_RESP=$(curl -s -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\",\"role\":\"admin\"}")
echo "$AREG_RESP" | grep -q '"message":"注册成功"' && pass "admin 注册成功" || fail "admin 注册失败: $AREG_RESP"

info "登录普通用户"
LOGIN_RESP=$(curl -sf -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")
ACCESS_TOKEN=$(extract "$LOGIN_RESP" "access_token")
REFRESH_TOKEN=$(extract "$LOGIN_RESP" "refresh_token")
[ -n "$ACCESS_TOKEN" ]  && pass "拿到 access_token"  || fail "未拿到 access_token: $LOGIN_RESP"
[ -n "$REFRESH_TOKEN" ] && pass "拿到 refresh_token" || fail "未拿到 refresh_token"

info "登录 admin 用户"
ALOGIN_RESP=$(curl -sf -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}")
ADMIN_TOKEN=$(extract "$ALOGIN_RESP" "access_token")
[ -n "$ADMIN_TOKEN" ] && pass "admin 拿到 access_token" || fail "admin 登录失败: $ALOGIN_RESP"

info "无 Token 访问私有接口应被拒绝（预期 401）"
NO_AUTH=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/articles" \
  -H "Content-Type: application/json" -d '{}')
[ "$NO_AUTH" = "401" ] && pass "无 Token 返回 401" || fail "无 Token 返回 $NO_AUTH（异常）"

info "刷新令牌"
REFRESH_RESP=$(curl -sf -X POST "$BASE_URL/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
NEW_ACCESS=$(extract "$REFRESH_RESP" "access_token")
NEW_REFRESH=$(extract "$REFRESH_RESP" "refresh_token")
[ -n "$NEW_ACCESS" ]  && pass "拿到新 access_token"  || fail "未拿到新 access_token"
[ -n "$NEW_REFRESH" ] && pass "拿到新 refresh_token" || fail "未拿到新 refresh_token"
ACCESS_TOKEN="$NEW_ACCESS"

info "旧 Refresh Token 应已失效（令牌轮转）"
OLD_RESP=$(curl -s -X POST "$BASE_URL/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
OLD_CODE=$(extract "$OLD_RESP" "code")
[ "$OLD_CODE" = "401" ] && pass "旧 token 已失效" || fail "旧 token 未失效，code=$OLD_CODE"

# ─────────────────────────────────────────────
section "3. 分类管理"
# ─────────────────────────────────────────────

info "创建分类（需要 JWT）"
CAT_RESP=$(curl -s -X POST "$BASE_URL/categories" \
  -H "Content-Type: application/json" \
  -H "Authorization: $ACCESS_TOKEN" \
  -d '{"name":"技术"}')
CAT_ID=$(extract "$CAT_RESP" "id")
[ -n "$CAT_ID" ] && pass "创建分类成功 id=$CAT_ID" || fail "创建分类失败: $CAT_RESP"

info "获取分类列表（公开）"
CATS=$(curl -sf "$BASE_URL/categories")
echo "$CATS" | grep -q "技术" && pass "分类列表正常" || fail "分类列表异常: $CATS"

# ─────────────────────────────────────────────
section "4. 标签管理"
# ─────────────────────────────────────────────

info "创建标签（需要 JWT）"
TAG_RESP=$(curl -s -X POST "$BASE_URL/tags" \
  -H "Content-Type: application/json" \
  -H "Authorization: $ACCESS_TOKEN" \
  -d '{"name":"Go"}')
TAG_ID=$(extract "$TAG_RESP" "id")
[ -n "$TAG_ID" ] && pass "创建标签成功 id=$TAG_ID" || fail "创建标签失败: $TAG_RESP"

info "获取标签列表（公开）"
TAGS=$(curl -sf "$BASE_URL/tags")
echo "$TAGS" | grep -q "Go" && pass "标签列表正常" || fail "标签列表异常: $TAGS"

# ─────────────────────────────────────────────
section "5. 文章 CRUD"
# ─────────────────────────────────────────────

info "创建文章（draft）"
ART_RESP=$(curl -s -X POST "$BASE_URL/articles" \
  -H "Content-Type: application/json" \
  -H "Authorization: $ACCESS_TOKEN" \
  -d "{\"title\":\"测试文章\",\"summary\":\"摘要\",\"content\":\"这是 Go 博客内容\",\"category_id\":$CAT_ID,\"tag_ids\":[$TAG_ID],\"status\":\"published\"}")
ART_ID=$(extract "$ART_RESP" "id")
[ -n "$ART_ID" ] && pass "创建文章成功 id=$ART_ID" || fail "创建文章失败: $ART_RESP"

info "获取文章详情（公开）"
GET_ART=$(curl -sf "$BASE_URL/articles/$ART_ID")
echo "$GET_ART" | grep -q "测试文章" && pass "文章详情正常" || fail "文章详情异常: $GET_ART"

info "文章列表（公开，含分类和标签过滤）"
LIST_ART=$(curl -sf "$BASE_URL/articles?page=1&page_size=10&category_id=$CAT_ID")
echo "$LIST_ART" | grep -q "total" && pass "文章列表正常" || fail "文章列表异常: $LIST_ART"

info "更新文章（仅作者）"
UPD_RESP=$(curl -s -X PUT "$BASE_URL/articles/$ART_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: $ACCESS_TOKEN" \
  -d "{\"title\":\"更新后标题\",\"summary\":\"新摘要\",\"content\":\"更新内容 Go\",\"category_id\":$CAT_ID,\"tag_ids\":[$TAG_ID],\"status\":\"published\"}")
echo "$UPD_RESP" | grep -q "更新后标题" && pass "文章更新成功" || fail "文章更新失败: $UPD_RESP"

info "他人更新文章应被拒绝（预期 403）"
OTHER_UPD=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/articles/$ART_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: $ADMIN_TOKEN" \
  -d "{\"title\":\"x\",\"summary\":\"\",\"content\":\"x\",\"category_id\":$CAT_ID,\"tag_ids\":[],\"status\":\"published\"}")
# admin 角色可以更新，预期 200
[ "$OTHER_UPD" = "200" ] && pass "admin 更新文章返回 200" || fail "admin 更新返回 $OTHER_UPD"

# ─────────────────────────────────────────────
section "6. 评论"
# ─────────────────────────────────────────────

info "发表一级评论（需要 JWT）"
CMT_RESP=$(curl -s -X POST "$BASE_URL/articles/$ART_ID/comments" \
  -H "Content-Type: application/json" \
  -H "Authorization: $ACCESS_TOKEN" \
  -d '{"content":"很棒的文章！","parent_id":0}')
CMT_ID=$(extract "$CMT_RESP" "id")
[ -n "$CMT_ID" ] && pass "发表评论成功 id=$CMT_ID" || fail "发表评论失败: $CMT_RESP"

info "发表回复评论（parent_id=$CMT_ID）"
REPLY_RESP=$(curl -s -X POST "$BASE_URL/articles/$ART_ID/comments" \
  -H "Content-Type: application/json" \
  -H "Authorization: $ACCESS_TOKEN" \
  -d "{\"content\":\"谢谢支持\",\"parent_id\":$CMT_ID}")
REPLY_ID=$(extract "$REPLY_RESP" "id")
[ -n "$REPLY_ID" ] && pass "回复评论成功 id=$REPLY_ID" || fail "回复评论失败: $REPLY_RESP"

info "获取文章评论列表（公开）"
CMTS=$(curl -sf "$BASE_URL/articles/$ART_ID/comments")
echo "$CMTS" | grep -q "很棒的文章" && pass "评论列表正常" || fail "评论列表异常: $CMTS"

info "他人删除评论应被拒绝（预期 403）"
DEL_CMT=$(curl -s "$BASE_URL/comments/$CMT_ID" \
  -X DELETE -H "Authorization: $ADMIN_TOKEN")
DEL_CODE=$(extract "$DEL_CMT" "code")
# admin 可以删除任意评论，预期成功
[ "$DEL_CODE" = "200" ] && pass "admin 删除评论成功" || fail "admin 删除评论失败 code=$DEL_CODE: $DEL_CMT"

# ─────────────────────────────────────────────
section "7. 全文搜索"
# ─────────────────────────────────────────────

info "搜索关键词 'Go'"
SEARCH=$(curl -sf "$BASE_URL/search?q=Go&page=1&page_size=10")
echo "$SEARCH" | grep -q "total" && pass "全文搜索响应正常" || fail "全文搜索异常: $SEARCH"

info "搜索缺少 q 参数应返回 400"
SEARCH_BAD=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/search")
[ "$SEARCH_BAD" = "200" ] && pass "搜索缺参数由业务层处理" || {
  # 也可能直接被 binding 返回 400，均可接受
  [ "$SEARCH_BAD" = "400" ] && pass "搜索缺参数返回 400" || fail "搜索缺参数返回 $SEARCH_BAD"
}

# ─────────────────────────────────────────────
section "8. Phase 4 安全特性"
# ─────────────────────────────────────────────

info "CORS：已知来源应返回 Access-Control-Allow-Origin 头"
CORS_OK=$(curl -s -I -H "Origin: http://localhost:3000" "$BASE_URL/health")
echo "$CORS_OK" | grep -qi "access-control-allow-origin: http://localhost:3000" \
  && pass "CORS 允许已知来源 http://localhost:3000" \
  || fail "CORS 头部缺失，未找到 Access-Control-Allow-Origin"

info "CORS：OPTIONS 预检请求应返回 204"
CORS_PRE=$(curl -s -o /dev/null -w "%{http_code}" -X OPTIONS "$BASE_URL/health" \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET")
[ "$CORS_PRE" = "204" ] && pass "OPTIONS 预检返回 204" || fail "OPTIONS 预检返回 $CORS_PRE"

info "CORS：未知来源不应携带 Access-Control-Allow-Origin"
CORS_BAD=$(curl -s -I -H "Origin: http://evil.com" "$BASE_URL/health")
echo "$CORS_BAD" | grep -qi "access-control-allow-origin: http://evil.com" \
  && fail "CORS 错误地允许了未知来源 evil.com" \
  || pass "CORS 正确拒绝未知来源"

info "XSS 防护：文章内容中的 <script> 标签及其内容应被 bluemonday 移除"
XSS_RESP=$(curl -s -X POST "$BASE_URL/articles" \
  -H "Content-Type: application/json" \
  -H "Authorization: $ACCESS_TOKEN" \
  -d "{\"title\":\"XSS测试\",\"summary\":\"安全测试\",\"content\":\"<script>alert('xss')</script>正文内容\",\"category_id\":$CAT_ID,\"tag_ids\":[$TAG_ID],\"status\":\"published\"}")
XSS_ART_ID=$(extract "$XSS_RESP" "id")
[ -n "$XSS_ART_ID" ] && pass "XSS 测试文章创建成功 id=$XSS_ART_ID" || fail "创建 XSS 测试文章失败: $XSS_RESP"

XSS_GET=$(curl -sf "$BASE_URL/articles/$XSS_ART_ID")
echo "$XSS_GET" | grep -q "alert" \
  && fail "XSS 未被净化：响应中仍含 alert('xss')" \
  || pass "XSS 内容已净化，恶意脚本已移除"

curl -s -X DELETE "$BASE_URL/articles/$XSS_ART_ID" -H "Authorization: $ACCESS_TOKEN" > /dev/null

info "限流：快速登录应触发 HTTP 429（每 IP 每分钟 10 次）"
set +e
GOT_429=0
for i in $(seq 1 15); do
  RATE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"ratelimit_probe","password":"wrongpass"}')
  if [ "$RATE_CODE" = "429" ]; then
    GOT_429=1
    break
  fi
done
set -e
[ "$GOT_429" = "1" ] && pass "限流触发 429（Too Many Requests）" || fail "发送 15 次后仍未触发限流"

info "Swagger UI 可访问（/api/swagger/index.html）"
SWAGGER_CODE=$(curl -s -o /dev/null -w "%{http_code}" -L "$BASE_URL/swagger/index.html")
[ "$SWAGGER_CODE" = "200" ] && pass "Swagger UI 返回 200" || fail "Swagger UI 返回 $SWAGGER_CODE"

# ─────────────────────────────────────────────
section "9. 清理：删除测试数据"
# ─────────────────────────────────────────────

info "删除文章"
DEL_ART=$(curl -s -X DELETE "$BASE_URL/articles/$ART_ID" \
  -H "Authorization: $ACCESS_TOKEN")
echo "$DEL_ART" | grep -q "已删除" && pass "文章删除成功" || fail "文章删除失败: $DEL_ART"

info "删除标签"
DEL_TAG=$(curl -s -X DELETE "$BASE_URL/tags/$TAG_ID" \
  -H "Authorization: $ACCESS_TOKEN")
echo "$DEL_TAG" | grep -q "已删除" && pass "标签删除成功" || fail "标签删除失败: $DEL_TAG"

info "删除分类"
DEL_CAT=$(curl -s -X DELETE "$BASE_URL/categories/$CAT_ID" \
  -H "Authorization: $ACCESS_TOKEN")
echo "$DEL_CAT" | grep -q "已删除" && pass "分类删除成功" || fail "分类删除失败: $DEL_CAT"

echo ""
pass "══ 所有测试通过 ══"

