<template>
  <div class="max-w-2xl mx-auto space-y-6">
    <div class="flex items-center justify-between">
      <button class="btn btn-outline btn-sm" @click="$router.back()">← Kembali</button>
      <h1 class="text-xl font-bold text-slate-900">Buat Tiket Baru</h1>
      <div class="w-16"></div>
    </div>

    <div class="card p-6">
      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div class="space-y-1.5">
          <label for="title" class="label">Judul Tiket</label>
          <input
            id="title"
            v-model.trim="form.title"
            type="text"
            class="input"
            placeholder="Ringkasan masalah Anda..."
          />
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <label for="category" class="label">Kategori</label>
            <select id="category" v-model="form.category" class="input">
              <option value="IT Support">IT Support</option>
              <option value="Hardware">Hardware</option>
              <option value="Software">Software</option>
              <option value="Network">Network</option>
              <option value="HR/Facility">HR/Facility</option>
            </select>
          </div>

          <div class="space-y-1.5">
            <label for="priority" class="label">Prioritas</label>
            <select id="priority" v-model="form.priority" class="input">
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
              <option value="urgent">Urgent</option>
            </select>
          </div>
        </div>

        <div class="space-y-1.5">
          <label for="description" class="label">Deskripsi Lengkap</label>
          <textarea
            id="description"
            v-model.trim="form.description"
            rows="5"
            class="input h-auto"
            placeholder="Jelaskan detail kendala yang dialami, langkah yang sudah dicoba, dll..."
          ></textarea>
        </div>

        <div v-if="error" class="rounded-md bg-rose-50 border border-rose-200 text-rose-700 text-sm p-3">
          {{ error }}
        </div>

        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="btn btn-outline" @click="$router.back()">Batal</button>
          <button type="submit" class="btn-primary" :disabled="submitting">
            <span v-if="submitting" class="w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin"></span>
            {{ submitting ? 'Menyimpan...' : 'Kirim Tiket' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ticketService } from '../services/ticket.service'

const router = useRouter()
const submitting = ref(false)
const error = ref('')

const form = reactive({
  title: '',
  category: 'IT Support',
  priority: 'medium',
  description: ''
})

const handleSubmit = async () => {
  error.value = ''
  if (!form.title || !form.description || !form.category || !form.priority) {
    error.value = 'Semua field wajib diisi'
    return
  }

  submitting.value = true
  try {
    const res = await ticketService.create(form)
    router.push(`/tickets/${res.data.id}`)
  } catch (err) {
    error.value = err.message || 'Gagal membuat tiket'
  } finally {
    submitting.value = false
  }
}
</script>