export interface AuthUser {
  id: string
  username: string
  email: string
  role?: number | string
  status?: number | string
  avatarURL?: string
  createdAt?: string
  updatedAt?: string
  bio?: string
  location?: string
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
    if (typeof globalThis.atob !== 'function') return null
    const json = globalThis.atob(padded)

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

function extractTokenExpiresAt(token: string) {
  const payload = decodeJwtPayload(token)
  if (!payload) return 0

  const exp = payload.exp
  if (typeof exp === 'number' && Number.isFinite(exp)) return exp
  if (typeof exp === 'string') {
    const value = Number(exp)
    return Number.isFinite(value) ? value : 0
  }

  return 0
}

function shouldRefreshSoon(token: string, advanceSeconds = 45) {
  if (!token) return false
  const exp = extractTokenExpiresAt(token)
  if (!exp) return false

  const now = Math.floor(Date.now() / 1000)
  return exp <= now + advanceSeconds
}

function normalizeAuthUser(input?: Record<string, unknown> | null): AuthUser | null {
  if (!input || typeof input !== 'object') return null

  return {
    id: String(input.id || ''),
    username: String(input.username || ''),
    email: String(input.email || ''),
    role: typeof input.role === 'string' || typeof input.role === 'number' ? input.role : undefined,
    status: typeof input.status === 'string' || typeof input.status === 'number' ? input.status : undefined,
    avatarURL: String(input.avatarURL || input.avatarUrl || input.avatar_u_r_l || input.avatar_url || ''),
    createdAt: typeof input.createdAt === 'string'
      ? input.createdAt
      : (typeof input.created_at === 'string' ? input.created_at : ''),
    updatedAt: typeof input.updatedAt === 'string'
      ? input.updatedAt
      : (typeof input.updated_at === 'string' ? input.updated_at : ''),
    bio: String(input.bio || ''),
    location: String(input.location || '')
  }
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
    const derivedUserId = extractUserIdFromToken(next.accessToken)
    session.value = {
      accessToken: next.accessToken || '',
      refreshToken: next.refreshToken || '',
      userId: derivedUserId || next.userId || ''
    }

    if (import.meta.client) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(session.value))
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
      const userId = extractUserIdFromToken(accessToken) || parsed.userId || ''

      saveSession({
        accessToken,
        refreshToken,
        userId
      })
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

  async function ensureAccessTokenFresh() {
    if (!session.value.accessToken || !session.value.refreshToken) return
    if (!shouldRefreshSoon(session.value.accessToken)) return
    await refreshTokens()
  }

  function getCurrentUserId() {
    const tokenUserId = extractUserIdFromToken(session.value.accessToken)
    if (tokenUserId && session.value.userId !== tokenUserId) {
      saveSession({
        ...session.value,
        userId: tokenUserId
      })
    }
    return tokenUserId || session.value.userId
  }

  async function authFetch<T>(url: string, options: AuthRequestOptions = {}, retry = true): Promise<T> {
    async function execute() {
      return await $fetch(url, {
        ...options,
        headers: withAuthHeaders(options.headers)
      }) as T
    }

    try {
      await ensureAccessTokenFresh()
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
    const currentUserId = getCurrentUserId()
    const userId = id && id === currentUserId ? id : currentUserId
    if (!userId) {
      return null
    }

    const response = await authFetch<{ user?: Record<string, unknown> }>(`/api/user/profile/${userId}`, {
      method: 'GET'
    })

    user.value = normalizeAuthUser(response.user)
    return user.value
  }

  async function updateProfile(payload: { username: string; email: string; avatarURL?: string }) {
    const userId = getCurrentUserId()
    if (!userId) {
      throw new Error('当前未登录，无法更新个人信息')
    }

    const response = await authFetch<{ user?: Record<string, unknown> }>(`/api/user/profile/${userId}`, {
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

    user.value = normalizeAuthUser(response.user) || user.value
    return user.value
  }

  const isAdmin = computed(() => {
    const role = user.value?.role
    if (role === undefined || role === null) return false
    if (typeof role === 'number') return role === 1

    return String(role).toUpperCase() === 'ADMIN'
  })

  return {
    session,
    user,
    initialized,
    isLoggedIn: computed(() => Boolean(session.value.accessToken)),
    isAdmin,
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
