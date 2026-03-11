<script setup lang="ts">
const route = useRoute()
const { getPostById } = useBlogData()

const post = computed(() => getPostById(route.params.id as string))

if (!post.value) {
  await navigateTo('/')
}
</script>

<template>
  <div v-if="post">
    <PostDetail :post="post" />
  </div>
</template>
