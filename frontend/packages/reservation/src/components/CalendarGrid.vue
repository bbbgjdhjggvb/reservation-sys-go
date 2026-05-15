<script setup lang="ts">
import { onMounted } from 'vue'
import { formatDate, formatDateShort, isToday, isPastDay, addDays } from '@/utils/date'
import { useCalendar } from '@/composables/useCalendar'
import { useToast } from '@/composables/useToast'
import SlotCell from './SlotCell.vue'

const cal = useCalendar()
const { weekDays, weekLabel, canGoPrev, canGoNext } = cal
const { showToast } = useToast()

const dn = ['一', '二', '三', '四', '五', '六', '日']

onMounted(() => {
  cal.fetchOccupiedSlots()
})

function handleToggleSlot(dateStr: string, startTime: string, endTime: string) {
  const ok = cal.toggleSlot(dateStr, startTime, endTime)
  if (!ok) {
    showToast(`最多只能选择${cal.MAX_SLOTS}个时间段`, 'warning')
  }
}
</script>

<template>
  <div>
    <!-- Week Navigation -->
    <div class="flex items-center justify-between mb-4">
      <button
        :disabled="!canGoPrev"
        class="px-3 py-1 text-sm rounded transition-opacity"
        :class="canGoPrev ? 'text-primary-500 hover:bg-primary-50' : 'text-gray-300 cursor-not-allowed'"
        @click="cal.changeWeek(-1)"
      >
        上一周
      </button>
      <span class="text-sm font-medium text-gray-700">{{ weekLabel }}</span>
      <button
        :disabled="!canGoNext"
        class="px-3 py-1 text-sm rounded transition-opacity"
        :class="canGoNext ? 'text-primary-500 hover:bg-primary-50' : 'text-gray-300 cursor-not-allowed'"
        @click="cal.changeWeek(1)"
      >
        下一周
      </button>
    </div>

    <!-- Calendar Grid -->
    <div class="overflow-x-auto -mx-4 sm:mx-0">
      <div
        class="grid min-w-[560px]"
        style="grid-template-columns: 72px repeat(7, 1fr);"
      >
        <!-- Header row: empty corner + 7 day columns -->
        <div class="p-2 text-xs text-gray-400"></div>
        <div
          v-for="(day, i) in weekDays"
          :key="i"
          class="p-2 text-center text-xs font-medium"
          :class="{
            'text-primary-500 bg-primary-50': isToday(day),
            'text-gray-500': !isToday(day) && !isPastDay(day),
            'text-gray-300': isPastDay(day),
          }"
        >
          <div>{{ dn[i] }}</div>
          <div :class="{ 'text-primary-600': isToday(day) }">{{ formatDateShort(day) }}</div>
        </div>

        <!-- Time slot rows -->
        <template v-for="slot in cal.TIME_SLOTS" :key="slot.label">
          <div class="p-2 text-xs text-gray-600 text-right pr-3">
            {{ slot.label }}
          </div>
          <div
            v-for="(day, j) in weekDays"
            :key="j"
          >
            <SlotCell
              :date="formatDate(day)"
              :start-time="slot.start"
              :end-time="slot.end"
              :label="slot.label"
              :day-label="`${dn[j]} ${formatDateShort(day)}`"
              :cell-state="cal.getCellState(formatDate(day), slot.start, slot.end, day)"
              @toggle="handleToggleSlot(formatDate(day), slot.start, slot.end)"
            />
          </div>
        </template>
      </div>
    </div>
  </div>
</template>
