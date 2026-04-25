import { joinURL } from 'ufo'
import type { H3Event } from 'h3'

interface SitemapPostRecord {
  id: string
  slug: string
  title: string
  summary: string
  createdAt: string
  updatedAt: string
  authorId: string
  authorName: string
}

interface SitemapCategoryRecord {
  id: string
  name: string
}

interface SitemapTagRecord {
  id: string
  name: string
}

function toRecord(value: unknown) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined
  return value as Record<string, unknown>
}

function toText(value: unknown) {
  if (value === undefined || value === null) return ''
  return String(value).trim()
}

function escapeXml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

export function getSeoSiteConfig(event: H3Event) {
  const config = useRuntimeConfig(event)
  return {
    siteUrl: String(config.public.siteUrl || 'http://127.0.0.1:3000').replace(/\/+$/, ''),
    siteName: String(config.public.siteName || 'GoPalette Blog')
  }
}

export async function fetchSeoPosts(event: H3Event) {
  const config = useRuntimeConfig(event)
  const response = await $fetch<Record<string, unknown>>(joinURL(config.gatewayBase, '/v1/blog/posts'), {
    method: 'GET',
    query: {
      page: '1',
      pageSize: '1000'
    }
  })

  const items = Array.isArray(response?.posts) ? response.posts : []

  return items.map((item) => {
    const record = toRecord(item) || {}
    const author = toRecord(record.author) || {}

    return {
      id: toText(record.id),
      slug: toText(record.slug),
      title: toText(record.title) || '未命名文章',
      summary: toText(record.summary),
      createdAt: toText(record.createdAt || record.created_at),
      updatedAt: toText(record.updatedAt || record.updated_at || record.createdAt || record.created_at),
      authorId: toText(author.id),
      authorName: toText(author.name) || '匿名作者'
    } satisfies SitemapPostRecord
  }).filter(item => item.slug)
}

export async function fetchSeoCategories(event: H3Event) {
  const config = useRuntimeConfig(event)
  const response = await $fetch<Record<string, unknown>>(joinURL(config.gatewayBase, '/v1/categories'), {
    method: 'GET',
    query: {
      page: '1',
      pageSize: '200'
    }
  })

  const items = Array.isArray(response?.categories) ? response.categories : []

  return items.map((item) => {
    const record = toRecord(item) || {}

    return {
      id: toText(record.id),
      name: toText(record.name)
    } satisfies SitemapCategoryRecord
  }).filter(item => item.name)
}

export async function fetchSeoTags(event: H3Event) {
  const config = useRuntimeConfig(event)
  const response = await $fetch<Record<string, unknown>>(joinURL(config.gatewayBase, '/v1/tags'), {
    method: 'GET',
    query: {
      page: '1',
      pageSize: '200'
    }
  })

  const items = Array.isArray(response?.tags) ? response.tags : []

  return items.map((item) => {
    const record = toRecord(item) || {}

    return {
      id: toText(record.id),
      name: toText(record.name)
    } satisfies SitemapTagRecord
  }).filter(item => item.name)
}

export { escapeXml }
