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

        <!-- Optional Attachment Upload -->
        <div class="space-y-1.5 pt-1 border-t border-slate-100 dark:border-slate-800">
          <div class="flex items-center justify-between">
            <label class="label text-slate-700 dark:text-slate-300">
              {{ t('attachments') }}
              <span class="text-xs font-normal text-slate-400 dark:text-slate-500">({{ t('optional_label') }})</span>
            </label>
          </div>

          <div class="flex flex-col sm:flex-row sm:items-center gap-3">
            <label class="btn btn-outline btn-sm cursor-pointer shrink-0 inline-flex items-center gap-2">
              <svg class="w-4 h-4 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
              </svg>
              <span>{{ t('add_attachment') }}</span>
              <input
                ref="fileInputRef"
                type="file"
                class="hidden"
                @change="handleFileSelected"
              />
            </label>

            <div v-if="selectedFile" class="flex items-center gap-2 text-xs bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700 px-3 py-1.5 rounded-md min-w-0">
              <span class="font-medium text-slate-800 dark:text-slate-200 truncate">{{ selectedFile.name }}</span>
              <span class="text-slate-400 dark:text-slate-500 shrink-0">({{ formatFileSize(selectedFile.size) }})</span>
              <button
                type="button"
                class="text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 ml-1"
                :title="t('cancel')"
                @click="removeFile"
              >
                ✕
              </button>
            </div>
          </div>

          <p class="text-[11px] text-slate-400 dark:text-slate-500 mt-1">
            {{ t('attachment_help_text') }}
          </p>
          <div v-if="fileError" class="text-xs text-rose-600 dark:text-rose-400 font-medium">
            {{ fileError }}
          </div>
        </div>

        <div v-if="error" class="rounded-md bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 text-rose-700 dark:text-rose-300 text-sm p-3">
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
const fileError = ref('')
const selectedFile = ref(null)
const fileInputRef = ref(null)

const form = reactive({
  title: '',
  category: 'IT Support',
  priority: 'medium',
  description: ''
})

const formatFileSize = (bytes) => {
  if (!bytes || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const handleFileSelected = (event) => {
  const file = event.target.files?.[0]
  if (!file) return

  fileError.value = ''
  // 5MB limit check
  if (file.size > 5 * 1024 * 1024) {
    fileError.value = t('file_too_large')
    if (fileInputRef.value) fileInputRef.value.value = ''
    selectedFile.value = null
    return
  }

  // Extension check (whitelist client-side)
  const allowed = ['jpg', 'jpeg', 'png', 'webp', 'gif', 'pdf', 'txt', 'csv', 'doc', 'docx', 'xls', 'xlsx', 'zip']
  const ext = file.name.split('.').pop()?.toLowerCase()
  if (!ext || !allowed.includes(ext)) {
    fileError.value = t('attachment_upload_failed') + ' (Tipe berkas tidak didukung)'
    if (fileInputRef.value) fileInputRef.value.value = ''
    selectedFile.value = null
    return
  }

  selectedFile.value = file
}

const removeFile = () => {
  selectedFile.value = null
  fileError.value = ''
  if (fileInputRef.value) fileInputRef.value.value = ''
}

const handleSubmit = async () => {
  error.value = ''
  fileError.value = ''
  if (!form.title || !form.description || !form.category || !form.priority) {
    error.value = t('required_fields')
    return
  }

  submitting.value = true
  try {
    const res = await ticketService.create(form)
    const ticketId = res.data.id

    // If optional file was selected, upload it
    if (selectedFile.value) {
      try {
        await ticketService.uploadAttachment(ticketId, selectedFile.value)
      } catch (attErr) {
        // Ticket created, but attachment failed - still redirect but maybe notify
        console.error('Attachment upload failed:', attErr)
      }
    }

    router.push(`/tickets/${ticketId}`)
  } catch (err) {
    error.value = err.message || t('error_title')
  } finally {
    submitting.value = false
  }
}
</script>