import { ensureAccessToken, readSessionHint } from '../../utils/auth'

export default defineEventHandler(async (event): Promise<{ success: true, loggedIn: boolean, userId: string }> => {
  await ensureAccessToken(event, { required: true, forceRefresh: true })
  return {
    success: true,
    ...readSessionHint(event)
  }
})
