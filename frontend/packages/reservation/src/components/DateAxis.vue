<script setup lang="ts">
import { ref, watch, onMounted, nextTick } from 'vue'
import { formatDateShort } from '@/utils/date'

const props = defineProps<{
  days: Date[]
  activeIndex: number
}>()

const emit = defineEmits<{
  'select-day': [index: number]
}>()

const scrollContainer = ref<HTMLElement | null>(null)
const dayButtonRefs = ref<(HTMLElement | null)[]>([])

function setDayRef(el: HTMLElement | null, index: number) {
  dayButtonRefs.value[index] = el
}

// Weekday labels in Chinese (Mon-Sun)
const WEEKDAY_CN = ['一', '二', '三', '四', '五', '六', '日']

function getWeekday(date: Date): string {
  const d = date.getDay()
  return WEEKDAY_CN[d === 0 ? 6 : d - 1]
}

function getDateShort(date: Date): string {
  return formatDateShort(date)
}

function isToday(date: Date): boolean {
  const now = new Date()
  return date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate()
}

function selectDay(index: number) {
  emit('select-day', index)
  centerActiveDay(index)
}

function centerActiveDay(index: number) {
  const container = scrollContainer.value
  const targetBtn = dayButtonRefs.value[index]
  if (!container || !targetBtn) return

  nextTick(() => {
    const containerRect = container.getBoundingClientRect()
    const btnRect = targetBtn.getBoundingClientRect()
    const btnCenter = btnRect.left + btnRect.width / 2
    const containerCenter = containerRect.left + containerRect.width / 2
    const offset = btnCenter - containerCenter
    const maxScroll = container.scrollWidth - container.clientWidth
    const clampedOffset = Math.max(-container.scrollLeft, Math.min(offset, maxScroll - container.scrollLeft))
    container.scrollBy({ left: clampedOffset, behavior: 'smooth' })
  })
}

// Wheel to horizontal scroll conversion
function onWheel(evt: WheelEvent) {
  if (evt.deltaX === 0) {
    evt.preventDefault()
    scrollContainer.value?.scrollBy({ left: evt.deltaY, behavior: 'auto' })
  }
}

// Center active day when activeIndex changes
watch(() => props.activeIndex, (idx) => {
  centerActiveDay(idx)
})

onMounted(() => {
  centerActiveDay(props.activeIndex)
})
</script>

<template>
  <div class="relative">
    <!-- Scroll container -->
    <div
      ref="scrollContainer"
      class="flex overflow-x-auto space-x-3 pb-1 no-scrollbar scroll-smooth pr-16"
      style="-webkit-overflow-scrolling: touch;"
      @wheel.passive="onWheel"
    >
      <button
        v-for="(day, idx) in days"
        :key="idx"
        :ref="(el: any) => setDayRef(el, idx)"
        @click="selectDay(idx)"
        class="date-btn flex-shrink-0 w-12 py-2.5 rounded-xl flex flex-col items-center transition-all duration-300"
        :class="idx === activeIndex
          ? 'bg-szu-red text-white shadow-md'
          : 'bg-white border border-gray-100 hover:border-szu-red/30'"
      >
        <span
          class="text-[10px] mb-0.5"
          :class="idx === activeIndex ? 'text-white opacity-85' : 'text-gray-400'"
        >
          {{ getWeekday(day) }}
        </span>
        <span
          class="text-sm font-bold"
          :class="idx === activeIndex ? 'text-white' : 'text-gray-800'"
        >
          {{ getDateShort(day) }}
        </span>
        <!-- Today dot indicator -->
        <span
          v-if="isToday(day)"
          class="w-1 h-1 rounded-full mt-1"
          :class="idx === activeIndex ? 'bg-white' : 'bg-szu-red'"
        />
        <span v-else class="w-1 h-1 mt-1" />
      </button>
    </div>

    <!-- Gradient fade mask on right edge -->
    <div class="absolute right-0 top-0 bottom-0 w-16 pointer-events-none z-20 date-axis-fade" />
  </div>
</template>

<style scoped>
.date-axis-fade {
  background: linear-gradient(to left, white 0%, rgba(255, 255, 255, 0.8) 40%, transparent 100%);
}
</style>
