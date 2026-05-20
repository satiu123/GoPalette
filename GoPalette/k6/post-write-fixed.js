import http from 'k6/http'
import { check } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'
import exec from 'k6/execution'
import { BASE_URL, POSTS_PATH, authHeader, getCategoryID, getOrCreateTokens, pick } from './write-helpers.js'

const RATE = Number(__ENV.RATE || __ENV.POST_RATE || 500)
const DURATION = __ENV.DURATION || '3m'
const PRE_ALLOCATED_VUS = Number(__ENV.PRE_ALLOCATED_VUS || 300)
const MAX_VUS = Number(__ENV.MAX_VUS || 1500)
const ACCOUNT_COUNT = Number(__ENV.ACCOUNT_COUNT || 100)
const POST_STATUS = Number(__ENV.POST_STATUS || 0)
const MAX_UNEXPECTED_STATUS_RATIO = Number(__ENV.MAX_UNEXPECTED_STATUS_RATIO || 0.01)
const CONTENT_REPEAT = Number(__ENV.CONTENT_REPEAT || 80)

const createSuccess = new Counter('post_create_success_total')
const createConflict = new Counter('post_create_conflict_total')
const unexpectedStatus = new Rate('post_unexpected_status_rate')
const writeDuration = new Trend('post_write_duration')

/*
k6 run k6/post-write-fixed.js \
  -e GATEWAY_BASE=http://localhost:18080 \
  -e RATE=500 \
  -e CONTENT_REPEAT=80 \
  -e DURATION=3m \
  -e ACCOUNT_COUNT=100
*/
export const options = {
  scenarios: {
    post_writes_fixed: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: PRE_ALLOCATED_VUS,
      maxVUs: MAX_VUS,
      tags: { workload: 'post-write-fixed', target_rps: String(RATE) },
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
    RATE,
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
    title: `k6 fixed ${RATE}rps 写入文章 ${suffix}`,
    summary: `k6 fixed post write ${RATE}rps summary ${suffix}`,
    slug: `k6-fixed-${RATE}rps-post-write-${suffix}`,
    content: `# k6 fixed ${RATE}rps post write ${suffix}\n\n这是一篇由 k6 固定 RPS 创建的高并发写入压测文章。\n\n${'content '.repeat(CONTENT_REPEAT)}`,
    status: POST_STATUS,
    categoryId: data.categoryID,
    tags: ['k6', 'post-write-fixed', `${RATE}rps`, `vu-${exec.vu.idInTest % 32}`],
  })

  const res = http.post(`${BASE_URL}${POSTS_PATH}`, payload, {
    headers: authHeader(token),
    tags: { endpoint: 'post-create', workload: 'post-write-fixed', target_rps: String(RATE) },
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
