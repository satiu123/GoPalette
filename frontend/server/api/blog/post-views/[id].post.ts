import { gatewayFetch, readSessionHint } from '../../../utils/auth'

const VIEWER_COOKIE = 'gopalette_viewer_id'
const VIEWER_COOKIE_MAX_AGE = 60 * 60 * 24 * 365

function isProd() {
  return !import.meta.dev
}

function bytesToHex(buffer: ArrayBuffer) {
  return Array.from(new Uint8Array(buffer))
    .map(byte => byte.toString(16).padStart(2, '0'))
    .join('')
}

async function hashText(value: string) {
  const encoded = new TextEncoder().encode(value)
  return bytesToHex(await globalThis.crypto.subtle.digest('SHA-256', encoded))
}

function createViewerId() {
  if (globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID()
  }

  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 14)}`
}

function isLikelyBot(userAgent: string) {
  const text = userAgent.toLowerCase()
  if (!text) return false

  return [
    'bot',
    'crawler',
    'spider',
    'slurp',
    'bingpreview',
    'facebookexternalhit',
    'whatsapp',
    'telegrambot',
    'discordbot',
    'preview',
    'lighthouse'
  ].some(token => text.includes(token))
}

function ensureViewerId(event: Parameters<typeof getCookie>[0]) {
  const existing = getCookie(event, VIEWER_COOKIE)
  if (existing) return existing

  const next = createViewerId()
  setCookie(event, VIEWER_COOKIE, next, {
    path: '/',
    httpOnly: true,
    sameSite: 'lax',
    secure: isProd(),
    maxAge: VIEWER_COOKIE_MAX_AGE
  })
  return next
}

export default defineEventHandler(async (event): Promise<unknown> => {
  const { id } = getRouterParams(event)
  if (!id) {
    throw createError({ statusCode: 400, statusMessage: '缺少文章 ID' })
  }

  const userAgent = getRequestHeader(event, 'user-agent') || ''
  if (isLikelyBot(userAgent)) {
    return {
      counted: false,
      skipped: 'bot'
    }
  }

  const session = readSessionHint(event)
  const viewerSource = session.loggedIn && session.userId
    ? `user:${session.userId}`
    : `anon:${ensureViewerId(event)}`
  const userAgentHash = await hashText(userAgent)
  const viewerKey = await hashText(`${viewerSource}|ua:${userAgentHash.slice(0, 16)}`)

  return await gatewayFetch(event, `/v1/posts/${id}/view`, {
    method: 'POST',
    auth: 'optional',
    body: {
      viewerKey
    }
  })
})
