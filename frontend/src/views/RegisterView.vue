<template>
  <div class="min-h-screen flex items-center justify-center bg-slate-100/60 p-4">
    <div class="card w-full max-w-md p-6 sm:p-8">
      <div class="text-center mb-8">
        <div class="w-12 h-12 rounded-xl bg-slate-900 text-white font-bold flex items-center justify-center text-lg mx-auto mb-4">
          SJ
        </div>
        <h1 class="text-2xl font-bold text-slate-900">Buat Akun</h1>
        <p class="text-sm text-slate-500 mt-1">Daftar sebagai karyawan (requester)</p>
      </div>

      <form @submit.prevent="handleRegister" class="space-y-4">
        <div class="space-y-1.5">
          <label for="name" class="label">Nama Lengkap</label>
          <input id="name" v-model.trim="form.name" type="text" class="input" placeholder="Nama Anda" />
        </div>

        <div class="space-y-1.5">
          <label for="email" class="label">Email</label>
          <input id="email" v-model.trim="form.email" type="email" class="input" placeholder="nama@simplejourney.id" />
        </div>

        <div class="space-y-1.5">
          <label for="password" class="label">Password</label>
          <input id="password" v-model="form.password" type="password" class="input" placeholder="Minimal 8 karakter" />
        </div>

        <div v-if="error" class="rounded-md bg-rose-50 border border-rose-200 text-rose-700 text-sm p-3">
          {{ error }}
        </div>

        <button type="submit" class="btn-primary w-full" :disabled="loading">
          <span v-if="loading" class="w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin"></span>
          {{ loading ? 'Memproses...' : 'Daftar' }}
        </button>
      </form>

      <p class="text-center text-sm text-slate-500 mt-6">
        Sudah punya akun?
        <router-link to="/login" class="font-medium text-slate-900 hover:underline">Masuk</router-link>
      </p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const form = reactive({ name: '', email: '', password: '' })
const error = ref('')
const loading = ref(false)

const handleRegister = async () => {
  error.value = ''
  if (!form.name || !form.email || !form.password) {
    error.value = 'Semua field wajib diisi'
    return
  }
  if (!/^\S+@\S+\.\S+$/.test(form.email)) {
    error.value = 'Format email tidak valid'
    return
  }
  if (form.password.length < 8) {
    error.value = 'Password minimal 8 karakter'
    return
  }

  loading.value = true
  try {
    await auth.register(form)
    router.push('/tickets')
  } catch (err) {
    error.value = err.message || 'Pendaftaran gagal'
  } finally {
    loading.value = false
  }
}
</script>