import { joinURL } from 'ufo'

export default defineEventHandler(async (event) => {
    const config = useRuntimeConfig(event)
    const query = getQuery(event)

    return await $fetch(joinURL(config.gatewayBase, '/v1/comments'), {
        method: 'GET',
        query: {
            postId: query.postId || '',
            page: query.page || '1',
            pageSize: query.pageSize || '50'
        }
    })
})
