#!/usr/bin/env bash
set -e

BASE_URL="http://localhost:8080/api"
USERNAME="testuser_$$"
PASSWORD="testpass123"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

# ── 1. 健康检查 ────────────────────────────────────────────────
info "1. 健康检查"
HEALTH=$(curl -sf "$BASE_URL/health")
echo "$HEALTH" | grep -q '"status":"ok"' && pass "health" || fail "health 响应异常: $HEALTH"

# ── 2. 注册 ───────────────────────────────────────────────────
info "2. 注册 (username: $USERNAME)"
REG_RESP=$(curl -s -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")
echo "   响应: $REG_RESP"
echo "$REG_RESP" | grep -q '"message":"注册成功"' && pass "注册成功" || fail "注册失败: $REG_RESP"

# ── 3. 登录，获取双令牌 ────────────────────────────────────────
info "3. 登录"
LOGIN_RESP=$(curl -sf -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")
echo "   响应: $LOGIN_RESP"

ACCESS_TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
REFRESH_TOKEN=$(echo "$LOGIN_RESP" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

[ -n "$ACCESS_TOKEN" ]  && pass "拿到 access_token"  || fail "未拿到 access_token"
[ -n "$REFRESH_TOKEN" ] && pass "拿到 refresh_token" || fail "未拿到 refresh_token"
info "   Access Token : ${ACCESS_TOKEN:0:40}..."
info "   Refresh Token: ${REFRESH_TOKEN:0:40}..."

# ── 4. 用 Access Token 访问受保护接口 ─────────────────────────
info "4. 使用 Access Token 访问受保护接口"
AUTH_RESP=$(curl -s -o /dev/null -w "%{http_code}" \
  "$BASE_URL/health" \
  -H "Authorization: Bearer $ACCESS_TOKEN")
[ "$AUTH_RESP" = "200" ] && pass "携带 Access Token 返回 200" || fail "返回码异常: $AUTH_RESP"

# ── 5. 无 Token 时应被拒绝（私有接口）─────────────────────────
info "5. 无 Token 访问私有接口应被拒绝（预期 401/404）"
NO_AUTH=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/protected_placeholder")
[ "$NO_AUTH" = "401" ] || [ "$NO_AUTH" = "404" ] \
  && pass "无 Token 返回 $NO_AUTH（符合预期）" \
  || fail "无 Token 返回 $NO_AUTH（异常）"

# ── 6. 用 Refresh Token 换新双令牌 ────────────────────────────
info "6. 刷新令牌"
REFRESH_RESP=$(curl -sf -X POST "$BASE_URL/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
echo "   响应: $REFRESH_RESP"

NEW_ACCESS=$(echo "$REFRESH_RESP" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
NEW_REFRESH=$(echo "$REFRESH_RESP" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

[ -n "$NEW_ACCESS" ]  && pass "拿到新 access_token"  || fail "未拿到新 access_token"
[ -n "$NEW_REFRESH" ] && pass "拿到新 refresh_token" || fail "未拿到新 refresh_token"
info "   新 Access Token : ${NEW_ACCESS:0:40}..."
info "   新 Refresh Token: ${NEW_REFRESH:0:40}..."

# ── 7. 旧 Refresh Token 应失效（令牌轮转）─────────────────────
info "7. 旧 Refresh Token 应已失效"
OLD_REFRESH_RESP=$(curl -s -X POST "$BASE_URL/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
OLD_CODE=$(echo "$OLD_REFRESH_RESP" | grep -o '"code":[0-9]*' | grep -o '[0-9]*')
[ "$OLD_CODE" = "401" ] \
  && pass "旧 Refresh Token 已失效（返回 code:401）" \
  || fail "旧 Refresh Token 返回 code:$OLD_CODE，轮转未生效"

# ── 8. 用新 Access Token 访问接口 ────────────────────────────
info "8. 用新 Access Token 访问接口"
NEW_AUTH_RESP=$(curl -s -o /dev/null -w "%{http_code}" \
  "$BASE_URL/health" \
  -H "Authorization: Bearer $NEW_ACCESS")
[ "$NEW_AUTH_RESP" = "200" ] && pass "新 Access Token 有效" || fail "新 Access Token 无效，返回 $NEW_AUTH_RESP"

echo ""
pass "所有测试通过"
