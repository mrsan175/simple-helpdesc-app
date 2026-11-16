<template>
  <div class="rounded-lg border border-rose-200 bg-rose-50 p-4">
    <div class="flex items-start gap-3">
      <svg class="w-5 h-5 text-rose-500 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <div class="flex-1">
        <p class="text-sm font-medium text-rose-800">{{ title }}</p>
        <p class="text-sm text-rose-700 mt-1">{{ message }}</p>
        <button v-if="showRetry" @click="$emit('retry')" class="mt-3 btn btn-sm border border-rose-300 text-rose-700 bg-white hover:bg-rose-50">
          Coba Lagi
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: { type: String, default: 'Terjadi kesalahan' },
  message: { type: String, default: '' },
  showRetry: { type: Boolean, default: false }
})
defineEmits(['retry'])
</script>