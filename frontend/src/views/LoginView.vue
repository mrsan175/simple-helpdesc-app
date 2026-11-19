<template>
  <div class="min-h-screen flex items-center justify-center bg-slate-100/60 dark:bg-slate-950 p-4">
    <div class="card w-full max-w-md p-6 sm:p-8">
      <div class="text-center mb-8">
        <div class="w-12 h-12 rounded-xl bg-slate-900 text-white font-bold flex items-center justify-center text-lg mx-auto mb-4">
          HD
        </div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">{{ t('login_title') }}</h1>
        <p class="text-sm text-slate-500 dark:text-slate-400 mt-1">{{ t('login_company') }}</p>
      </div>

      <form @submit.prevent="handleLogin" class="space-y-4">
        <div class="space-y-1.5">
          <label for="email" class="label">{{ t('email') }} <span class="text-rose-500">*</span></label>
          <input
            id="email"
            v-model.trim="form.email"
            type="email"
            class="input"
            placeholder="nama@helpdesk.local"
          />
        </div>

        <div class="space-y-1.5">
          <label for="password" class="label">{{ t('password') }} <span class="text-rose-500">*</span></label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            class="input"
            placeholder="••••••••"
          />
        </div>

        <div v-if="error" class="rounded-md bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 text-rose-700 dark:text-rose-300 text-sm p-3">
          {{ error }}
        </div>

        <button type="submit" class="btn-primary w-full" :disabled="loading">
          <span v-if="loading" class="w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin"></span>
          {{ loading ? t('processing') : t('login_btn') }}
        </button>
      </form>

      <div class="mt-6">
        <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wide mb-2">{{ t('demo_accounts') }}</p>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
          <button
            v-for="acc in demoAccounts"
            :key="acc.label"
            type="button"
            class="btn btn-outline btn-sm text-xs truncate"
            @click="fillDemo(acc)"
          >
            {{ acc.label }}
          </button>
        </div>
      </div>

      <p class="text-center text-sm text-slate-500 dark:text-slate-400 mt-6">
        {{ t('no_account') }}
        <router-link to="/register" class="font-medium text-slate-900 dark:text-slate-100 hover:underline">{{ t('register_link') }}</router-link>
      </p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const { t } = useI18n()

const form = reactive({ email: '', password: '' })
const error = ref('')
const loading = ref(false)

const demoAccounts = [
  { label: 'Admin', email: 'admin@helpdesk.local', password: 'Password123!' },
  { label: 'Agent', email: 'agent@helpdesk.local', password: 'Password123!' },
  { label: 'Requester A', email: 'requester1@helpdesk.local', password: 'Password123!' },
  { label: 'Requester B', email: 'requester2@helpdesk.local', password: 'Password123!' }
]

const fillDemo = (acc) => {
  form.email = acc.email
  form.password = acc.password
}

const handleLogin = async () => {
  error.value = ''
  if (!form.email || !form.password) {
    error.value = t('required_fields')
    return
  }

  loading.value = true
  try {
    await auth.login({ email: form.email, password: form.password })
    // Role-based default: admin → stats, everyone else → tickets
    const redirect = route.query.redirect || (auth.user?.role === 'admin' ? '/admin/stats' : '/tickets')
    router.push(redirect)
  } catch (err) {
    error.value = err.message || t('login_failed')
  } finally {
    loading.value = false
  }
}
</script>