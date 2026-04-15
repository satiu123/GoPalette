<script setup lang="ts">
const toast = useToast()
const router = useRouter()

const { user, isLoggedIn, initAuth, fetchProfile, updateProfile, clearSession } = useAuth()

const loading = ref(true)
const saving = ref(false)

const form = reactive({
  username: '',
  email: '',
  avatarURL: ''
})

onMounted(async () => {
  initAuth()

  if (!isLoggedIn.value) {
    await router.replace('/login')
    return
  }

  try {
    const profile = await fetchProfile()
    if (profile) {
      form.username = profile.username || ''
      form.email = profile.email || ''
      form.avatarURL = profile.avatarURL || ''
    }
  } catch (error: any) {
    toast.add({
      color: 'error',
      title: '获取个人信息失败',
      description: error?.data?.message || error?.message || '请稍后重试'
    })
  } finally {
    loading.value = false
  }
})

async function onSave() {
  if (saving.value) return

  saving.value = true
  try {
    await updateProfile({
      username: form.username.trim(),
      email: form.email.trim(),
      avatarURL: form.avatarURL.trim()
    })

    toast.add({
      color: 'success',
      title: '个人信息已更新'
    })
  } catch (error: any) {
    toast.add({
      color: 'error',
      title: '更新失败',
      description: error?.data?.message || error?.message || '请稍后再试'
    })
  } finally {
    saving.value = false
  }
}

async function onLogout() {
  clearSession()
  await router.push('/login')
}

useSeoMeta({
  title: '个人信息 - GoPalette',
  description: '查看和编辑你的 GoPalette 个人资料。'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton
        color="neutral"
        variant="subtle"
        icon="i-lucide-log-out"
        label="退出登录"
        @click="onLogout"
      />
    </AppHeader>

    <main class="mx-auto w-full max-w-2xl px-4 pb-20 pt-10 sm:px-0">
      <UCard>
        <template #header>
          <div class="space-y-1">
            <h1 class="text-xl font-semibold text-highlighted">
              个人信息
            </h1>
            <p class="text-sm text-toned">
              维护你的公开资料，便于文章页展示作者信息。
            </p>
          </div>
        </template>

        <div
          v-if="loading"
          class="py-10 text-center text-sm text-toned"
        >
          正在加载个人资料...
        </div>

        <form
          v-else
          class="space-y-4"
          @submit.prevent="onSave"
        >
          <UFormField label="用户 ID" name="id">
            <UInput
              :model-value="user?.id || ''"
              disabled
            />
          </UFormField>

          <UFormField label="用户名" name="username">
            <UInput
              v-model="form.username"
              required
            />
          </UFormField>

          <UFormField label="邮箱" name="email">
            <UInput
              v-model="form.email"
              type="email"
              required
            />
          </UFormField>

          <UFormField label="头像 URL" name="avatarURL">
            <UInput v-model="form.avatarURL" />
          </UFormField>

          <UButton
            type="submit"
            :loading="saving"
            label="保存修改"
          />
        </form>
      </UCard>
    </main>
  </div>
</template>
