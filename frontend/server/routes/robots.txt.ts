import { joinURL } from 'ufo'
import { getSeoSiteConfig } from '../utils/seo'

export default defineEventHandler((event) => {
  const { siteUrl } = getSeoSiteConfig(event)

  setHeader(event, 'content-type', 'text/plain; charset=utf-8')
  setHeader(event, 'cache-control', 'public, max-age=3600')

  return [
    'User-agent: *',
    'Allow: /',
    `Sitemap: ${joinURL(siteUrl, '/sitemap.xml')}`
  ].join('\n')
})
