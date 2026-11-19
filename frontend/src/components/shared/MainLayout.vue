<template>
  <div class="min-h-screen flex bg-slate-50 dark:bg-slate-950">
    <!-- Desktop Sidebar (Full Height) -->
    <aside
      class="hidden lg:flex lg:shrink-0 lg:flex-col lg:sticky lg:top-0 lg:h-screen border-r border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 transition-[width] duration-200 z-30"
      :class="sidebarCollapsed ? 'lg:w-16' : 'lg:w-64'"
    >
      <!-- Sidebar Logo Header -->
      <div
        class="h-14 border-b border-slate-200 dark:border-slate-800 flex items-center px-3"
        :class="sidebarCollapsed ? 'justify-center' : 'gap-2.5'"
      >
        <router-link to="/" class="flex items-center gap-2.5 min-w-0" :title="t('app_title')">
          <div class="w-8 h-8 rounded-lg bg-slate-900 dark:bg-slate-100 text-white dark:text-slate-900 font-bold flex items-center justify-center text-sm shrink-0">
            HD
          </div>
          <div v-if="!sidebarCollapsed" class="min-w-0 leading-tight">
            <h1 class="font-bold text-sm text-slate-900 dark:text-slate-100 truncate">{{ t('app_title') }}</h1>
            <p class="text-[11px] text-slate-500 dark:text-slate-400 truncate">{{ t('app_subtitle') }}</p>
          </div>
        </router-link>
      </div>

      <!-- Sidebar Navigation Links -->
      <nav class="flex-1 p-2 space-y-1 overflow-y-auto">
        <router-link
          v-if="auth.isAdmin"
          to="/admin/stats"
          :title="t('nav_stats')"
          class="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors"
          :class="[
            isStatsActive ? 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-semibold' : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 hover:text-slate-900 dark:hover:text-slate-100',
            sidebarCollapsed ? 'justify-center' : ''
          ]"
        >
          <svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          <span v-if="!sidebarCollapsed" class="truncate">{{ t('nav_stats') }}</span>
        </router-link>

        <router-link
          to="/tickets"
          :title="t('nav_tickets')"
          class="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors"
          :class="[
            isTicketsActive ? 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-semibold' : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 hover:text-slate-900 dark:hover:text-slate-100',
            sidebarCollapsed ? 'justify-center' : ''
          ]"
        >
          <svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 002 2 2 2 0 010 4 2 2 0 00-2 2v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 00-2-2 2 2 0 010-4 2 2 0 002-2V7a2 2 0 00-2-2H5z" />
          </svg>
          <span v-if="!sidebarCollapsed" class="truncate">{{ t('nav_tickets') }}</span>
        </router-link>

        <router-link
          v-if="auth.isAdmin"
          to="/users"
          :title="t('nav_users')"
          class="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors"
          :class="[
            isUsersActive ? 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-semibold' : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 hover:text-slate-900 dark:hover:text-slate-100',
            sidebarCollapsed ? 'justify-center' : ''
          ]"
        >
          <svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
          </svg>
          <span v-if="!sidebarCollapsed" class="truncate">{{ t('nav_users') }}</span>
        </router-link>
      </nav>

      <!-- Sidebar Footer -->
      <div class="p-2 border-t border-slate-200 dark:border-slate-800">
        <button
          @click="handleLogout"
          :title="t('logout')"
          class="w-full btn btn-outline btn-sm text-slate-600 dark:text-slate-300 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/30 border-slate-200 dark:border-slate-700"
          :class="sidebarCollapsed ? 'px-0 justify-center' : ''"
        >
          <span v-if="!sidebarCollapsed">{{ t('logout') }}</span>
          <svg v-else class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
        </button>
      </div>
    </aside>

    <!-- Main Column: Top Navbar + Page Content -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- Top Navbar -->
      <header class="sticky top-0 z-20 h-14 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between gap-2 px-3 sm:px-4">
        <!-- Left: Collapse button (desktop) & Mobile Menu button (mobile) -->
        <div class="flex items-center gap-2 min-w-0">
          <!-- Desktop Collapse Button (berada pada navbar disamping logo sidebar) -->
          <button
            type="button"
            class="hidden lg:flex items-center justify-center p-2 -ml-1 rounded-md text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 hover:text-slate-900 dark:hover:text-slate-100 transition-colors"
            :title="sidebarCollapsed ? t('expand_sidebar') : t('collapse_sidebar')"
            :aria-label="sidebarCollapsed ? t('expand_sidebar') : t('collapse_sidebar')"
            @click="toggleSidebar"
          >
            <svg class="w-5 h-5 transition-transform" :class="sidebarCollapsed ? 'rotate-180' : ''" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7M20 19l-7-7 7-7" />
            </svg>
          </button>

          <!-- Mobile Hamburger Toggle -->
          <button
            type="button"
            @click="mobileMenuOpen = !mobileMenuOpen"
            class="lg:hidden p-2 -ml-1 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-400"
            :aria-label="t('nav_tickets')"
          >
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>

          <!-- Mobile-only Brand (karena sidebar disembunyikan pada viewport kecil) -->
          <router-link to="/" class="lg:hidden flex items-center gap-2 min-w-0">
            <div class="w-7 h-7 rounded-lg bg-slate-900 dark:bg-slate-100 text-white dark:text-slate-900 font-bold flex items-center justify-center text-xs shrink-0">
              HD
            </div>
            <div class="min-w-0 leading-tight">
              <h1 class="font-bold text-sm text-slate-900 dark:text-slate-100 truncate">{{ t('app_title') }}</h1>
              <p class="text-[10px] text-slate-500 dark:text-slate-400 truncate hidden sm:block">{{ t('app_subtitle') }}</p>
            </div>
          </router-link>
        </div>

        <!-- Right: Tools + User Dropdown -->
        <div class="flex items-center gap-0.5 shrink-0">
          <button
            type="button"
            class="p-2 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-400"
            :title="settings.fullscreen ? t('exit_fullscreen') : t('fullscreen')"
            :aria-label="settings.fullscreen ? t('exit_fullscreen') : t('fullscreen')"
            @click="toggleFullscreen"
          >
            <svg v-if="!settings.fullscreen" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4h4M20 8V4h-4M4 16v4h4M20 16v4h-4" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 4v5H4m11-5v5h5M9 20v-5H4m11 5v-5h5" />
            </svg>
          </button>

          <button
            type="button"
            class="px-2 py-2 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-400 text-[11px] font-bold tracking-wide"
            :title="t('switch_lang')"
            :aria-label="t('switch_lang')"
            @click="settings.toggleLanguage()"
          >
            {{ settings.langLabel }}
          </button>

          <button
            type="button"
            class="p-2 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-400"
            :title="settings.isDark ? t('theme_light') : t('theme_dark')"
            :aria-label="settings.isDark ? t('theme_light') : t('theme_dark')"
            @click="settings.toggleTheme()"
          >
            <svg v-if="settings.isDark" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.36 6.36l-.7-.7M7.34 5.34l-.7-.7m12.02 0l-.7.7M7.34 18.66l-.7.7M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
            </svg>
          </button>

          <!-- User Menu Dropdown Trigger -->
          <div class="relative ml-1 sm:ml-2 pl-2 border-l border-slate-200 dark:border-slate-800">
            <div class="flex items-center gap-2">
              <button
                type="button"
                class="rounded-full p-0.5 hover:ring-2 hover:ring-slate-300 dark:hover:ring-slate-600 transition"
                :aria-label="t('app_routes')"
                :aria-expanded="userMenuOpen"
                @click="userMenuOpen = !userMenuOpen"
              >
                <div class="w-7 h-7 rounded-full bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-200 font-semibold flex items-center justify-center text-[11px] shrink-0">
                  {{ auth.user?.name?.charAt(0).toUpperCase() }}
                </div>
              </button>
              <div class="hidden sm:block leading-tight max-w-[10rem] text-left">
                <p class="text-xs font-medium text-slate-900 dark:text-slate-100 truncate">{{ auth.user?.name }}</p>
                <p class="text-[10px] text-slate-500 dark:text-slate-400 capitalize truncate">{{ auth.user?.role }}</p>
              </div>
            </div>

            <!-- Backdrop -->
            <div v-if="userMenuOpen" class="fixed inset-0 z-40" @click="userMenuOpen = false"></div>

            <!-- Dropdown Menu -->
            <div
              v-if="userMenuOpen"
              class="absolute right-0 mt-2 w-60 z-50 card p-1.5 shadow-lg shadow-slate-900/10 dark:shadow-black/40"
            >
              <div class="flex items-center gap-2.5 px-2 py-2 border-b border-slate-100 dark:border-slate-800 mb-1">
                <div class="w-8 h-8 rounded-full bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-200 font-semibold flex items-center justify-center text-xs shrink-0">
                  {{ auth.user?.name?.charAt(0).toUpperCase() }}
                </div>
                <div class="min-w-0">
                  <p class="text-xs font-semibold text-slate-900 dark:text-slate-100 truncate">{{ auth.user?.name }}</p>
                  <p class="text-[11px] text-slate-500 dark:text-slate-400 truncate">{{ auth.user?.email }}</p>
                </div>
              </div>

              <p class="px-2 pt-1.5 pb-1 text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">{{ t('app_routes') }}</p>

              <router-link
                v-if="auth.isAdmin"
                to="/admin/stats"
                class="flex items-center gap-2 px-2 py-1.5 text-xs rounded-md text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800"
                @click="userMenuOpen = false"
              >
                {{ t('nav_stats') }}
              </router-link>
              <router-link
                to="/tickets"
                class="flex items-center gap-2 px-2 py-1.5 text-xs rounded-md text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800"
                @click="userMenuOpen = false"
              >
                {{ t('nav_tickets') }}
              </router-link>
              <router-link
                v-if="!auth.isAgent"
                to="/tickets/create"
                class="flex items-center gap-2 px-2 py-1.5 text-xs rounded-md text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800"
                @click="userMenuOpen = false"
              >
                {{ t('create_ticket') }}
              </router-link>
              <router-link
                v-if="auth.isAdmin"
                to="/users"
                class="flex items-center gap-2 px-2 py-1.5 text-xs rounded-md text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800"
                @click="userMenuOpen = false"
              >
                {{ t('nav_users') }}
              </router-link>

              <div class="my-1 border-t border-slate-100 dark:border-slate-800"></div>

              <button
                type="button"
                class="w-full flex items-center gap-2 px-2 py-1.5 text-xs rounded-md text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/30"
                @click="userMenuOpen = false; handleLogout()"
              >
                {{ t('logout') }}
              </button>
            </div>
          </div>
        </div>
      </header>

      <!-- Main Content Area -->
      <main class="flex-1 min-w-0">
        <div class="max-w-7xl mx-auto p-4 sm:p-6 lg:p-8">
          <router-view />
        </div>
      </main>
    </div>

    <!-- Mobile Drawer Overlay -->
    <div v-if="mobileMenuOpen" class="lg:hidden fixed inset-0 bg-slate-900/50 z-40" @click="mobileMenuOpen = false"></div>
    <div v-if="mobileMenuOpen" class="lg:hidden fixed inset-y-0 left-0 w-64 bg-white dark:bg-slate-900 z-50 p-4 flex flex-col justify-between border-r border-slate-200 dark:border-slate-800">
      <div>
        <div class="flex items-center justify-between mb-6 pb-4 border-b border-slate-100 dark:border-slate-800">
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-lg bg-slate-900 dark:bg-slate-100 text-white dark:text-slate-900 font-bold flex items-center justify-center text-sm shrink-0">
              HD
            </div>
            <div class="font-bold text-slate-900 dark:text-slate-100">{{ t('app_title') }}</div>
          </div>
          <button @click="mobileMenuOpen = false" class="text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:hover:text-slate-400" :aria-label="t('close')">
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <nav class="space-y-1">
          <router-link
            v-if="auth.isAdmin"
            to="/admin/stats"
            @click="mobileMenuOpen = false"
            class="block px-3 py-2 text-sm font-medium rounded-md transition-colors"
            :class="isStatsActive ? 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-semibold' : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'"
          >
            {{ t('nav_stats') }}
          </router-link>
          <router-link
            to="/tickets"
            @click="mobileMenuOpen = false"
            class="block px-3 py-2 text-sm font-medium rounded-md transition-colors"
            :class="isTicketsActive ? 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-semibold' : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'"
          >
            {{ t('nav_tickets') }}
          </router-link>
          <router-link
            v-if="auth.isAdmin"
            to="/users"
            @click="mobileMenuOpen = false"
            class="block px-3 py-2 text-sm font-medium rounded-md transition-colors"
            :class="isUsersActive ? 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-semibold' : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'"
          >
            {{ t('nav_users') }}
          </router-link>
        </nav>
      </div>

      <div class="border-t border-slate-100 dark:border-slate-800 pt-4">
        <p class="text-sm font-medium text-slate-900 dark:text-slate-100">{{ auth.user?.name }}</p>
        <p class="text-xs text-slate-500 dark:text-slate-400 capitalize mb-3">{{ auth.user?.role }}</p>
        <button @click="handleLogout" class="w-full btn btn-outline btn-sm text-rose-600 border-rose-200">
          {{ t('logout') }}
        </button>
      </div>
    </div>

    <Toast :message="toast.message" :type="toast.type" :visible="toast.visible" />
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useI18n } from '../../i18n'
import { useRealtime } from '../../services/realtime.service'
import Toast from './Toast.vue'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const mobileMenuOpen = ref(false)
const userMenuOpen = ref(false)
const sidebarCollapsed = ref(localStorage.getItem('sidebar_collapsed') === 'true')
const { t, settings } = useI18n()

const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
  localStorage.setItem('sidebar_collapsed', sidebarCollapsed.value ? 'true' : 'false')
}

const toast = ref({ message: '', visible: false, type: 'success' })
const realtime = useRealtime()
let showToastTimer = null

const showToast = (msg, type = 'success') => {
  toast.value = { message: msg, type, visible: true }
  if (showToastTimer) clearTimeout(showToastTimer)
  showToastTimer = setTimeout(() => {
    toast.value.visible = false
  }, 4000)
}

const isTicketsActive = computed(() => route.path.startsWith('/tickets'))
const isUsersActive = computed(() => route.path.startsWith('/users'))
const isStatsActive = computed(() => route.path.startsWith('/admin/stats'))

const handleLogout = () => {
  realtime.disconnect()
  auth.logout()
  router.push('/login')
}

const toggleFullscreen = async () => {
  if (!document.fullscreenElement) {
    try {
      await document.documentElement.requestFullscreen()
      if (!settings.fullscreen) settings.toggleFullscreen()
    } catch {}
  } else {
    try {
      await document.exitFullscreen()
      if (settings.fullscreen) settings.toggleFullscreen()
    } catch {}
  }
}

let unsubs = []

onMounted(() => {
  document.addEventListener('fullscreenchange', () => {
    const isFull = !!document.fullscreenElement
    if (settings.fullscreen !== isFull) settings.fullscreen = isFull
  })

  // SSE real-time notifications: notifikasi tiket dibuat hanya untuk agent dan admin
  unsubs.push(realtime.onEvent('ticket_created', (data) => {
    if (auth.isAgent || auth.isAdmin) {
      showToast(`${t('rt_ticket_created')}: ${data?.title || ''}`)
    }
  }))
  unsubs.push(realtime.onEvent('ticket_updated', () => {
    showToast(t('rt_ticket_updated'))
  }))
  unsubs.push(realtime.onEvent('comment_added', () => {
    showToast(t('rt_comment_added'))
  }))
  unsubs.push(realtime.onEvent('attachment_added', () => {
    showToast(t('rt_attachment_added'))
  }))
})

onUnmounted(() => {
  unsubs.forEach((unsub) => unsub())
})
</script>