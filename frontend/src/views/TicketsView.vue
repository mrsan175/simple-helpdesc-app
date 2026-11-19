<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">{{ t('tickets_title') }}</h1>
        <p class="text-sm text-slate-500 dark:text-slate-400 mt-1">
          {{ isRequester ? t('tickets_subtitle_requester') : t('tickets_subtitle_staff') }}
        </p>
      </div>
      <button v-if="!auth.isAgent" class="btn-primary" @click="$router.push('/tickets/create')">
        <span class="text-lg leading-none">+</span> {{ t('create_ticket') }}
      </button>
    </div>

    <!-- Stats (Agent/Admin) -->
    <div v-if="!isRequester && stats" class="grid grid-cols-2 lg:grid-cols-4 gap-3">
      <div class="card p-4">
        <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">{{ t('stat_open') }}</p>
        <p class="text-2xl font-bold text-slate-900 dark:text-slate-100 mt-1">{{ stats.data.open_tickets }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">{{ t('stat_in_progress') }}</p>
        <p class="text-2xl font-bold text-amber-600 mt-1">{{ stats.data.in_progress_tickets }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">{{ t('stat_resolved') }}</p>
        <p class="text-2xl font-bold text-emerald-600 mt-1">{{ stats.data.resolved_tickets }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">{{ t('stat_urgent') }}</p>
        <p class="text-2xl font-bold text-rose-600 mt-1">{{ stats.data.urgent_tickets }}</p>
      </div>
    </div>

    <!-- Filter Bar -->
    <div class="card p-4">
      <div
        class="grid grid-cols-1 sm:grid-cols-2 gap-3"
        :class="isRequester ? 'lg:grid-cols-3' : 'lg:grid-cols-4'"
      >
        <div class="sm:col-span-2 lg:col-span-1 space-y-1.5">
          <label for="filter-search" class="block text-xs font-medium text-slate-700 dark:text-slate-300">{{ t('filter_search') }}</label>
          <div class="relative">
            <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400 dark:text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              id="filter-search"
              v-model="filters.search"
              type="search"
              :placeholder="t('filter_search_ph')"
              class="input pl-9 text-xs sm:text-sm h-9"
              @input="handleSearchInput"
            />
          </div>
        </div>

        <div class="space-y-1.5">
          <label for="filter-status" class="block text-xs font-medium text-slate-700 dark:text-slate-300">{{ t('filter_status') }}</label>
          <select id="filter-status" v-model="filters.status" class="input text-xs sm:text-sm h-9" @change="applyFilters">
            <option value="">{{ t('filter_all_status') }}</option>
            <option value="open">Open</option>
            <option value="in_progress">In Progress</option>
            <option value="resolved">Resolved</option>
            <option value="closed">Closed</option>
          </select>
        </div>

        <div class="space-y-1.5">
          <label for="filter-priority" class="block text-xs font-medium text-slate-700 dark:text-slate-300">{{ t('filter_priority') }}</label>
          <select id="filter-priority" v-model="filters.priority" class="input text-xs sm:text-sm h-9" @change="applyFilters">
            <option value="">{{ t('filter_all_priority') }}</option>
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
            <option value="urgent">Urgent</option>
          </select>
        </div>

        <div v-if="!isRequester" class="space-y-1.5">
          <label for="filter-assignee" class="block text-xs font-medium text-slate-700 dark:text-slate-300">{{ t('filter_assignee') }}</label>
          <select id="filter-assignee" v-model="filters.assignee_id" class="input text-xs sm:text-sm h-9" @change="applyFilters">
            <option value="">{{ t('filter_all_assignee') }}</option>
            <option value="unassigned">{{ t('filter_unassigned') }}</option>
            <option value="me">{{ t('filter_my_tickets') }}</option>
            <option v-for="u in assignees" :key="u.id" :value="u.id">{{ u.name }}</option>
          </select>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <LoadingSkeleton v-if="loading" :rows="5" />

    <!-- Error State -->
    <ErrorAlert v-else-if="error" :message="error" :show-retry="true" @retry="fetchTickets" />

    <!-- Empty State -->
    <EmptyState
      v-else-if="tickets.length === 0"
      :title="t('empty_tickets_title')"
      :message="t('empty_tickets_desc')"
    >
      <button v-if="hasActiveFilters" class="btn btn-outline btn-sm" @click="resetFilters">{{ t('reset_filter') }}</button>
      <button v-if="!auth.isAgent" class="btn btn-primary btn-sm ml-2" @click="$router.push('/tickets/create')">{{ t('create_ticket') }}</button>
    </EmptyState>

    <!-- Desktop Table -->
    <div v-else class="card overflow-hidden hidden md:block">
      <div class="overflow-x-auto">
        <table class="w-full text-xs">
          <thead class="border-b bg-slate-50 dark:bg-slate-800/60">
            <tr>
              <th class="text-left font-medium text-slate-500 dark:text-slate-400 px-3 py-2">{{ t('th_id') }}</th>
              <th class="text-left font-medium text-slate-500 dark:text-slate-400 px-3 py-2">{{ t('th_keterangan') }}</th>
              <th v-if="!isRequester" class="text-left font-medium text-slate-500 dark:text-slate-400 px-3 py-2">{{ t('th_requester') }}</th>
              <th class="text-left font-medium text-slate-500 dark:text-slate-400 px-3 py-2">{{ t('th_status') }}</th>
              <th class="text-left font-medium text-slate-500 dark:text-slate-400 px-3 py-2">{{ t('th_priority') }}</th>
              <th class="text-left font-medium text-slate-500 dark:text-slate-400 px-3 py-2">{{ t('th_assignee') }}</th>
              <th class="text-left font-medium text-slate-500 dark:text-slate-400 px-3 py-2">{{ t('th_created') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 dark:divide-slate-800">
            <template v-for="ticket in tickets" :key="ticket.id">
              <tr
                class="hover:bg-slate-50/80 dark:hover:bg-slate-800/60 cursor-pointer transition-colors"
                @click="toggleExpand(ticket.id)"
              >
                <td class="px-3 py-2 align-top">
                  <span class="font-mono text-[11px] text-slate-400 dark:text-slate-500">{{ ticket.id.slice(0, 8) }}</span>
                </td>
                <td class="px-3 py-2 align-top max-w-[22rem]">
                  <div class="flex items-start gap-1.5">
                    <svg
                      class="w-3.5 h-3.5 text-slate-400 dark:text-slate-500 shrink-0 mt-0.5 transition-transform"
                      :class="isExpanded(ticket.id) ? 'rotate-90' : ''"
                      fill="none" viewBox="0 0 24 24" stroke="currentColor"
                    >
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                    </svg>
                    <div class="min-w-0">
                      <p class="font-medium text-slate-900 dark:text-slate-100 leading-snug truncate">{{ ticket.title }}</p>
                      <p class="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">
                        {{ ticket.category }}
                        <span v-if="ticket.is_overdue" class="ml-1.5 text-rose-600 font-medium">{{ t('sla_badge') }}</span>
                      </p>
                    </div>
                  </div>
                </td>
                <td v-if="!isRequester" class="px-3 py-2 align-top">
                  <p class="font-medium text-slate-900 dark:text-slate-100 leading-tight truncate max-w-[12rem]">{{ ticket.requester?.name || '-' }}</p>
                  <p class="text-[11px] text-slate-500 dark:text-slate-400 truncate max-w-[12rem]">{{ ticket.requester?.email || '-' }}</p>
                </td>
                <td class="px-3 py-2 align-top whitespace-nowrap"><StatusBadge :status="ticket.status" /></td>
                <td class="px-3 py-2 align-top whitespace-nowrap"><PriorityBadge :priority="ticket.priority" /></td>
                <td class="px-3 py-2 align-top text-slate-600 dark:text-slate-300 whitespace-nowrap">
                  <span v-if="ticket.assignee">{{ ticket.assignee.name }}</span>
                  <span v-else class="text-slate-400 dark:text-slate-500 italic">{{ t('unassigned_label') }}</span>
                </td>
                <td class="px-3 py-2 align-top text-slate-500 dark:text-slate-400 whitespace-nowrap">
                  <p>{{ formatDate(ticket.created_at) }}</p>
                  <p class="text-[11px] text-slate-400 dark:text-slate-500">{{ formatTime(ticket.created_at) }}</p>
                </td>
              </tr>
              <tr v-if="isExpanded(ticket.id)" class="bg-slate-50/70 dark:bg-slate-800/40">
                <td :colspan="isRequester ? 6 : 7" class="px-3 sm:px-4 py-3 border-t border-slate-100 dark:border-slate-800">
                  <div class="flex items-start justify-between gap-4">
                    <div class="min-w-0">
                      <p class="text-[10px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-wider mb-1">Deskripsi</p>
                      <p class="text-xs text-slate-700 dark:text-slate-300 whitespace-pre-wrap leading-relaxed break-words">
                        {{ ticket.description || '-' }}
                      </p>
                    </div>
                    <button
                      class="btn btn-outline btn-sm shrink-0"
                      @click.stop="$router.push(`/tickets/${ticket.id}`)"
                    >
                      Detail
                    </button>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Mobile Card List -->
    <div v-if="!loading && !error && tickets.length > 0" class="md:hidden space-y-2">
      <div
        v-for="ticket in tickets"
        :key="ticket.id"
        class="card p-3 cursor-pointer hover:bg-slate-50/80 dark:hover:bg-slate-800/60 transition-colors"
        @click="toggleExpand(ticket.id)"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0 flex-1">
            <p class="text-[10px] font-mono text-slate-400 dark:text-slate-500 leading-none">
              {{ ticket.id.slice(0, 8) }}
            </p>
            <p class="font-medium text-sm text-slate-900 dark:text-slate-100 leading-snug mt-1">{{ ticket.title }}</p>
            <p class="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">
              {{ ticket.category }}
              <span v-if="ticket.is_overdue" class="ml-1.5 text-rose-600 font-medium">{{ t('sla_badge') }}</span>
            </p>
          </div>
          <div class="flex flex-col items-end gap-1 shrink-0">
            <StatusBadge :status="ticket.status" />
            <PriorityBadge :priority="ticket.priority" />
          </div>
        </div>

        <!-- Expandable description -->
        <div
          v-if="isExpanded(ticket.id)"
          class="mt-2 pt-2 border-t border-slate-100 dark:border-slate-800 flex flex-col gap-2"
          @click.stop
        >
           <router-link
            :to="`/tickets/${ticket.id}`"
            class="btn btn-outline btn-sm mt-2.5 inline-flex items-center"
          >
            {{ t('open_detail') }}
          </router-link>
          <p class="text-[10px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-wider mb-1">Deskripsi</p>
          <p class="text-xs text-slate-700 dark:text-slate-300 whitespace-pre-wrap leading-relaxed break-words">
            {{ ticket.description || '-' }}
          </p>
        </div>


        <div class="mt-2 pt-2 border-t border-slate-100 dark:border-slate-800 flex flex-wrap gap-x-3 gap-y-0.5 text-[11px] text-slate-500 dark:text-slate-400">
          <p v-if="!isRequester">
            <span class="text-slate-400 dark:text-slate-500">{{ t('th_requester') }}:</span>
            {{ ticket.requester?.name || '-' }}
          </p>
          <p>
            <span class="text-slate-400 dark:text-slate-500">{{ t('th_assignee') }}:</span>
            {{ ticket.assignee?.name || t('unassigned_label') }}
          </p>
          <p>
            <span class="text-slate-400 dark:text-slate-500">{{ t('th_created') }}:</span>
            {{ formatDate(ticket.created_at) }} · {{ formatTime(ticket.created_at) }}
          </p>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="!loading && !error && meta.total_pages > 0" class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
      <p class="text-sm text-slate-500 dark:text-slate-400">
        {{
          t('showing_pagination', {
            from: (meta.current_page - 1) * meta.limit + 1,
            to: Math.min(meta.current_page * meta.limit, meta.total_items),
            total: meta.total_items
          })
        }}
      </p>
      <div class="flex items-center gap-2">
        <button
          class="btn btn-outline btn-sm"
          :disabled="meta.current_page <= 1"
          @click="changePage(meta.current_page - 1)"
        >
          {{ t('prev') }}
        </button>
        <span class="text-sm text-slate-600 dark:text-slate-300 px-2">
          {{ t('page_info', { current: meta.current_page, total: meta.total_pages }) }}
        </span>
        <button
          class="btn btn-outline btn-sm"
          :disabled="meta.current_page >= meta.total_pages"
          @click="changePage(meta.current_page + 1)"
        >
          {{ t('next') }}
        </button>
      </div>
    </div>

    <Toast :message="toast.message" :type="toast.type" :visible="toast.visible" />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ticketService } from '../services/ticket.service'
import { userService } from '../services/user.service'
import { useRealtime } from '../services/realtime.service'
import StatusBadge from '../components/shared/StatusBadge.vue'
import PriorityBadge from '../components/shared/PriorityBadge.vue'
import LoadingSkeleton from '../components/shared/LoadingSkeleton.vue'
import ErrorAlert from '../components/shared/ErrorAlert.vue'
import EmptyState from '../components/shared/EmptyState.vue'
import Toast from '../components/shared/Toast.vue'
import { useI18n } from '../i18n'

const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()

const isRequester = computed(() => auth.isRequester)

const tickets = ref([])
const meta = ref({ total_items: 0, total_pages: 0, current_page: 1, limit: 10 })
const loading = ref(false)
const error = ref('')
const stats = ref(null)
const assignees = ref([])
const toast = reactive({ message: '', type: 'success', visible: false })
const expandedTickets = ref(new Set())

const toggleExpand = (id) => {
  const next = new Set(expandedTickets.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  expandedTickets.value = next
}

const isExpanded = (id) => expandedTickets.value.has(id)

const filters = reactive({
  search: '',
  status: '',
  priority: '',
  assignee_id: ''
})

const hasActiveFilters = computed(() =>
  Boolean(filters.search || filters.status || filters.priority || filters.assignee_id)
)

let searchTimer = null
let toastTimer = null

const fetchTickets = async () => {
  loading.value = true
  error.value = ''
  try {
    const params = {
      page: meta.value.current_page,
      limit: meta.value.limit,
      search: filters.search || undefined,
      status: filters.status || undefined,
      priority: filters.priority || undefined
    }
    if (!isRequester.value && filters.assignee_id) {
      params.assignee_id = filters.assignee_id
    }
    const res = await ticketService.list(params)
    tickets.value = res.data || []
    meta.value = res.meta || { total_items: 0, total_pages: 0, current_page: 1, limit: 10 }
  } catch (err) {
    error.value = err.message || t('load_tickets_failed')
  } finally {
    loading.value = false
  }
}

const fetchStats = async () => {
  if (isRequester.value) return
  try {
    stats.value = await ticketService.stats()
  } catch {
    stats.value = null
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

const handleSearchInput = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    meta.value.current_page = 1
    fetchTickets()
  }, 300)
}

const applyFilters = () => {
  meta.value.current_page = 1
  fetchTickets()
}

const resetFilters = () => {
  filters.search = ''
  filters.status = ''
  filters.priority = ''
  filters.assignee_id = ''
  meta.value.current_page = 1
  fetchTickets()
}

const changePage = (page) => {
  meta.value.current_page = page
  fetchTickets()
}

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

onMounted(async () => {
  await fetchTickets()
  await Promise.all([fetchStats(), fetchAssignees()])

  // Real-time subscribe for current user's ticket changes
  const myTickets = tickets.value?.filter((t) => {
    // Only requester sees their own tickets in the list
    return true
  }) || []

  const listUnsubs = []
  if (myTickets?.length > 0) {
    myTickets.forEach((t) => {
      listUnsubs.push(
        realtime.onEvent('ticket_updated', (p) => {
          if (p?.id === t.id) fetchTickets()
        })
      )
      listUnsubs.push(
        realtime.onEvent('comment_added', (p) => {
          if (p?.ticket_id === t.id) fetchTickets()
        })
      )
      listUnsubs.push(
        realtime.onEvent('attachment_added', (p) => {
          if (p?.ticket_id === t.id) fetchTickets()
        })
      )
    })
  }
})
</script>