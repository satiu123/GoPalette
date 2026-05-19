import http from 'k6/http'
import { check, fail } from 'k6'

export const BASE_URL = __ENV.GATEWAY_BASE || 'http://localhost:8080'
export const POSTS_PATH = '/v1/posts'
export const CATEGORIES_PATH = '/v1/categories'
export const USERS_REGISTER_PATH = '/v1/users/register'
export const USERS_LOGIN_PATH = '/v1/users/login'

export function splitCSV(value) {
  return String(value || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

export function normalizeToken(token) {
  return token.replace(/^Bearer\s+/i, '').trim()
}

export function authHeader(token) {
  return {
    Authorization: `Bearer ${normalizeToken(token)}`,
    'Content-Type': 'application/json',
    Accept: 'application/json',
  }
}

export function json(res) {
  try {
    return res.json()
  } catch (_e) {
    return null
  }
}

export function pick(items, offset) {
  return items[Math.abs(offset) % items.length]
}

export function getOrCreateTokens(defaultAccountCount) {
  const tokens = splitCSV(__ENV.AUTH_TOKENS || __ENV.ACCESS_TOKENS || __ENV.AUTH_TOKEN || __ENV.ACCESS_TOKEN)
    .map(normalizeToken)
  if (tokens.length > 0) {
    return tokens
  }

  const accountCount = Number(__ENV.ACCOUNT_COUNT || defaultAccountCount)
  if (accountCount <= 0) {
    fail('ACCOUNT_COUNT 必须大于 0，或直接设置 AUTH_TOKENS')
  }

  const password = __ENV.ACCOUNT_PASSWORD || 'K6LoadTest123!'
  const emailDomain = __ENV.ACCOUNT_EMAIL_DOMAIN || 'k6.gopalette.local'
  const prefix = __ENV.ACCOUNT_PREFIX || `k6-${Date.now()}`

  const registeredTokens = []
  for (let i = 0; i < accountCount; i += 1) {
    const email = `${prefix}-${i}@${emailDomain}`
    const username = `${prefix}-${i}`
    registerAccount(username, email, password)
    registeredTokens.push(loginAccount(email, password))
  }
  return registeredTokens
}

function registerAccount(username, email, password) {
  const res = http.post(
    `${BASE_URL}${USERS_REGISTER_PATH}`,
    JSON.stringify({ username, email, password }),
    { headers: { 'Content-Type': 'application/json', Accept: 'application/json' }, tags: { endpoint: 'user-register' } },
  )
  const ok = check(res, {
    'register status is 200 or 409': (r) => r.status === 200 || r.status === 409,
  })
  if (!ok) {
    fail(`注册压测账号失败，email=${email}, status=${res.status}, body=${res.body}`)
  }
}

function loginAccount(email, password) {
  const res = http.post(
    `${BASE_URL}${USERS_LOGIN_PATH}`,
    JSON.stringify({ email, password }),
    { headers: { 'Content-Type': 'application/json', Accept: 'application/json' }, tags: { endpoint: 'user-login' } },
  )
  const body = json(res)
  const token = body?.accessToken || body?.access_token
  const ok = check(res, {
    'login status is 200': (r) => r.status === 200,
    'login returns access token': () => !!token,
  })
  if (!ok) {
    fail(`登录压测账号失败，email=${email}, status=${res.status}, body=${res.body}`)
  }
  return normalizeToken(token)
}

export function getPostIDs(headers, createSeedPost) {
  const ids = splitCSV(__ENV.POST_IDS || __ENV.POST_ID)
    .map((item) => Number(item))
    .filter((id) => id > 0)
  if (ids.length > 0) {
    return ids
  }

  const pageSize = Number(__ENV.POST_POOL_SIZE || 50)
  const res = http.get(`${BASE_URL}${POSTS_PATH}?page=1&pageSize=${pageSize}`, {
    headers,
    tags: { endpoint: 'post-list-for-comment-seed' },
  })
  const body = json(res)
  const listedIDs = (body?.posts || [])
    .map((post) => Number(post.id || post.info?.id))
    .filter((id) => id > 0)

  if (listedIDs.length > 0) {
    return listedIDs
  }

  const seedID = createSeedPost(headers)
  if (seedID > 0) {
    return [seedID]
  }

  fail(`无法获取或创建有效 post_id。list_status=${res.status}, list_body=${res.body}`)
}

export function getCategoryID(headers) {
  const envCategoryID = Number(__ENV.CATEGORY_ID || 0)
  if (envCategoryID > 0) {
    return envCategoryID
  }

  const categoryName = __ENV.CATEGORY_NAME || 'K6 Load Test'
  const res = http.get(`${BASE_URL}${CATEGORIES_PATH}?page=1&pageSize=100`, {
    headers,
    tags: { endpoint: 'category-list-for-k6' },
  })
  const body = json(res)
  const categories = body?.categories || []
  const category = categories.find((item) => item?.name === categoryName)
  const categoryID = Number(category?.id || category?.info?.id || 0)

  const ok = check(res, {
    'list categories status is 200': (r) => r.status === 200,
    'k6 category exists': () => categoryID > 0,
  })
  if (!ok) {
    fail(
      `无法获取压测分类，请先启动 compose.k6.yaml 或设置 CATEGORY_ID。` +
      ` categoryName=${categoryName}, status=${res.status}, body=${res.body}`,
    )
  }
  return categoryID
}
