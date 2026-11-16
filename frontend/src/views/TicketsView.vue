<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-slate-900">Tiket</h1>
        <p class="text-sm text-slate-500 mt-1">
          {{ isRequester ? 'Pantau dan kelola tiket dukungan Anda.' : 'Kelola semua tiket dukungan.' }}
        </p>
      </div>
      <button v-if="!auth.isAgent" class="btn-primary" @click="$router.push('/tickets/create')">
        <span class="text-lg leading-none">+</span> Buat Tiket
      </button>
    </div>

    <!-- Stats (Agent/Admin) -->
    <div v-if="!isRequester && stats" class="grid grid-cols-2 lg:grid-cols-4 gap-3">
      <div class="card p-4">
        <p class="text-xs font-medium text-slate-500 uppercase">Open</p>
        <p class="text-2xl font-bold text-slate-900 mt-1">{{ stats.data.open_tickets }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs font-medium text-slate-500 uppercase">In Progress</p>
        <p class="text-2xl font-bold text-amber-600 mt-1">{{ stats.data.in_progress_tickets }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs font-medium text-slate-500 uppercase">Resolved</p>
        <p class="text-2xl font-bold text-emerald-600 mt-1">{{ stats.data.resolved_tickets }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs font-medium text-slate-500 uppercase">Urgent</p>
        <p class="text-2xl font-bold text-rose-600 mt-1">{{ stats.data.urgent_tickets }}</p>
      </div>
    </div>

    <!-- Filter Bar -->
    <div class="card p-4">
      <div class="flex flex-col sm:flex-row gap-3">
        <div class="relative flex-1">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            v-model="filters.search"
            type="search"
            placeholder="Cari berdasarkan judul tiket..."
            class="input pl-9"
            @input="handleSearchInput"
          />
        </div>

        <select v-model="filters.status" class="input sm:w-40" @change="applyFilters">
          <option value="">Semua Status</option>
          <option value="open">Open</option>
          <option value="in_progress">In Progress</option>
          <option value="resolved">Resolved</option>
          <option value="closed">Closed</option>
        </select>

        <select v-model="filters.priority" class="input sm:w-40" @change="applyFilters">
          <option value="">Semua Prioritas</option>
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
          <option value="urgent">Urgent</option>
        </select>

        <select v-if="!isRequester" v-model="filters.assignee_id" class="input sm:w-48" @change="applyFilters">
          <option value="">Semua Assignee</option>
          <option value="unassigned">Belum Di-assign</option>
          <option value="me">Tiket Saya</option>
          <option v-for="u in assignees" :key="u.id" :value="u.id">{{ u.name }}</option>
        </select>
      </div>
    </div>

    <!-- Loading State -->
    <LoadingSkeleton v-if="loading" :rows="5" />

    <!-- Error State -->
    <ErrorAlert v-else-if="error" :message="error" :show-retry="true" @retry="fetchTickets" />

    <!-- Empty State -->
    <EmptyState
      v-else-if="tickets.length === 0"
      title="Tidak ada tiket ditemukan"
      message="Belum ada tiket yang sesuai dengan filter Anda, atau Anda belum membuat tiket apa pun."
    >
      <button v-if="hasActiveFilters" class="btn btn-outline btn-sm" @click="resetFilters">Reset Filter</button>
      <button v-if="!auth.isAgent" class="btn btn-primary btn-sm ml-2" @click="$router.push('/tickets/create')">Buat Tiket</button>
    </EmptyState>

    <!-- Desktop Table -->
    <div v-else class="card overflow-hidden hidden md:block">
      <div class="overflow-x-auto">
        <table class="w-full text-xs">
          <thead class="border-b bg-slate-50">
            <tr>
              <th class="text-left font-medium text-slate-500 px-3 py-2">ID</th>
              <th class="text-left font-medium text-slate-500 px-3 py-2" title="Klik baris untuk melihat deskripsi">Keterangan ⤢</th>
              <th v-if="!isRequester" class="text-left font-medium text-slate-500 px-3 py-2">Pelapor</th>
              <th class="text-left font-medium text-slate-500 px-3 py-2">Status</th>
              <th class="text-left font-medium text-slate-500 px-3 py-2">Prioritas</th>
              <th class="text-left font-medium text-slate-500 px-3 py-2">Ditangani Oleh</th>
              <th class="text-left font-medium text-slate-500 px-3 py-2">Dibuat</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <template v-for="ticket in tickets" :key="ticket.id">
              <tr
                class="hover:bg-slate-50/80 cursor-pointer transition-colors"
                @click="toggleExpand(ticket.id)"
              >
                <td class="px-3 py-2 align-top">
                  <span class="font-mono text-[11px] text-slate-400">{{ ticket.id.slice(0, 8) }}</span>
                </td>
                <td class="px-3 py-2 align-top max-w-[22rem]">
                  <div class="flex items-start gap-1.5">
                    <svg
                      class="w-3.5 h-3.5 text-slate-400 shrink-0 mt-0.5 transition-transform"
                      :class="isExpanded(ticket.id) ? 'rotate-90' : ''"
                      fill="none" viewBox="0 0 24 24" stroke="currentColor"
                    >
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                    </svg>
                    <div class="min-w-0">
                      <p class="font-medium text-slate-900 leading-snug truncate">{{ ticket.title }}</p>
                      <p class="text-[11px] text-slate-500 mt-0.5">
                        {{ ticket.category }}
                        <span v-if="ticket.is_overdue" class="ml-1.5 text-rose-600 font-medium">● SLA</span>
                      </p>
                    </div>
                  </div>
                </td>
                <td v-if="!isRequester" class="px-3 py-2 align-top">
                  <p class="font-medium text-slate-900 leading-tight truncate max-w-[12rem]">{{ ticket.requester?.name || '-' }}</p>
                  <p class="text-[11px] text-slate-500 truncate max-w-[12rem]">{{ ticket.requester?.email || '-' }}</p>
                </td>
                <td class="px-3 py-2 align-top whitespace-nowrap"><StatusBadge :status="ticket.status" /></td>
                <td class="px-3 py-2 align-top whitespace-nowrap"><PriorityBadge :priority="ticket.priority" /></td>
                <td class="px-3 py-2 align-top text-slate-600 whitespace-nowrap">
                  <span v-if="ticket.assignee">{{ ticket.assignee.name }}</span>
                  <span v-else class="text-slate-400 italic">Belum di-tangani</span>
                </td>
                <td class="px-3 py-2 align-top text-slate-500 whitespace-nowrap">
                  <p>{{ formatDate(ticket.created_at) }}</p>
                  <p class="text-[11px] text-slate-400">{{ formatTime(ticket.created_at) }}</p>
                </td>
              </tr>
              <tr v-if="isExpanded(ticket.id)" class="bg-slate-50/70">
                <td :colspan="isRequester ? 6 : 7" class="px-3 sm:px-4 py-3 border-t border-slate-100">
                  <div class="flex items-start justify-between gap-4">
                    <div class="min-w-0">
                      <p class="text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1">Deskripsi</p>
                      <p class="text-xs text-slate-700 whitespace-pre-wrap leading-relaxed break-words">
                        {{ ticket.description || '-' }}
                      </p>
                    </div>
                    <button
                      class="btn btn-outline btn-sm shrink-0"
                      @click.stop="$router.push(`/tickets/${ticket.id}`)"
                    >
                      Buka Detail →
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
        class="card p-3 cursor-pointer hover:bg-slate-50/80 transition-colors"
        @click="toggleExpand(ticket.id)"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0 flex-1">
            <p class="text-[10px] font-mono text-slate-400 leading-none">
              {{ ticket.id.slice(0, 8) }}
            </p>
            <p class="font-medium text-sm text-slate-900 leading-snug mt-1">{{ ticket.title }}</p>
            <p class="text-[11px] text-slate-500 mt-0.5">
              {{ ticket.category }}
              <span v-if="ticket.is_overdue" class="ml-1.5 text-rose-600 font-medium">● SLA</span>
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
          class="mt-2 pt-2 border-t border-slate-100"
          @click.stop
        >
          <p class="text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1">Deskripsi</p>
          <p class="text-xs text-slate-700 whitespace-pre-wrap leading-relaxed break-words">
            {{ ticket.description || '-' }}
          </p>
          <router-link
            :to="`/tickets/${ticket.id}`"
            class="btn btn-outline btn-sm mt-2.5 inline-flex items-center"
          >
            Buka Detail →
          </router-link>
        </div>

        <div class="mt-2 pt-2 border-t border-slate-100 flex flex-wrap gap-x-3 gap-y-0.5 text-[11px] text-slate-500">
          <p v-if="!isRequester">
            <span class="text-slate-400">Pelapor:</span>
            {{ ticket.requester?.name || '-' }}
          </p>
          <p>
            <span class="text-slate-400">Ditangani:</span>
            {{ ticket.assignee?.name || 'Belum di-tangani' }}
          </p>
          <p>
            <span class="text-slate-400">Dibuat:</span>
            {{ formatDate(ticket.created_at) }} · {{ formatTime(ticket.created_at) }}
          </p>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="!loading && !error && meta.total_pages > 0" class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
      <p class="text-sm text-slate-500">
        Menampilkan {{ (meta.current_page - 1) * meta.limit + 1 }}-{{ Math.min(meta.current_page * meta.limit, meta.total_items) }}
        dari {{ meta.total_items }} tiket
      </p>
      <div class="flex items-center gap-2">
        <button
          class="btn btn-outline btn-sm"
          :disabled="meta.current_page <= 1"
          @click="changePage(meta.current_page - 1)"
        >
          Previous
        </button>
        <span class="text-sm text-slate-600 px-2">
          Halaman {{ meta.current_page }} dari {{ meta.total_pages }}
        </span>
        <button
          class="btn btn-outline btn-sm"
          :disabled="meta.current_page >= meta.total_pages"
          @click="changePage(meta.current_page + 1)"
        >
          Next
        </button>
      </div>
    </div>

    <Toast :message="toast.message" :type="toast.type" :visible="toast.visible" />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ticketService } from '../services/ticket.service'
import { userService } from '../services/user.service'
import StatusBadge from '../components/shared/StatusBadge.vue'
import PriorityBadge from '../components/shared/PriorityBadge.vue'
import LoadingSkeleton from '../components/shared/LoadingSkeleton.vue'
import ErrorAlert from '../components/shared/ErrorAlert.vue'
import EmptyState from '../components/shared/EmptyState.vue'
import Toast from '../components/shared/Toast.vue'

const router = useRouter()
const auth = useAuthStore()

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
    error.value = err.message || 'Gagal memuat daftar tiket'
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
  return new Date(value).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' })
}

const formatTime = (value) => {
  if (!value) return '-'
  return new Date(value).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

onMounted(async () => {
  await fetchTickets()
  await Promise.all([fetchStats(), fetchAssignees()])
})
</script>