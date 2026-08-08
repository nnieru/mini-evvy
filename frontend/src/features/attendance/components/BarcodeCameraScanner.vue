<script setup lang="ts">
import { Html5Qrcode, Html5QrcodeSupportedFormats } from 'html5-qrcode'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Button from '@/shared/ui/Button.vue'
import Select from '@/shared/ui/Select.vue'
import Alert from '@/shared/ui/Alert.vue'

type CameraDevice = { id: string; label: string }

const props = defineProps<{
  paused?: boolean
}>()

const emit = defineEmits<{
  scan: [barcode: string]
}>()

const scannerId = `scanner-${Math.random().toString(36).slice(2)}`
const cameras = ref<CameraDevice[]>([])
const selectedCameraId = ref('')
const scanning = ref(false)
const error = ref('')
const lastCode = ref('')
const lastScanAt = ref(0)

let scanner: Html5Qrcode | null = null

const formats = [
  Html5QrcodeSupportedFormats.QR_CODE,
  Html5QrcodeSupportedFormats.CODE_128,
  Html5QrcodeSupportedFormats.CODE_39,
  Html5QrcodeSupportedFormats.EAN_13,
]

function pickDefaultCamera(devices: CameraDevice[]) {
  const rear = devices.find((device) =>
    /back|rear|environment/i.test(device.label),
  )
  return rear?.id ?? devices[0]?.id ?? ''
}

async function loadCameras() {
  try {
    const devices = await Html5Qrcode.getCameras()
    cameras.value = devices.map((device) => ({
      id: device.id,
      label: device.label || `Camera ${device.id.slice(0, 6)}`,
    }))
    if (!selectedCameraId.value) {
      selectedCameraId.value = pickDefaultCamera(cameras.value)
    }
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : 'Unable to access camera devices'
  }
}

async function stopScanner() {
  if (!scanner) return
  try {
    if (scanner.isScanning) {
      await scanner.stop()
    }
    scanner.clear()
  } catch {
    // ignore cleanup errors
  } finally {
    scanning.value = false
  }
}

async function startScanner() {
  error.value = ''
  if (!selectedCameraId.value) {
    await loadCameras()
  }
  if (!selectedCameraId.value) {
    error.value = 'No camera found on this device'
    return
  }

  if (!scanner) {
    scanner = new Html5Qrcode(scannerId, { formatsToSupport: formats, verbose: false })
  } else if (scanner.isScanning) {
    await scanner.stop()
  }

  try {
    await scanner.start(
      selectedCameraId.value,
      { fps: 10, qrbox: { width: 260, height: 260 }, aspectRatio: 1 },
      (decoded) => {
        if (props.paused) return
        const now = Date.now()
        if (decoded === lastCode.value && now - lastScanAt.value < 2000) return
        lastCode.value = decoded
        lastScanAt.value = now
        emit('scan', decoded.trim())
      },
      () => undefined,
    )
    scanning.value = true
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to start camera'
    scanning.value = false
  }
}

async function toggleScanner() {
  if (scanning.value) {
    await stopScanner()
  } else {
    await startScanner()
  }
}

async function switchCamera(cameraId: string) {
  selectedCameraId.value = cameraId
  if (scanning.value) {
    await stopScanner()
    await startScanner()
  }
}

function onVisibilityChange() {
  if (document.hidden) {
    void stopScanner()
  }
}

watch(
  () => props.paused,
  async (paused) => {
    if (paused && scanning.value) {
      await stopScanner()
    }
  },
)

onMounted(async () => {
  await loadCameras()
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onBeforeUnmount(async () => {
  document.removeEventListener('visibilitychange', onVisibilityChange)
  await stopScanner()
  scanner = null
})
</script>

<template>
  <div class="space-y-4">
    <Alert v-if="error" tone="error" :title="error" />

    <div class="overflow-hidden rounded-2xl border border-border bg-ink">
      <div :id="scannerId" class="min-h-[280px] w-full bg-ink sm:min-h-[360px]" />
    </div>

    <div class="grid gap-3 sm:grid-cols-[1fr_auto_auto] sm:items-end">
      <Select
        v-if="cameras.length > 0"
        v-model="selectedCameraId"
        label="Camera"
        :options="cameras.map((camera) => ({ label: camera.label, value: camera.id }))"
        @update:model-value="switchCamera"
      />
      <Button variant="secondary" @click="loadCameras">Refresh cameras</Button>
      <Button @click="toggleScanner">
        {{ scanning ? 'Stop camera' : 'Start camera' }}
      </Button>
    </div>

    <p class="text-xs text-ink-muted">
      Works on phone and laptop cameras. Use HTTPS or localhost for browser camera access.
    </p>
  </div>
</template>

<style scoped>
:deep(#qr-shaded-region) {
  border-radius: 1rem;
}
</style>
