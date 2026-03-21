interface FeedArticle {
  id: number
  slug?: string
  updated_at?: string
  created_at?: string
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

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const siteUrl = String(config.public.siteUrl || 'http://localhost:3000').replace(/\/$/, '')
  const apiBase = String(config.public.apiBase || '/api')

  const response = await $fetch<ArticleListResponse>(`${apiBase}/articles`, {
    query: { page: 1, page_size: 500 }
  })

  const articles = response?.data?.articles ?? []

  const urls = [
    `  <url>\n    <loc>${xmlEscape(siteUrl)}/</loc>\n    <changefreq>daily</changefreq>\n    <priority>1.0</priority>\n  </url>`
  ]

  for (const article of articles) {
    const identifier = encodeURIComponent((article.slug || String(article.id)).trim())
    const updatedAt = article.updated_at || article.created_at
    const lastmod = updatedAt ? `\n    <lastmod>${xmlEscape(new Date(updatedAt).toISOString())}</lastmod>` : ''
    urls.push(
      `  <url>\n    <loc>${xmlEscape(siteUrl)}/post/${identifier}</loc>${lastmod}\n    <changefreq>weekly</changefreq>\n    <priority>0.8</priority>\n  </url>`
    )
  }

  const xml = `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls.join('\n')}\n</urlset>`

  setHeader(event, 'Content-Type', 'application/xml; charset=utf-8')
  setHeader(event, 'Cache-Control', 'public, max-age=600')
  return xml
})
