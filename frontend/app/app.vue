<script setup lang="ts">
const { fetchMe } = useAuth()
// 客户端刷新时从后端恢复用户信息（非阻塞）
if (import.meta.client) fetchMe()
</script>

<template>
  <div class="min-h-screen bg-m3-sys-light-background text-m3-sys-light-on-background font-sans selection:bg-m3-sys-light-primary-container selection:text-m3-sys-light-on-primary-container">
    <NuxtRouteAnnouncer />
    <AppHeader />
    <main>
      <NuxtPage />
    </main>
    <AppFooter />
    <AppConfirmDialog />
  </div>
</template>
