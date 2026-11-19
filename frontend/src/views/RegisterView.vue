<template>
  <div class="min-h-screen flex items-center justify-center bg-slate-100/60 dark:bg-slate-950 p-4">
    <div class="card w-full max-w-md p-6 sm:p-8">
      <div class="text-center mb-8">
        <div class="w-12 h-12 rounded-xl bg-slate-900 text-white font-bold flex items-center justify-center text-lg mx-auto mb-4">
          SJ
        </div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">{{ t('register_title') }}</h1>
        <p class="text-sm text-slate-500 dark:text-slate-400 mt-1">{{ t('register_subtitle') }}</p>
      </div>

      <form @submit.prevent="handleRegister" class="space-y-4">
        <div class="space-y-1.5">
          <label for="name" class="label">{{ t('full_name') }} <span class="text-rose-500">*</span></label>
          <input id="name" v-model.trim="form.name" type="text" class="input" :placeholder="t('full_name')" />
        </div>

        <div class="space-y-1.5">
          <label for="email" class="label">{{ t('email') }} <span class="text-rose-500">*</span></label>
          <input id="email" v-model.trim="form.email" type="email" class="input" placeholder="nama@simplejourney.id" />
        </div>

        <div class="space-y-1.5">
          <label for="password" class="label">{{ t('password') }} <span class="text-rose-500">*</span></label>
          <input id="password" v-model="form.password" type="password" class="input" :placeholder="t('password_min')" />
        </div>

        <div v-if="error" class="rounded-md bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 text-rose-700 dark:text-rose-300 text-sm p-3">
          {{ error }}
        </div>

        <button type="submit" class="btn-primary w-full" :disabled="loading">
          <span v-if="loading" class="w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin"></span>
          {{ loading ? t('processing') : t('register_btn') }}
        </button>
      </form>

      <p class="text-center text-sm text-slate-500 dark:text-slate-400 mt-6">
        {{ t('have_account') }}
        <router-link to="/login" class="font-medium text-slate-900 dark:text-slate-100 hover:underline">{{ t('login_link') }}</router-link>
      </p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'

const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()
const form = reactive({ name: '', email: '', password: '' })
const error = ref('')
const loading = ref(false)

const handleRegister = async () => {
  error.value = ''
  if (!form.name || !form.email || !form.password) {
    error.value = t('required_fields')
    return
  }
  if (!/^\S+@\S+\.\S+$/.test(form.email)) {
    error.value = t('invalid_email')
    return
  }
  if (form.password.length < 8) {
    error.value = t('password_min_error')
    return
  }

  loading.value = true
  try {
    await auth.register(form)
    router.push('/')
  } catch (err) {
    error.value = err.message || t('register_failed')
  } finally {
    loading.value = false
  }
}
</script>