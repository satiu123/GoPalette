import { joinURL } from 'ufo'
import { gatewayFetch } from '../../utils/auth'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const query = getQuery(event)
  const postId = String(query.postId || '').trim()

  if (!postId) {
    return await gatewayFetch(event, '/v1/comments', {
      method: 'GET',
      auth: 'required',
      query: {
        page: query.page || '1',
        pageSize: query.pageSize || '50'
      }
    })
  }

  return await $fetch(joinURL(config.gatewayBase, '/v1/comments'), {
    method: 'GET',
    query: {
      postId,
      page: query.page || '1',
      pageSize: query.pageSize || '50'
    }
  })
})
