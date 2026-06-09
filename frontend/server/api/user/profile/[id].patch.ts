import { gatewayFetch } from '../../../utils/auth'

export default defineEventHandler(async (event): Promise<unknown> => {
  const { id } = getRouterParams(event)
  const body = await readBody(event)
  const query = getQuery(event)

  if (!id) {
    throw createError({ statusCode: 400, statusMessage: '缺少用户 ID' })
  }

  return await gatewayFetch(event, `/v1/users/${id}`, {
    method: 'PATCH',
    query: {
      updateMask: query.updateMask || 'username,email,avatarURL'
    },
    auth: 'required',
    body
  })
})
