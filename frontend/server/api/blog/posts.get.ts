import { joinURL } from 'ufo'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const query = getQuery(event)

  return await $fetch(joinURL(config.gatewayBase, '/v1/blog/posts'), {
    method: 'GET',
    query: {
      page: query.page || '1',
      pageSize: query.pageSize || '20'
    }
  })
})
