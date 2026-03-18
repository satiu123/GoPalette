<script setup lang="ts">
import { AlertTriangle } from 'lucide-vue-next'

const { state, confirm, cancel } = useConfirmDialog()

const confirmBtnClass = computed(() =>
  state.value.tone === 'danger'
    ? 'bg-m3-sys-light-error-container text-m3-sys-light-on-error-container hover:opacity-90'
    : 'bg-m3-sys-light-primary text-m3-sys-light-on-primary hover:opacity-90'
)

function onEsc(event: KeyboardEvent) {
  if (event.key === 'Escape' && state.value.visible) cancel()
}

watch(
  () => state.value.visible,
  (visible) => {
    if (!import.meta.client) return
    document.body.style.overflow = visible ? 'hidden' : ''
  }
)

onMounted(() => {
  window.addEventListener('keydown', onEsc)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onEsc)
  if (import.meta.client) document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="state.visible"
        class="fixed inset-0 z-[100] bg-black/35 backdrop-blur-[2px] flex items-center justify-center px-4"
        @click.self="cancel"
      >
        <Transition
          enter-active-class="transition duration-200 ease-out"
          enter-from-class="opacity-0 translate-y-2 scale-95"
          enter-to-class="opacity-100 translate-y-0 scale-100"
          leave-active-class="transition duration-150 ease-in"
          leave-from-class="opacity-100 translate-y-0 scale-100"
          leave-to-class="opacity-0 translate-y-2 scale-95"
        >
          <div class="w-full max-w-md bg-m3-sys-light-surface rounded-[1.75rem] border border-m3-sys-light-outline-variant shadow-2xl p-6">
            <div class="flex items-start gap-3">
              <div
                class="w-10 h-10 rounded-xl flex items-center justify-center"
                :class="state.tone === 'danger'
                  ? 'bg-m3-sys-light-error-container text-m3-sys-light-on-error-container'
                  : 'bg-m3-sys-light-primary-container text-m3-sys-light-on-primary-container'"
              >
                <AlertTriangle class="w-5 h-5" />
              </div>
              <div class="flex-1">
                <h3 class="text-lg font-black tracking-tight text-m3-sys-light-on-surface">{{ state.title }}</h3>
                <p class="mt-1 text-sm text-m3-sys-light-on-surface-variant leading-relaxed">{{ state.message }}</p>
              </div>
            </div>

            <div class="mt-6 flex justify-end gap-3">
              <button
                type="button"
                class="px-4 py-2.5 rounded-full bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant font-medium hover:bg-m3-sys-light-secondary-container transition-colors"
                @click="cancel"
              >
                {{ state.cancelText }}
              </button>
              <button
                type="button"
                class="px-4 py-2.5 rounded-full font-medium transition-opacity"
                :class="confirmBtnClass"
                @click="confirm"
              >
                {{ state.confirmText }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
