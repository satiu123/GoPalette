import { gatewayFetch } from '../../../utils/auth'

export default defineEventHandler(async (event): Promise<unknown> => {
  const { id } = getRouterParams(event)
  const body = await readBody(event)

  if (!id) {
    throw createError({ statusCode: 400, statusMessage: 'id is required' })
  }

  return await gatewayFetch(event, `/v1/comments/${id}`, {
    method: 'PATCH',
    auth: 'required',
    body: {
      id,
      ...body
    }
  })
})
