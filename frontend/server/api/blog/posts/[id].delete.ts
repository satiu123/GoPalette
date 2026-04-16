import { joinURL } from 'ufo'

export default defineEventHandler(async (event) => {
    const config = useRuntimeConfig(event)
    const { id } = getRouterParams(event)
    const authorization = getHeader(event, 'authorization')

    if (!id) {
        throw createError({ statusCode: 400, statusMessage: 'id is required' })
    }

    return await $fetch(joinURL(config.gatewayBase, `/v1/posts/${id}`), {
        method: 'DELETE',
        headers: authorization ? { authorization } : undefined
    })
})
