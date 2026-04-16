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

export interface CommentInfo {
    id: string
    postId: string
    userId: string
    content: string
    parentId?: string
    rootId?: string
    status?: number | string
    likeCount?: string
    author?: {
        id?: string
        name?: string
        avatarUrl?: string
    }
    createdAt?: string
    updatedAt?: string
    replies?: CommentInfo[]
}

function normalizeComment(input: Record<string, any>): CommentInfo {
    const repliesRaw = Array.isArray(input?.replies) ? input.replies : []

    return {
        id: String(input?.id || ''),
        postId: String(input?.postId || input?.post_id || ''),
        userId: String(input?.userId || input?.user_id || ''),
        content: String(input?.content || ''),
        parentId: String(input?.parentId || input?.parent_id || ''),
        rootId: String(input?.rootId || input?.root_id || ''),
        status: input?.status,
        likeCount: String(input?.likeCount || input?.like_count || '0'),
        author: input?.author
            ? {
                id: String(input.author.id || ''),
                name: String(input.author.name || ''),
                avatarUrl: String(input.author.avatarUrl || input.author.avatar_url || '')
            }
            : undefined,
        createdAt: String(input?.createdAt || input?.created_at || ''),
        updatedAt: String(input?.updatedAt || input?.updated_at || ''),
        replies: repliesRaw.map((reply: Record<string, any>) => normalizeComment(reply))
    }
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

export interface SearchPostItem {
    id: string
    title: string
    summary: string
    slug: string
    categoryName: string
    tags: string[]
    createdAt?: string
}

export const POST_STATUS_DRAFT = 0
export const POST_STATUS_PUBLISHED = 1
export const POST_STATUS_ARCHIVED = 2

function toPostStatus(value: number | string | undefined) {
    if (typeof value === 'number') return value

    const text = String(value || '').trim().toUpperCase()
    if (!text) return POST_STATUS_DRAFT
    if (text === 'PUBLISHED') return POST_STATUS_PUBLISHED
    if (text === 'ARCHIVED') return POST_STATUS_ARCHIVED
    if (text === 'DRAFT') return POST_STATUS_DRAFT

    const parsed = Number(text)
    return Number.isFinite(parsed) ? parsed : POST_STATUS_DRAFT
}

export function isVisiblePostStatus(status: number | string) {
    const value = toPostStatus(status)
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
        status: toPostStatus(raw.status),
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
        status: toPostStatus(info.status),
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

export async function searchPosts(keyword: string, page = 1, pageSize = 20): Promise<{ results: SearchPostItem[]; total: number; totalPages: number }> {
    const query = keyword.trim()
    if (!query) {
        return {
            results: [],
            total: 0,
            totalPages: 0
        }
    }

    const response = await $fetch<any>('/api/blog/search', {
        query: {
            query,
            page,
            pageSize,
            page_size: pageSize
        }
    })

    const items = (response?.results || response?.items || response?.data?.results || response?.data?.items || []) as Array<any>

    return {
        results: items.map(item => ({
            id: String(item?.id || ''),
            title: String(item?.title || '未命名文章'),
            summary: String(item?.summary || '暂无摘要'),
            slug: String(item?.slug || ''),
            categoryName: String(item?.categoryName || item?.category_name || '未分类'),
            tags: Array.isArray(item?.tags) ? item.tags.map((tag: any) => String(tag)) : [],
            createdAt: typeof item?.createdAt === 'string'
                ? item.createdAt
                : (typeof item?.created_at === 'string' ? item.created_at : '')
        })),
        total: Number(response?.total || response?.data?.total || 0),
        totalPages: Number(response?.totalPages || response?.total_pages || response?.data?.totalPages || response?.data?.total_pages || 0)
    }
}

export async function fetchComments(postId: string, page = 1, pageSize = 50): Promise<{ comments: CommentInfo[]; total: number }> {
    const response = await $fetch<any>('/api/blog/comments', {
        query: {
            postId,
            page,
            pageSize
        }
    })

    return {
        comments: (response?.comments || response?.data?.comments || []).map((item: Record<string, any>) => normalizeComment(item)),
        total: Number(response?.total || response?.data?.total || 0)
    }
}

export async function createComment(payload: { postId: string; content: string; parentId?: string }): Promise<CommentInfo | null> {
    const response = await $fetch<any>('/api/blog/comments', {
        method: 'POST',
        headers: withCsrfHeaders(withAuthHeaders()),
        body: payload
    })

    const comment = response?.comment || response?.data?.comment || response?.data || response
    return comment ? normalizeComment(comment as Record<string, any>) : null
}

export async function deleteComment(id: string): Promise<boolean> {
    const response = await $fetch<any>(`/api/blog/comments/${id}`, {
        method: 'DELETE',
        headers: withCsrfHeaders(withAuthHeaders())
    })

    return Boolean(response?.success ?? response?.data?.success ?? true)
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
