<template>
  <Teleport to="body">
    <transition name="toast">
      <div
        v-if="visible"
        class="fixed top-4 inset-x-4 sm:top-auto sm:bottom-4 sm:left-auto sm:right-4 sm:w-full sm:max-w-sm z-[9999] pointer-events-none"
      >
        <div
          :class="[
            'pointer-events-auto rounded-lg border p-3.5 sm:p-4 shadow-xl text-xs sm:text-sm flex items-start gap-3',
            isError
              ? 'bg-rose-50 dark:bg-rose-950/95 border-rose-200 dark:border-rose-800 text-rose-900 dark:text-rose-200'
              : 'bg-emerald-50 dark:bg-emerald-950/95 border-emerald-200 dark:border-emerald-800 text-emerald-900 dark:text-emerald-200'
          ]"
        >
          <span class="text-sm sm:text-base font-bold shrink-0 leading-none mt-0.5">{{ isError ? '✕' : '✓' }}</span>
          <p class="flex-1 font-medium leading-snug break-words">{{ message }}</p>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  message: { type: String, default: '' },
  type: { type: String, default: 'success' },
  visible: { type: Boolean, default: false }
})

const isError = computed(() => props.type === 'error')
</script>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
@media (min-width: 640px) {
  .toast-enter-from,
  .toast-leave-to {
    transform: translateY(8px);
  }
}
</style>
