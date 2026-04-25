import { joinURL } from 'ufo'
import { clearAuthCookies, ensureAccessToken, readSessionHint } from '../../utils/auth'

export default defineEventHandler(async (event): Promise<{ success: true }> => {
  const config = useRuntimeConfig(event)
  const session = readSessionHint(event)

  try {
    const accessToken = await ensureAccessToken(event, { required: false })
    if (accessToken && session.userId) {
      await $fetch(joinURL(config.gatewayBase, '/v1/users/logout'), {
        method: 'POST',
        headers: {
          authorization: `Bearer ${accessToken}`
        },
        body: {
          userId: Number(session.userId)
        }
      })
    }
  } catch {
    // Best effort only; local cookie cleanup is the hard guarantee.
  }

  clearAuthCookies(event)
  return { success: true }
})
