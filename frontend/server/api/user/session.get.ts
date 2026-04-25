import { readSessionHint } from '../../utils/auth'

export default defineEventHandler(async (event): Promise<{ loggedIn: boolean, userId: string }> => {
  return readSessionHint(event)
})
