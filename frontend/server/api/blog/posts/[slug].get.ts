import { joinURL } from 'ufo'

export default defineEventHandler(async (event) => {
    const config = useRuntimeConfig(event)
    const params = getRouterParams(event)
    const pathname = getRequestURL(event).pathname
    const tail = pathname.split('/').filter(Boolean).pop() || ''

    const slug = params.slug || decodeURIComponent(tail)

    if (!slug) {
        throw createError({ statusCode: 400, statusMessage: 'slug is required' })
    }

    return await $fetch(joinURL(config.gatewayBase, `/v1/posts/slug/${slug}`), {
        method: 'GET'
    })
})
