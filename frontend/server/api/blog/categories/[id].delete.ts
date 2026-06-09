import { gatewayFetch } from '../../../utils/auth'

export default defineEventHandler(async (event): Promise<unknown> => {
  const { id } = getRouterParams(event)

  if (!id) {
    throw createError({ statusCode: 400, statusMessage: '缺少分类 ID' })
  }

  return await gatewayFetch(event, `/v1/categories/${id}`, {
    method: 'DELETE',
    auth: 'required'
  })
})
