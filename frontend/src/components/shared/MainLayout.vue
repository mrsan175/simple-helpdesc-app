<template>
  <div class="min-h-screen bg-slate-50 dark:bg-slate-950">
    <!-- Top Navbar -->
    <header class="sticky top-0 z-30 h-14 bg-white dark:bg-slate-900 border-b flex items-center justify-between gap-2 px-3 sm:px-4">
      <div class="flex items-center gap-2 min-w-0">
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

        <router-link to="/tickets" class="flex items-center gap-2 min-w-0">
          <div class="w-8 h-8 rounded-lg bg-slate-900 text-white font-bold flex items-center justify-center text-sm shrink-0">
            SJ
          </div>
          <div class="min-w-0 leading-tight">
            <h1 class="font-bold text-sm text-slate-900 dark:text-slate-100 truncate">{{ t('app_title') }}</h1>
            <p class="text-[11px] text-slate-500 dark:text-slate-400 truncate hidden sm:block">{{ t('app_subtitle') }}</p>
          </div>
        </router-link>
      </div>

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

        <div class="hidden sm:flex items-center gap-2 ml-2 pl-2 border-l">
          <div class="w-7 h-7 rounded-full bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-200 font-semibold flex items-center justify-center text-[11px] shrink-0">
            {{ auth.user?.name?.charAt(0).toUpperCase() }}
          </div>
          <div class="leading-tight max-w-[10rem]">
            <p class="text-xs font-medium text-slate-900 dark:text-slate-100 truncate">{{ auth.user?.name }}</p>
            <p class="text-[10px] text-slate-500 dark:text-slate-400 capitalize truncate">{{ auth.user?.role }}</p>
          </div>
        </div>
      </div>
    </header>

    <div class="flex">
      <!-- Desktop Sidebar -->
      <aside class="hidden lg:flex lg:w-64 lg:shrink-0 lg:flex-col lg:sticky lg:top-14 lg:h-[calc(100vh-3.5rem)] border-r bg-white dark:bg-slate-900">
        <nav class="flex-1 p-4 space-y-1 overflow-y-auto">
          <router-link
            to="/tickets"
            class="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors"
            :class="isTicketsActive ? 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-semibold' : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 hover:text-slate-900 dark:hover:text-slate-100'"
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 002 2 2 2 0 010 4 2 2 0 00-2 2v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 00-2-2 2 2 0 010-4 2 2 0 002-2V7a2 2 0 00-2-2H5z" />
            </svg>
            {{ t('nav_tickets') }}
          </router-link>

          <router-link
            v-if="auth.isAdmin"
            to="/users"
            class="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors"
            :class="isUsersActive ? 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-semibold' : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 hover:text-slate-900 dark:hover:text-slate-100'"
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
            {{ t('nav_users') }}
          </router-link>
        </nav>

        <div class="p-4 border-t">
          <button @click="handleLogout" class="w-full btn btn-outline btn-sm text-slate-600 dark:text-slate-300 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/30 border-slate-200 dark:border-slate-700">
            {{ t('logout') }}
          </button>
        </div>
      </aside>

      <!-- Mobile Drawer Overlay -->
      <div v-if="mobileMenuOpen" class="lg:hidden fixed inset-0 bg-slate-900/50 z-40" @click="mobileMenuOpen = false"></div>
      <div v-if="mobileMenuOpen" class="lg:hidden fixed inset-y-0 left-0 w-64 bg-white dark:bg-slate-900 z-50 p-4 flex flex-col justify-between border-r">
        <div>
          <div class="flex items-center justify-between mb-6 pb-4 border-b">
            <div class="font-bold text-slate-900 dark:text-slate-100">{{ t('app_title') }}</div>
            <button @click="mobileMenuOpen = false" class="text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:hover:text-slate-400" :aria-label="t('close')">
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

        <div class="border-t pt-4">
          <p class="text-sm font-medium text-slate-900 dark:text-slate-100">{{ auth.user?.name }}</p>
          <p class="text-xs text-slate-500 dark:text-slate-400 capitalize mb-3">{{ auth.user?.role }}</p>
          <button @click="handleLogout" class="w-full btn btn-outline btn-sm text-rose-600 border-rose-200">
            {{ t('logout') }}
          </button>
        </div>
      </div>

      <!-- Main Content Area -->
      <main class="flex-1 min-w-0">
        <div class="max-w-7xl mx-auto p-4 sm:p-6 lg:p-8">
          <router-view />
        </div>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useI18n } from '../../i18n'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const mobileMenuOpen = ref(false)
const { t, settings } = useI18n()

const isTicketsActive = computed(() => route.path.startsWith('/tickets'))
const isUsersActive = computed(() => route.path.startsWith('/users'))

const handleLogout = () => {
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

onMounted(() => {
  document.addEventListener('fullscreenchange', () => {
    const isFull = !!document.fullscreenElement
    if (settings.fullscreen !== isFull) settings.fullscreen = isFull
  })
})
</script>