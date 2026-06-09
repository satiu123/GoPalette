import { gatewayFetch } from '../../../utils/auth'

export default defineEventHandler(async (event): Promise<unknown> => {
  const { id } = getRouterParams(event)
  if (!id) {
    throw createError({ statusCode: 400, statusMessage: '缺少文章 ID' })
  }

  return await gatewayFetch(event, `/v1/posts/${id}/like`, {
    method: 'POST',
    auth: 'required',
    body: {
      id
    }
  })
})
