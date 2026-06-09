<script setup lang="ts">
const toast = useToast()
const router = useRouter()

const { register } = useAuth()

const form = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: ''
})

const submitting = ref(false)

async function onSubmit() {
  if (submitting.value) return

  if (form.password !== form.confirmPassword) {
    toast.add({
      color: 'error',
      title: '两次密码不一致'
    })
    return
  }

  submitting.value = true
  try {
    await register({
      username: form.username.trim(),
      email: form.email.trim(),
      password: form.password
    })

    toast.add({
      color: 'success',
      title: '注册成功，请登录'
    })

    await router.push('/login')
  } catch (error: unknown) {
    toast.add({
      color: 'error',
      title: '注册失败',
      description: getRequestErrorMessage(error, '请检查用户名、邮箱和密码后重试')
    })
  } finally {
    submitting.value = false
  }
}

useSeoMeta({
  title: '注册 - GoPalette',
  description: '注册 GoPalette 账号，开启写作与个人主页功能。'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader />

    <main class="mx-auto flex w-full max-w-md px-4 pb-20 pt-12 sm:px-0">
      <UCard class="motion-fade-up motion-panel w-full">
        <template #header>
          <h1 class="text-xl font-semibold text-highlighted">
            创建账号
          </h1>
        </template>

        <form
          class="space-y-4"
          @submit.prevent="onSubmit"
        >
          <UFormField
            label="用户名"
            name="username"
          >
            <UInput
              v-model="form.username"
              placeholder="请输入用户名"
              required
            />
          </UFormField>

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

          <UFormField
            label="确认密码"
            name="confirmPassword"
          >
            <UInput
              v-model="form.confirmPassword"
              type="password"
              placeholder="请再次输入密码"
              required
            />
          </UFormField>

          <UButton
            type="submit"
            block
            :loading="submitting"
            label="注册"
          />
        </form>

        <template #footer>
          <p class="text-sm text-toned">
            已有账号？
            <NuxtLink
              to="/login"
              class="font-medium text-primary"
            >
              去登录
            </NuxtLink>
          </p>
        </template>
      </UCard>
    </main>
  </div>
</template>
