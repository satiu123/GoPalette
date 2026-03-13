<script setup lang="ts">
import { Search, Menu, User, PenSquare, LogOut, LayoutDashboard } from 'lucide-vue-next'
import { userAvatarUrl } from '~/composables/useBlogData'

const router = useRouter()
const { isLoggedIn, user, logout } = useAuth()

const searchQuery = ref('')
const showSearch  = ref(false)

function navigate(to: 'home' | 'write') {
  if (to === 'home') router.push('/')
  else router.push('/write')
}

function doSearch() {
  const q = searchQuery.value.trim()
  if (q) router.push(`/?q=${encodeURIComponent(q)}`)
  showSearch.value = false
  searchQuery.value = ''
}

function handleLogout() {
  logout()
  router.push('/')
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
          GoPalette.
        </button>
      </div>

      <!-- 搜索框（展开态） -->
      <div v-if="showSearch" class="flex-1 mx-6">
        <form @submit.prevent="doSearch" class="flex items-center bg-m3-sys-light-surface-variant rounded-full px-5 py-2 gap-3">
          <Search class="w-5 h-5 text-m3-sys-light-on-surface-variant shrink-0" />
          <input
            v-model="searchQuery"
            autofocus
            placeholder="Search articles…"
            class="flex-1 bg-transparent focus:outline-none text-m3-sys-light-on-surface"
            @keydown.esc="showSearch = false"
          />
          <button type="button" @click="showSearch = false" class="text-m3-sys-light-on-surface-variant hover:text-m3-sys-light-on-surface text-sm">✕</button>
        </form>
      </div>

      <nav v-else class="hidden md:flex items-center gap-8">
        <button @click="navigate('home')" class="font-medium text-m3-sys-light-on-surface hover:text-m3-sys-light-primary transition-colors">Home</button>
        <button @click="navigate('home')" class="font-medium text-m3-sys-light-on-surface hover:text-m3-sys-light-primary transition-colors">Articles</button>
        <button @click="navigate('write')" class="font-medium text-m3-sys-light-on-surface hover:text-m3-sys-light-primary transition-colors">Write</button>
      </nav>

      <div class="flex items-center gap-2">
        <button
          @click="showSearch = !showSearch"
          class="p-3 rounded-full hover:bg-m3-sys-light-surface-variant transition-colors text-m3-sys-light-on-surface"
        >
          <Search class="w-6 h-6" />
        </button>

        <button
          @click="navigate('write')"
          class="hidden sm:flex items-center gap-2 px-6 py-3 bg-m3-sys-light-secondary-container text-m3-sys-light-on-secondary-container rounded-full font-medium hover:bg-m3-sys-light-secondary hover:text-m3-sys-light-on-secondary transition-all duration-300"
        >
          <PenSquare class="w-5 h-5" />
          <span>Write</span>
        </button>

        <!-- 已登录：显示用户名 + Profile + Admin(if admin) + 登出 -->
        <template v-if="isLoggedIn">
          <NuxtLink
            to="/profile"
            class="hidden sm:flex items-center gap-2 px-4 py-2 rounded-full hover:bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface font-medium transition-colors"
          >
            <img
              :src="userAvatarUrl(user)"
              :alt="user?.username"
              class="w-6 h-6 rounded-full object-cover"
              referrerpolicy="no-referrer"
            />
            <span>{{ user?.username }}</span>
          </NuxtLink>
          <NuxtLink
            v-if="user?.role === 'admin'"
            to="/admin"
            class="hidden sm:flex items-center gap-2 px-4 py-2 bg-m3-sys-light-tertiary-container text-m3-sys-light-on-tertiary-container rounded-full font-medium hover:opacity-90 transition-opacity"
          >
            <LayoutDashboard class="w-4 h-4" />
            <span>Admin</span>
          </NuxtLink>
          <button
            @click="handleLogout"
            class="hidden sm:flex items-center gap-2 px-5 py-2 bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant rounded-full font-medium hover:bg-m3-sys-light-error-container hover:text-m3-sys-light-on-error-container transition-all duration-300"
          >
            <LogOut class="w-4 h-4" />
          </button>
        </template>

        <!-- 未登录：Sign In -->
        <NuxtLink
          v-else
          to="/login"
          class="hidden sm:flex items-center gap-2 px-6 py-3 bg-m3-sys-light-primary-container text-m3-sys-light-on-primary-container rounded-full font-medium hover:bg-m3-sys-light-primary hover:text-m3-sys-light-on-primary transition-all duration-300"
        >
          <User class="w-5 h-5" />
          <span>Sign In</span>
        </NuxtLink>
      </div>
    </div>
  </header>
</template>


