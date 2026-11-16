<template>
  <span :class="['badge', badgeClasses]">
    <span class="inline-block w-1.5 h-1.5 rounded-full" :class="dotClasses"></span>
    {{ label }}
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  status: {
    type: String,
    required: true
  }
})

const label = computed(() => {
  switch (props.status) {
    case 'open':
      return 'Open'
    case 'in_progress':
      return 'In Progress'
    case 'resolved':
      return 'Resolved'
    case 'closed':
      return 'Closed'
    default:
      return props.status
  }
})

const badgeClasses = computed(() => {
  switch (props.status) {
    case 'open':
      return 'bg-sky-50 text-sky-700 border-sky-200'
    case 'in_progress':
      return 'bg-amber-50 text-amber-700 border-amber-200'
    case 'resolved':
      return 'bg-emerald-50 text-emerald-700 border-emerald-200'
    case 'closed':
      return 'bg-slate-100 text-slate-700 border-slate-200'
    default:
      return 'bg-slate-100 text-slate-700 border-slate-200'
  }
})

const dotClasses = computed(() => {
  switch (props.status) {
    case 'open':
      return 'bg-sky-500'
    case 'in_progress':
      return 'bg-amber-500 animate-pulse'
    case 'resolved':
      return 'bg-emerald-500'
    case 'closed':
      return 'bg-slate-400'
    default:
      return 'bg-slate-400'
  }
})
</script>