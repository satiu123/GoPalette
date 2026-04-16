import { joinURL } from 'ufo'

export default defineEventHandler(async (event) => {
    const config = useRuntimeConfig(event)
    const { id } = getRouterParams(event)
    const body = await readBody(event)
    const authorization = getHeader(event, 'authorization')

    if (!id) {
        throw createError({ statusCode: 400, statusMessage: 'id is required' })
    }

    return await $fetch(joinURL(config.gatewayBase, `/v1/categories/${id}`), {
        method: 'PATCH',
        headers: authorization ? { authorization } : undefined,
        body
    })
})
