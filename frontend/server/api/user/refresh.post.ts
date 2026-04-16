import { joinURL } from 'ufo'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const body = await readBody<Record<string, unknown>>(event)
  const refreshToken = String(body?.refreshToken || body?.refresh_token || '').trim()

  if (!refreshToken) {
    throw createError({
      statusCode: 400,
      statusMessage: 'refreshToken is required'
    })
  }

  return await $fetch(joinURL(config.gatewayBase, '/v1/users/refresh'), {
    method: 'POST',
    body: {
      refreshToken
    }
  })
})
