<script setup lang="ts">
const slots = useSlots()
</script>

<template>
  <UHeader
    :toggle="{ class: 'md:hidden' }"
    :ui="{
      root: 'border-b border-default bg-default/95 backdrop-blur',
      container: 'gap-3 px-4 py-3 sm:px-14!',
      left: 'min-w-0 items-center gap-3',
      center: 'hidden min-w-0 flex-1 justify-center px-2 md:flex',
      right: 'items-center justify-end gap-1 sm:gap-2'
    }"
  >
    <template #left>
      <NuxtLink
        to="/"
        class="flex shrink-0 items-center gap-2"
      >
        <span class="text-lg font-semibold tracking-tight text-highlighted sm:text-xl">GoPalette</span>
      </NuxtLink>

      <div class="hidden md:flex">
        <TemplateMenu />
      </div>
    </template>

    <template v-if="slots.search">
      <div class="w-full max-w-md">
        <slot name="search" />
      </div>
    </template>

    <template #right>
      <div
        v-if="slots.default"
        class="flex items-center gap-1 sm:gap-2"
      >
        <slot />
      </div>

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

    <template #body>
      <div class="space-y-3 p-4 md:hidden">
        <TemplateMenu vertical />
      </div>
    </template>

    <template v-if="slots.search" #bottom>
      <div class="border-t border-default px-4 pb-3 pt-2 md:hidden">
        <slot name="search" />
      </div>
    </template>
  </UHeader>
</template>
