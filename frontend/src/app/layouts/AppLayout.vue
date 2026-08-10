<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/features/auth/stores/auth'
import { useUiStore } from '@/app/stores/ui'
import { useMediaQuery } from '@/shared/composables/useMediaQuery'
import { useEventOrgRole } from '@/features/organizations/composables/useEventOrgRole'
import Button from '@/shared/ui/Button.vue'
import ToastHost from '@/shared/ui/ToastHost.vue'
import { cn } from '@/shared/lib/cn'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()
const isDesktop = useMediaQuery('(min-width: 1024px)')

const eventId = computed(() => route.params.eventId as string | undefined)
const orgId = computed(() => route.params.orgId as string | undefined)

const { isStaff: isEventStaff } = useEventOrgRole(computed(() => eventId.value ?? ''))

const allEventNavItems = [
  { label: 'Overview', to: (id: string) => `/events/${id}`, staffOnly: true },
  { label: 'Categories', to: (id: string) => `/events/${id}/categories`, staffOnly: true },
  { label: 'Seats', to: (id: string) => `/events/${id}/seats`, staffOnly: true },
  { label: 'Guests', to: (id: string) => `/events/${id}/guests`, staffOnly: false },
  { label: 'Seating draft', to: (id: string) => `/events/${id}/seating-draft`, staffOnly: true },
  { label: 'Bookings', to: (id: string) => `/events/${id}/bookings`, staffOnly: false },
  { label: 'Check-in', to: (id: string) => `/events/${id}/attendance`, staffOnly: false },
  { label: 'Jobs', to: (id: string) => `/events/${id}/jobs`, staffOnly: true },
  { label: 'Invite email', to: (id: string) => `/events/${id}/invite-email`, staffOnly: true },
] as const

const navItems = computed(() => {
  if (eventId.value) {
    const id = eventId.value
    return allEventNavItems
      .filter((item) => isEventStaff.value || !item.staffOnly)
      .map((item) => ({ label: item.label, to: item.to(id) }))
  }
  if (orgId.value) {
    return [
      { label: 'Organization', to: `/orgs/${orgId.value}` },
      { label: 'Events', to: `/orgs/${orgId.value}/events` },
    ]
  }
  return [{ label: 'Organizations', to: '/orgs' }]
})

function logout() {
  auth.logout()
  router.push('/login')
}

function closeSidebarOnNavigate() {
  if (!isDesktop.value) ui.closeSidebar()
}
</script>

<template>
  <div class="min-h-dvh bg-surface">
    <header class="sticky top-0 z-40 border-b border-border bg-surface/90 backdrop-blur">
      <div class="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6">
        <div class="flex items-center gap-3">
          <button
            class="rounded-lg border border-border px-3 py-2 text-sm lg:hidden"
            @click="ui.toggleSidebar()"
          >
            Menu
          </button>
          <RouterLink to="/orgs" class="font-display text-xl font-semibold text-ink">
            mini-evvy
          </RouterLink>
        </div>
        <div class="flex items-center gap-3">
          <span class="hidden text-sm text-ink-muted sm:inline">{{ auth.user?.email }}</span>
          <Button variant="ghost" size="sm" @click="logout">Sign out</Button>
        </div>
      </div>
    </header>

    <div class="mx-auto flex max-w-7xl gap-6 px-4 py-6 sm:px-6">
      <aside
        :class="
          cn(
            'fixed inset-y-0 left-0 z-30 w-72 border-r border-border bg-surface-raised p-4 pt-20 transition-transform lg:static lg:translate-x-0 lg:rounded-2xl lg:border lg:pt-4',
            ui.sidebarOpen ? 'translate-x-0' : '-translate-x-full',
          )
        "
      >
        <nav class="space-y-1">
          <RouterLink
            v-for="item in navItems"
            :key="item.to"
            :to="item.to"
            class="block rounded-lg px-3 py-2 text-sm font-medium text-ink-muted transition hover:bg-accent-soft hover:text-accent"
            active-class="!bg-accent-soft !text-accent"
            @click="closeSidebarOnNavigate"
          >
            {{ item.label }}
          </RouterLink>
        </nav>
      </aside>

      <div
        v-if="ui.sidebarOpen && !isDesktop"
        class="fixed inset-0 z-20 bg-ink/30 lg:hidden"
        @click="ui.closeSidebar()"
      />

      <main class="min-w-0 flex-1 space-y-6">
        <RouterView />
      </main>
    </div>
    <ToastHost />
  </div>
</template>
