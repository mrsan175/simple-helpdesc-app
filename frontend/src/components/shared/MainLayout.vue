<template>
  <div class="min-h-screen flex flex-col lg:flex-row bg-slate-50">
    <!-- Desktop Sidebar -->
    <aside class="hidden lg:flex lg:w-64 lg:flex-col lg:fixed lg:inset-y-0 border-r bg-white">
      <div class="p-6 border-b flex items-center gap-3">
        <div class="w-8 h-8 rounded-lg bg-slate-900 text-white font-bold flex items-center justify-center text-sm">
          SJ
        </div>
        <div>
          <h1 class="font-bold text-sm leading-tight text-slate-900">Simple Journey</h1>
          <p class="text-xs text-slate-500">Helpdesk System</p>
        </div>
      </div>

      <nav class="flex-1 p-4 space-y-1">
        <router-link
          to="/tickets"
          class="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors"
          :class="isTicketsActive ? 'bg-slate-100 text-slate-900 font-semibold' : 'text-slate-600 hover:bg-slate-50 hover:text-slate-900'"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 002 2 2 2 0 010 4 2 2 0 00-2 2v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 00-2-2 2 2 0 010-4 2 2 0 002-2V7a2 2 0 00-2-2H5z" />
          </svg>
          Tiket
        </router-link>

        <router-link
          v-if="auth.isAdmin"
          to="/users"
          class="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors"
          :class="isUsersActive ? 'bg-slate-100 text-slate-900 font-semibold' : 'text-slate-600 hover:bg-slate-50 hover:text-slate-900'"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
          </svg>
          Manajemen User
        </router-link>
      </nav>

      <div class="p-4 border-t bg-slate-50/50">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-3 overflow-hidden">
            <div class="w-8 h-8 rounded-full bg-slate-200 text-slate-700 font-semibold flex items-center justify-center text-xs shrink-0">
              {{ auth.user?.name?.charAt(0).toUpperCase() }}
            </div>
            <div class="overflow-hidden">
              <p class="text-sm font-medium text-slate-900 truncate">{{ auth.user?.name }}</p>
              <p class="text-xs text-slate-500 capitalize truncate">{{ auth.user?.role }}</p>
            </div>
          </div>
        </div>
        <button @click="handleLogout" class="w-full btn btn-outline btn-sm text-slate-600 hover:text-rose-600 hover:bg-rose-50 border-slate-200">
          Logout
        </button>
      </div>
    </aside>

    <!-- Mobile Header -->
    <header class="lg:hidden bg-white border-b px-4 py-3 flex items-center justify-between sticky top-0 z-20">
      <div class="flex items-center gap-2">
        <div class="w-7 h-7 rounded bg-slate-900 text-white font-bold flex items-center justify-center text-xs">
          SJ
        </div>
        <span class="font-bold text-sm text-slate-900">Simple Journey Helpdesk</span>
      </div>
      <button @click="mobileMenuOpen = !mobileMenuOpen" class="p-2 rounded-md hover:bg-slate-100 text-slate-600">
        <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
    </header>

    <!-- Mobile Drawer Overlay -->
    <div v-if="mobileMenuOpen" class="lg:hidden fixed inset-0 bg-slate-900/50 z-30" @click="mobileMenuOpen = false"></div>
    <div v-if="mobileMenuOpen" class="lg:hidden fixed inset-y-0 left-0 w-64 bg-white z-40 p-4 flex flex-col justify-between border-r">
      <div>
        <div class="flex items-center justify-between mb-6 pb-4 border-b">
          <div class="font-bold text-slate-900">Simple Journey</div>
          <button @click="mobileMenuOpen = false" class="text-slate-400 hover:text-slate-600" aria-label="Tutup menu">
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <nav class="space-y-1">
          <router-link
            to="/tickets"
            @click="mobileMenuOpen = false"
            class="block px-3 py-2 text-sm font-medium rounded-md transition-colors"
            :class="isTicketsActive ? 'bg-slate-100 text-slate-900 font-semibold' : 'text-slate-700 hover:bg-slate-100'"
          >
            Tiket
          </router-link>
          <router-link
            v-if="auth.isAdmin"
            to="/users"
            @click="mobileMenuOpen = false"
            class="block px-3 py-2 text-sm font-medium rounded-md transition-colors"
            :class="isUsersActive ? 'bg-slate-100 text-slate-900 font-semibold' : 'text-slate-700 hover:bg-slate-100'"
          >
            Manajemen User
          </router-link>
        </nav>
      </div>

      <div class="border-t pt-4">
        <p class="text-sm font-medium text-slate-900">{{ auth.user?.name }}</p>
        <p class="text-xs text-slate-500 capitalize mb-3">{{ auth.user?.role }}</p>
        <button @click="handleLogout" class="w-full btn btn-outline btn-sm text-rose-600 border-rose-200">
          Logout
        </button>
      </div>
    </div>

    <!-- Main Content Area -->
    <main class="flex-1 lg:pl-64 min-h-screen">
      <div class="max-w-7xl mx-auto p-4 sm:p-6 lg:p-8">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const mobileMenuOpen = ref(false)

const isTicketsActive = computed(() => route.path.startsWith('/tickets'))
const isUsersActive = computed(() => route.path.startsWith('/users'))

const handleLogout = () => {
  auth.logout()
  router.push('/login')
}
</script>