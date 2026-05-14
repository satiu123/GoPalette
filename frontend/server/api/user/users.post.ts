import { gatewayFetch } from '../../utils/auth'

export default defineEventHandler(async (event): Promise<unknown> => {
  const body = await readBody(event)

  return await gatewayFetch(event, '/v1/users', {
    method: 'POST',
    auth: 'required',
    body
  })
})
