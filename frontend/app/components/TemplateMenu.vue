<script setup lang="ts">
const route = useRoute()
const { isLoggedIn, isAdmin, initAuth, session, user, fetchProfile } = useAuth()

onMounted(() => {
  initAuth()
  if (session.value.userId && !user.value) {
    void fetchProfile(session.value.userId)
  }
})

const items = computed(() => {
  const common = [
    { label: '首页', to: '/' },
    { label: '文章', to: '/posts' },
    { label: '写作', to: '/write' }
  ]

  if (isLoggedIn.value) {
    return isAdmin.value
      ? [...common, { label: '管理', to: '/admin' }, { label: '我的', to: '/profile' }]
      : [...common, { label: '我的', to: '/profile' }]
  }

  return [...common, { label: '登录', to: '/login' }, { label: '注册', to: '/register' }]
})

function isActive(path: string) {
  if (path === '/') {
    return route.path === '/'
  }

  return route.path.startsWith(path)
}
</script>

<template>
  <div class="flex items-center gap-1 rounded-full bg-muted/60 p-1">
    <UButton v-for="item in items" :key="item.to" :to="item.to" :prefetch="false" size="xs" :label="item.label" class="rounded-full"
      :variant="isActive(item.to) ? 'solid' : 'ghost'" :color="isActive(item.to) ? 'primary' : 'neutral'" />
  </div>
</template>
