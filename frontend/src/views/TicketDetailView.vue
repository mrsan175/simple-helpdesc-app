<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <button class="btn btn-outline btn-sm" @click="$router.back()">← Kembali</button>
      <span v-if="loading">Memuat...</span>
    </div>

    <LoadingSkeleton v-if="loading" :rows="3" />

    <ErrorAlert v-else-if="error" :message="error" :show-retry="true" @retry="fetchTicket" />

    <div v-else-if="ticket" class="grid grid-cols-1 lg:grid-cols-3 gap-4 sm:gap-6">
      <!-- Left column: content & comments -->
      <div class="lg:col-span-2 space-y-4 sm:space-y-6 order-2 lg:order-1">
        <div class="card p-4 sm:p-6">
          <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <span class="text-[11px] font-mono text-slate-400 select-all">{{ ticket.id }}</span>
              <h1 class="text-lg sm:text-xl font-semibold text-slate-900 mt-1 leading-snug break-words">
                {{ ticket.title }}
              </h1>
              <p class="text-xs sm:text-sm text-slate-500 mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-0.5">
                <span>Pelapor: <strong class="font-medium text-slate-700">{{ ticket.requester?.name || '-' }}</strong></span>
                <span class="text-slate-300">•</span>
                <span>{{ formatDate(ticket.created_at) }} ({{ formatTime(ticket.created_at) }})</span>
              </p>
            </div>
            <div class="flex items-center sm:flex-col sm:items-end gap-1.5 shrink-0 self-start">
              <StatusBadge :status="ticket.status" />
              <PriorityBadge :priority="ticket.priority" />
            </div>
          </div>

          <div v-if="ticket.is_overdue" class="mt-3 inline-flex items-center gap-1.5 rounded-md bg-rose-50 border border-rose-200 px-2.5 py-1 text-xs font-medium text-rose-700">
            <span class="w-1.5 h-1.5 rounded-full bg-rose-500 animate-pulse"></span>
            SLA Overdue
          </div>

          <hr class="my-4 border-slate-100" />
          <p class="text-xs sm:text-sm text-slate-700 whitespace-pre-wrap leading-relaxed break-words">{{ ticket.description }}</p>
        </div>

        <div class="card p-4 sm:p-6">
          <h2 class="text-sm sm:text-base font-semibold text-slate-900 mb-3 sm:mb-4">
            Diskusi ({{ (ticket.comments || []).length }})
          </h2>

          <div v-if="!(ticket.comments || []).length" class="text-center py-6 text-xs sm:text-sm text-slate-400">
            Belum ada balasan pada tiket ini.
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="comment in ticket.comments || []"
              :key="comment.id"
              :class="[
                'p-3 sm:p-4 rounded-lg border text-xs sm:text-sm transition-colors',
                comment.is_internal ? 'bg-amber-50/70 border-amber-200' : 'bg-white border-slate-200'
              ]"
            >
              <div class="flex items-center justify-between gap-2 mb-1.5">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="w-6 h-6 rounded-full bg-slate-200 text-slate-700 font-semibold flex items-center justify-center text-[10px] shrink-0">
                    {{ (comment.author?.name || 'U').charAt(0).toUpperCase() }}
                  </div>
                  <span class="font-medium text-slate-900 truncate">{{ comment.author?.name || 'User' }}</span>
                  <span
                    v-if="comment.is_internal"
                    class="text-[9px] font-semibold uppercase tracking-wider px-1.5 py-0.5 rounded bg-amber-100 text-amber-800 border border-amber-200 shrink-0"
                  >
                    Internal Note
                  </span>
                </div>
                <span class="text-[11px] text-slate-400 shrink-0">
                  {{ formatDate(comment.created_at) }} · {{ formatTime(comment.created_at) }}
                </span>
              </div>
              <p class="text-slate-700 whitespace-pre-wrap leading-relaxed pl-8 break-words">{{ comment.body }}</p>
            </div>
          </div>

          <div class="mt-5 sm:mt-6 pt-4 border-t border-slate-100">
            <label for="comment" class="label text-xs sm:text-sm">Tulis Balasan <span class="text-rose-500">*</span></label>
            <textarea
              id="comment"
              v-model="newComment.body"
              rows="3"
              class="input mt-1.5 text-xs sm:text-sm"
              placeholder="Tulis balasan atau penjelasan kendala..."
            ></textarea>

            <label v-if="isAgentOrAdmin" class="flex items-start sm:items-center gap-2 mt-2.5 text-xs text-slate-700 cursor-pointer select-none">
              <input
                v-model="newComment.is_internal"
                type="checkbox"
                class="rounded border-slate-300 text-slate-900 focus:ring-slate-900/10 mt-0.5 sm:mt-0"
              />
              <span>Jadikan catatan internal <span class="text-slate-400">(hanya dilihat Agent & Admin)</span></span>
            </label>

            <div class="mt-3 flex justify-end">
              <button
                class="btn-primary btn-sm sm:h-10 sm:px-4 sm:text-sm w-full sm:w-auto"
                :disabled="!newComment.body.trim() || submitting"
                @click="submitComment"
              >
                <span v-if="submitting" class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></span>
                {{ submitting ? 'Mengirim...' : 'Kirim Balasan' }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Right column: controls -->
      <div class="space-y-4 sm:space-y-6 order-1 lg:order-2">
        <!-- Aksi Status: hanya muncul jika ada aksi yang dapat dilakukan -->
        <div v-if="availableActions.length > 0" class="card p-4 sm:p-5 border-slate-900/10 shadow-sm">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2.5">Aksi Status</h2>
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
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3 pb-2 border-b border-slate-100">
            Keterangan Tiket
          </h2>

          <dl class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-1 gap-3 sm:gap-4 text-xs">
            <div>
              <dt class="font-medium text-slate-400 uppercase tracking-wide text-[11px]">Status</dt>
              <dd class="mt-1">
                <StatusBadge :status="ticket.status" />
              </dd>
            </div>

            <div>
              <dt class="font-medium text-slate-400 uppercase tracking-wide text-[11px]">Prioritas</dt>
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
              <dt class="font-medium text-slate-400 uppercase tracking-wide text-[11px]">Ditangani Oleh</dt>
              <dd class="mt-1">
                <select
                  v-if="auth.isAdmin"
                  :value="ticket.assignee_id || ''"
                  class="input text-xs py-1 h-8"
                  @change="updateAssignee($event.target.value)"
                >
                  <option value="">Unassigned</option>
                  <option v-for="u in assignees" :key="u.id" :value="u.id">{{ u.name }}</option>
                </select>
                <div v-else-if="ticket.assignee" class="flex items-center gap-2">
                  <div class="w-6 h-6 rounded-full bg-slate-200 text-slate-700 flex items-center justify-center text-[10px] font-semibold shrink-0">
                    {{ ticket.assignee.name.charAt(0).toUpperCase() }}
                  </div>
                  <div class="min-w-0">
                    <p class="font-medium text-slate-900 truncate">{{ ticket.assignee.name }}</p>
                    <p class="text-[11px] text-slate-500 truncate">{{ ticket.assignee.email }}</p>
                  </div>
                </div>
                <span v-else class="text-slate-400 italic">Belum di-tangani</span>
              </dd>
            </div>

            <div>
              <dt class="font-medium text-slate-400 uppercase tracking-wide text-[11px]">Kategori</dt>
              <dd class="mt-1">
                <span class="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium bg-slate-100 text-slate-700 border border-slate-200">
                  {{ ticket.category }}
                </span>
              </dd>
            </div>

            <div>
              <dt class="font-medium text-slate-400 uppercase tracking-wide text-[11px]">Pelapor</dt>
              <dd class="mt-1">
                <div class="flex items-center gap-2">
                  <div class="w-6 h-6 rounded-full bg-slate-100 text-slate-600 flex items-center justify-center text-[10px] font-semibold shrink-0">
                    {{ ticket.requester?.name?.charAt(0).toUpperCase() || '?' }}
                  </div>
                  <div class="min-w-0">
                    <p class="font-medium text-slate-900 truncate">{{ ticket.requester?.name || '-' }}</p>
                    <p class="text-[11px] text-slate-500 truncate">{{ ticket.requester?.email || '-' }}</p>
                  </div>
                </div>
              </dd>
            </div>

            <div class="sm:col-span-2 lg:col-span-1 pt-2 border-t border-slate-100">
              <dt class="font-medium text-slate-400 uppercase tracking-wide text-[11px]">Waktu Dibuat</dt>
              <dd class="text-slate-700 mt-0.5">
                {{ formatDate(ticket.created_at) }} <span class="text-slate-400">· {{ formatTime(ticket.created_at) }}</span>
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
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ticketService } from '../services/ticket.service'
import { userService } from '../services/user.service'
import StatusBadge from '../components/shared/StatusBadge.vue'
import PriorityBadge from '../components/shared/PriorityBadge.vue'
import LoadingSkeleton from '../components/shared/LoadingSkeleton.vue'
import ErrorAlert from '../components/shared/ErrorAlert.vue'
import ConfirmDialog from '../components/shared/ConfirmDialog.vue'
import Toast from '../components/shared/Toast.vue'
import { canTransition } from '../constants/ticket'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const ticket = ref(null)
const loading = ref(false)
const error = ref('')
const assignees = ref([])
const submitting = ref(false)
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
    actions.push({ target: 'in_progress', label: 'Proses Tiket (In Progress)', variant: 'btn-primary' })
  }
  if (canTransition(status, 'resolved', role)) {
    actions.push({ target: 'resolved', label: 'Tandai Resolved', variant: 'btn-primary' })
  }
  if (canTransition(status, 'closed', role)) {
    actions.push({ target: 'closed', label: 'Tutup Tiket', variant: 'btn-destructive', destructive: true })
  }
  if (canTransition(status, 'open', role)) {
    actions.push({ target: 'open', label: 'Buka Kembali (Reopen)', variant: 'btn-outline' })
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
    error.value = err.message || 'Gagal memuat detail tiket'
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
    in_progress: 'Proses Tiket',
    resolved: 'Tandai Resolved',
    closed: 'Tutup Tiket',
    open: 'Reopen Tiket'
  }[target]

  pendingAction.value = {
    target,
    title: label,
    message: `Apakah Anda yakin ingin mengubah status tiket ke "${label}"?`,
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
    showToast('Status tiket berhasil diperbarui')
  } catch (err) {
    showToast(err.message || 'Gagal memperbarui status', 'error')
  }
}

const updatePriority = async (priority) => {
  try {
    const res = await ticketService.update(ticket.value.id, { priority })
    ticket.value = res.data
    showToast('Prioritas diperbarui')
  } catch (err) {
    showToast(err.message || 'Gagal memperbarui prioritas', 'error')
  }
}

const updateAssignee = async (assigneeId) => {
  try {
    const res = await ticketService.update(ticket.value.id, { assignee_id: assigneeId || null })
    ticket.value = res.data
    showToast('Assignee diperbarui')
  } catch (err) {
    showToast(err.message || 'Gagal memperbarui assignee', 'error')
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
    showToast(err.message || 'Gagal mengirim komentar', 'error')
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
  return new Date(value).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' })
}

const formatTime = (value) => {
  if (!value) return '-'
  return new Date(value).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

onMounted(async () => {
  await fetchTicket()
  await fetchAssignees()
})
</script>