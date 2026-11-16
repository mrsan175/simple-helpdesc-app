<template>
  <Teleport to="body">
    <transition name="toast">
      <div v-if="visible" class="fixed bottom-4 right-4 z-[100] w-full max-w-sm">
        <div :class="['rounded-lg border p-4 shadow-lg text-sm flex items-start gap-3', isError ? 'bg-rose-50 border-rose-200 text-rose-800' : 'bg-emerald-50 border-emerald-200 text-emerald-800']">
          <span class="text-base">{{ isError ? '✕' : '✓' }}</span>
          <p class="flex-1">{{ message }}</p>
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
  transition: all 0.2s ease;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>