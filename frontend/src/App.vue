<template>
  <div class="min-h-screen bg-slate-50 dark:bg-slate-800">
    <!-- Top page loading bar -->
    <div class="fixed top-0 inset-x-0 z-[100] h-0.5 pointer-events-none" aria-hidden="true">
      <div
        class="h-full bg-sky-500 transition-[width,opacity] duration-200 ease-out"
        :class="barVisible ? 'opacity-100' : 'opacity-0'"
        :style="{ width: `${barProgress}%` }"
      ></div>
    </div>

    <router-view v-slot="{ Component }">
      <transition name="fade" mode="out-in">
        <component :is="Component" />
      </transition>
    </router-view>
  </div>
</template>

<script setup>
import { onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const barProgress = ref(0)
const barVisible = ref(false)
let growTimer = null
let hideTimer = null

const startBar = () => {
  clearTimeout(hideTimer)
  clearInterval(growTimer)
  barVisible.value = true
  barProgress.value = 8
  growTimer = setInterval(() => {
    barProgress.value = Math.min(90, barProgress.value + (90 - barProgress.value) * 0.15)
  }, 200)
}

const finishBar = () => {
  clearInterval(growTimer)
  barProgress.value = 100
  hideTimer = setTimeout(() => {
    barVisible.value = false
    hideTimer = setTimeout(() => { barProgress.value = 0 }, 200)
  }, 150)
}

router.beforeEach((to, from, next) => {
  if (to.fullPath !== from.fullPath) startBar()
  next()
})
router.afterEach(finishBar)
router.onError(finishBar)

onUnmounted(() => {
  clearInterval(growTimer)
  clearTimeout(hideTimer)
})
</script>

<style>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>