<script setup lang="ts">
const slots = useSlots()
</script>

<template>
  <UHeader
    :toggle="false"
    :ui="{
      container: 'sm:px-14!',
      right: 'justify-end-safe overflow-x-auto py-2'
    }"
  >
    <template #left>
      <NuxtLink to="/">
        <AppLogo class="w-auto h-6 shrink-0" />
      </NuxtLink>

      <TemplateMenu />
    </template>

    <template #right>
      <slot />

      <USeparator
        v-if="slots.default"
        orientation="vertical"
        class="h-7"
      />

      <div
        role="group"
        class="flex items-center gap-0.5"
      >
        <UColorModeButton size="sm" />

        <UButton
          icon="i-simple-icons-github"
          color="neutral"
          variant="ghost"
          size="sm"
          to="https://github.com/satiu123/GoPalette"
          target="_blank"
        />
      </div>
    </template>
  </UHeader>
</template>
