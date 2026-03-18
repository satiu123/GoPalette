type ConfirmTone = 'primary' | 'danger'

interface ConfirmDialogOptions {
  title?: string
  message?: string
  confirmText?: string
  cancelText?: string
  tone?: ConfirmTone
}

interface ConfirmDialogState {
  visible: boolean
  title: string
  message: string
  confirmText: string
  cancelText: string
  tone: ConfirmTone
}

let pendingResolver: ((value: boolean) => void) | null = null

const defaultState: ConfirmDialogState = {
  visible: false,
  title: '请确认操作',
  message: '是否继续执行当前操作？',
  confirmText: '确认',
  cancelText: '取消',
  tone: 'primary'
}

export function useConfirmDialog() {
  const state = useState<ConfirmDialogState>('confirm-dialog-state', () => ({ ...defaultState }))

  function closeWith(result: boolean) {
    state.value.visible = false
    if (pendingResolver) {
      pendingResolver(result)
      pendingResolver = null
    }
  }

  function confirm() {
    closeWith(true)
  }

  function cancel() {
    closeWith(false)
  }

  function askConfirm(options: ConfirmDialogOptions = {}) {
    if (pendingResolver) {
      pendingResolver(false)
      pendingResolver = null
    }

    state.value = {
      visible: true,
      title: options.title ?? defaultState.title,
      message: options.message ?? defaultState.message,
      confirmText: options.confirmText ?? defaultState.confirmText,
      cancelText: options.cancelText ?? defaultState.cancelText,
      tone: options.tone ?? defaultState.tone
    }

    return new Promise<boolean>((resolve) => {
      pendingResolver = resolve
    })
  }

  return {
    state,
    askConfirm,
    confirm,
    cancel
  }
}
