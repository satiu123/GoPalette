import { joinURL } from 'ufo'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const query = getQuery(event)

  return await $fetch(joinURL(config.gatewayBase, '/v1/search/posts'), {
    method: 'GET',
    query: {
      query: query.query || '',
      page: query.page || '1',
      pageSize: query.pageSize || query.page_size || '20',
      page_size: query.page_size || query.pageSize || '20'
    }
  })
})
