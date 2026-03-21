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
  const updated = articles.length
    ? new Date(articles[0]?.updated_at || articles[0]?.created_at || Date.now()).toISOString()
    : new Date().toISOString()

  const entries = articles.map((article) => {
    const identifier = encodeURIComponent((article.slug || String(article.id)).trim())
    const link = `${siteUrl}/post/${identifier}`
    const publishTime = new Date(article.created_at || article.updated_at || Date.now()).toISOString()
    const title = xmlEscape(article.title || 'Untitled')
    const summary = xmlEscape((article.summary || plainText(article.content || '')).slice(0, 240))

    return `  <entry>\n    <title>${title}</title>\n    <id>${xmlEscape(link)}</id>\n    <link href="${xmlEscape(link)}" />\n    <updated>${xmlEscape(publishTime)}</updated>\n    <summary>${summary}</summary>\n  </entry>`
  })

  const xml = `<?xml version="1.0" encoding="utf-8"?>\n<feed xmlns="http://www.w3.org/2005/Atom">\n  <title>GoPalette</title>\n  <id>${xmlEscape(siteUrl)}</id>\n  <updated>${xmlEscape(updated)}</updated>\n  <link href="${xmlEscape(siteUrl)}" />\n  <link href="${xmlEscape(siteUrl)}/atom.xml" rel="self" />\n${entries.join('\n')}\n</feed>`

  setHeader(event, 'Content-Type', 'application/atom+xml; charset=utf-8')
  setHeader(event, 'Cache-Control', 'public, max-age=600')
  return xml
})
