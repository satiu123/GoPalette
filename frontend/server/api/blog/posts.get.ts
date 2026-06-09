import { gatewayFetch } from '../../utils/auth'

export default defineEventHandler(async (event) => {
  const query = getQuery(event)

  return await gatewayFetch(event, '/v1/blog/posts', {
    method: 'GET',
    auth: 'none',
    query: {
      page: query.page || '1',
      pageSize: query.pageSize || '20'
    }
  })
})
