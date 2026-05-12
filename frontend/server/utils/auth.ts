import { joinURL } from 'ufo'
import type { H3Event } from 'h3'

const ACCESS_COOKIE = 'gopalette_access_token'
const REFRESH_COOKIE = 'gopalette_refresh_token'
const LOGGED_IN_COOKIE = 'gopalette_logged_in'
const USER_ID_COOKIE = 'gopalette_user_id'
const REFRESH_AHEAD_SECONDS = 45

interface TokenPair {
  accessToken: string
  refreshToken: string
}

function extractErrorStatus(error: unknown) {
  if (!error || typeof error !== 'object') return 0

  const typed = error as { status?: unknown, statusCode?: unknown, response?: { status?: unknown } }
  return Number(typed.status ?? typed.statusCode ?? typed.response?.status ?? 0)
}

function isProd() {
  return !import.meta.dev
}

function authCookieSecure() {
  const config = useRuntimeConfig()
  const value = config.authCookieSecure
  if (typeof value === 'boolean') return value
  if (typeof value === 'string' && value.trim()) return value.toLowerCase() === 'true'

  const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env || {}
  const nitroValue = env.NITRO_AUTH_COOKIE_SECURE
  if (typeof nitroValue === 'string' && nitroValue.trim()) {
    return nitroValue.toLowerCase() === 'true'
  }

  return isProd()
}

function decodeJwtPayload(token: string) {
  try {
    const part = token.split('.')[1]
    if (!part) return null

    const normalized = part.replace(/-/g, '+').replace(/_/g, '/')
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
    if (typeof globalThis.atob !== 'function') return null
    const json = globalThis.atob(padded)
    return JSON.parse(json) as Record<string, unknown>
  } catch {
    return null
  }
}

function extractTokenExpiresAt(token: string) {
  const payload = decodeJwtPayload(token)
  if (!payload) return 0

  const exp = payload.exp
  if (typeof exp === 'number' && Number.isFinite(exp)) return exp
  if (typeof exp === 'string') {
    const parsed = Number(exp)
    return Number.isFinite(parsed) ? parsed : 0
  }

  return 0
}

function extractUserIdFromToken(token: string) {
  const payload = decodeJwtPayload(token)
  if (!payload) return ''

  const candidate = payload.userId || payload.user_id || payload.sub || payload.id
  if (typeof candidate === 'string') return candidate
  if (typeof candidate === 'number' && Number.isFinite(candidate)) return String(Math.trunc(candidate))

  return ''
}

function shouldRefreshSoon(token: string, advanceSeconds = REFRESH_AHEAD_SECONDS) {
  const exp = extractTokenExpiresAt(token)
  if (!exp) return false

  return exp <= Math.floor(Date.now() / 1000) + advanceSeconds
}

function authCookieBase() {
  return {
    path: '/',
    sameSite: 'lax' as const,
    secure: authCookieSecure()
  }
}

function authCookieMaxAge(token: string) {
  const exp = extractTokenExpiresAt(token)
  if (!exp) return undefined

  const delta = exp - Math.floor(Date.now() / 1000)
  return delta > 0 ? delta : 0
}

function normalizeTokenPair(input: Record<string, unknown> | null | undefined): TokenPair {
  return {
    accessToken: String(input?.accessToken || input?.access_token || ''),
    refreshToken: String(input?.refreshToken || input?.refresh_token || '')
  }
}

export function clearAuthCookies(event: H3Event) {
  const cookieOptions = authCookieBase()

  deleteCookie(event, ACCESS_COOKIE, { ...cookieOptions, httpOnly: true })
  deleteCookie(event, REFRESH_COOKIE, { ...cookieOptions, httpOnly: true })
  deleteCookie(event, LOGGED_IN_COOKIE, cookieOptions)
  deleteCookie(event, USER_ID_COOKIE, cookieOptions)
}

export function setAuthCookies(event: H3Event, tokens: TokenPair) {
  const accessMaxAge = authCookieMaxAge(tokens.accessToken)
  const refreshMaxAge = authCookieMaxAge(tokens.refreshToken)
  const userId = extractUserIdFromToken(tokens.accessToken)
  const cookieOptions = authCookieBase()

  setCookie(event, ACCESS_COOKIE, tokens.accessToken, {
    ...cookieOptions,
    httpOnly: true,
    maxAge: accessMaxAge
  })
  setCookie(event, REFRESH_COOKIE, tokens.refreshToken, {
    ...cookieOptions,
    httpOnly: true,
    maxAge: refreshMaxAge
  })
  setCookie(event, LOGGED_IN_COOKIE, '1', {
    ...cookieOptions,
    httpOnly: false,
    maxAge: refreshMaxAge
  })
  setCookie(event, USER_ID_COOKIE, userId, {
    ...cookieOptions,
    httpOnly: false,
    maxAge: refreshMaxAge
  })
}

export function readSessionHint(event: H3Event) {
  const userId = getCookie(event, USER_ID_COOKIE) || ''
  const loggedIn = getCookie(event, LOGGED_IN_COOKIE) === '1' && Boolean(userId)

  return { loggedIn, userId }
}

export async function refreshServerSession(event: H3Event) {
  const config = useRuntimeConfig(event)
  const refreshToken = getCookie(event, REFRESH_COOKIE) || ''
  if (!refreshToken) {
    clearAuthCookies(event)
    return ''
  }

  try {
    const response = await $fetch<Record<string, unknown>>(joinURL(config.gatewayBase, '/v1/users/refresh'), {
      method: 'POST',
      body: {
        refreshToken
      }
    })

    const tokens = normalizeTokenPair(response)
    if (!tokens.accessToken || !tokens.refreshToken) {
      clearAuthCookies(event)
      return ''
    }

    setAuthCookies(event, tokens)
    return tokens.accessToken
  } catch (error: unknown) {
    clearAuthCookies(event)
    const status = extractErrorStatus(error)
    if (status === 401) {
      throw createError({ statusCode: 401, statusMessage: 'Session expired' })
    }
    throw error
  }
}

export async function ensureAccessToken(event: H3Event, options?: { required?: boolean, forceRefresh?: boolean }) {
  const required = options?.required ?? true
  const forceRefresh = options?.forceRefresh ?? false
  const accessToken = getCookie(event, ACCESS_COOKIE) || ''

  if (!forceRefresh && accessToken && !shouldRefreshSoon(accessToken)) {
    return accessToken
  }

  const refreshToken = getCookie(event, REFRESH_COOKIE) || ''
  if (!refreshToken) {
    clearAuthCookies(event)
    if (required) {
      throw createError({ statusCode: 401, statusMessage: 'Unauthorized' })
    }
    return ''
  }

  try {
    const nextToken = await refreshServerSession(event)
    if (!nextToken && required) {
      throw createError({ statusCode: 401, statusMessage: 'Unauthorized' })
    }
    return nextToken
  } catch (error: unknown) {
    if (required) {
      throw error
    }
    return ''
  }
}

export async function gatewayFetch<T>(
  event: H3Event,
  path: string,
  options?: {
    method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'
    query?: Record<string, unknown>
    body?: unknown
    headers?: Record<string, string>
    auth?: 'required' | 'optional' | 'none'
  }
): Promise<T> {
  const config = useRuntimeConfig(event)
  const authMode = options?.auth || 'none'

  const execute = async (authorization?: string): Promise<T> => {
    const headers = {
      ...(options?.headers || {})
    }
    if (authorization) {
      headers.authorization = authorization
    }

    return await $fetch<T>(joinURL(config.gatewayBase, path), {
      method: options?.method,
      query: options?.query,
      body: options?.body as BodyInit | Record<string, any> | null | undefined,
      headers: Object.keys(headers).length ? headers : undefined
    })
  }

  let accessToken = ''
  if (authMode !== 'none') {
    accessToken = await ensureAccessToken(event, { required: authMode === 'required' })
  }

  try {
    return await execute(accessToken ? `Bearer ${accessToken}` : '')
  } catch (error: unknown) {
    const status = extractErrorStatus(error)
    if (status !== 401 || authMode === 'none') {
      throw error
    }

    const refreshedToken = await ensureAccessToken(event, {
      required: authMode === 'required',
      forceRefresh: true
    })

    if (!refreshedToken && authMode === 'optional') {
      return await execute('')
    }

    return await execute(refreshedToken ? `Bearer ${refreshedToken}` : '')
  }
}
