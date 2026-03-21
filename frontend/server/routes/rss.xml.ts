interface FeedArticle {
  id: number
  slug?: string
  title?: string
  summary?: string
  content?: string
  created_at?: string
  updated_at?: string
}

interface ArticleListResponse {
  code: number
  msg: string
  data?: {
    total?: number
    articles?: FeedArticle[]
  }
}

function xmlEscape(input: string): string {
  return input
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

function plainText(input: string): string {
  return input.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const siteUrl = String(config.public.siteUrl || 'http://localhost:3000').replace(/\/$/, '')
  const apiBase = String(config.public.apiBase || '/api')

  const response = await $fetch<ArticleListResponse>(`${apiBase}/articles`, {
    query: { page: 1, page_size: 100 }
  })

  const articles = response?.data?.articles ?? []

  const items = articles.map((article) => {
    const identifier = encodeURIComponent((article.slug || String(article.id)).trim())
    const link = `${siteUrl}/post/${identifier}`
    const pubDate = new Date(article.created_at || article.updated_at || Date.now()).toUTCString()
    const title = xmlEscape(article.title || 'Untitled')
    const description = xmlEscape((article.summary || plainText(article.content || '')).slice(0, 240))

    return `  <item>\n    <title>${title}</title>\n    <link>${xmlEscape(link)}</link>\n    <guid>${xmlEscape(link)}</guid>\n    <description>${description}</description>\n    <pubDate>${xmlEscape(pubDate)}</pubDate>\n  </item>`
  })

  const now = new Date().toUTCString()
  const xml = `<?xml version="1.0" encoding="UTF-8"?>\n<rss version="2.0">\n<channel>\n  <title>GoPalette</title>\n  <link>${xmlEscape(siteUrl)}</link>\n  <description>GoPalette technical blog feed</description>\n  <language>zh-cn</language>\n  <lastBuildDate>${xmlEscape(now)}</lastBuildDate>\n${items.join('\n')}\n</channel>\n</rss>`

  setHeader(event, 'Content-Type', 'application/rss+xml; charset=utf-8')
  setHeader(event, 'Cache-Control', 'public, max-age=600')
  return xml
})
