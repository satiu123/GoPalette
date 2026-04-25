<script setup lang="ts">
const toast = useToast()
const router = useRouter()

const { login, initAuth, isLoggedIn } = useAuth()

const form = reactive({
  email: '',
  password: ''
})

const submitting = ref(false)

onMounted(() => {
  initAuth()
  if (isLoggedIn.value) {
    router.replace('/profile')
  }
})

async function onSubmit() {
  if (submitting.value) return

  submitting.value = true
  try {
    await login({
      email: form.email.trim(),
      password: form.password
    })

    toast.add({
      color: 'success',
      title: '登录成功'
    })

    await router.push('/profile')
  } catch (error: any) {
    toast.add({
      color: 'error',
      title: '登录失败',
      description: error?.data?.message || error?.message || '请检查邮箱和密码'
    })
  } finally {
    submitting.value = false
  }
}

useSeoMeta({
  title: '登录 - GoPalette',
  description: '登录 GoPalette，管理个人资料与文章发布。'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader />

    <main class="mx-auto flex w-full max-w-md px-4 pb-20 pt-12 sm:px-0">
      <UCard class="motion-fade-up motion-panel w-full">
        <template #header>
          <div class="space-y-1">
            <h1 class="text-xl font-semibold text-highlighted">
              登录账号
            </h1>
            <p class="text-sm text-toned">
              使用邮箱密码登录后可进入写作与个人信息管理。
            </p>
          </div>
        </template>

        <form
          class="space-y-4"
          @submit.prevent="onSubmit"
        >
          <UFormField
            label="邮箱"
            name="email"
          >
            <UInput
              v-model="form.email"
              type="email"
              placeholder="you@example.com"
              required
            />
          </UFormField>

          <UFormField
            label="密码"
            name="password"
          >
            <UInput
              v-model="form.password"
              type="password"
              placeholder="请输入密码"
              required
            />
          </UFormField>

          <UButton
            type="submit"
            block
            :loading="submitting"
            label="登录"
          />
        </form>

        <template #footer>
          <p class="text-sm text-toned">
            还没有账号？
            <NuxtLink
              to="/register"
              class="font-medium text-primary"
            >
              去注册
            </NuxtLink>
          </p>
        </template>
      </UCard>
    </main>
  </div>
</template>
