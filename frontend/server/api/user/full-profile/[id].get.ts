import { gatewayFetch } from '../../../utils/auth'

export default defineEventHandler(async (event): Promise<unknown> => {
  const { id } = getRouterParams(event)

  if (!id) {
    throw createError({ statusCode: 400, statusMessage: '缺少用户 ID' })
  }

  return await gatewayFetch(event, `/v1/profiles/${id}`, {
    method: 'GET',
    auth: 'optional'
  })
})
