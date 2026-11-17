<template>
  <div class="w-full space-y-6">
    <div class="flex items-center justify-between">
      <button class="btn btn-outline btn-sm" @click="$router.back()">← {{ t('back') }}</button>
      <h1 class="text-xl font-bold text-slate-900 dark:text-slate-100">{{ t('create_ticket_title') }}</h1>
      <div class="w-16"></div>
    </div>

    <div class="card p-4 sm:p-6">
      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div class="space-y-1.5">
          <label for="title" class="label">{{ t('ticket_title') }} <span class="text-rose-500">*</span></label>
          <input
            id="title"
            v-model.trim="form.title"
            type="text"
            class="input"
            :placeholder="t('ticket_title_ph')"
          />
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <label for="category" class="label">{{ t('ticket_category') }} <span class="text-rose-500">*</span></label>
            <select id="category" v-model="form.category" class="input">
              <option value="IT Support">IT Support</option>
              <option value="Hardware">Hardware</option>
              <option value="Software">Software</option>
              <option value="Network">Network</option>
              <option value="HR/Facility">HR/Facility</option>
            </select>
          </div>

          <div class="space-y-1.5">
            <label for="priority" class="label">{{ t('ticket_priority') }} <span class="text-rose-500">*</span></label>
            <select id="priority" v-model="form.priority" class="input">
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
              <option value="urgent">Urgent</option>
            </select>
          </div>
        </div>

        <div class="space-y-1.5">
          <label for="description" class="label">{{ t('ticket_desc') }} <span class="text-rose-500">*</span></label>
          <textarea
            id="description"
            v-model.trim="form.description"
            rows="5"
            class="input h-auto"
            :placeholder="t('ticket_desc_ph')"
          ></textarea>
        </div>

        <div v-if="error" class="rounded-md bg-rose-50 border border-rose-200 text-rose-700 text-sm p-3">
          {{ error }}
        </div>

        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="btn btn-outline" @click="$router.back()">{{ t('cancel') }}</button>
          <button type="submit" class="btn-primary" :disabled="submitting">
            <span v-if="submitting" class="w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin"></span>
            {{ submitting ? t('saving') : t('submit_ticket') }}
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
import { useI18n } from '../i18n'

const router = useRouter()
const { t } = useI18n()
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