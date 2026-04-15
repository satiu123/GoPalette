import { joinURL } from 'ufo'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const body = await readBody(event)

  return await $fetch(joinURL(config.gatewayBase, '/v1/users/register'), {
    method: 'POST',
    body
  })
})
