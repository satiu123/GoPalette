<script setup lang="ts">
const route = useRoute()

const { data, error } = await useArticle(route.params.id as string)

if (error.value || !data.value?.data) {
  await navigateTo('/')
}

const post = computed(() => data.value?.data ?? null)
</script>

<template>
  <div v-if="post">
    <PostDetail :post="post" />
  </div>
</template>
