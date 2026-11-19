<template>
  <div class="divide-y divide-slate-100 dark:divide-slate-800">
    <div
      v-for="attachment in attachments"
      :key="attachment.id"
      :data-attachment-id="attachment.id"
      class="py-2.5 flex flex-wrap items-center justify-between gap-3 text-xs"
    >
      <div class="min-w-0 flex-1">
        <p class="font-medium text-slate-900 dark:text-slate-100 break-all">{{ attachment.filename }}</p>
        <p class="text-[11px] text-slate-400 dark:text-slate-500 mt-0.5">
          {{ formatFileSize(attachment.file_size) }} · {{ attachment.uploader?.name || 'User' }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2 shrink-0">
        <button type="button" class="btn btn-outline btn-sm text-xs" @click="$emit('download', attachment.id)">
          {{ t('download') }}
        </button>
        <template v-if="canManage">
          <label class="btn btn-outline btn-sm text-xs cursor-pointer" :class="{ 'opacity-50 pointer-events-none': busy }">
            {{ t('replace_attachment') }}
            <input
              type="file"
              class="hidden"
              :disabled="busy"
              :aria-label="t('replace_attachment') + ': ' + attachment.filename"
              @change="selectReplacement(attachment, $event)"
            />
          </label>
          <button
            type="button"
            class="btn btn-outline btn-sm text-xs text-rose-600 dark:text-rose-400"
            :disabled="busy"
            :aria-label="t('delete_attachment') + ': ' + attachment.filename"
            @click="$emit('delete', attachment)"
          >
            {{ t('delete') }}
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useI18n } from '../../i18n'

const { t } = useI18n()
defineProps({
  attachments: { type: Array, default: () => [] },
  canManage: { type: Boolean, default: false },
  busy: { type: Boolean, default: false }
})
const emit = defineEmits(['download', 'replace', 'delete'])

const selectReplacement = (attachment, event) => {
  const file = event.target.files?.[0]
  if (file) emit('replace', { attachment, file })
  event.target.value = ''
}

const formatFileSize = (bytes) => {
  if (!bytes || bytes <= 0) return '0 B'
  const unit = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), 3)
  return `${parseFloat((bytes / Math.pow(1024, unit)).toFixed(1))} ${['B', 'KB', 'MB', 'GB'][unit]}`
}
</script>
