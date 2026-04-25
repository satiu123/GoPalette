import { joinURL } from 'ufo'
import { escapeXml, fetchSeoPosts, getSeoSiteConfig } from '../utils/seo'

export default defineEventHandler(async (event) => {
  const { siteName, siteUrl } = getSeoSiteConfig(event)
  const posts = await fetchSeoPosts(event)
  const latestPosts = posts.slice(0, 50)

  setHeader(event, 'content-type', 'application/rss+xml; charset=utf-8')

  const items = latestPosts.map((post) => {
    const link = joinURL(siteUrl, `/posts/${encodeURIComponent(post.slug)}`)
    const pubDate = new Date(post.updatedAt || post.createdAt || Date.now()).toUTCString()

    return [
      '<item>',
      `<title>${escapeXml(post.title)}</title>`,
      `<link>${escapeXml(link)}</link>`,
      `<guid>${escapeXml(link)}</guid>`,
      `<description>${escapeXml(post.summary || post.title)}</description>`,
      `<author>${escapeXml(post.authorName)}</author>`,
      `<pubDate>${escapeXml(pubDate)}</pubDate>`,
      '</item>'
    ].join('')
  }).join('')

  return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>${escapeXml(siteName)}</title>
    <link>${escapeXml(siteUrl)}</link>
    <description>${escapeXml(`${siteName} 最新文章订阅`)}</description>
    <language>zh-CN</language>
    ${items}
  </channel>
</rss>`
})
