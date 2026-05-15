<script setup lang="ts">
const route = useRoute()
const { initAuth, isLoggedIn } = useAuth()
const { ensureLoaded, isReady, errorMessage } = useWriteResources()

const WriteEditorWorkspace = defineAsyncComponent(() => import('~/components/write/WriteEditorWorkspace.client.vue'))

const checkingAccess = ref(true)
const loadFailed = ref(false)

onMounted(async () => {
  initAuth()

  if (!isLoggedIn.value) {
    await navigateTo(`/login?redirect=${encodeURIComponent(route.fullPath)}`)
    return
  }

  const loaded = await ensureLoaded()
  loadFailed.value = !loaded
  checkingAccess.value = false
})

useSeoMeta({
  title: '写作工作台',
  description: 'GoPalette 富文本写作工作台，支持草稿保存与文章发布。'
})
</script>

<template>
  <div class="min-h-screen">
    <ClientOnly>
      <component
        :is="WriteEditorWorkspace"
        v-if="!checkingAccess && isLoggedIn && isReady"
      />

      <div
        v-else
        class="mx-auto flex min-h-screen w-full max-w-5xl items-start justify-center px-4 py-24 sm:px-8"
      >
        <div class="w-full max-w-3xl space-y-6">
          <div class="space-y-3">
            <div class="loading-shimmer h-8 w-40 rounded-full" />
            <div class="loading-shimmer h-4 w-72 rounded-full" />
          </div>

          <div class="rounded-3xl border border-default bg-default p-6 shadow-sm">
            <div class="mb-6 grid gap-4 md:grid-cols-2">
              <div class="loading-shimmer h-11 rounded-xl" />
              <div class="loading-shimmer h-11 rounded-xl" />
              <div class="loading-shimmer h-24 rounded-2xl md:col-span-2" />
            </div>

            <div class="space-y-3">
              <div class="loading-shimmer h-5 w-28 rounded-full" />
              <div class="loading-shimmer h-6 w-full rounded-full" />
              <div class="loading-shimmer h-6 w-11/12 rounded-full" />
              <div class="loading-shimmer h-6 w-10/12 rounded-full" />
            </div>
          </div>

          <UAlert
            v-if="loadFailed"
            color="error"
            variant="soft"
            title="写作页初始化失败"
            :description="errorMessage || '编辑器资源加载失败，请刷新后重试。'"
          />
        </div>
      </div>
    </ClientOnly>
  </div>
</template>
