<script setup lang="ts">
import { deleteComment, fetchComments, fetchPosts, searchPosts } from '~/composables/useBlogApi'
import type { BlogPostItem, CommentInfo } from '~/composables/useBlogApi'

const toast = useToast()
const { initAuth, isLoggedIn } = useAuth()

const postKeyword = ref('')
const postPage = ref(1)
const pageSize = 10
const postLoading = ref(false)
const postTotal = ref(0)
const postRows = ref<BlogPostItem[]>([])

const commentPostId = ref('')
const commentLoading = ref(false)
const commentRows = ref<CommentInfo[]>([])
const commentTotal = ref(0)
const deletingCommentId = ref('')

function toCover(seed: string) {
    return `https://picsum.photos/seed/${encodeURIComponent(seed || 'gopalette')}/1200/640`
}

async function loadPosts() {
    postLoading.value = true

    try {
        const query = postKeyword.value.trim()
        if (query) {
            const response = await searchPosts(query, postPage.value, pageSize)
            postTotal.value = response.total
            postRows.value = response.results.map(item => ({
                id: item.id,
                title: item.title,
                summary: item.summary,
                slug: item.slug,
                status: 1,
                tags: item.tags,
                category: item.categoryName,
                categoryId: '',
                author: '未知作者',
                publishedAt: item.createdAt || '未知时间',
                readingMinutes: Math.max(1, Math.ceil(item.summary.length / 300)),
                cover: toCover(item.slug || item.id)
            }))
            return
        }

        const response = await fetchPosts(postPage.value, pageSize)
        postTotal.value = response.total
        postRows.value = response.posts
    } catch (error: any) {
        toast.add({
            title: '加载文章失败',
            description: error?.data?.message || error?.message || '请稍后重试',
            color: 'error'
        })
    } finally {
        postLoading.value = false
    }
}

async function loadComments() {
    const postId = commentPostId.value.trim()
    if (!postId) {
        commentRows.value = []
        commentTotal.value = 0
        return
    }

    commentLoading.value = true
    try {
        const response = await fetchComments(postId, 1, 100)
        commentRows.value = response.comments
        commentTotal.value = response.total
    } catch (error: any) {
        toast.add({
            title: '加载评论失败',
            description: error?.data?.message || error?.message || '请检查文章 ID 是否正确',
            color: 'error'
        })
    } finally {
        commentLoading.value = false
    }
}

async function removeComment(id: string) {
    if (!id) return

    deletingCommentId.value = id
    try {
        await deleteComment(id)
        toast.add({ title: '评论已删除', color: 'success' })
        await loadComments()
    } catch (error: any) {
        toast.add({
            title: '删除评论失败',
            description: error?.data?.message || error?.message || '请稍后重试',
            color: 'error'
        })
    } finally {
        deletingCommentId.value = ''
    }
}

const totalPages = computed(() => Math.max(1, Math.ceil(postTotal.value / pageSize)))

watch(postPage, () => {
    loadPosts()
})

onMounted(async () => {
    initAuth()

    if (!isLoggedIn.value) {
        await navigateTo('/login?redirect=/admin')
        return
    }

    await loadPosts()
})

useSeoMeta({
    title: '后台管理',
    description: 'GoPalette 管理后台，支持文章检索与评论审核。'
})
</script>

<template>
    <div class="min-h-screen bg-default">
        <AppHeader />

        <main class="mx-auto w-full max-w-6xl px-4 pb-20 pt-10 sm:px-14">
            <div class="mb-6 flex flex-wrap items-end justify-between gap-4">
                <div>
                    <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
                        后台管理
                    </h1>
                    <p class="mt-1 text-sm text-toned">
                        文章检索、编辑入口与评论审核
                    </p>
                </div>
            </div>

            <section class="rounded-2xl border border-default bg-default p-5 sm:p-6">
                <div class="flex flex-wrap items-end gap-3">
                    <UFormField label="文章关键词" class="min-w-56 flex-1">
                        <UInput v-model="postKeyword" placeholder="按标题/摘要检索" icon="i-lucide-search" />
                    </UFormField>

                    <UButton :loading="postLoading" label="查询文章" icon="i-lucide-filter" @click="loadPosts" />
                </div>

                <div class="mt-5 space-y-3">
                    <article v-for="item in postRows" :key="item.id" class="rounded-xl border border-default p-4">
                        <div class="flex flex-wrap items-start justify-between gap-3">
                            <div>
                                <p class="text-base font-semibold text-highlighted">
                                    {{ item.title }}
                                </p>
                                <p class="mt-1 text-xs text-toned">
                                    ID: {{ item.id }} · 分类: {{ item.category }}
                                </p>
                            </div>

                            <UButton :to="`/write?slug=${item.slug}`" size="xs" color="primary" variant="soft"
                                icon="i-lucide-square-pen" label="编辑" />
                        </div>
                    </article>

                    <UAlert v-if="!postLoading && postRows.length === 0" title="暂无文章" description="请更换关键词或稍后重试。"
                        icon="i-lucide-file-text" color="neutral" variant="soft" />
                </div>

                <div class="mt-5 flex items-center justify-end gap-2" v-if="postRows.length > 0">
                    <UButton size="xs" color="neutral" variant="soft" icon="i-lucide-chevron-left"
                        :disabled="postPage <= 1" @click="postPage = Math.max(1, postPage - 1)" />
                    <span class="text-xs text-toned">第 {{ postPage }} / {{ totalPages }} 页</span>
                    <UButton size="xs" color="neutral" variant="soft" icon="i-lucide-chevron-right"
                        :disabled="postPage >= totalPages" @click="postPage = Math.min(totalPages, postPage + 1)" />
                </div>
            </section>

            <section class="mt-8 rounded-2xl border border-default bg-default p-5 sm:p-6">
                <div class="flex flex-wrap items-end gap-3">
                    <UFormField label="评论审核（按文章 ID）" class="min-w-56 flex-1">
                        <UInput v-model="commentPostId" placeholder="输入文章 ID 拉取评论" icon="i-lucide-message-circle" />
                    </UFormField>

                    <UButton :loading="commentLoading" label="查询评论" icon="i-lucide-list-filter" color="neutral"
                        @click="loadComments" />
                </div>

                <p class="mt-4 text-xs text-toned">
                    共 {{ commentTotal || commentRows.length }} 条评论
                </p>

                <div class="mt-4 space-y-3">
                    <article v-for="item in commentRows" :key="item.id" class="rounded-xl border border-default p-4">
                        <div class="flex items-start justify-between gap-3">
                            <div>
                                <p class="text-sm font-medium text-highlighted">
                                    {{ item.author?.name || `用户 ${item.userId}` }}
                                </p>
                                <p class="mt-1 text-xs text-toned">评论ID: {{ item.id }}</p>
                            </div>

                            <UButton :loading="deletingCommentId === item.id" size="xs" color="error" variant="ghost"
                                icon="i-lucide-trash-2" label="删除" @click="removeComment(item.id)" />
                        </div>

                        <p class="mt-3 whitespace-pre-wrap text-sm text-toned">
                            {{ item.content }}
                        </p>
                    </article>

                    <UAlert v-if="!commentLoading && commentRows.length === 0" title="暂无评论数据"
                        description="输入文章 ID 后点击查询评论。" icon="i-lucide-message-square-off" color="neutral"
                        variant="soft" />
                </div>
            </section>
        </main>
    </div>
</template>
