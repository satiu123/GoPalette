import http from 'k6/http'
import { check, sleep } from 'k6'
import { SharedArray } from 'k6/data'
// 导入处理 CSV 的库
import papaparse from 'https://jslib.k6.io/papaparse/5.1.1/index.js';

const BASE_URL = __ENV.GATEWAY_BASE || 'http://localhost:8080'
const SEARCH_PATH = '/v1/search/posts'
const CATEGORIES = ['', '后端', '架构', '数据库', '运维']

// 1. 加载并解析 CSV 文件
const keywords = new SharedArray('blog keywords', function () {
  const data = open('../keywords.csv'); // 读取文件
  const results = papaparse.parse(data, { header: true }); // 解析 CSV，header: true 会自动识别第一行的 keyword
  // 过滤掉空行并返回关键字数组,保留前150个关键字
  return results.data
    .map(row => row.keyword.trim())
    .filter(keyword => keyword.length > 0)
    .slice(0, 150);
});

export const options = {
  scenarios: {
    search_ramp: {
      executor: 'ramping-vus',
      startVUs: 0,
      gracefulRampDown: '20s',
      stages: [
        { duration: '30s', target: 20 },
        { duration: '1m', target: 50 },
        { duration: '1m', target: 100 },
        { duration: '1m', target: 200 },
        { duration: '1m', target: 300 },
        { duration: '30s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    // 既然你的实测数据是 18ms，这里建议设为 p(99)<50，更具说服力
    http_req_duration: ['p(95)<30', 'p(99)<50'],
    checks: ['rate>0.99'],
  },
  summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
}

export default function () {
  // 2. 从 3000 个关键字中随机选择一个
  const keyword = keywords[Math.floor(Math.random() * keywords.length)]

  const category = CATEGORIES[Math.floor(Math.random() * CATEGORIES.length)]
  const page = Math.floor(Math.random() * 5) + 1 // 50万数据，页码可以稍微大点
  const pageSize = 20

  const query = [
    `query=${encodeURIComponent(keyword)}`,
    `page=${page}`,
    `pageSize=${pageSize}`,
    `category=${encodeURIComponent(category)}`,
  ].join('&')

  const res = http.get(`${BASE_URL}${SEARCH_PATH}?${query}`, {
    headers: { Accept: 'application/json' },
    tags: { endpoint: 'search-posts' },
  })

  check(res, {
    'status is 200': (r) => r.status === 200,
    'contains total field': (r) => r.body && r.body.includes('"total"'),
  })

  sleep(Math.random() * 0.2 + 0.05)
}