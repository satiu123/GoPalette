import { joinURL } from 'ufo'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const { id } = getRouterParams(event)
  const authorization = getHeader(event, 'authorization')
  const body = await readBody(event)
  const query = getQuery(event)

  if (!id) {
    throw createError({ statusCode: 400, statusMessage: 'id is required' })
  }

  return await $fetch(joinURL(config.gatewayBase, `/v1/users/${id}`), {
    method: 'PATCH',
    query: {
      updateMask: query.updateMask || 'username,email,avatarURL'
    },
    headers: authorization ? { authorization } : undefined,
    body
  })
})
