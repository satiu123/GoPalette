import { gatewayFetch, setAuthCookies } from '../../utils/auth'

function decodeJwtPayload(token: string) {
  try {
    const part = token.split('.')[1]
    if (!part) return null

    const normalized = part.replace(/-/g, '+').replace(/_/g, '/')
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
    if (typeof globalThis.atob !== 'function') return null
    return JSON.parse(globalThis.atob(padded)) as Record<string, unknown>
  } catch {
    return null
  }
}

function extractUserId(token: string) {
  const payload = decodeJwtPayload(token)
  if (!payload) return ''

  const candidate = payload.userId || payload.user_id || payload.sub || payload.id
  if (typeof candidate === 'string') return candidate
  if (typeof candidate === 'number' && Number.isFinite(candidate)) return String(Math.trunc(candidate))

  return ''
}

export default defineEventHandler(async (event): Promise<{ success: true, userId: string }> => {
  const body = await readBody(event)

  const response = await gatewayFetch<Record<string, unknown>>(event, '/v1/users/login', {
    method: 'POST',
    auth: 'none',
    body
  })

  const accessToken = String(response.accessToken || response.access_token || '')
  const refreshToken = String(response.refreshToken || response.refresh_token || '')
  if (!accessToken || !refreshToken) {
    throw createError({
      statusCode: 502,
      statusMessage: '登录服务返回异常，请稍后再试'
    })
  }

  setAuthCookies(event, { accessToken, refreshToken })

  return {
    success: true,
    userId: extractUserId(accessToken)
  }
})
