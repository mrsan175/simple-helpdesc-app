<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div class="absolute inset-0 bg-slate-900/50" @click="$emit('close')"></div>
      <div class="relative bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <h3 class="text-lg font-semibold text-slate-900">{{ title }}</h3>
        <p class="text-sm text-slate-500 mt-2">{{ message }}</p>
        <div class="flex justify-end gap-2 mt-6">
          <button class="btn btn-outline btn-sm" @click="$emit('close')">{{ cancelLabel }}</button>
          <button :class="['btn', 'btn-sm', destructive ? 'btn-destructive' : 'btn-primary']" @click="$emit('confirm')">
            {{ confirmLabel }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: 'Konfirmasi' },
  message: { type: String, default: 'Apakah Anda yakin?' },
  confirmLabel: { type: String, default: 'Ya' },
  cancelLabel: { type: String, default: 'Batal' },
  destructive: { type: Boolean, default: false }
})
defineEmits(['confirm', 'close'])
</script>