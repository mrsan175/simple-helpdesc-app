<template>
  <div class="space-y-3">
    <div v-for="i in rows" :key="i" class="card p-4 flex items-center gap-4">
      <div class="h-10 w-10 rounded bg-slate-100 animate-pulse"></div>
      <div class="flex-1 space-y-2">
        <div class="h-4 bg-slate-100 rounded animate-pulse" :class="i % 2 ? 'w-3/4' : 'w-1/2'"></div>
        <div class="h-3 bg-slate-100 rounded animate-pulse w-1/3"></div>
      </div>
      <div class="h-6 w-20 bg-slate-100 rounded animate-pulse"></div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  rows: { type: Number, default: 5 }
})
</script>