import http from 'k6/http'
import { check, fail } from 'k6'
import { Counter, Rate } from 'k6/metrics'
import exec from 'k6/execution'

const BASE_URL = __ENV.GATEWAY_BASE || 'http://localhost:8080'
const COMMENT_PATH = '/v1/comments'
const POSTS_PATH = '/v1/posts?page=1&pageSize=1'

const RATE = Number(__ENV.RATE || 300)
const DURATION = __ENV.DURATION || '2m'
const PRE_ALLOCATED_VUS = Number(__ENV.PRE_ALLOCATED_VUS || 300)
const MAX_VUS = Number(__ENV.MAX_VUS || 1500)
const MIN_429_RATIO = Number(__ENV.MIN_429_RATIO || 0.6)

const rateLimitedCount = new Counter('comment_rate_limited_total')
const successCount = new Counter('comment_create_success_total')
const unexpectedStatusRate = new Rate('comment_unexpected_status_rate')
const rateLimitedRatio = new Rate('comment_rate_limited_ratio')
/*  
k6 run k6/comment-write-abuse-rate-limit.js \
   -e GATEWAY_BASE=http://localhost:8080 \
   -e AUTH_TOKEN='Bearer令牌' \
   -e POST_ID=1 \
   -e RATE=500 \
   -e DURATION=3m \
   -e MIN_429_RATIO=0.6
*/
export const options = {
  scenarios: {
    abusive_comment_writes: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: PRE_ALLOCATED_VUS,
      maxVUs: MAX_VUS,
    },
  },
  thresholds: {
    checks: ['rate>0.99'],
    comment_unexpected_status_rate: ['rate==0'],
    comment_rate_limited_ratio: [`rate>${MIN_429_RATIO}`],
    http_req_duration: ['p(95)<200'],
  },
  summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
}

http.setResponseCallback(http.expectedStatuses(200, 429))

function getAccessToken() {
  const token = (__ENV.AUTH_TOKEN || __ENV.ACCESS_TOKEN || '').trim()
  if (token) {
    return token
  }

  const email = (__ENV.LOGIN_EMAIL || '').trim()
  const password = (__ENV.LOGIN_PASSWORD || '').trim()
  if (!email || !password) {
    fail('缺少认证信息：请设置 AUTH_TOKEN，或同时设置 LOGIN_EMAIL/LOGIN_PASSWORD')
  }

  const loginRes = http.post(
    `${BASE_URL}/v1/users/login`,
    JSON.stringify({ email, password }),
    { headers: { 'Content-Type': 'application/json', Accept: 'application/json' } },
  )

  let loginBody = null
  try {
    loginBody = loginRes.json()
  } catch (_e) {
    loginBody = null
  }
  const accessToken = loginBody?.accessToken || loginBody?.access_token
  const ok = check(loginRes, {
    'login status is 200': (r) => r.status === 200,
    'login returns access token': () => !!accessToken,
  })
  if (!ok) {
    fail(`登录失败，状态码=${loginRes.status}，响应=${loginRes.body}`)
  }
  return accessToken
}

function getPostID(authHeaders) {
  const envPostID = Number(__ENV.POST_ID || 0)
  if (envPostID > 0) {
    return envPostID
  }

  const postRes = http.get(`${BASE_URL}${POSTS_PATH}`, {
    headers: authHeaders,
  })
  let postBody = null
  try {
    postBody = postRes.json()
  } catch (_e) {
    postBody = null
  }

  const postID = Number(postBody?.posts?.[0]?.id || 0)
  const ok = check(postRes, {
    'list posts status is 200': (r) => r.status === 200,
    'has post id': () => postID > 0,
  })
  if (!ok) {
    fail(
      '无法获取有效 post_id，请设置 POST_ID 或确保 /v1/posts 至少存在一篇文章。' +
      `status=${postRes.status}, body=${postRes.body}`,
    )
  }
  return postID
}

export function setup() {
  const token = getAccessToken()
  const headers = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
    Accept: 'application/json',
  }
  const postID = getPostID(headers)
  return { token, postID }
}

export default function (data) {
  const content = `abuse-check vu=${exec.vu.idInTest} iter=${exec.scenario.iterationInTest} ts=${Date.now()}`
  const payload = JSON.stringify({
    postId: data.postID,
    content,
    parentId: 0,
  })

  const res = http.post(`${BASE_URL}${COMMENT_PATH}`, payload, {
    headers: {
      Authorization: `Bearer ${data.token}`,
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    tags: { endpoint: 'comment-create', test: 'abusive-rate-limit' },
  })

  const isSuccess = res.status === 200
  const isRateLimited = res.status === 429
  const isExpected = isSuccess || isRateLimited

  if (isSuccess) {
    successCount.add(1)
  }
  if (isRateLimited) {
    rateLimitedCount.add(1)
  }
  unexpectedStatusRate.add(isExpected ? 0 : 1)
  rateLimitedRatio.add(isRateLimited ? 1 : 0)

  check(res, {
    'status is 200 or 429': () => isExpected,
  })
}
