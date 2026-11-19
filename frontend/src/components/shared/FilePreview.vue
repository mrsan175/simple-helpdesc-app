<template>
  <dialog
    ref="dialog"
    class="file-preview-dialog rounded-xl border border-slate-200 bg-white p-0 text-slate-900 shadow-2xl dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
    :aria-labelledby="`${id}-title ${id}-name`"
    :aria-describedby="`${id}-notice`"
    aria-modal="true"
    @cancel.prevent="close"
    @click="onBackdropClick"
    @keydown="onKeydown"
  >
    <section class="flex max-h-[90dvh] flex-col">
      <header class="flex shrink-0 items-start justify-between gap-4 border-b border-slate-200 p-4 dark:border-slate-700">
        <div class="min-w-0">
          <p :id="`${id}-title`" class="text-xs font-medium text-slate-500 dark:text-slate-400">{{ t('preview_title') }}</p>
          <h2 :id="`${id}-name`" class="break-all text-base font-semibold">{{ filename }}</h2>
          <p v-if="contextLabel" class="mt-1 whitespace-pre-wrap break-words text-sm text-slate-500 dark:text-slate-400">{{ contextLabel }}</p>
        </div>
        <button ref="closeButton" type="button" class="btn btn-outline btn-sm shrink-0" autofocus @click="close">{{ t('preview_close') }}</button>
      </header>

      <div class="min-h-0 overflow-auto p-4" :aria-busy="loading">
        <p :id="`${id}-notice`" class="mb-3 text-xs text-slate-500 dark:text-slate-400">{{ t(file ? 'preview_draft_notice' : 'preview_private_notice') }}</p>
        <p v-if="loading" role="status" class="py-12 text-center text-sm">{{ t('preview_loading') }}</p>
        <p v-else-if="errorKey" role="alert" class="rounded-lg bg-rose-50 p-4 text-sm text-rose-700 dark:bg-rose-950 dark:text-rose-200">{{ t(errorKey) }}</p>
        <template v-else>
          <img v-if="type === 'image' && objectUrl" :src="objectUrl" :alt="t('preview_image_alt', { name: filename })" class="mx-auto max-h-[65dvh] max-w-full object-contain" @error="imageFailed" />
          <template v-else-if="type === 'pdf' && objectUrl">
            <p class="mb-3 text-xs text-slate-500 dark:text-slate-400">{{ t('preview_pdf_notice') }}</p>
            <a :href="objectUrl" target="_blank" rel="noopener noreferrer" class="btn btn-outline btn-sm mb-3">{{ t('preview_pdf_open') }}</a>
            <iframe :src="objectUrl" :title="t('preview_pdf_frame', { name: filename })" sandbox="allow-scripts" referrerpolicy="no-referrer" class="h-[60dvh] w-full rounded border border-slate-200 bg-white dark:border-slate-700"></iframe>
          </template>
          <template v-else-if="type === 'text' || type === 'office'">
            <p v-if="type === 'office'" class="mb-3 text-xs text-slate-500 dark:text-slate-400">{{ t('preview_office_notice') }}</p>
            <pre v-if="text" class="whitespace-pre-wrap break-words rounded-lg bg-slate-50 p-4 font-mono text-sm dark:bg-slate-950">{{ text }}</pre>
            <p v-else class="py-8 text-center text-sm text-slate-500 dark:text-slate-400">{{ t('preview_no_text') }}</p>
            <p v-if="truncated" role="status" class="mt-3 text-xs text-amber-700 dark:text-amber-300">{{ t('preview_text_truncated') }}</p>
          </template>
          <template v-else-if="type === 'zip'">
            <p class="mb-2 text-xs text-slate-500 dark:text-slate-400">{{ t('preview_zip_notice') }}</p>
            <p class="mb-3 text-xs text-slate-500 dark:text-slate-400">{{ t('preview_zip_encoding') }}</p>
            <p v-if="!entries.length" class="py-8 text-center text-sm">{{ t('preview_archive_empty') }}</p>
            <table v-else class="w-full table-fixed text-left text-sm">
              <thead><tr class="border-b border-slate-200 dark:border-slate-700"><th scope="col" class="p-2">{{ t('preview_archive_name') }}</th><th scope="col" class="w-28 p-2">{{ t('preview_archive_size') }}</th></tr></thead>
              <tbody>
                <tr v-for="(entry, index) in entries" :key="index" class="border-b border-slate-100 dark:border-slate-800">
                  <td class="break-words p-2"><span class="whitespace-pre-wrap">{{ entry.name }}</span><span v-if="entry.directory" class="ml-2 text-xs text-slate-500">({{ t('preview_archive_folder') }})</span><span v-if="entry.encrypted" class="ml-2 text-xs text-amber-700 dark:text-amber-300">({{ t('preview_archive_encrypted') }})</span></td>
                  <td class="p-2 align-top">{{ formatPreviewSize(entry.size) }}</td>
                </tr>
              </tbody>
            </table>
          </template>
        </template>
      </div>
    </section>
  </dialog>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
import { useI18n } from '../../i18n'
import { ticketService } from '../../services/ticket.service'
import { parseInPreviewWorker } from '../../services/file-preview-client.js'
import { decodePreviewText, formatPreviewSize, getPreviewMime, getPreviewType, previewErrorKey, PreviewError, validatePdfSignature, validatePreviewFile } from '../../services/file-preview.js'

const props = defineProps({
  file: { type: File, default: null },
  attachment: { type: Object, default: null },
  contextLabel: { type: String, default: '' }
})
const emit = defineEmits(['close'])
const { t } = useI18n()
const id = useId()
const dialog = ref(null)
const closeButton = ref(null)
const filename = computed(() => props.file?.name || props.attachment?.filename || '')
const type = computed(() => getPreviewType(filename.value))
const loading = ref(true)
const errorKey = ref('')
const objectUrl = ref('')
const text = ref('')
const truncated = ref(false)
const entries = ref([])
let controller
let previousFocus

function release() {
  controller?.abort()
  if (objectUrl.value) URL.revokeObjectURL(objectUrl.value)
  objectUrl.value = ''
}

async function load() {
  release()
  const current = new AbortController()
  controller = current
  loading.value = true
  errorKey.value = ''
  text.value = ''
  truncated.value = false
  entries.value = []
  const name = filename.value
  const previewType = getPreviewType(name)
  try {
    let blob = props.file
    if (!blob) {
      if (!props.attachment?.id) throw new PreviewError('invalid')
      validatePreviewFile({ size: Number(props.attachment.file_size || 0) }, name)
      try {
        blob = await ticketService.previewAttachment(props.attachment.id, current.signal)
      } catch (error) {
        if (current.signal.aborted) return
        throw new PreviewError('remote')
      }
    }
    if (current.signal.aborted) return
    validatePreviewFile(blob, name)
    if (previewType === 'office' || previewType === 'zip') {
      const result = await parseInPreviewWorker(blob, name, { signal: current.signal })
      if (current.signal.aborted) return
      text.value = result.text || ''
      truncated.value = Boolean(result.truncated)
      entries.value = result.entries || []
    } else if (previewType === 'text') {
      const buffer = await blob.arrayBuffer()
      if (current.signal.aborted) return
      const result = decodePreviewText(buffer)
      text.value = result.text
      truncated.value = result.truncated
    } else {
      if (previewType === 'pdf') validatePdfSignature(await blob.slice(0, 1024).arrayBuffer())
      if (current.signal.aborted) return
      // Never trust a server-supplied MIME type or display arbitrary HTML/SVG.
      objectUrl.value = URL.createObjectURL(new Blob([blob], { type: getPreviewMime(name) }))
    }
  } catch (error) {
    if (!current.signal.aborted) errorKey.value = previewErrorKey(error)
  } finally {
    if (!current.signal.aborted) loading.value = false
  }
}

function imageFailed() {
  if (objectUrl.value) URL.revokeObjectURL(objectUrl.value)
  objectUrl.value = ''
  errorKey.value = 'preview_error_image'
}

function close() {
  release()
  dialog.value?.close()
  emit('close')
}

function onBackdropClick(event) {
  if (event.target !== dialog.value) return
  const box = dialog.value.getBoundingClientRect()
  if (event.clientX < box.left || event.clientX > box.right || event.clientY < box.top || event.clientY > box.bottom) close()
}

function onKeydown(event) {
  if (event.key === 'Escape') {
    event.preventDefault()
    close()
  } else if (event.key === 'Tab') {
    const focusable = [...dialog.value.querySelectorAll('button:not(:disabled), a[href], iframe, [tabindex="0"]')]
    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    if ((event.shiftKey && document.activeElement === first) || (!event.shiftKey && document.activeElement === last)) {
      event.preventDefault()
      const target = event.shiftKey ? last : first
      target?.focus()
    }
  }
}

watch(() => [props.file, props.attachment, props.attachment?.id, props.attachment?.filename, props.attachment?.file_size], load, { immediate: true })
onMounted(() => {
  previousFocus = document.activeElement
  // Native modal dialog makes background content inert, including keyboard focus.
  dialog.value.showModal()
  closeButton.value?.focus()
})
onBeforeUnmount(() => {
  release()
  dialog.value?.close()
  if (previousFocus?.isConnected) previousFocus.focus()
})
</script>

<style scoped>
.file-preview-dialog {
  width: min(64rem, calc(100vw - 2rem));
  max-width: none;
  max-height: 90dvh;
  overflow: hidden;
}
.file-preview-dialog::backdrop {
  background: rgb(15 23 42 / 65%);
}
</style>
