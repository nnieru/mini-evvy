import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/features/auth/stores/auth'
import { isStaffForEvent } from '@/app/router/roleGuards'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/orgs',
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/features/auth/pages/LoginPage.vue'),
      meta: { guest: true },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/features/auth/pages/RegisterPage.vue'),
      meta: { guest: true },
    },
    {
      path: '/',
      component: () => import('@/app/layouts/AppLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: 'orgs',
          name: 'orgs',
          component: () => import('@/features/organizations/pages/OrganizationsPage.vue'),
        },
        {
          path: 'orgs/:orgId',
          name: 'org-detail',
          component: () => import('@/features/organizations/pages/OrganizationDetailPage.vue'),
        },
        {
          path: 'orgs/:orgId/events',
          name: 'org-events',
          component: () => import('@/features/events/pages/EventsPage.vue'),
        },
        {
          path: 'events/:eventId',
          name: 'event-detail',
          component: () => import('@/features/events/pages/EventOverviewPage.vue'),
          meta: { staffOnly: true },
        },
        {
          path: 'events/:eventId/categories',
          name: 'event-categories',
          component: () => import('@/features/categories/pages/CategoriesPage.vue'),
          meta: { staffOnly: true },
        },
        {
          path: 'events/:eventId/seats',
          name: 'event-seats',
          component: () => import('@/features/seats/pages/SeatsPage.vue'),
          meta: { staffOnly: true },
        },
        {
          path: 'events/:eventId/guests',
          name: 'event-guests',
          component: () => import('@/features/guests/pages/GuestsPage.vue'),
        },
        {
          path: 'events/:eventId/bookings',
          name: 'event-bookings',
          component: () => import('@/features/bookings/pages/BookingsPage.vue'),
        },
        {
          path: 'events/:eventId/attendance',
          name: 'event-attendance',
          component: () => import('@/features/attendance/pages/AttendancePage.vue'),
        },
        {
          path: 'events/:eventId/jobs',
          name: 'event-jobs',
          component: () => import('@/features/jobs/pages/JobsPage.vue'),
          meta: { staffOnly: true },
        },
        {
          path: 'events/:eventId/invite-email',
          name: 'event-invite-email',
          component: () => import('@/features/email-templates/pages/InviteEmailPage.vue'),
          meta: { staffOnly: true },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.initialized) {
    await auth.bootstrap()
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.meta.guest && auth.isAuthenticated) {
    return { name: 'orgs' }
  }

  const eventId = to.params.eventId as string | undefined
  if (eventId && to.meta.staffOnly && auth.token) {
    try {
      const staff = await isStaffForEvent(eventId, auth.token)
      if (!staff) {
        return { name: 'event-bookings', params: { eventId } }
      }
    } catch {
      return { name: 'event-bookings', params: { eventId } }
    }
  }

  return true
})

export default router
