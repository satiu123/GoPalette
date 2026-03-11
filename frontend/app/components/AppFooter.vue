<script setup lang="ts">
import { Send, Twitter, Github, Linkedin } from 'lucide-vue-next'
</script>

<template>
  <footer class="mt-24 px-4 sm:px-6 lg:px-8 pb-8 max-w-7xl mx-auto">
    <div
      v-motion
      :initial="{ opacity: 0, y: 40 }"
      :visible-once="{ opacity: 1, y: 0, transition: { duration: 800, ease: [0.22, 1, 0.36, 1] } }"
      class="bg-m3-sys-light-secondary-container text-m3-sys-light-on-secondary-container rounded-[3rem] p-12 sm:p-20 relative overflow-hidden"
    >
      <div class="absolute top-0 right-0 w-96 h-96 bg-m3-sys-light-tertiary-container rounded-full mix-blend-multiply filter blur-3xl opacity-50 animate-blob"></div>
      <div class="absolute bottom-0 left-0 w-96 h-96 bg-m3-sys-light-primary-container rounded-full mix-blend-multiply filter blur-3xl opacity-50 animate-blob animation-delay-2000"></div>

      <div class="relative z-10 grid grid-cols-1 lg:grid-cols-2 gap-16 items-center">
        <div>
          <h2 class="text-5xl sm:text-6xl font-black tracking-tighter leading-[1.1] mb-6">
            Stay in the loop.
          </h2>
          <p class="text-xl opacity-80 leading-relaxed max-w-md">
            Get the latest articles, design resources, and updates delivered straight to your inbox. No spam, ever.
          </p>

          <form class="mt-10 flex flex-col sm:flex-row gap-4 max-w-md" @submit.prevent>
            <input
              type="email"
              placeholder="Enter your email"
              class="flex-grow px-6 py-4 rounded-full bg-m3-sys-light-surface text-m3-sys-light-on-surface placeholder:text-m3-sys-light-on-surface-variant focus:outline-none focus:ring-4 focus:ring-m3-sys-light-primary/20 transition-all shadow-inner"
              required
            />
            <button
              type="submit"
              class="flex items-center justify-center gap-2 px-8 py-4 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full font-bold hover:bg-m3-sys-light-on-primary-container transition-colors shadow-lg shadow-m3-sys-light-primary/20 whitespace-nowrap"
            >
              Subscribe
              <Send class="w-5 h-5" />
            </button>
          </form>
        </div>

        <div class="flex flex-col gap-12 lg:items-end">
          <div class="flex gap-8">
            <a href="#" class="p-4 bg-m3-sys-light-surface rounded-full hover:bg-m3-sys-light-primary hover:text-m3-sys-light-on-primary transition-all duration-300 shadow-sm hover:shadow-xl hover:-translate-y-1">
              <Twitter class="w-6 h-6" />
            </a>
            <a href="#" class="p-4 bg-m3-sys-light-surface rounded-full hover:bg-m3-sys-light-primary hover:text-m3-sys-light-on-primary transition-all duration-300 shadow-sm hover:shadow-xl hover:-translate-y-1">
              <Github class="w-6 h-6" />
            </a>
            <a href="#" class="p-4 bg-m3-sys-light-surface rounded-full hover:bg-m3-sys-light-primary hover:text-m3-sys-light-on-primary transition-all duration-300 shadow-sm hover:shadow-xl hover:-translate-y-1">
              <Linkedin class="w-6 h-6" />
            </a>
          </div>

          <div class="flex flex-col gap-4 lg:text-right">
            <a href="#" class="text-lg font-medium hover:text-m3-sys-light-primary transition-colors">About Us</a>
            <a href="#" class="text-lg font-medium hover:text-m3-sys-light-primary transition-colors">Privacy Policy</a>
            <a href="#" class="text-lg font-medium hover:text-m3-sys-light-primary transition-colors">Terms of Service</a>
          </div>
        </div>
      </div>

      <div class="relative z-10 mt-20 pt-8 border-t border-current/10 flex flex-col sm:flex-row justify-between items-center gap-4">
        <p class="font-bold tracking-tight text-2xl">Expressive.</p>
        <p class="opacity-60 text-sm">© 2026 Expressive Blog. All rights reserved.</p>
      </div>
    </div>
  </footer>
</template>
