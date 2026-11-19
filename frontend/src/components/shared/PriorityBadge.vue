<template>
  <span :class="['badge', badgeClasses]">
    {{ label }}
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from '../../i18n'

const props = defineProps({
  priority: {
    type: String,
    default: ''
  }
})

const { t } = useI18n()

const label = computed(() => {
  switch (props.priority) {
    case 'low':
      return 'Low'
    case 'medium':
      return 'Medium'
    case 'high':
      return 'High'
    case 'urgent':
      return 'Urgent'
    default:
      return t('priority_unset')
  }
})

const badgeClasses = computed(() => {
  switch (props.priority) {
    case 'low':
      return 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-800'
    case 'medium':
      return 'bg-blue-50 text-blue-700 border-blue-200'
    case 'high':
      return 'bg-orange-50 text-orange-700 border-orange-200'
    case 'urgent':
      return 'bg-rose-50 text-rose-700 border-rose-200 font-semibold'
    default:
      return 'bg-slate-100 dark:bg-slate-800 text-slate-400 dark:text-slate-500 border-slate-200 dark:border-slate-800 italic font-normal'
  }
})
</script>