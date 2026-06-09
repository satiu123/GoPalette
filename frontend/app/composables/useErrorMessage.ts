type ErrorRecord = Record<string, unknown>

const REASON_MESSAGES: Record<string, string> = {
  EMAIL_CONFLICT: '该邮箱已被注册，请直接登录或换一个邮箱',
  USERNAME_CONFLICT: '该用户名已被使用，请换一个用户名',
  PASSWORD_INCORRECT: '邮箱或密码不正确',
  USER_NOT_FOUND: '用户不存在，请检查邮箱是否正确',
  ACCOUNT_DISABLED: '账号已被停用，请联系管理员',
  UNAUTHENTICATED: '登录状态已过期，请重新登录',
  ACCESS_DENIED: '当前账号没有权限执行该操作',
  TOKEN_EXPIRED: '登录状态已过期，请重新登录',
  TOKEN_INVALID: '登录状态无效，请重新登录',
  REFRESH_TOKEN_EXPIRED: '登录状态已过期，请重新登录',
  INVALID_ARGUMENT: '提交内容有误，请检查后再试',
  DATABASE_ERROR: '服务暂时不可用，请稍后再试',
  INTERNAL_SERVER_ERROR: '服务暂时不可用，请稍后再试',
  UNKNOWN_ERROR: '服务暂时不可用，请稍后再试',
  SLUG_CONFLICT: '文章路径已存在，请换一个标题或路径',
  POST_NOT_FOUND: '文章不存在或已被删除',
  CATEGORY_NOT_FOUND: '分类不存在，请重新选择分类',
  TAG_NOT_FOUND: '标签不存在，请刷新后重试',
  COMMENT_NOT_FOUND: '评论不存在或已被删除',
  PARENT_NOT_FOUND: '回复的评论不存在',
  RATE_LIMITED: '操作过于频繁，请稍后再试',
  SEARCH_FAILED: '搜索服务暂时不可用，请稍后再试',
  INDEX_SYNC_FAILED: '搜索索引同步失败，请稍后再试',
  REBUILD_IN_PROGRESS: '索引正在重建，请稍后再试'
}

const STATUS_MESSAGES: Record<number, string> = {
  400: '提交内容有误，请检查后再试',
  401: '登录状态已过期，请重新登录',
  403: '当前账号没有权限执行该操作',
  404: '请求的内容不存在',
  409: '内容冲突，请检查是否已存在',
  413: '文件过大，请压缩后再上传',
  429: '操作过于频繁，请稍后再试',
  500: '服务暂时不可用，请稍后再试',
  502: '上游服务暂时不可用，请稍后再试',
  503: '服务正在维护，请稍后再试',
  504: '请求超时，请稍后再试'
}

const COMMON_MESSAGE_TRANSLATIONS: Record<string, string> = {
  'Unauthorized': '登录状态已过期，请重新登录',
  'Session expired': '登录状态已过期，请重新登录',
  'Forbidden': '当前账号没有权限执行该操作',
  'Bad Request': '提交内容有误，请检查后再试',
  'Not Found': '请求的内容不存在',
  'Conflict': '内容冲突，请检查是否已存在',
  'Payload Too Large': '文件过大，请压缩后再上传',
  'Too Many Requests': '操作过于频繁，请稍后再试',
  'Internal Server Error': '服务暂时不可用，请稍后再试',
  'Bad Gateway': '上游服务暂时不可用，请稍后再试',
  'Service Unavailable': '服务正在维护，请稍后再试',
  'Gateway Timeout': '请求超时，请稍后再试'
}

function isRecord(value: unknown): value is ErrorRecord {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value))
}

function cleanText(value: unknown) {
  if (value === undefined || value === null) return ''
  const text = String(value).trim()
  return text && text !== '[object Object]' ? text : ''
}

function isReasonCode(value: string) {
  return /^[A-Z][A-Z0-9_]+$/.test(value)
}

function isUsefulMessage(value: string) {
  if (!value) return false
  if (isReasonCode(value)) return false
  if (/^\[[A-Z]+\]\s+".*":\s+\d{3}/.test(value)) return false
  return true
}

function getNestedRecord(record: ErrorRecord, key: string) {
  const value = record[key]
  return isRecord(value) ? value : undefined
}

function collectRecords(error: unknown) {
  const records: ErrorRecord[] = []
  const seen = new Set<ErrorRecord>()

  const add = (value: unknown) => {
    if (!isRecord(value) || seen.has(value)) return
    seen.add(value)
    records.push(value)
  }

  if (isRecord(error)) {
    add(getNestedRecord(error, 'data'))
    const response = getNestedRecord(error, 'response')
    add(getNestedRecord(response || {}, '_data'))
    add(getNestedRecord(response || {}, 'data'))
    add(error)
    add(getNestedRecord(error, 'cause'))

    for (let i = 0; i < records.length; i += 1) {
      const record = records[i]
      if (!record) continue
      add(getNestedRecord(record, 'data'))
      add(getNestedRecord(record, '_data'))
      add(getNestedRecord(record, 'cause'))
    }
  }

  return records
}

function extractStatus(records: ErrorRecord[]) {
  for (const record of records) {
    const status = Number(record.statusCode ?? record.status ?? record.code ?? 0)
    if (Number.isFinite(status) && status >= 400 && status <= 599) {
      return status
    }
  }
  return 0
}

function extractReason(records: ErrorRecord[]) {
  for (const record of records) {
    const reason = cleanText(record.reason ?? record.errorReason ?? record.error_reason)
    if (reason) return reason
  }
  return ''
}

function extractMessage(records: ErrorRecord[]) {
  const fields = ['message', 'statusMessage', 'statusText', 'detail', 'error_description', 'error']
  for (const record of records) {
    for (const field of fields) {
      const text = cleanText(record[field])
      if (!text) continue
      if (COMMON_MESSAGE_TRANSLATIONS[text]) return COMMON_MESSAGE_TRANSLATIONS[text]
      if (REASON_MESSAGES[text]) return REASON_MESSAGES[text]
      if (isUsefulMessage(text)) return text
    }
  }
  return ''
}

export function getRequestErrorMessage(error: unknown, fallback = '请稍后重试') {
  const direct = cleanText(error)
  if (direct && !isRecord(error)) return direct

  const records = collectRecords(error)
  const message = extractMessage(records)
  if (message) return message

  const reason = extractReason(records)
  if (reason && REASON_MESSAGES[reason]) return REASON_MESSAGES[reason]

  const status = extractStatus(records)
  if (status && STATUS_MESSAGES[status]) return STATUS_MESSAGES[status]

  return fallback
}
