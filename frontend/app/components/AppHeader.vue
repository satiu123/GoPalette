<script setup lang="ts">
const props = defineProps<{
  compact?: boolean
}>()

const slots = useSlots()
</script>

<template>
  <UHeader
    :toggle="{ class: 'md:hidden' }"
    :ui="{
      root: props.compact
        ? 'sticky top-0 z-50 border-b border-default bg-default/96 transition-colors duration-200'
        : 'border-b border-default bg-default/96 transition-colors duration-200',
      container: 'gap-3 px-4 py-3 sm:px-14!',
      left: 'min-w-0 items-center gap-3',
      center: 'hidden min-w-0 flex-1 justify-center px-2 md:flex',
      right: 'items-center justify-end gap-1 sm:gap-2'
    }"
  >
    <template #left>
      <NuxtLink
        to="/"
        class="motion-link flex shrink-0 items-center gap-2"
        :class="props.compact ? 'max-[360px]:hidden' : ''"
      >
        <span class="text-lg font-semibold tracking-tight text-highlighted sm:text-xl">GoPalette</span>
      </NuxtLink>

      <div class="hidden md:flex">
        <TemplateMenu />
      </div>
    </template>

    <template v-if="slots.search && !props.compact">
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
        v-if="slots.default && !props.compact"
        orientation="vertical"
        class="h-7"
      />

      <div
        role="group"
        class="flex items-center gap-0.5"
        :class="props.compact ? 'hidden sm:flex' : ''"
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

    <template
      v-if="slots.search"
      #bottom
    >
      <div class="border-t border-default bg-default/96 px-4 pb-3 pt-2 shadow-sm backdrop-blur md:hidden">
        <slot name="search" />
      </div>
    </template>
  </UHeader>
</template>
