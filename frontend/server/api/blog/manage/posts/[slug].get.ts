import { gatewayFetch } from '../../../../utils/auth'

export default defineEventHandler(async (event): Promise<unknown> => {
  const params = getRouterParams(event)
  const pathname = getRequestURL(event).pathname
  const tail = pathname.split('/').filter(Boolean).pop() || ''
  const slug = params.slug || decodeURIComponent(tail)

  if (!slug) {
    throw createError({ statusCode: 400, statusMessage: '缺少文章路径' })
  }

  return await gatewayFetch(event, `/v1/posts/slug/${slug}`, {
    method: 'GET',
    auth: 'required'
  })
})
