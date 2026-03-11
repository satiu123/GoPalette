import type { ApiUser, ApiResponse } from '~/composables/useBlogData'

interface TokenData {
  access_token: string
  refresh_token: string
}

export function useAuth() {
  const config = useRuntimeConfig()

  // useCookie 在 SSR + 客户端均可访问，避免 hydration 不一致
  const accessToken = useCookie<string | null>('gp_access', {
    maxAge: 60 * 15,      // 15 分钟
    sameSite: 'lax',
    secure: false         // 开发环境 http
  })
  const refreshToken = useCookie<string | null>('gp_refresh', {
    maxAge: 60 * 60 * 24 * 7, // 7 天
    sameSite: 'lax',
    secure: false
  })

  // 当前登录用户信息（仅客户端持久化）
  const user = useState<ApiUser | null>('auth.user', () => null)

  const isLoggedIn = computed(() => !!accessToken.value)

  /** 构造带 Authorization 头的 headers */
  function authHeaders(): HeadersInit {
    return accessToken.value ? { Authorization: accessToken.value } : {}
  }

  /** 登录 */
  async function login(username: string, password: string): Promise<void> {
    const res = await $fetch<ApiResponse<TokenData>>(`${config.public.apiBase}/login`, {
      method: 'POST',
      body: { username, password }
    })
    if (res.code !== 200) throw new Error(res.msg)
    accessToken.value = res.data.access_token
    refreshToken.value = res.data.refresh_token
    user.value = { id: 0, username, role: 'user' } // 简单回填，重载后从 token 解析
  }

  /** 注册 */
  async function register(username: string, password: string): Promise<void> {
    const res = await $fetch<ApiResponse<unknown>>(`${config.public.apiBase}/register`, {
      method: 'POST',
      body: { username, password }
    })
    if (res.code !== 200) throw new Error(res.msg)
  }

  /** 使用 refresh token 换取新双令牌 */
  async function refresh(): Promise<boolean> {
    if (!refreshToken.value) return false
    try {
      const res = await $fetch<ApiResponse<TokenData>>(`${config.public.apiBase}/refresh`, {
        method: 'POST',
        body: { refresh_token: refreshToken.value }
      })
      if (res.code === 200) {
        accessToken.value = res.data.access_token
        refreshToken.value = res.data.refresh_token
        return true
      }
    } catch { /* ignore */ }
    return false
  }

  /** 注销 */
  function logout() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
  }

  /**
   * 带自动 token 刷新的 $fetch 封装。
   * 401 时尝试刷新一次，若仍失败则抛出错误。
   */
  async function authFetch<T>(url: string, options: Parameters<typeof $fetch>[1] = {}): Promise<T> {
    const fullUrl = url.startsWith('http') ? url : `${config.public.apiBase}${url}`
    try {
      return await $fetch<T>(fullUrl, {
        ...options,
        headers: { ...authHeaders(), ...(options?.headers as Record<string, string> ?? {}) }
      })
    } catch (err: unknown) {
      const status = (err as { status?: number })?.status
      if (status === 401) {
        const ok = await refresh()
        if (ok) {
          return $fetch<T>(fullUrl, {
            ...options,
            headers: { ...authHeaders(), ...(options?.headers as Record<string, string> ?? {}) }
          })
        }
        logout()
      }
      throw err
    }
  }

  return { user, isLoggedIn, accessToken, login, register, refresh, logout, authFetch }
}
