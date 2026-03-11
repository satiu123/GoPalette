<script setup lang="ts">
const { isLoggedIn, login, register } = useAuth()
const router = useRouter()

// 已登录直接跳首页
if (isLoggedIn.value) {
  await navigateTo('/')
}

const mode     = ref<'login' | 'register'>('login')
const username = ref('')
const password = ref('')
const loading  = ref(false)
const error    = ref('')

async function submit() {
  error.value = ''
  if (!username.value.trim() || !password.value) {
    error.value = '请填写用户名和密码'
    return
  }
  loading.value = true
  try {
    if (mode.value === 'login') {
      await login(username.value.trim(), password.value)
      router.push('/')
    } else {
      await register(username.value.trim(), password.value)
      // 注册成功后自动登录
      await login(username.value.trim(), password.value)
      router.push('/')
    }
  } catch (e: unknown) {
    error.value = (e as Error)?.message ?? (mode.value === 'login' ? '用户名或密码错误' : '注册失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-[80vh] flex items-center justify-center px-4">
    <div
      v-motion
      :initial="{ opacity: 0, scale: 0.96 }"
      :enter="{ opacity: 1, scale: 1, transition: { duration: 400, ease: [0.22, 1, 0.36, 1] } }"
      class="w-full max-w-md bg-m3-sys-light-surface-variant rounded-[2.5rem] p-10 shadow-xl"
    >
      <!-- 标题 -->
      <h1 class="text-4xl font-black tracking-tighter text-m3-sys-light-on-surface mb-2">
        {{ mode === 'login' ? 'Welcome back.' : 'Join us.' }}
      </h1>
      <p class="text-m3-sys-light-on-surface-variant mb-10">
        {{ mode === 'login' ? 'Sign in to your account.' : 'Create a new account.' }}
      </p>

      <form @submit.prevent="submit" class="space-y-5">
        <div>
          <label class="block text-sm font-semibold text-m3-sys-light-on-surface-variant mb-1.5">Username</label>
          <input
            v-model="username"
            type="text"
            autocomplete="username"
            placeholder="e.g. satiu123"
            class="w-full bg-m3-sys-light-surface text-m3-sys-light-on-surface rounded-2xl px-5 py-3.5 focus:outline-none focus:ring-4 focus:ring-m3-sys-light-primary/20 transition-all"
          />
        </div>
        <div>
          <label class="block text-sm font-semibold text-m3-sys-light-on-surface-variant mb-1.5">Password</label>
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            placeholder="••••••••"
            class="w-full bg-m3-sys-light-surface text-m3-sys-light-on-surface rounded-2xl px-5 py-3.5 focus:outline-none focus:ring-4 focus:ring-m3-sys-light-primary/20 transition-all"
          />
        </div>

        <p v-if="error" class="text-red-500 text-sm pt-1">{{ error }}</p>

        <button
          type="submit"
          :disabled="loading"
          class="w-full py-4 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-2xl font-bold text-lg hover:bg-m3-sys-light-on-primary-container transition-all shadow-lg shadow-m3-sys-light-primary/20 disabled:opacity-50 mt-2"
        >
          {{ loading ? 'Please wait…' : (mode === 'login' ? 'Sign In' : 'Register') }}
        </button>
      </form>

      <p class="text-center text-m3-sys-light-on-surface-variant mt-8 text-sm">
        {{ mode === 'login' ? "Don't have an account?" : 'Already have an account?' }}
        <button
          @click="mode = mode === 'login' ? 'register' : 'login'; error = ''"
          class="text-m3-sys-light-primary font-semibold hover:underline ml-1"
        >
          {{ mode === 'login' ? 'Register' : 'Sign In' }}
        </button>
      </p>
    </div>
  </div>
</template>
