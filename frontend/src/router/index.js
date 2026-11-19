import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/LoginView.vue'),
    meta: { guestOnly: true }
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('../views/RegisterView.vue'),
    meta: { guestOnly: true }
  },
  {
    path: '/',
    component: () => import('../components/shared/MainLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        redirect: () => {
          const auth = useAuthStore()
          return auth.user?.role === 'admin' ? '/admin/stats' : '/tickets'
        }
      },
      {
        path: 'tickets',
        name: 'tickets',
        component: () => import('../views/TicketsView.vue')
      },
      {
        path: 'tickets/create',
        name: 'ticket-create',
        component: () => import('../views/CreateTicketView.vue'),
        meta: { roles: ['requester', 'admin'] }
      },
      {
        path: 'tickets/:id',
        name: 'ticket-detail',
        component: () => import('../views/TicketDetailView.vue')
      },
      {
        path: 'users',
        name: 'users',
        component: () => import('../views/UsersView.vue'),
        meta: { roles: ['admin'] }
      },
      {
        path: 'admin/stats',
        name: 'admin-stats',
        component: () => import('../views/AdminStatsView.vue'),
        meta: { roles: ['admin'] }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: () => {
      const auth = useAuthStore()
      return auth.isAuthenticated
        ? (auth.user?.role === 'admin' ? '/admin/stats' : '/tickets')
        : '/login'
    }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const auth = useAuthStore()

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return next({ name: 'login', query: { redirect: to.fullPath } })
  }

  // Authenticated user hitting login/register → role-based landing
  if (to.meta.guestOnly && auth.isAuthenticated) {
    return next(auth.user?.role === 'admin' ? '/admin/stats' : '/tickets')
  }

  // Role-protected routes
  if (to.meta.roles && !to.meta.roles.includes(auth.user?.role)) {
    return next({ name: 'tickets' })
  }

  next()
})

export default router
