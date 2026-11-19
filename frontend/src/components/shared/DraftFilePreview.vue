<template>
  <section v-if="file" class="rounded-lg border border-dashed border-slate-300 bg-slate-50 p-3 dark:border-slate-600 dark:bg-slate-950" :aria-label="t('preview_draft')">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0 flex-1">
        <p class="text-xs font-semibold text-amber-700 dark:text-amber-300">{{ t('preview_draft') }}</p>
        <p class="mt-1 break-all text-sm font-medium text-slate-900 dark:text-slate-100">{{ file.name }}</p>
        <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('preview_file_size') }}: {{ formatPreviewSize(file.size) }} · {{ t('preview_file_type') }}: {{ extension.toUpperCase() || file.type || '—' }}</p>
      </div>
      <div class="flex gap-2">
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy" :aria-label="`${t('preview_open')}: ${file.name}`" @click="$emit('preview', file)">{{ t('preview_open') }}</button>
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy" :aria-label="`${t('preview_remove')}: ${file.name}`" @click="$emit('remove', file)">{{ t('preview_remove') }}</button>
      </div>
    </div>
    <p class="mt-2 text-xs text-slate-500 dark:text-slate-400">{{ t('preview_draft_notice') }}</p>
    <p v-if="errorKey" role="status" class="mt-2 text-xs text-rose-700 dark:text-rose-300">{{ t(errorKey) }}</p>
    <img v-else-if="objectUrl" :src="objectUrl" :alt="t('preview_image_alt', { name: file.name })" class="mt-3 max-h-40 max-w-full rounded object-contain" @error="imageFailed" />
    <div v-else-if="snippet" class="mt-3">
      <pre class="max-h-32 overflow-auto whitespace-pre-wrap break-words font-mono text-xs text-slate-700 dark:text-slate-300">{{ snippet }}</pre>
      <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('preview_snippet') }}</p>
    </div>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from '../../i18n'
import { decodePreviewText, formatPreviewSize, getFileExtension, getPreviewMime, getPreviewType, previewErrorKey, validatePreviewFile } from '../../services/file-preview.js'

const props = defineProps({ file: { type: File, default: null }, busy: { type: Boolean, default: false } })
defineEmits(['preview', 'remove'])
const { t } = useI18n()
const objectUrl = ref('')
const snippet = ref('')
const errorKey = ref('')
const extension = computed(() => getFileExtension(props.file?.name))
let generation = 0

function release() {
  generation++
  if (objectUrl.value) URL.revokeObjectURL(objectUrl.value)
  objectUrl.value = ''
}

function imageFailed() {
  release()
  errorKey.value = 'preview_error_image'
}

watch(() => props.file, async (file) => {
  release()
  const current = generation
  snippet.value = ''
  errorKey.value = ''
  if (!file) return
  try {
    validatePreviewFile(file)
    const type = getPreviewType(file)
    if (type === 'image') {
      objectUrl.value = URL.createObjectURL(new Blob([file], { type: getPreviewMime(file.name) }))
    } else if (type === 'text') {
      const buffer = await file.slice(0, 8192).arrayBuffer()
      if (generation === current) snippet.value = decodePreviewText(buffer, 600).text
    }
  } catch (error) {
    if (generation === current) errorKey.value = previewErrorKey(error)
  }
}, { immediate: true })
onBeforeUnmount(release)
</script>
