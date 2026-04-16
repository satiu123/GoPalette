import { joinURL } from 'ufo'

export default defineEventHandler(async (event) => {
    const config = useRuntimeConfig(event)
    const body = await readBody(event)
    const authorization = getHeader(event, 'authorization')

    return await $fetch(joinURL(config.gatewayBase, '/v1/categories'), {
        method: 'POST',
        headers: authorization ? { authorization } : undefined,
        body
    })
})
