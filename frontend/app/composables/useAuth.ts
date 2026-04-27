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
  loggedIn: boolean
  userId: string
}

interface AuthRequestOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'
  query?: Record<string, unknown>
  body?: Record<string, unknown> | BodyInit | null
  headers?: Record<string, string>
}

interface SessionReply {
  loggedIn?: boolean
  userId?: string
  user_id?: string
}

const LOGGED_IN_COOKIE = 'gopalette_logged_in'
const USER_ID_COOKIE = 'gopalette_user_id'

function isUnauthorizedError(error: unknown) {
  if (!error || typeof error !== 'object') return false

  const typed = error as { status?: unknown, statusCode?: unknown, response?: { status?: unknown } }
  const status = Number(typed.status ?? typed.statusCode ?? typed.response?.status ?? 0)

  return status === 401
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

function normalizeSessionReply(input?: SessionReply | null): AuthSession {
  return {
    loggedIn: Boolean(input?.loggedIn),
    userId: String(input?.userId || input?.user_id || '')
  }
}

export function useAuth() {
  const loggedInCookie = useCookie<string | null>(LOGGED_IN_COOKIE)
  const userIdCookie = useCookie<string | null>(USER_ID_COOKIE)

  const session = useState<AuthSession>('auth.session', () => ({
    loggedIn: loggedInCookie.value === '1' && Boolean(userIdCookie.value),
    userId: String(userIdCookie.value || '')
  }))
  const user = useState<AuthUser | null>('auth.user', () => null)
  const initialized = useState<boolean>('auth.initialized', () => false)

  function withCsrfHeaders(headers?: Record<string, string>) {
    const { csrf, headerName } = useCsrf()
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

  function syncSessionFromCookies() {
    const next: AuthSession = {
      loggedIn: loggedInCookie.value === '1' && Boolean(userIdCookie.value),
      userId: String(userIdCookie.value || '')
    }

    session.value = next
    if (!next.userId) {
      user.value = null
    }
    return next
  }

  function saveSession(next: AuthSession) {
    session.value = {
      loggedIn: Boolean(next.loggedIn && next.userId),
      userId: String(next.userId || '')
    }

    loggedInCookie.value = session.value.loggedIn ? '1' : ''
    userIdCookie.value = session.value.userId || ''

    if (!session.value.userId) {
      user.value = null
    }
  }

  function clearSession() {
    saveSession({
      loggedIn: false,
      userId: ''
    })
  }

  function initAuth() {
    if (initialized.value) return

    initialized.value = true
    syncSessionFromCookies()
  }

  async function refreshSession() {
    try {
      const response = await $fetch<SessionReply>('/api/user/refresh', {
        method: 'POST',
        headers: withCsrfHeaders()
      })

      const next = normalizeSessionReply(response)
      if (!next.loggedIn || !next.userId) {
        clearSession()
        return false
      }

      saveSession(next)
      return true
    } catch {
      clearSession()
      return false
    }
  }

  async function authFetch<T>(url: string, options: AuthRequestOptions = {}, retry = true): Promise<T> {
    try {
      return await $fetch<T>(url, options)
    } catch (error: unknown) {
      if (!retry || !isUnauthorizedError(error)) {
        throw error
      }

      const refreshed = await refreshSession()
      if (!refreshed) {
        throw error
      }

      return await $fetch<T>(url, options)
    }
  }

  async function login(payload: { email: string, password: string }) {
    const response = await $fetch<SessionReply>('/api/user/login', {
      method: 'POST',
      headers: withCsrfHeaders(),
      body: payload
    })

    const next = normalizeSessionReply({
      loggedIn: true,
      userId: response.userId || response.user_id
    })
    if (!next.userId) {
      throw new Error('登录失败：服务端未返回用户信息')
    }

    saveSession(next)
    await fetchProfile(next.userId)

    return response
  }

  async function logout() {
    try {
      await $fetch('/api/user/logout', {
        method: 'POST',
        headers: withCsrfHeaders()
      })
    } finally {
      clearSession()
    }
  }

  async function register(payload: { username: string, email: string, password: string }) {
    return await $fetch<{ user?: AuthUser }>('/api/user/register', {
      method: 'POST',
      headers: withCsrfHeaders(),
      body: payload
    })
  }

  async function fetchProfile(id?: string) {
    const userId = String(id || session.value.userId || '')
    if (!userId) {
      return null
    }

    const response = await authFetch<{ user?: Record<string, unknown> }>(`/api/user/profile/${userId}`, {
      method: 'GET'
    })

    user.value = normalizeAuthUser(response.user)
    if (user.value?.id) {
      saveSession({
        loggedIn: true,
        userId: user.value.id
      })
    }
    return user.value
  }

  async function updateProfile(payload: { username: string, email: string, avatarURL?: string }) {
    const userId = String(session.value.userId || user.value?.id || '')
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
    isLoggedIn: computed(() => Boolean(session.value.loggedIn && session.value.userId)),
    isAdmin,
    initAuth,
    login,
    logout,
    register,
    refreshTokens: refreshSession,
    authFetch,
    fetchProfile,
    updateProfile,
    clearSession
  }
}
