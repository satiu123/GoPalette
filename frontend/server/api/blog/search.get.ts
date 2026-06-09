import { gatewayFetch } from '../../utils/auth'

export default defineEventHandler(async (event) => {
  const query = getQuery(event)

  return await gatewayFetch(event, '/v1/search/posts', {
    method: 'GET',
    auth: 'none',
    query: {
      query: query.query || '',
      page: query.page || '1',
      pageSize: query.pageSize || query.page_size || '20',
      page_size: query.page_size || query.pageSize || '20'
    }
  })
})
