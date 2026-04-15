export interface PostInfo {
    id: string
    title: string
    summary: string
    slug: string
    status: number
    viewCount?: string
    likeCount?: string
    commentCount?: string
    author?: {
        id?: string
        name?: string
    }
    category?: {
        id?: string
        name?: string
    }
    tags?: string[]
    createdAt?: string
    updatedAt?: string
}

export interface PostDetail {
    info?: PostInfo
    content?: string
    originalContent?: string
}

export interface BlogPostItem {
    id: string
    title: string
    summary: string
    slug: string
    status: number
    tags: string[]
    category: string
    categoryId: string
    author: string
    publishedAt: string
    readingMinutes: number
    cover: string
    content?: string
}

export const POST_STATUS_DRAFT = 0
export const POST_STATUS_PUBLISHED = 1
export const POST_STATUS_ARCHIVED = 2

export function isVisiblePostStatus(status: number | string) {
    const value = Number(status)
    if (!Number.isFinite(value)) return false

    return value === POST_STATUS_PUBLISHED || value === POST_STATUS_ARCHIVED
}

function formatDate(input?: string) {
    if (!input) return '未知时间'

    const date = new Date(input)
    if (Number.isNaN(date.getTime())) return '未知时间'

    return new Intl.DateTimeFormat('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
    }).format(date)
}

function estimateReadingMinutes(content: string, summary: string) {
    const text = `${content} ${summary}`.trim()
    const words = text.length

    return Math.max(1, Math.ceil(words / 300))
}

function buildCover(seed: string) {
    const safeSeed = encodeURIComponent(seed || 'gopalette')
    return `https://picsum.photos/seed/${safeSeed}/1200/640`
}

type PostInfoLike = PostInfo | { info?: PostInfo } | Record<string, any>

interface AuthSessionLike {
    accessToken?: string
}

function withCsrfHeaders(headers?: Record<string, string>) {
    const { csrf, headerName } = useCsrf()
    const token = unref(csrf)
    const name = unref(headerName)

    if (!token || !name) {
        return headers
    }

    return {
        ...(headers || {}),
        [name]: token
    }
}

function withAuthHeaders(headers?: Record<string, string>) {
    const session = useState<AuthSessionLike>('auth.session', () => ({ accessToken: '' }))
    const accessToken = session.value?.accessToken || ''

    if (!accessToken) {
        return headers
    }

    return {
        ...(headers || {}),
        authorization: `Bearer ${accessToken}`
    }
}

function unwrapPostInfo(post: PostInfoLike): PostInfo {
    const withInfo = post as { info?: PostInfo }
    const raw = withInfo?.info && typeof withInfo.info === 'object' ? withInfo.info : post

    return (raw || {}) as PostInfo
}

export function normalizePostInfo(post: PostInfoLike): BlogPostItem {
    const raw = unwrapPostInfo(post)

    return {
        id: raw.id || '',
        title: raw.title || '未命名文章',
        summary: raw.summary || '暂无摘要',
        slug: raw.slug || '',
        status: Number(raw.status || 0),
        tags: raw.tags || [],
        category: raw.category?.name || '未分类',
        categoryId: raw.category?.id || '',
        author: raw.author?.name || '匿名作者',
        publishedAt: formatDate(raw.createdAt),
        readingMinutes: estimateReadingMinutes('', raw.summary || ''),
        cover: buildCover(raw.slug || raw.id || 'gopalette')
    }
}

export function normalizePostDetail(detail: PostDetail): BlogPostItem {
    const info = detail.info || ({} as PostInfo)
    const content = detail.content || detail.originalContent || ''

    return {
        id: info.id || '',
        title: info.title || '未命名文章',
        summary: info.summary || '暂无摘要',
        slug: info.slug || '',
        status: info.status || POST_STATUS_DRAFT,
        tags: info.tags || [],
        category: info.category?.name || '未分类',
        categoryId: info.category?.id || '',
        author: info.author?.name || '匿名作者',
        publishedAt: formatDate(info.createdAt),
        readingMinutes: estimateReadingMinutes(content, info.summary || ''),
        cover: buildCover(info.slug || info.id || 'gopalette'),
        content
    }
}

export async function fetchPosts(page = 1, pageSize = 30): Promise<{ posts: BlogPostItem[]; total: number }> {
    const response = await $fetch<any>('/api/blog/posts', {
        query: {
            page,
            pageSize
        }
    })

    const sourcePosts = (response?.posts || response?.items || response?.data?.posts || []) as PostInfoLike[]
    const sourceTotal = response?.total || response?.data?.total || 0
    const posts = sourcePosts.map((item: PostInfoLike) => normalizePostInfo(item))

    return {
        posts,
        total: Number(sourceTotal || 0)
    }
}

export async function fetchTags(page = 1, pageSize = 200): Promise<{ tags: Array<{ id: string; name: string }>; total: number }> {
    const response = await $fetch<{ tags?: Array<{ id: string; name: string }>; total?: string }>('/api/blog/tags', {
        query: {
            page,
            pageSize
        }
    })

    return {
        tags: response.tags || [],
        total: Number(response.total || 0)
    }
}

export async function fetchCategories(page = 1, pageSize = 200): Promise<{ categories: Array<{ id: string; name: string }>; total: number }> {
    const response = await $fetch<{ categories?: Array<{ id: string; name: string }>; total?: string }>('/api/blog/categories', {
        query: {
            page,
            pageSize
        }
    })

    return {
        categories: response.categories || [],
        total: Number(response.total || 0)
    }
}

export async function fetchPostBySlug(slug: string): Promise<BlogPostItem | null> {
    const response = await $fetch<any>(`/api/blog/posts/${slug}`)
    const post = response?.post || response?.data?.post || response?.data || response
    if (!post) return null

    return normalizePostDetail(post)
}

export async function createPost(payload: {
    title: string
    summary: string
    slug: string
    content: string
    status: number
    categoryId?: string
    tags?: string[]
}): Promise<BlogPostItem | null> {
    const response = await $fetch<any>('/api/blog/posts', {
        method: 'POST',
        headers: withCsrfHeaders(withAuthHeaders()),
        body: payload
    })

    const post = response?.post || response?.data?.post || response?.data || response

    return post ? normalizePostDetail(post) : null
}

export async function updatePost(id: string, payload: {
    title: string
    summary: string
    slug: string
    content: string
    status: number
    categoryId?: string
    tags?: string[]
    updateMask?: string
}): Promise<BlogPostItem | null> {
    const response = await $fetch<any>(`/api/blog/posts/${id}`, {
        method: 'PATCH',
        headers: withCsrfHeaders(withAuthHeaders()),
        body: {
            id,
            ...payload,
            updateMask: payload.updateMask || 'title,summary,slug,content,status,categoryId,tags'
        }
    })

    const post = response?.post || response?.data?.post || response?.data || response

    return post ? normalizePostDetail(post) : null
}
