import { joinURL } from 'ufo'
import { escapeXml, fetchSeoPosts, getSeoSiteConfig } from '../utils/seo'

export default defineEventHandler(async (event) => {
  const { siteUrl } = getSeoSiteConfig(event)
  const posts = await fetchSeoPosts(event)
  const authorIds = Array.from(new Set(posts.map(post => post.authorId).filter(Boolean)))

  setHeader(event, 'content-type', 'application/xml; charset=utf-8')

  const staticUrls = [
    { loc: joinURL(siteUrl, '/'), lastmod: '' },
    { loc: joinURL(siteUrl, '/posts'), lastmod: '' },
    { loc: joinURL(siteUrl, '/rss.xml'), lastmod: '' }
  ]

  const postUrls = posts.map(post => ({
    loc: joinURL(siteUrl, `/posts/${encodeURIComponent(post.slug)}`),
    lastmod: post.updatedAt || post.createdAt
  }))

  const authorUrls = authorIds.map(authorId => ({
    loc: joinURL(siteUrl, `/authors/${encodeURIComponent(authorId)}`),
    lastmod: ''
  }))

  const nodes = [...staticUrls, ...postUrls, ...authorUrls].map((item) => {
    const parts = [`<loc>${escapeXml(item.loc)}</loc>`]
    if (item.lastmod) {
      const iso = new Date(item.lastmod).toISOString()
      parts.push(`<lastmod>${escapeXml(iso)}</lastmod>`)
    }
    return `<url>${parts.join('')}</url>`
  }).join('')

  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  ${nodes}
</urlset>`
})
