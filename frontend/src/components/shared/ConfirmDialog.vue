<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div class="absolute inset-0 bg-slate-900/50" @click="$emit('close')"></div>
      <div class="relative bg-white dark:bg-slate-900 rounded-lg shadow-xl w-full max-w-md p-6">
        <h3 class="text-lg font-semibold text-slate-900 dark:text-slate-100">{{ title || t('confirmation') }}</h3>
        <p class="text-sm text-slate-500 dark:text-slate-400 mt-2">{{ message || t('sure_question') }}</p>
        <div class="flex justify-end gap-2 mt-6">
          <button class="btn btn-outline btn-sm" @click="$emit('close')">{{ cancelLabel || t('cancel') }}</button>
          <button :class="['btn', 'btn-sm', destructive ? 'btn-destructive' : 'btn-primary']" @click="$emit('confirm')">
            {{ confirmLabel || t('yes') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { useI18n } from '../../i18n'

defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: '' },
  message: { type: String, default: '' },
  confirmLabel: { type: String, default: '' },
  cancelLabel: { type: String, default: '' },
  destructive: { type: Boolean, default: false }
})
defineEmits(['confirm', 'close'])

const { t } = useI18n()
</script>
