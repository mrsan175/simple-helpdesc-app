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

        <!-- Lampiran utama hanya dikelola pemilik tiket. -->
        <div class="card p-4 sm:p-6" data-testid="ticket-attachments">
          <div class="flex items-center justify-between mb-3 sm:mb-4">
            <h2 class="text-sm sm:text-base font-semibold text-slate-900 dark:text-slate-100 flex items-center gap-2">
              <svg class="w-4 h-4 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
              </svg>
              {{ t('ticket_attachments') }} ({{ (ticket.attachments || []).length }})
            </h2>
            <label
              v-if="isTicketOwner"
              class="btn btn-outline btn-sm cursor-pointer text-xs flex items-center gap-1.5"
              :class="{ 'opacity-50 pointer-events-none': isMutating }"
            >
              <input type="file" class="hidden" @change="handleFileUpload" :disabled="isMutating" />
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

          <AttachmentList
            v-else
            :attachments="ticket.attachments || []"
            :can-manage="isTicketOwner"
            :busy="isMutating"
            @preview="openTicketAttachmentPreview"
            @download="downloadAttachment"
            @replace="replaceAttachment"
            @delete="confirmDeleteAttachment"
          />

          <p class="mt-3 text-xs text-slate-500 dark:text-slate-400">
            {{ t('ticket_attachment_owner_only') }}
          </p>

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
              :data-comment-id="comment.id"
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
                <span class="text-[11px] text-slate-400 dark:text-slate-500 text-right">
                  {{ formatDate(comment.created_at) }} · {{ formatTime(comment.created_at) }}
                  <span v-if="comment.updated_at" class="block" :title="formatDate(comment.updated_at) + ' · ' + formatTime(comment.updated_at)">{{ t('edited') }}</span>
                </span>
              </div>
              <div v-if="!isClosed && editingCommentId === comment.id" class="pl-8 space-y-2">
                <label :for="'edit-comment-' + comment.id" class="label">{{ t('edit_comment') }}</label>
                <textarea :id="'edit-comment-' + comment.id" v-model="editCommentBody" rows="3" class="input" :disabled="isMutating"></textarea>
                <div class="flex justify-end gap-2">
                  <button type="button" class="btn btn-outline btn-sm" :disabled="isMutating" @click="cancelEditComment">{{ t('cancel') }}</button>
                  <button type="button" class="btn-primary btn-sm" :disabled="isMutating || !editCommentBody.trim()" @click="saveComment(comment.id)">{{ t('save') }}</button>
                </div>
              </div>
              <p v-else class="text-slate-700 dark:text-slate-300 whitespace-pre-wrap leading-relaxed pl-8 break-words">{{ comment.body }}</p>

              <div v-if="comment.attachments?.length" class="mt-3 ml-8 border-t border-slate-200 dark:border-slate-700 pt-2">
                <p class="text-[11px] font-medium text-slate-500 dark:text-slate-400">{{ t('comment_attachment') }}</p>
                <AttachmentList
                  :attachments="comment.attachments"
                  :can-manage="isCommentAuthor(comment) && !isClosed"
                  :busy="isMutating"
                  @preview="openCommentAttachmentPreview(comment, $event)"
                  @download="downloadAttachment"
                  @replace="replaceAttachment"
                  @delete="confirmDeleteAttachment"
                />
              </div>

              <div v-if="isCommentAuthor(comment) && !isClosed && editingCommentId !== comment.id" class="flex justify-end gap-2 mt-3">
                <button type="button" class="btn btn-outline btn-sm text-xs" :disabled="isMutating" @click="startEditComment(comment)">{{ t('edit_comment') }}</button>
                <button type="button" class="btn btn-outline btn-sm text-xs text-rose-600 dark:text-rose-400" :disabled="isMutating" @click="confirmDeleteComment(comment)">{{ t('delete_comment') }}</button>
              </div>
            </div>
          </div>

          <div v-if="!isClosed" class="mt-5 sm:mt-6 pt-4 border-t border-slate-100 dark:border-slate-800">
            <label for="comment" class="label text-xs sm:text-sm">{{ t('write_reply') }} <span class="text-rose-500">*</span></label>
            <textarea
              id="comment"
              v-model="newComment.body"
              rows="3"
              class="input mt-1.5 text-xs sm:text-sm"
              :placeholder="t('write_reply_ph')"
              :disabled="isMutating || !!pendingCommentId"
            ></textarea>
            <p v-if="pendingCommentId" class="mt-2 text-xs text-amber-700 dark:text-amber-300" role="status">{{ t('comment_upload_pending') }}</p>

            <label v-if="isAgentOrAdmin" class="flex items-start sm:items-center gap-2 mt-2.5 text-xs text-slate-700 dark:text-slate-300 cursor-pointer select-none">
              <input
                v-model="newComment.is_internal"
                :disabled="isMutating || !!pendingCommentId"
                type="checkbox"
                class="rounded border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 focus:ring-slate-900/10 mt-0.5 sm:mt-0"
              />
              <span>{{ t('mark_internal') }}</span>
            </label>

            <div class="mt-2.5 flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-3">
              <label class="btn btn-outline btn-sm cursor-pointer inline-flex items-center gap-1.5 text-xs shrink-0 self-start" :class="{ 'opacity-50 pointer-events-none': isMutating }">
                <svg class="w-3.5 h-3.5 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
                </svg>
                <span>{{ t('attach_file') }}</span>
                <input
                  ref="commentFileInput"
                  :disabled="isMutating"
                  type="file"
                  class="hidden"
                  @change="handleCommentFileSelected"
                />
              </label>
            </div>

            <DraftFilePreview
              :file="commentFile"
              :busy="isMutating"
              class="mt-2.5"
              @preview="openDraftPreview"
              @remove="removeCommentFile"
            />

            <div v-if="commentFileError" class="text-[11px] text-rose-600 dark:text-rose-400 font-medium mt-1">
              {{ commentFileError }}
            </div>

            <div class="mt-3 flex justify-end">
              <button
                class="btn-primary btn-sm sm:h-10 sm:px-4 sm:text-sm w-full sm:w-auto"
                :disabled="isMutating || (!pendingCommentId && !newComment.body.trim())"
                @click="submitComment"
              >
                <span v-if="submitting" class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></span>
                {{ submitting ? t('sending') : pendingCommentId ? (commentFile ? t('retry_attachment') : t('finish_comment')) : t('send_reply') }}
              </button>
            </div>
          </div>

          <div v-else class="mt-5 sm:mt-6 rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs text-slate-600 dark:border-slate-700 dark:bg-slate-800/60 dark:text-slate-300" role="status">
            {{ t('discussion_closed_notice') }}
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
              :disabled="isMutating"
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
                  :value="ticket.priority || ''"
                  :disabled="isMutating"
                  class="input text-xs py-1 h-8"
                  @change="updatePriority($event.target.value)"
                >
                  <option value="" disabled>{{ t('select_priority') }}</option>
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
                  :disabled="isMutating"
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
                <select
                  v-if="auth.isAdmin"
                  :value="ticket.requester_id || ''"
                  :disabled="isMutating"
                  class="input text-xs py-1 h-8"
                  @change="updateRequester($event.target.value)"
                >
                  <option v-for="u in requesters" :key="u.id" :value="u.id">{{ u.name }} ({{ u.email }})</option>
                </select>
                <div v-else class="flex items-center gap-2">
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

    <FilePreview
      v-if="previewTarget"
      :file="previewTarget.file"
      :attachment="previewTarget.attachment"
      :context-label="previewTarget.contextLabel"
      @close="previewTarget = null"
    />

    <ConfirmDialog
      v-if="pendingAction"
      :open="!!pendingAction"
      :title="pendingAction.title"
      :message="pendingAction.message"
      :confirm-label="pendingAction.confirmLabel"
      :destructive="pendingAction.destructive"
      :busy="contentBusy"
      @confirm="confirmAction"
      @close="closePendingAction"
    />

    <Toast :message="toast.message" :type="toast.type" :visible="toast.visible" />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
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
import AttachmentList from '../components/shared/AttachmentList.vue'
import FilePreview from '../components/shared/FilePreview.vue'
import DraftFilePreview from '../components/shared/DraftFilePreview.vue'
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
const requesters = ref([])
const submitting = ref(false)
const uploadingFile = ref(false)
const uploadError = ref('')
const commentFile = ref(null)
const commentFileError = ref('')
const commentFileInput = ref(null)
const toast = reactive({ message: '', type: 'success', visible: false })
const pendingAction = ref(null)
const contentBusy = ref(false)
const editingCommentId = ref(null)
const editCommentBody = ref('')
// Keep a saved comment ID when its separate upload fails, so retrying cannot duplicate it.
const pendingCommentId = ref(null)
const previewTarget = ref(null)
const isMutating = computed(() => contentBusy.value || submitting.value || uploadingFile.value)
const isTicketOwner = computed(() => !!ticket.value && ticket.value.requester_id === auth.user?.id)
const isClosed = computed(() => ticket.value?.status === 'closed')
const isCommentAuthor = (comment) => comment.author_id === auth.user?.id

const newComment = reactive({ body: '', is_internal: false })

const isAgentOrAdmin = computed(() => auth.isAgent || auth.isAdmin)

const STATUS_LABEL_KEYS = {
  open: 'stat_open',
  in_progress: 'stat_in_progress',
  resolved: 'stat_resolved',
  closed: 'stat_closed'
}

// Admin dan Agent dapat mengatur prioritas tiket
const canEditPriority = computed(() => {
  return auth.isAdmin || auth.isAgent
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

let fetchSequence = 0
const fetchTicket = async ({ silent = false } = {}) => {
  const sequence = ++fetchSequence
  if (!silent) loading.value = true
  try {
    const res = await ticketService.get(route.params.id)
    if (sequence !== fetchSequence) return
    ticket.value = res.data
    error.value = ''
  } catch (err) {
    if (sequence === fetchSequence) error.value = err.message || t('load_ticket_failed')
  } finally {
    if (sequence === fetchSequence) loading.value = false
  }
}

const fetchUsers = async () => {
  if (!auth.isAdmin) return
  try {
    const res = await userService.list()
    const users = res.data || []
    assignees.value = users.filter((u) => u.role === 'agent' || u.role === 'admin')
    requesters.value = users
  } catch {
    assignees.value = []
    requesters.value = []
  }
}

const changeStatus = (target) => {
  const label = {
    in_progress: t('action_in_progress'),
    resolved: t('action_resolved'),
    closed: t('action_closed'),
    open: t('action_reopen')
  }[target]
  const statusKey = {
    open: 'stat_open',
    in_progress: 'stat_in_progress',
    resolved: 'stat_resolved',
    closed: 'stat_closed'
  }[target]
  const impactKey = {
    open: 'impact_reopen',
    in_progress: 'impact_in_progress',
    resolved: 'impact_resolved',
    closed: 'impact_closed'
  }[target]

  pendingAction.value = {
    kind: 'status',
    target,
    title: t('status_change_title'),
    message: [
      `${t('status_from')}: ${t(STATUS_LABEL_KEYS[ticket.value.status] || 'th_status')}`,
      `${t('status_to')}: ${t(statusKey)}`,
      `${t('status_impact')}: ${t(impactKey)}`
    ].join('\n'),
    confirmLabel: label,
    destructive: target === 'closed'
  }
}

const closePendingAction = () => {
  if (!contentBusy.value) pendingAction.value = null
}

const confirmAction = async () => {
  if (!pendingAction.value || isMutating.value) return
  const action = pendingAction.value
  contentBusy.value = true
  try {
    if (action.kind === 'status') {
      if (!(await updateStatus(action.target))) return
    } else if (action.kind === 'deleteAttachment') {
      await ticketService.deleteAttachment(action.id)
      showToast(t('attachment_deleted'))
      await fetchTicket({ silent: true })
    } else if (action.kind === 'deleteComment') {
      await ticketService.deleteComment(action.id)
      if (editingCommentId.value === action.id) cancelEditComment()
      if (pendingCommentId.value === action.id) resetCommentDraft()
      showToast(t('comment_deleted'))
      await fetchTicket({ silent: true })
    }
    pendingAction.value = null
  } catch (err) {
    showToast(err.message || t('error_title'), 'error')
  } finally {
    contentBusy.value = false
  }
}

const updateStatus = async (status) => {
  try {
    const res = await ticketService.update(ticket.value.id, { status })
    ticket.value = res.data
    showToast(t('status_updated'))
    return true
  } catch (err) {
    showToast(err.message || t('status_update_failed'), 'error')
    return false
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

const updateRequester = async (requesterId) => {
  if (!requesterId) return
  try {
    const res = await ticketService.update(ticket.value.id, { requester_id: requesterId })
    ticket.value = res.data
    showToast(t('requester_updated'))
  } catch (err) {
    showToast(err.message || t('requester_update_failed'), 'error')
  }
}

const startEditComment = (comment) => {
  if (isClosed.value || !isCommentAuthor(comment) || isMutating.value) return
  editingCommentId.value = comment.id
  editCommentBody.value = comment.body
}

const cancelEditComment = () => {
  editingCommentId.value = null
  editCommentBody.value = ''
}

const saveComment = async (id) => {
  if (isClosed.value || isMutating.value || !editCommentBody.value.trim()) return
  contentBusy.value = true
  try {
    await ticketService.updateComment(id, { body: editCommentBody.value.trim() })
    cancelEditComment()
    showToast(t('comment_updated'))
    await fetchTicket({ silent: true })
  } catch (err) {
    showToast(err.message || t('error_title'), 'error')
  } finally {
    contentBusy.value = false
  }
}

const confirmDeleteComment = (comment) => {
  if (isClosed.value || !isCommentAuthor(comment) || isMutating.value) return
  pendingAction.value = {
    kind: 'deleteComment', id: comment.id,
    title: t('delete_comment'), message: t('confirm_delete_comment'),
    confirmLabel: t('delete'), destructive: true
  }
}

const confirmDeleteAttachment = (attachment) => {
  if (isMutating.value) return
  pendingAction.value = {
    kind: 'deleteAttachment', id: attachment.id,
    title: t('delete_attachment'), message: t('confirm_delete_attachment', { name: attachment.filename }),
    confirmLabel: t('delete'), destructive: true
  }
}

const validateFile = (file) => {
  if (file.size > 5 * 1024 * 1024) return t('file_too_large')
  const allowed = ['jpg', 'jpeg', 'png', 'webp', 'gif', 'pdf', 'txt', 'csv', 'doc', 'docx', 'xls', 'xlsx', 'zip']
  const ext = file.name.split('.').pop()?.toLowerCase()
  return allowed.includes(ext) ? '' : t('unsupported_file_type')
}

const replaceAttachment = async ({ attachment, file }) => {
  if (isMutating.value) return
  const message = validateFile(file)
  if (message) return showToast(message, 'error')
  contentBusy.value = true
  try {
    await ticketService.replaceAttachment(attachment.id, file)
    showToast(t('attachment_replaced'))
    await fetchTicket({ silent: true })
  } catch (err) {
    showToast(err.message || t('attachment_upload_failed'), 'error')
  } finally {
    contentBusy.value = false
  }
}

const handleCommentFileSelected = (event) => {
  const file = event.target.files?.[0]
  if (!file) return
  commentFileError.value = validateFile(file)
  if (commentFileError.value) {
    event.target.value = ''
    commentFile.value = null
    return
  }
  commentFile.value = file
}

const removeCommentFile = () => {
  commentFile.value = null
  commentFileError.value = ''
  if (commentFileInput.value) commentFileInput.value.value = ''
}

const resetCommentDraft = () => {
  pendingCommentId.value = null
  newComment.body = ''
  newComment.is_internal = false
  removeCommentFile()
}

const submitComment = async () => {
  if (isClosed.value || isMutating.value) return
  if (!pendingCommentId.value && !newComment.body.trim()) {
    commentFileError.value = t('comment_body_required')
    return
  }
  submitting.value = true
  commentFileError.value = ''
  try {
    if (!pendingCommentId.value) {
      const res = await ticketService.addComment(ticket.value.id, {
        body: newComment.body.trim(), is_internal: newComment.is_internal
      })
      pendingCommentId.value = res.data.id
    }
    if (commentFile.value) {
      await ticketService.uploadCommentAttachment(pendingCommentId.value, commentFile.value)
    }
    resetCommentDraft()
    showToast(t('comment_sent'))
  } catch (err) {
    commentFileError.value = err.message || t('comment_failed')
    showToast(commentFileError.value, 'error')
  } finally {
    await fetchTicket({ silent: true })
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
  if (!file || !isTicketOwner.value || isMutating.value) return

  uploadError.value = validateFile(file)
  if (uploadError.value) {
    event.target.value = ''
    return
  }
  uploadingFile.value = true
  try {
    await ticketService.uploadAttachment(ticket.value.id, file)
    showToast(t('attachment_uploaded'))
    await fetchTicket({ silent: true })
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

const openTicketAttachmentPreview = (attachment) => {
  previewTarget.value = {
    attachment,
    contextLabel: t('preview_context_ticket')
  }
}

const openCommentAttachmentPreview = (comment, attachment) => {
  previewTarget.value = {
    attachment,
    contextLabel: `${t('preview_context_comment')} — ${comment.author?.name || 'User'}`
  }
}

const openDraftPreview = (file) => {
  previewTarget.value = {
    file,
    contextLabel: t('preview_context_comment')
  }
}

let detailUnsubs = []

onMounted(async () => {
  await fetchTicket()
  await fetchUsers()

  for (const event of ['comment_added', 'comment_updated', 'comment_deleted', 'attachment_added', 'attachment_updated', 'attachment_deleted']) {
    detailUnsubs.push(realtime.onEvent(event, (payload) => {
      if (payload?.ticket_id === route.params.id) fetchTicket({ silent: true })
    }))
  }
  detailUnsubs.push(realtime.onEvent('ticket_updated', (payload) => {
    if (payload?.id === route.params.id) fetchTicket({ silent: true })
  }))
})

watch(() => route.params.id, async () => {
  cancelEditComment()
  resetCommentDraft()
  pendingAction.value = null
  uploadError.value = ''
  ticket.value = null
  await fetchTicket()
})

onUnmounted(() => {
  fetchSequence++
  detailUnsubs.forEach((fn) => fn())
  if (toastTimer) clearTimeout(toastTimer)
})
</script>