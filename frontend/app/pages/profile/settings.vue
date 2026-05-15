<script setup lang="ts">
import { formatDashboardDate, getErrorMessage, toAvatarFallback } from '~/composables/useProfileDashboard'

const toast = useToast()
const router = useRouter()
const fileUploadRef = useTemplateRef('fileUploadRef')

const { session, user, isLoggedIn, initAuth, fetchProfile, updateProfile } = useAuth()
const { csrf, headerName } = useCsrf()
const upload = useUpload('/api/upload', {
  formKey: 'file',
  multiple: false,
  headers: { [headerName]: csrf }
})

const loading = ref(true)
const saving = ref(false)
const uploadingAvatar = ref(false)

const form = reactive({
  username: '',
  email: '',
  avatarURL: ''
})

const displayCreatedAt = computed(() => formatDashboardDate(user.value?.createdAt))

function fillForm() {
  form.username = user.value?.username || ''
  form.email = user.value?.email || ''
  form.avatarURL = user.value?.avatarURL || ''
}

onMounted(async () => {
  initAuth()

  if (!isLoggedIn.value) {
    await router.replace('/login')
    return
  }

  try {
    await fetchProfile()
    fillForm()
  } catch (error: unknown) {
    toast.add({
      color: 'error',
      title: '获取个人信息失败',
      description: getErrorMessage(error, '请稍后重试')
    })
  } finally {
    loading.value = false
  }
})

async function onAvatarChange() {
  const target = fileUploadRef.value?.inputRef
  if (!target) return

  uploadingAvatar.value = true
  try {
    const result = await upload(target)
    form.avatarURL = result.url || `/images/${result.pathname}`
    toast.add({
      color: 'success',
      title: '头像已上传',
      description: '保存资料后头像会同步到账号。'
    })
  } catch (error: unknown) {
    toast.add({
      color: 'error',
      title: '头像上传失败',
      description: getErrorMessage(error, '请确认文件类型和大小后重试')
    })
  } finally {
    uploadingAvatar.value = false
  }
}

async function onSave() {
  if (saving.value) return

  if (!form.username.trim()) {
    toast.add({
      color: 'error',
      title: '请填写显示名称'
    })
    return
  }

  if (!form.email.trim()) {
    toast.add({
      color: 'error',
      title: '请填写邮箱'
    })
    return
  }

  saving.value = true
  try {
    await updateProfile({
      username: form.username.trim(),
      email: form.email.trim(),
      avatarURL: form.avatarURL.trim()
    })
    await fetchProfile(session.value.userId)

    toast.add({
      color: 'success',
      title: '个人资料已保存'
    })
  } catch (error: unknown) {
    toast.add({
      color: 'error',
      title: '保存失败',
      description: getErrorMessage(error, '请稍后再试')
    })
  } finally {
    saving.value = false
  }
}

useSeoMeta({
  title: '资料设置 - GoPalette',
  description: '更新个人资料、头像和账号展示信息。'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader />

    <main class="mx-auto w-full max-w-4xl px-4 pb-20 pt-10 sm:px-14">
      <div class="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="flex items-center gap-1 text-sm text-toned">
            <NuxtLink
              to="/profile"
              class="transition-colors hover:text-primary"
            >
              个人中心
            </NuxtLink>
            <span>/</span>
            <span>设置</span>
          </p>
          <h1 class="mt-1 text-2xl font-semibold text-highlighted">
            资料设置
          </h1>
        </div>
        <UButton
          to="/profile"
          color="neutral"
          variant="soft"
          icon="i-lucide-arrow-left"
          label="返回概览"
        />
      </div>

      <section class="rounded-lg border border-default bg-default">
        <div class="border-b border-default p-4 sm:p-5">
          <h2 class="text-base font-semibold text-highlighted">
            基本资料
          </h2>
          <p class="mt-1 text-xs text-toned">
            这些信息会用于作者展示与互动场景。
          </p>
        </div>

        <div
          v-if="loading"
          class="space-y-4 p-4 sm:p-5"
        >
          <USkeleton class="h-20 w-full" />
          <USkeleton class="h-10 w-full" />
        </div>

        <form
          v-else
          class="p-4 sm:p-5"
          @submit.prevent="onSave"
        >
          <div class="grid gap-6 lg:grid-cols-[240px_minmax(0,1fr)]">
            <aside class="space-y-4">
              <div class="flex items-center gap-4 lg:block lg:space-y-4">
                <UAvatar
                  :src="form.avatarURL"
                  :alt="form.username"
                  :text="toAvatarFallback(form.username)"
                  size="3xl"
                />
                <div class="min-w-0">
                  <p class="text-sm font-medium text-highlighted">
                    {{ form.username || '未命名用户' }}
                  </p>
                  <p class="mt-1 text-xs text-toned">
                    注册时间 {{ displayCreatedAt }}
                  </p>
                </div>
              </div>

              <UFileUpload
                ref="fileUploadRef"
                accept="image/*"
                label="上传头像"
                description="支持图片文件，最大 5MB"
                :preview="false"
                :disabled="uploadingAvatar"
                class="min-h-32"
                @update:model-value="onAvatarChange"
              >
                <template #leading>
                  <UAvatar
                    :icon="uploadingAvatar ? 'i-lucide-loader-circle' : 'i-lucide-image-up'"
                    size="lg"
                    :ui="{ icon: uploadingAvatar ? 'animate-spin' : '' }"
                  />
                </template>
              </UFileUpload>

              <UButton
                v-if="form.avatarURL"
                type="button"
                block
                color="neutral"
                variant="ghost"
                icon="i-lucide-x"
                label="移除头像"
                @click="form.avatarURL = ''"
              />
            </aside>

            <div class="space-y-4">
              <UFormField
                label="显示名称"
                name="username"
                required
              >
                <UInput
                  v-model="form.username"
                  required
                  placeholder="你的公开显示名称"
                />
              </UFormField>

              <UFormField
                label="邮箱"
                name="email"
                required
                hint="当前版本会直接同步邮箱，后续可接入验证流程。"
              >
                <UInput
                  v-model="form.email"
                  type="email"
                  required
                  placeholder="name@example.com"
                />
              </UFormField>

              <div class="rounded-lg border border-dashed border-default p-4 text-sm text-toned">
                个人简介、网站和社交链接需要后端资料字段支持；当前页面先保留清晰入口，避免提交后丢数据。
              </div>

              <div class="flex flex-wrap items-center gap-2 pt-2">
                <UButton
                  type="submit"
                  :loading="saving"
                  icon="i-lucide-save"
                  label="保存修改"
                />
                <UButton
                  type="button"
                  color="neutral"
                  variant="soft"
                  icon="i-lucide-rotate-ccw"
                  label="重置"
                  @click="fillForm"
                />
              </div>
            </div>
          </div>
        </form>
      </section>
    </main>
  </div>
</template>
