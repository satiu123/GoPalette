import { gatewayFetch } from '../../../utils/auth'

export default defineEventHandler(async (event): Promise<unknown> => {
  const query = getQuery(event)

  return await gatewayFetch(event, '/v1/posts', {
    method: 'GET',
    auth: 'required',
    query: {
      page: query.page || '1',
      pageSize: query.pageSize || '20'
    }
  })
})
