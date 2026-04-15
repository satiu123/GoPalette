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
  return typeof candidate === 'string' ? candidate : ''
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
      session.value = {
        accessToken: parsed.accessToken || '',
        refreshToken: parsed.refreshToken || '',
        userId: parsed.userId || ''
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

  async function login(payload: { email: string; password: string }) {
    const response = await $fetch<{ accessToken?: string; refreshToken?: string }>('/api/user/login', {
      method: 'POST',
      headers: withCsrfHeaders(),
      body: payload
    })

    const accessToken = response.accessToken || ''
    const refreshToken = response.refreshToken || ''

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
    const userId = id || session.value.userId
    if (!userId) {
      return null
    }

    const response = await $fetch<{ user?: AuthUser }>(`/api/user/profile/${userId}`, {
      headers: session.value.accessToken
        ? {
          authorization: `Bearer ${session.value.accessToken}`
        }
        : undefined
    })

    user.value = response.user || null
    return user.value
  }

  async function updateProfile(payload: { username: string; email: string; avatarURL?: string }) {
    const userId = session.value.userId
    if (!userId) {
      throw new Error('当前未登录，无法更新个人信息')
    }

    const response = await $fetch<{ user?: AuthUser }>(`/api/user/profile/${userId}`, {
      method: 'PATCH',
      query: {
        updateMask: 'username,email,avatarURL'
      },
      headers: withCsrfHeaders(
        session.value.accessToken
          ? {
            authorization: `Bearer ${session.value.accessToken}`
          }
          : undefined
      ),
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
    fetchProfile,
    updateProfile,
    clearSession
  }
}
