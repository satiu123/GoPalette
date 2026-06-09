import { gatewayFetch } from '../../utils/auth'

export default defineEventHandler(async (event) => {
  const body = await readBody(event)

  return await gatewayFetch(event, '/v1/users/register', {
    method: 'POST',
    auth: 'none',
    body
  })
})
