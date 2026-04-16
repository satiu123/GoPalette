export interface AuthUser {
  id: string
  username: string
  email: string
  role?: number
  status?: number
  avatarURL?: string
  createdAt?: string
  updatedAt?: string
}

interface AuthSession {
  accessToken: string
  refreshToken: string
  userId: string
}

interface AuthTokenResponse {
  accessToken?: string
  refreshToken?: string
  access_token?: string
  refresh_token?: string
}

interface AuthRequestOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'
  query?: Record<string, unknown>
  body?: Record<string, unknown> | BodyInit | null
  headers?: Record<string, string>
}

const STORAGE_KEY = 'gopalette.auth.session'

function decodeJwtPayload(token: string) {
  try {
    const part = token.split('.')[1]
    if (!part) return null

    const normalized = part.replace(/-/g, '+').replace(/_/g, '/')
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
    const json = atob(padded)

    return JSON.parse(json) as Record<string, unknown>
  } catch {
    return null
  }
}

function extractUserIdFromToken(token: string) {
  const payload = decodeJwtPayload(token)
  if (!payload) return ''

  const candidate = payload.userId || payload.user_id || payload.sub || payload.id
  if (typeof candidate === 'string') return candidate
  if (typeof candidate === 'number' && Number.isFinite(candidate)) return String(Math.trunc(candidate))

  return ''
}

function isUnauthorizedError(error: unknown) {
  if (!error || typeof error !== 'object') return false

  const typed = error as { status?: unknown, statusCode?: unknown, response?: { status?: unknown } }
  const status = Number(typed.status ?? typed.statusCode ?? typed.response?.status ?? 0)

  return status === 401
}

export function useAuth() {
  const { csrf, headerName } = useCsrf()

  const session = useState<AuthSession>('auth.session', () => ({
    accessToken: '',
    refreshToken: '',
    userId: ''
  }))

  const user = useState<AuthUser | null>('auth.user', () => null)
  const initialized = useState<boolean>('auth.initialized', () => false)
  const refreshingPromise = useState<Promise<boolean> | null>('auth.refreshing.promise', () => null)

  function saveSession(next: AuthSession) {
    session.value = next

    if (import.meta.client) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
    }
  }

  function clearSession() {
    session.value = {
      accessToken: '',
      refreshToken: '',
      userId: ''
    }
    user.value = null

    if (import.meta.client) {
      localStorage.removeItem(STORAGE_KEY)
    }
  }

  function initAuth() {
    if (initialized.value || !import.meta.client) return

    initialized.value = true
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return

    try {
      const parsed = JSON.parse(raw) as Partial<AuthSession>
      const accessToken = parsed.accessToken || ''
      const refreshToken = parsed.refreshToken || ''
      const userId = parsed.userId || extractUserIdFromToken(accessToken)

      session.value = {
        accessToken,
        refreshToken,
        userId
      }
    } catch {
      clearSession()
    }
  }

  function withCsrfHeaders(headers?: Record<string, string>) {
    const csrfToken = unref(csrf)
    const name = unref(headerName)

    if (!csrfToken || !name) {
      return headers
    }

    return {
      ...(headers || {}),
      [name]: csrfToken
    }
  }

  function withAuthHeaders(headers?: Record<string, string>) {
    if (!session.value.accessToken) {
      return headers
    }

    return {
      ...(headers || {}),
      authorization: `Bearer ${session.value.accessToken}`
    }
  }

  function normalizeTokenResponse(response: AuthTokenResponse) {
    return {
      accessToken: response.accessToken || response.access_token || '',
      refreshToken: response.refreshToken || response.refresh_token || ''
    }
  }

  async function refreshTokens() {
    if (!session.value.refreshToken) {
      return false
    }

    if (refreshingPromise.value) {
      return await refreshingPromise.value
    }

    refreshingPromise.value = (async () => {
      try {
        const response = await $fetch<AuthTokenResponse>('/api/user/refresh', {
          method: 'POST',
          headers: withCsrfHeaders(),
          body: {
            refreshToken: session.value.refreshToken
          }
        })

        const next = normalizeTokenResponse(response)
        if (!next.accessToken || !next.refreshToken) {
          return false
        }

        const userId = extractUserIdFromToken(next.accessToken) || session.value.userId
        saveSession({
          accessToken: next.accessToken,
          refreshToken: next.refreshToken,
          userId
        })

        return true
      } catch (error: unknown) {
        if (isUnauthorizedError(error)) {
          clearSession()
        }
        return false
      } finally {
        refreshingPromise.value = null
      }
    })()

    return await refreshingPromise.value
  }

  async function authFetch<T>(url: string, options: AuthRequestOptions = {}, retry = true): Promise<T> {
    async function execute() {
      return await $fetch(url, {
        ...options,
        headers: withAuthHeaders(options.headers)
      }) as T
    }

    try {
      return await execute()
    } catch (error: unknown) {
      if (!retry || !isUnauthorizedError(error)) {
        throw error
      }

      const refreshed = await refreshTokens()
      if (!refreshed) {
        throw error
      }

      return await execute()
    }
  }

  async function login(payload: { email: string; password: string }) {
    const response = await $fetch<AuthTokenResponse>('/api/user/login', {
      method: 'POST',
      headers: withCsrfHeaders(),
      body: payload
    })

    const { accessToken, refreshToken } = normalizeTokenResponse(response)

    if (!accessToken || !refreshToken) {
      throw new Error('登录失败：服务端未返回令牌')
    }

    const userId = extractUserIdFromToken(accessToken)

    saveSession({
      accessToken,
      refreshToken,
      userId
    })

    if (userId) {
      await fetchProfile(userId)
    }

    return response
  }

  async function register(payload: { username: string; email: string; password: string }) {
    return await $fetch<{ user?: AuthUser }>('/api/user/register', {
      method: 'POST',
      headers: withCsrfHeaders(),
      body: payload
    })
  }

  async function fetchProfile(id?: string) {
    const userId = id || session.value.userId || extractUserIdFromToken(session.value.accessToken)
    if (!userId) {
      return null
    }

    if (!session.value.userId) {
      saveSession({
        ...session.value,
        userId
      })
    }

    const response = await authFetch<{ user?: AuthUser }>(`/api/user/profile/${userId}`, {
      method: 'GET'
    })

    user.value = response.user || null
    return user.value
  }

  async function updateProfile(payload: { username: string; email: string; avatarURL?: string }) {
    const userId = session.value.userId
    if (!userId) {
      throw new Error('当前未登录，无法更新个人信息')
    }

    const response = await authFetch<{ user?: AuthUser }>(`/api/user/profile/${userId}`, {
      method: 'PATCH',
      query: {
        updateMask: 'username,email,avatarURL'
      },
      headers: withCsrfHeaders(),
      body: {
        id: userId,
        username: payload.username,
        email: payload.email,
        avatarURL: payload.avatarURL || ''
      }
    })

    user.value = response.user || user.value
    return user.value
  }

  return {
    session,
    user,
    initialized,
    isLoggedIn: computed(() => Boolean(session.value.accessToken)),
    initAuth,
    login,
    register,
    refreshTokens,
    authFetch,
    fetchProfile,
    updateProfile,
    clearSession
  }
}
