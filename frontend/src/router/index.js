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
        redirect: '/tickets'
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
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/tickets'
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

  if (to.meta.guestOnly && auth.isAuthenticated) {
    return next({ name: 'tickets' })
  }

  if (to.meta.roles && !to.meta.roles.includes(auth.user?.role)) {
    return next({ name: 'tickets' })
  }

  next()
})

export default router
