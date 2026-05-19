import http from 'k6/http'
import { check } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'
import exec from 'k6/execution'
import { BASE_URL, POSTS_PATH, authHeader, getCategoryID, getOrCreateTokens, getPostIDs, json, pick } from './write-helpers.js'

const COMMENTS_PATH = '/v1/comments'

const RATE = Number(__ENV.RATE || __ENV.COMMENT_RATE || 200)
const PRE_ALLOCATED_VUS = Number(__ENV.PRE_ALLOCATED_VUS || 300)
const MAX_VUS = Number(__ENV.MAX_VUS || 1500)
const ACCOUNT_COUNT = Number(__ENV.ACCOUNT_COUNT || 100)
const COMMENT_PARENT_RATIO = Number(__ENV.COMMENT_PARENT_RATIO || 0)
const MIN_SUCCESS_RATIO = Number(__ENV.MIN_SUCCESS_RATIO || __ENV.MIN_COMMENT_SUCCESS_RATIO || 0.01)
const MAX_UNEXPECTED_STATUS_RATIO = Number(__ENV.MAX_UNEXPECTED_STATUS_RATIO || 0.01)

const createSuccess = new Counter('comment_create_success_total')
const rateLimited = new Counter('comment_rate_limited_total')
const unexpectedStatus = new Rate('comment_unexpected_status_rate')
const successRatio = new Rate('comment_success_ratio')
const writeDuration = new Trend('comment_write_duration')

function parseStages(value, fallback) {
  const raw = String(value || '').trim()
  if (!raw) {
    return fallback
  }
  return raw.split(',').map((item) => {
    const [duration, target] = item.split(':').map((part) => part.trim())
    if (!duration || !target) {
      throw new Error(`RAMP_STAGES 格式错误: ${item}，示例 30s:100,1m:300,30s:0`)
    }
    return { duration, target: Number(target) }
  })
}

const RAMP_STAGES = parseStages(__ENV.RAMP_STAGES || __ENV.COMMENT_RAMP_STAGES, [
  { duration: '30s', target: RATE },
  { duration: '1m', target: RATE * 2 },
  { duration: '1m', target: RATE * 4 },
  { duration: '1m', target: RATE * 6 },
  { duration: '30s', target: 0 },
])

/*
k6 run k6/comment-write.js \
  -e GATEWAY_BASE=http://localhost:8080 \
  -e ACCOUNT_COUNT=500 \
  -e POST_IDS='1,2,3' \
  -e RATE=500 \
  -e RAMP_STAGES='30s:200,1m:500,1m:800,1m:1200,30s:0'

评论服务按用户每分钟限制 3 次创建。要压成功写入吞吐，请提高 ACCOUNT_COUNT 或传入 AUTH_TOKENS。
*/
export const options = {
  scenarios: {
    comment_writes: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      stages: RAMP_STAGES,
      preAllocatedVUs: PRE_ALLOCATED_VUS,
      maxVUs: MAX_VUS,
      tags: { workload: 'comment-write' },
    },
  },
  thresholds: {
    checks: ['rate>0.99'],
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<300', 'p(99)<800'],
    comment_unexpected_status_rate: [`rate<${MAX_UNEXPECTED_STATUS_RATIO}`],
    comment_success_ratio: [`rate>${MIN_SUCCESS_RATIO}`],
    comment_write_duration: ['p(95)<300', 'p(99)<800'],
  },
  summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
}

http.setResponseCallback(http.expectedStatuses(200, 429))

export function setup() {
  const tokens = getOrCreateTokens(ACCOUNT_COUNT)
  const headers = authHeader(tokens[0])
  const categoryID = getCategoryID(headers)
  const postIDs = getPostIDs(headers, (seedHeaders) => createSeedPost(seedHeaders, categoryID))
  return { tokens, postIDs }
}

function createSeedPost(headers, categoryID) {
  const now = Date.now()
  const payload = JSON.stringify({
    title: `k6 comment seed ${now}`,
    summary: 'k6 高并发评论写入种子文章',
    slug: `k6-comment-seed-${now}`,
    content: `# k6 comment seed\n\ncreated_at=${new Date(now).toISOString()}`,
    status: 1,
    categoryId: categoryID,
    tags: ['k6', 'comment-write'],
  })
  const res = http.post(`${BASE_URL}${POSTS_PATH}`, payload, {
    headers,
    tags: { endpoint: 'post-create-comment-seed' },
  })
  const body = json(res)
  const id = Number(body?.post?.info?.id || body?.post?.id || body?.id || 0)
  check(res, {
    'seed post status is 200': (r) => r.status === 200,
    'seed post returns id': () => id > 0,
  })
  return id
}

export default function (data) {
  const token = pick(data.tokens, exec.vu.idInTest + exec.scenario.iterationInTest)
  const postID = pick(data.postIDs, exec.scenario.iterationInTest)
  const parentID = COMMENT_PARENT_RATIO > 0 && Math.random() < COMMENT_PARENT_RATIO ? Number(__ENV.PARENT_COMMENT_ID || 0) : 0
  const payload = JSON.stringify({
    postId: postID,
    content: `k6 高并发评论 vu=${exec.vu.idInTest} iter=${exec.scenario.iterationInTest} ts=${Date.now()}`,
    parentId: parentID,
  })

  const res = http.post(`${BASE_URL}${COMMENTS_PATH}`, payload, {
    headers: authHeader(token),
    tags: { endpoint: 'comment-create', workload: 'comment-write' },
  })

  writeDuration.add(res.timings.duration)

  const isSuccess = res.status === 200
  const isRateLimited = res.status === 429
  const isExpected = isSuccess || isRateLimited
  if (isSuccess) {
    createSuccess.add(1)
  }
  if (isRateLimited) {
    rateLimited.add(1)
  }
  unexpectedStatus.add(isExpected ? 0 : 1)
  successRatio.add(isSuccess ? 1 : 0)

  check(res, {
    'comment create status is 200 or 429': () => isExpected,
  })
}
