import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUiStore = defineStore('ui', () => {
  const sidebarOpen = ref(false)
  const currentOrgId = ref<string | null>(null)
  const currentEventId = ref<string | null>(null)

  function toggleSidebar() {
    sidebarOpen.value = !sidebarOpen.value
  }

  function closeSidebar() {
    sidebarOpen.value = false
  }

  function setCurrentOrg(orgId: string | null) {
    currentOrgId.value = orgId
  }

  function setCurrentEvent(eventId: string | null) {
    currentEventId.value = eventId
  }

  return {
    sidebarOpen,
    currentOrgId,
    currentEventId,
    toggleSidebar,
    closeSidebar,
    setCurrentOrg,
    setCurrentEvent,
  }
})
