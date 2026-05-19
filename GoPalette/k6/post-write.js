import http from 'k6/http'
import { check } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'
import exec from 'k6/execution'
import { BASE_URL, POSTS_PATH, authHeader, getCategoryID, getOrCreateTokens, pick } from './write-helpers.js'

const RATE = Number(__ENV.RATE || __ENV.POST_RATE || 50)
const PRE_ALLOCATED_VUS = Number(__ENV.PRE_ALLOCATED_VUS || 150)
const MAX_VUS = Number(__ENV.MAX_VUS || 800)
const ACCOUNT_COUNT = Number(__ENV.ACCOUNT_COUNT || 20)
const POST_STATUS = Number(__ENV.POST_STATUS || 0)
const MAX_UNEXPECTED_STATUS_RATIO = Number(__ENV.MAX_UNEXPECTED_STATUS_RATIO || 0.01)

const createSuccess = new Counter('post_create_success_total')
const createConflict = new Counter('post_create_conflict_total')
const unexpectedStatus = new Rate('post_unexpected_status_rate')
const writeDuration = new Trend('post_write_duration')

function parseStages(value, fallback) {
  const raw = String(value || '').trim()
  if (!raw) {
    return fallback
  }
  return raw.split(',').map((item) => {
    const [duration, target] = item.split(':').map((part) => part.trim())
    if (!duration || !target) {
      throw new Error(`RAMP_STAGES 格式错误: ${item}，示例 30s:50,1m:100,30s:0`)
    }
    return { duration, target: Number(target) }
  })
}

const RAMP_STAGES = parseStages(__ENV.RAMP_STAGES || __ENV.POST_RAMP_STAGES, [
  { duration: '30s', target: RATE },
  { duration: '1m', target: RATE * 2 },
  { duration: '1m', target: RATE * 4 },
  { duration: '1m', target: RATE * 8 },
  { duration: '30s', target: 0 },
])

/*
k6 run k6/post-write.js \
  -e GATEWAY_BASE=http://localhost:8080 \
  -e ACCOUNT_COUNT=50 \
  -e RATE=100 \
  -e RAMP_STAGES='30s:100,1m:200,1m:400,1m:800,30s:0'
*/
export const options = {
  scenarios: {
    post_writes: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      stages: RAMP_STAGES,
      preAllocatedVUs: PRE_ALLOCATED_VUS,
      maxVUs: MAX_VUS,
      tags: { workload: 'post-write' },
    },
  },
  thresholds: {
    checks: ['rate>0.99'],
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    post_unexpected_status_rate: [`rate<${MAX_UNEXPECTED_STATUS_RATIO}`],
    post_write_duration: ['p(95)<500', 'p(99)<1000'],
  },
  summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
}

http.setResponseCallback(http.expectedStatuses(200, 409))

function uniqueSuffix() {
  return [
    exec.vu.idInTest,
    exec.scenario.iterationInTest,
    Date.now(),
    Math.floor(Math.random() * 1000000),
  ].join('-')
}

export function setup() {
  const tokens = getOrCreateTokens(ACCOUNT_COUNT)
  const categoryID = getCategoryID(authHeader(tokens[0]))
  return { tokens, categoryID }
}

export default function (data) {
  const token = pick(data.tokens, exec.vu.idInTest + exec.scenario.iterationInTest)
  const suffix = uniqueSuffix()
  const payload = JSON.stringify({
    title: `k6 高并发写入文章 ${suffix}`,
    summary: `k6 post write stress summary ${suffix}`,
    slug: `k6-post-write-${suffix}`,
    content: `# k6 post write ${suffix}\n\n这是一篇由 k6 创建的高并发写入压测文章。\n\n${'content '.repeat(80)}`,
    status: POST_STATUS,
    categoryId: data.categoryID,
    tags: ['k6', 'post-write', `vu-${exec.vu.idInTest % 32}`],
  })

  const res = http.post(`${BASE_URL}${POSTS_PATH}`, payload, {
    headers: authHeader(token),
    tags: { endpoint: 'post-create', workload: 'post-write' },
  })

  writeDuration.add(res.timings.duration)

  const isSuccess = res.status === 200
  const isConflict = res.status === 409
  const isExpected = isSuccess || isConflict
  if (isSuccess) {
    createSuccess.add(1)
  }
  if (isConflict) {
    createConflict.add(1)
  }
  unexpectedStatus.add(isExpected ? 0 : 1)

  check(res, {
    'post create status is 200 or 409': () => isExpected,
  })
}
