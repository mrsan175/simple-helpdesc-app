<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">{{ t('stats_title') }}</h1>
        <p class="text-sm text-slate-500 dark:text-slate-400 mt-1">{{ t('stats_subtitle') }}</p>
      </div>
      <button class="btn btn-outline btn-sm self-start sm:self-auto" @click="fetchStats" :disabled="loading">
        <svg class="w-4 h-4 mr-1.5" :class="{ 'animate-spin': loading }" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        {{ t('refresh') }}
      </button>
    </div>

    <LoadingSkeleton v-if="loading" :rows="4" />

    <ErrorAlert v-else-if="error" :message="error" :show-retry="true" @retry="fetchStats" />

    <div v-else-if="stats" class="space-y-6">
      <!-- KPI Top Summary Cards -->
      <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
        <div class="card p-4">
          <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">{{ t('stat_total') }}</p>
          <p class="text-2xl font-bold text-slate-900 dark:text-slate-100 mt-1">{{ stats.total_tickets }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">{{ t('stat_open') }}</p>
          <p class="text-2xl font-bold text-slate-900 dark:text-slate-100 mt-1">{{ stats.open_tickets }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">{{ t('stat_in_progress') }}</p>
          <p class="text-2xl font-bold text-amber-600 mt-1">{{ stats.in_progress_tickets }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">{{ t('stat_resolved') }}</p>
          <p class="text-2xl font-bold text-emerald-600 mt-1">{{ stats.resolved_tickets }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">{{ t('stat_closed') }}</p>
          <p class="text-2xl font-bold text-slate-900 dark:text-slate-100 mt-1">{{ stats.closed_tickets }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">{{ t('stat_sla_overdue') }}</p>
          <p class="text-2xl font-bold text-rose-600 mt-1">{{ stats.overdue_tickets }}</p>
        </div>
      </div>

      <!-- Grid: Categories & Priorities -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <!-- Breakdown by Category -->
        <div class="card p-4 sm:p-5">
          <h2 class="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-4 pb-2 border-b border-slate-100 dark:border-slate-800">
            {{ t('category_breakdown') }}
          </h2>
          <div v-if="!stats.categories?.length" class="text-xs text-slate-400 py-4 text-center">
            {{ t('empty_stats') }}
          </div>
          <div v-else class="space-y-3.5">
            <div v-for="c in stats.categories" :key="c.category">
              <div class="flex justify-between text-xs font-medium mb-1">
                <span class="text-slate-700 dark:text-slate-300">{{ c.category }}</span>
                <span class="text-slate-500 dark:text-slate-400">{{ c.count }} ({{ calcPercent(c.count, stats.total_tickets) }}%)</span>
              </div>
              <div class="w-full bg-slate-100 dark:bg-slate-800 h-2 rounded-full overflow-hidden">
                <div
                  class="bg-sky-500 h-2 rounded-full transition-all duration-300"
                  :style="{ width: `${calcPercent(c.count, stats.total_tickets)}%` }"
                ></div>
              </div>
            </div>
          </div>
        </div>

        <!-- Breakdown by Priority (active tickets) -->
        <div class="card p-4 sm:p-5">
          <h2 class="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-4 pb-2 border-b border-slate-100 dark:border-slate-800">
            {{ t('priority_breakdown') }}
          </h2>
          <div v-if="!stats.priorities?.length" class="text-xs text-slate-400 py-4 text-center">
            {{ t('empty_stats') }}
          </div>
          <div v-else class="space-y-3.5">
            <div v-for="p in stats.priorities" :key="p.priority">
              <div class="flex justify-between text-xs font-medium mb-1">
                <span class="text-slate-700 dark:text-slate-300 capitalize">{{ p.priority }}</span>
                <span class="text-slate-500 dark:text-slate-400">{{ p.count }}</span>
              </div>
              <div class="w-full bg-slate-100 dark:bg-slate-800 h-2 rounded-full overflow-hidden">
                <div
                  :class="['h-2 rounded-full transition-all duration-300', getPriorityColorClass(p.priority)]"
                  :style="{ width: `${calcPercent(p.count, totalActivePriorities)}%` }"
                ></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Agent Workload Table -->
      <div class="card overflow-hidden">
        <div class="p-4 border-b border-slate-100 dark:border-slate-800">
          <h2 class="text-sm font-semibold text-slate-900 dark:text-slate-100">{{ t('agent_workload') }}</h2>
          <p class="text-xs text-slate-500 dark:text-slate-400 mt-0.5">{{ t('agent_workload_sub') }}</p>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-xs">
            <thead class="border-b bg-slate-50 dark:bg-slate-800/60 font-medium text-slate-600 dark:text-slate-400">
              <tr>
                <th class="text-left px-4 py-2.5">{{ t('th_name') }}</th>
                <th class="text-left px-4 py-2.5">{{ t('th_assigned_tickets') }}</th>
                <th class="text-left px-4 py-2.5">{{ t('workload_status') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100 dark:divide-slate-800">
              <tr v-if="!stats.agents?.length">
                <td colspan="3" class="text-center py-6 text-xs text-slate-400">{{ t('no_agents') }}</td>
              </tr>
              <tr v-for="a in stats.agents" :key="a.agent_id" class="hover:bg-slate-50/60 dark:hover:bg-slate-800/40">
                <td class="px-4 py-2.5 font-medium text-slate-900 dark:text-slate-100">
                  <div class="flex items-center gap-2">
                    <div class="w-6 h-6 rounded-full bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 flex items-center justify-center text-[10px] font-bold shrink-0">
                      {{ a.agent_name.charAt(0).toUpperCase() }}
                    </div>
                    <span>{{ a.agent_name }}</span>
                  </div>
                </td>
                <td class="px-4 py-2.5 text-slate-700 dark:text-slate-300 font-semibold">{{ a.count }}</td>
                <td class="px-4 py-2.5">
                  <span
                    :class="[
                      'text-[11px] font-medium px-2 py-0.5 rounded-full inline-block',
                      a.count >= 5
                        ? 'bg-rose-100 dark:bg-rose-950/60 text-rose-700 dark:text-rose-300'
                        : a.count >= 2
                        ? 'bg-amber-100 dark:bg-amber-950/60 text-amber-700 dark:text-amber-300'
                        : 'bg-emerald-100 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300'
                    ]"
                  >
                    {{ a.count >= 5 ? t('load_high') : a.count >= 2 ? t('load_normal') : t('load_light') }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ticketService } from '../services/ticket.service'
import { useI18n } from '../i18n'
import LoadingSkeleton from '../components/shared/LoadingSkeleton.vue'
import ErrorAlert from '../components/shared/ErrorAlert.vue'

const { t } = useI18n()
const loading = ref(false)
const error = ref('')
const stats = ref(null)

const totalActivePriorities = computed(() => {
  if (!stats.value?.priorities) return 1
  return stats.value.priorities.reduce((acc, p) => acc + p.count, 0) || 1
})

const calcPercent = (count, total) => {
  if (!total || total <= 0) return 0
  return Math.round((count / total) * 100)
}

const getPriorityColorClass = (priority) => {
  switch (priority) {
    case 'urgent':
      return 'bg-rose-500'
    case 'high':
      return 'bg-orange-500'
    case 'medium':
      return 'bg-blue-500'
    default:
      return 'bg-slate-400'
  }
}

const fetchStats = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await ticketService.adminStats()
    stats.value = res.data
  } catch (err) {
    error.value = err.message || t('error_title')
  } finally {
    loading.value = false
  }
}

onMounted(fetchStats)
</script>
