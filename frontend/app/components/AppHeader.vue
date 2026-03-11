<script setup lang="ts">
import { Search, Menu, User, PenSquare } from 'lucide-vue-next'

const router = useRouter()

function navigate(to: 'home' | 'write') {
  if (to === 'home') router.push('/')
  else router.push('/write')
}
</script>

<template>
  <header class="sticky top-0 z-50 bg-m3-sys-light-surface/80 backdrop-blur-md border-b border-m3-sys-light-surface-variant">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-20 flex items-center justify-between">
      <div class="flex items-center gap-4">
        <button class="p-3 rounded-full hover:bg-m3-sys-light-surface-variant transition-colors text-m3-sys-light-on-surface">
          <Menu class="w-6 h-6" />
        </button>
        <button
          @click="navigate('home')"
          class="text-2xl font-bold tracking-tight text-m3-sys-light-primary hover:opacity-80 transition-opacity"
        >
          Expressive.
        </button>
      </div>

      <nav class="hidden md:flex items-center gap-8">
        <button @click="navigate('home')" class="font-medium text-m3-sys-light-on-surface hover:text-m3-sys-light-primary transition-colors">Design</button>
        <button @click="navigate('home')" class="font-medium text-m3-sys-light-on-surface hover:text-m3-sys-light-primary transition-colors">Development</button>
        <button @click="navigate('home')" class="font-medium text-m3-sys-light-on-surface hover:text-m3-sys-light-primary transition-colors">Culture</button>
        <button @click="navigate('home')" class="font-medium text-m3-sys-light-on-surface hover:text-m3-sys-light-primary transition-colors">About</button>
      </nav>

      <div class="flex items-center gap-2">
        <button class="p-3 rounded-full hover:bg-m3-sys-light-surface-variant transition-colors text-m3-sys-light-on-surface">
          <Search class="w-6 h-6" />
        </button>

        <button
          @click="navigate('write')"
          class="hidden sm:flex items-center gap-2 px-6 py-3 bg-m3-sys-light-secondary-container text-m3-sys-light-on-secondary-container rounded-full font-medium hover:bg-m3-sys-light-secondary hover:text-m3-sys-light-on-secondary transition-all duration-300"
        >
          <PenSquare class="w-5 h-5" />
          <span>Write</span>
        </button>

        <button class="hidden sm:flex items-center gap-2 px-6 py-3 bg-m3-sys-light-primary-container text-m3-sys-light-on-primary-container rounded-full font-medium hover:bg-m3-sys-light-primary hover:text-m3-sys-light-on-primary transition-all duration-300">
          <User class="w-5 h-5" />
          <span>Sign In</span>
        </button>
      </div>
    </div>
  </header>
</template>
