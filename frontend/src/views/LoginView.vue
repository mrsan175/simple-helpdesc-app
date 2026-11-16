<template>
  <div class="min-h-screen flex items-center justify-center bg-slate-100/60 p-4">
    <div class="card w-full max-w-md p-6 sm:p-8">
      <div class="text-center mb-8">
        <div class="w-12 h-12 rounded-xl bg-slate-900 text-white font-bold flex items-center justify-center text-lg mx-auto mb-4">
          SJ
        </div>
        <h1 class="text-2xl font-bold text-slate-900">Masuk ke Helpdesk</h1>
        <p class="text-sm text-slate-500 mt-1">PT Simple Journey Indonesia</p>
      </div>

      <form @submit.prevent="handleLogin" class="space-y-4">
        <div class="space-y-1.5">
          <label for="email" class="label">Email <span class="text-rose-500">*</span></label>
          <input
            id="email"
            v-model.trim="form.email"
            type="email"
            class="input"
            placeholder="nama@simplejourney.id"
          />
        </div>

        <div class="space-y-1.5">
          <label for="password" class="label">Password <span class="text-rose-500">*</span></label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            class="input"
            placeholder="Masukkan password"
          />
        </div>

        <div v-if="error" class="rounded-md bg-rose-50 border border-rose-200 text-rose-700 text-sm p-3">
          {{ error }}
        </div>

        <button type="submit" class="btn-primary w-full" :disabled="loading">
          <span v-if="loading" class="w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin"></span>
          {{ loading ? 'Memproses...' : 'Masuk' }}
        </button>
      </form>

      <div class="mt-6">
        <p class="text-xs font-medium text-slate-500 uppercase tracking-wide mb-2">Akun Demo</p>
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

      <p class="text-center text-sm text-slate-500 mt-6">
        Belum punya akun?
        <router-link to="/register" class="font-medium text-slate-900 hover:underline">Daftar</router-link>
      </p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const form = reactive({ email: '', password: '' })
const error = ref('')
const loading = ref(false)

const demoAccounts = [
  { label: 'Admin', email: 'admin@simplejourney.id', password: 'Password123!' },
  { label: 'Agent', email: 'agent@simplejourney.id', password: 'Password123!' },
  { label: 'Requester A', email: 'requester1@simplejourney.id', password: 'Password123!' },
  { label: 'Requester B', email: 'requester2@simplejourney.id', password: 'Password123!' }
]

const fillDemo = (acc) => {
  form.email = acc.email
  form.password = acc.password
}

const handleLogin = async () => {
  error.value = ''
  if (!form.email || !form.password) {
    error.value = 'Email dan password wajib diisi'
    return
  }

  loading.value = true
  try {
    await auth.login({ email: form.email, password: form.password })
    const redirect = route.query.redirect || '/tickets'
    router.push(redirect)
  } catch (err) {
    error.value = err.message || 'Email atau password salah'
  } finally {
    loading.value = false
  }
}
</script>