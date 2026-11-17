<template>
  <div class="card border-dashed p-12 text-center">
    <div class="w-12 h-12 rounded-full bg-slate-100 dark:bg-slate-800 flex items-center justify-center mx-auto mb-4">
      <svg class="w-6 h-6 text-slate-400 dark:text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
      </svg>
    </div>
    <h3 class="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-1">{{ title || t('empty_tickets_title') }}</h3>
    <p class="text-sm text-slate-500 dark:text-slate-400 mb-4">{{ message || t('empty_tickets_desc') }}</p>
    <slot />
  </div>
</template>

<script setup>
import { useI18n } from '../../i18n'

defineProps({
  title: { type: String, default: '' },
  message: { type: String, default: '' }
})

const { t } = useI18n()
</script>
