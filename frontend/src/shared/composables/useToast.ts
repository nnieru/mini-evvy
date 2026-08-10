import { ref } from 'vue'

export type ToastTone = 'info' | 'success' | 'warning' | 'error'

type ToastState = {
  message: string
  tone: ToastTone
}

const toast = ref<ToastState | null>(null)
let toastTimer: ReturnType<typeof setTimeout> | undefined

export function useToast() {
  function showToast(message: string, tone: ToastTone = 'info') {
    toast.value = { message, tone }
    if (toastTimer) clearTimeout(toastTimer)
    toastTimer = setTimeout(() => {
      toast.value = null
    }, 5000)
  }

  function clearToast() {
    toast.value = null
    if (toastTimer) clearTimeout(toastTimer)
  }

  return { toast, showToast, clearToast }
}
