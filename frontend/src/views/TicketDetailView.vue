<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <button class="btn btn-outline btn-sm" @click="$router.back()">← {{ t('back') }}</button>
      <span v-if="loading" class="text-slate-500 dark:text-slate-400">{{ t('loading') }}</span>
    </div>

    <LoadingSkeleton v-if="loading" :rows="3" />

    <ErrorAlert v-else-if="error" :message="error" :show-retry="true" @retry="fetchTicket" />

    <div v-else-if="ticket" class="grid grid-cols-1 lg:grid-cols-3 gap-4 sm:gap-6">
      <!-- Left column: content & comments -->
      <div class="lg:col-span-2 space-y-4 sm:space-y-6 order-2 lg:order-1">
        <div class="card p-4 sm:p-6">
          <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <span class="text-[11px] font-mono text-slate-400 dark:text-slate-500 select-all">{{ ticket.id }}</span>
              <h1 class="text-lg sm:text-xl font-semibold text-slate-900 dark:text-slate-100 mt-1 leading-snug break-words">
                {{ ticket.title }}
              </h1>
              <p class="text-xs sm:text-sm text-slate-500 dark:text-slate-400 mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-0.5">
                <span>{{ t('reporter') }}: <strong class="font-medium text-slate-700 dark:text-slate-300">{{ ticket.requester?.name || '-' }}</strong></span>
                <span class="text-slate-300 dark:text-slate-600">•</span>
                <span>{{ formatDate(ticket.created_at) }} ({{ formatTime(ticket.created_at) }})</span>
              </p>
            </div>
            <div class="flex items-center sm:flex-col sm:items-end gap-1.5 shrink-0 self-start">
              <StatusBadge :status="ticket.status" />
              <PriorityBadge :priority="ticket.priority" />
            </div>
          </div>

          <div v-if="ticket.is_overdue" class="mt-3 inline-flex items-center gap-1.5 rounded-md bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 px-2.5 py-1 text-xs font-medium text-rose-700 dark:text-rose-300">
            <span class="w-1.5 h-1.5 rounded-full bg-rose-500 animate-pulse"></span>
            {{ t('sla_overdue') }}
          </div>

          <hr class="my-4 border-slate-100 dark:border-slate-800" />
          <p class="text-xs sm:text-sm text-slate-700 dark:text-slate-300 whitespace-pre-wrap leading-relaxed break-words">{{ ticket.description }}</p>
        </div>

        <!-- Lampiran / Attachments Section -->
        <div class="card p-4 sm:p-6">
          <div class="flex items-center justify-between mb-3 sm:mb-4">
            <h2 class="text-sm sm:text-base font-semibold text-slate-900 dark:text-slate-100 flex items-center gap-2">
              <svg class="w-4 h-4 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
              </svg>
              {{ t('attachments') }} ({{ (ticket.attachments || []).length }})
            </h2>
            <label
              class="btn btn-outline btn-sm cursor-pointer text-xs flex items-center gap-1.5"
              :class="{ 'opacity-50 pointer-events-none': uploadingFile }"
            >
              <input type="file" class="hidden" @change="handleFileUpload" :disabled="uploadingFile" />
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
              <span>{{ uploadingFile ? t('uploading') : t('add_attachment') }}</span>
            </label>
          </div>

          <div v-if="uploadError" class="mb-3 text-xs rounded-md bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 text-rose-700 dark:text-rose-300 p-2.5">
            {{ uploadError }}
          </div>

          <div v-if="!(ticket.attachments || []).length" class="text-center py-5 text-xs text-slate-400 dark:text-slate-500">
            {{ t('no_attachments') }}
          </div>

          <div v-else class="divide-y divide-slate-100 dark:divide-slate-800">
            <div
              v-for="att in ticket.attachments || []"
              :key="att.id"
              class="py-2.5 flex items-center justify-between gap-3 text-xs"
            >
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="w-7 h-7 rounded bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 flex items-center justify-center shrink-0">
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                  </svg>
                </div>
                <div class="min-w-0">
                  <p class="font-medium text-slate-900 dark:text-slate-100 truncate">{{ att.filename }}</p>
                  <p class="text-[11px] text-slate-400 dark:text-slate-500">
                    {{ formatFileSize(att.file_size) }} · {{ att.uploader?.name || 'User' }}
                  </p>
                </div>
              </div>

              <button
                type="button"
                class="btn btn-outline btn-sm text-xs shrink-0 py-1 px-2.5"
                @click="downloadAttachment(att.id)"
              >
                {{ t('download') }}
              </button>
            </div>
          </div>

          <p class="mt-3 text-[11px] text-slate-400 dark:text-slate-500">
            {{ t('attachment_help_text') }}
          </p>
        </div>

        <div class="card p-4 sm:p-6">
          <h2 class="text-sm sm:text-base font-semibold text-slate-900 dark:text-slate-100 mb-3 sm:mb-4">
            {{ t('discussion') }} ({{ (ticket.comments || []).length }})
          </h2>

          <div v-if="!(ticket.comments || []).length" class="text-center py-6 text-xs sm:text-sm text-slate-400 dark:text-slate-500">
            {{ t('no_comments') }}
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="comment in ticket.comments || []"
              :key="comment.id"
              :class="[
                'p-3 sm:p-4 rounded-lg border text-xs sm:text-sm transition-colors',
                comment.is_internal
                  ? 'bg-amber-50/70 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800'
                  : 'bg-white dark:bg-slate-800/60 border-slate-200 dark:border-slate-700'
              ]"
            >
              <div class="flex items-center justify-between gap-2 mb-1.5">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="w-6 h-6 rounded-full bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 font-semibold flex items-center justify-center text-[10px] shrink-0">
                    {{ (comment.author?.name || 'U').charAt(0).toUpperCase() }}
                  </div>
                  <span class="font-medium text-slate-900 dark:text-slate-100 truncate">{{ comment.author?.name || 'User' }}</span>
                  <span
                    v-if="comment.is_internal"
                    class="text-[9px] font-semibold uppercase tracking-wider px-1.5 py-0.5 rounded bg-amber-100 dark:bg-amber-900/60 text-amber-800 dark:text-amber-200 border border-amber-200 dark:border-amber-700 shrink-0"
                  >
                    {{ t('internal_note') }}
                  </span>
                </div>
                <span class="text-[11px] text-slate-400 dark:text-slate-500 shrink-0">
                  {{ formatDate(comment.created_at) }} · {{ formatTime(comment.created_at) }}
                </span>
              </div>
              <p class="text-slate-700 dark:text-slate-300 whitespace-pre-wrap leading-relaxed pl-8 break-words">{{ comment.body }}</p>
            </div>
          </div>

          <div class="mt-5 sm:mt-6 pt-4 border-t border-slate-100 dark:border-slate-800">
            <label for="comment" class="label text-xs sm:text-sm">{{ t('write_reply') }} <span class="text-rose-500">*</span></label>
            <textarea
              id="comment"
              v-model="newComment.body"
              rows="3"
              class="input mt-1.5 text-xs sm:text-sm"
              :placeholder="t('write_reply_ph')"
            ></textarea>

            <label v-if="isAgentOrAdmin" class="flex items-start sm:items-center gap-2 mt-2.5 text-xs text-slate-700 dark:text-slate-300 cursor-pointer select-none">
              <input
                v-model="newComment.is_internal"
                type="checkbox"
                class="rounded border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 focus:ring-slate-900/10 mt-0.5 sm:mt-0"
              />
              <span>{{ t('mark_internal') }}</span>
            </label>

            <div class="mt-3 flex justify-end">
              <button
                class="btn-primary btn-sm sm:h-10 sm:px-4 sm:text-sm w-full sm:w-auto"
                :disabled="!newComment.body.trim() || submitting"
                @click="submitComment"
              >
                <span v-if="submitting" class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></span>
                {{ submitting ? t('sending') : t('send_reply') }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Right column: controls -->
      <div class="space-y-4 sm:space-y-6 order-1 lg:order-2">
        <!-- Aksi Status: hanya muncul jika ada aksi yang dapat dilakukan -->
        <div v-if="availableActions.length > 0" class="card p-4 sm:p-5 border-slate-900/10 dark:border-slate-800 shadow-sm">
          <h2 class="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-2.5">{{ t('status_actions') }}</h2>
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-1 gap-2">
            <button
              v-for="action in availableActions"
              :key="action.target"
              :class="['btn w-full btn-sm sm:h-10 text-xs sm:text-sm font-medium', action.variant]"
              @click="changeStatus(action.target)"
            >
              {{ action.label }}
            </button>
          </div>
        </div>

        <!-- Keterangan Tiket -->
        <div class="card p-4 sm:p-5">
          <h2 class="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-3 pb-2 border-b border-slate-100 dark:border-slate-800">
            {{ t('ticket_info') }}
          </h2>

          <dl class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-1 gap-3 sm:gap-4 text-xs">
            <div>
              <dt class="font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wide text-[11px]">{{ t('ticket_id') }}</dt>
              <dd class="mt-1">
                <span class="inline-block text-[11px] font-mono bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 px-2 py-0.5 rounded break-all select-all">
                  {{ ticket.id }}
                </span>
              </dd>
            </div>

            <div>
              <dt class="font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wide text-[11px]">{{ t('th_status') }}</dt>
              <dd class="mt-1">
                <StatusBadge :status="ticket.status" />
              </dd>
            </div>

            <div>
              <dt class="font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wide text-[11px]">{{ t('th_priority') }}</dt>
              <dd class="mt-1">
                <select
                  v-if="canEditPriority"
                  :value="ticket.priority"
                  class="input text-xs py-1 h-8"
                  @change="updatePriority($event.target.value)"
                >
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="urgent">Urgent</option>
                </select>
                <PriorityBadge v-else :priority="ticket.priority" />
              </dd>
            </div>

            <div>
              <dt class="font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wide text-[11px]">{{ t('th_assignee') }}</dt>
              <dd class="mt-1">
                <select
                  v-if="auth.isAdmin"
                  :value="ticket.assignee_id || ''"
                  class="input text-xs py-1 h-8"
                  @change="updateAssignee($event.target.value)"
                >
                  <option value="">{{ t('filter_unassigned') }}</option>
                  <option v-for="u in assignees" :key="u.id" :value="u.id">{{ u.name }}</option>
                </select>
                <div v-else-if="ticket.assignee" class="flex items-center gap-2">
                  <div class="w-6 h-6 rounded-full bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 flex items-center justify-center text-[10px] font-semibold shrink-0">
                    {{ ticket.assignee.name.charAt(0).toUpperCase() }}
                  </div>
                  <div class="min-w-0">
                    <p class="font-medium text-slate-900 dark:text-slate-100 truncate">{{ ticket.assignee.name }}</p>
                    <p class="text-[11px] text-slate-500 dark:text-slate-400 truncate">{{ ticket.assignee.email }}</p>
                  </div>
                </div>
                <span v-else class="text-slate-400 dark:text-slate-500 italic">{{ t('unassigned_label') }}</span>
              </dd>
            </div>

            <div>
              <dt class="font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wide text-[11px]">{{ t('ticket_category') }}</dt>
              <dd class="mt-1">
                <span class="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-800">
                  {{ ticket.category }}
                </span>
              </dd>
            </div>

            <div>
              <dt class="font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wide text-[11px]">{{ t('th_requester') }}</dt>
              <dd class="mt-1">
                <div class="flex items-center gap-2">
                  <div class="w-6 h-6 rounded-full bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 flex items-center justify-center text-[10px] font-semibold shrink-0">
                    {{ ticket.requester?.name?.charAt(0).toUpperCase() || '?' }}
                  </div>
                  <div class="min-w-0">
                    <p class="font-medium text-slate-900 dark:text-slate-100 truncate">{{ ticket.requester?.name || '-' }}</p>
                    <p class="text-[11px] text-slate-500 dark:text-slate-400 truncate">{{ ticket.requester?.email || '-' }}</p>
                  </div>
                </div>
              </dd>
            </div>

            <div class="sm:col-span-2 lg:col-span-1 pt-2 border-t border-slate-100 dark:border-slate-800">
              <dt class="font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wide text-[11px]">{{ t('created_at') }}</dt>
              <dd class="text-slate-700 dark:text-slate-300 mt-0.5">
                {{ formatDate(ticket.created_at) }} <span class="text-slate-400 dark:text-slate-500">· {{ formatTime(ticket.created_at) }}</span>
              </dd>
            </div>
          </dl>
        </div>
      </div>
    </div>

    <ConfirmDialog
      v-if="pendingAction"
      :open="!!pendingAction"
      :title="pendingAction.title"
      :message="pendingAction.message"
      :confirm-label="pendingAction.confirmLabel"
      :destructive="pendingAction.destructive"
      @confirm="confirmAction"
      @close="pendingAction = null"
    />

    <Toast :message="toast.message" :type="toast.type" :visible="toast.visible" />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ticketService } from '../services/ticket.service'
import { userService } from '../services/user.service'
import { useRealtime } from '../services/realtime.service'
import StatusBadge from '../components/shared/StatusBadge.vue'
import PriorityBadge from '../components/shared/PriorityBadge.vue'
import LoadingSkeleton from '../components/shared/LoadingSkeleton.vue'
import ErrorAlert from '../components/shared/ErrorAlert.vue'
import ConfirmDialog from '../components/shared/ConfirmDialog.vue'
import Toast from '../components/shared/Toast.vue'
import { canTransition } from '../constants/ticket'
import { useI18n } from '../i18n'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()
const realtime = useRealtime()

const ticket = ref(null)
const loading = ref(false)
const error = ref('')
const assignees = ref([])
const submitting = ref(false)
const uploadingFile = ref(false)
const uploadError = ref('')
const toast = reactive({ message: '', type: 'success', visible: false })
const pendingAction = ref(null)

const newComment = reactive({ body: '', is_internal: false })

const isAgentOrAdmin = computed(() => auth.isAgent || auth.isAdmin)

// Admin selalu bisa ubah prioritas; agent hanya untuk tiket yang di-assign ke dirinya.
const canEditPriority = computed(() => {
  if (auth.isAdmin) return true
  if (auth.isAgent) return Boolean(ticket.value?.assignee_id && ticket.value.assignee_id === auth.user?.id)
  return false
})

const availableActions = computed(() => {
  if (!ticket.value) return []
  const role = auth.user?.role
  const status = ticket.value.status
  const actions = []

  if (canTransition(status, 'in_progress', role)) {
    actions.push({ target: 'in_progress', label: t('action_in_progress'), variant: 'btn-primary' })
  }
  if (canTransition(status, 'resolved', role)) {
    actions.push({ target: 'resolved', label: t('action_resolved'), variant: 'btn-primary' })
  }
  if (canTransition(status, 'closed', role)) {
    actions.push({ target: 'closed', label: t('action_closed'), variant: 'btn-destructive', destructive: true })
  }
  if (canTransition(status, 'open', role)) {
    actions.push({ target: 'open', label: t('action_reopen'), variant: 'btn-outline' })
  }

  return actions
})

const fetchTicket = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await ticketService.get(route.params.id)
    ticket.value = res.data
  } catch (err) {
    error.value = err.message || t('load_ticket_failed')
  } finally {
    loading.value = false
  }
}

const fetchAssignees = async () => {
  if (!auth.isAdmin) return
  try {
    const res = await userService.list()
    assignees.value = (res.data || []).filter((u) => u.role === 'agent' || u.role === 'admin')
  } catch {
    assignees.value = []
  }
}

const changeStatus = (target) => {
  const label = {
    in_progress: t('action_in_progress'),
    resolved: t('action_resolved'),
    closed: t('action_closed'),
    open: t('action_reopen')
  }[target]

  pendingAction.value = {
    target,
    title: label,
    message: t('confirm_status', { label }),
    confirmLabel: label,
    destructive: target === 'closed'
  }
}

const confirmAction = async () => {
  if (!pendingAction.value) return
  await updateStatus(pendingAction.value.target)
  pendingAction.value = null
}

const updateStatus = async (status) => {
  try {
    const res = await ticketService.update(ticket.value.id, { status })
    ticket.value = res.data
    showToast(t('status_updated'))
  } catch (err) {
    showToast(err.message || t('status_update_failed'), 'error')
  }
}

const updatePriority = async (priority) => {
  try {
    const res = await ticketService.update(ticket.value.id, { priority })
    ticket.value = res.data
    showToast(t('priority_updated'))
  } catch (err) {
    showToast(err.message || t('priority_update_failed'), 'error')
  }
}

const updateAssignee = async (assigneeId) => {
  try {
    const res = await ticketService.update(ticket.value.id, { assignee_id: assigneeId || null })
    ticket.value = res.data
    showToast(t('assignee_updated'))
  } catch (err) {
    showToast(err.message || t('assignee_update_failed'), 'error')
  }
}

const submitComment = async () => {
  if (!newComment.body.trim()) return
  submitting.value = true
  try {
    await ticketService.addComment(ticket.value.id, newComment)
    newComment.body = ''
    newComment.is_internal = false
    await fetchTicket()
  } catch (err) {
    showToast(err.message || t('comment_failed'), 'error')
  } finally {
    submitting.value = false
  }
}

let toastTimer = null
const showToast = (message, type = 'success') => {
  toast.message = message
  toast.type = type
  toast.visible = true
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toast.visible = false
  }, 3000)
}

const formatDate = (value) => {
  if (!value) return '-'
  return new Date(value).toLocaleDateString('id-ID', { day: '2-digit', month: 'numeric', year: 'numeric' })
}

const formatTime = (value) => {
  if (!value) return '-'
  return new Date(value).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

const formatFileSize = (bytes) => {
  if (!bytes || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const handleFileUpload = async (event) => {
  const file = event.target.files?.[0]
  if (!file) return

  // Validate size client-side (5MB)
  if (file.size > 5 * 1024 * 1024) {
    uploadError.value = t('file_too_large')
    event.target.value = ''
    return
  }

  uploadError.value = ''
  uploadingFile.value = true
  try {
    await ticketService.uploadAttachment(ticket.value.id, file)
    showToast(t('attachment_uploaded'))
    await fetchTicket()
  } catch (err) {
    uploadError.value = err.message || t('attachment_upload_failed')
  } finally {
    uploadingFile.value = false
    event.target.value = ''
  }
}

const downloadAttachment = (attachmentId) => {
  ticketService.downloadAttachment(attachmentId)
}

let detailUnsubs = []

onMounted(async () => {
  await fetchTicket()
  await fetchAssignees()

  // Real-time auto-reload for this ticket
  detailUnsubs.push(
    realtime.onEvent('comment_added', (p) => {
      if (p?.ticket_id === route.params.id) fetchTicket()
    })
  )
  detailUnsubs.push(
    realtime.onEvent('attachment_added', (p) => {
      if (p?.ticket_id === route.params.id) fetchTicket()
    })
  )
  detailUnsubs.push(
    realtime.onEvent('ticket_updated', (p) => {
      if (p?.id === route.params.id) fetchTicket()
    })
  )
})

onUnmounted(() => {
  detailUnsubs.forEach((fn) => fn())
})
</script>