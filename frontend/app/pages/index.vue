<script setup lang="ts">
const { featuredPost, regularPosts } = useBlogData()
</script>

<template>
  <div>
    <AppHero :post="featuredPost" />
    <PostGrid :posts="regularPosts" />
  </div>
</template>
