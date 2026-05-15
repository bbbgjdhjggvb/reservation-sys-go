import { ref, reactive, computed } from 'vue'
import { getMonday, addDays, formatDate, isPastDay, isBeyondBookable, isSlotInPast, extractTime } from '@/utils/date'
import { api } from '@/api/client'
import { MAX_SLOTS, WEEKDAYS, TIME_SLOTS } from '@reservation/shared'
import type { SelectedSlot, OccupiedSlot } from '@reservation/shared'

// Module-level state — shared across all components that call useCalendar()
const currentWeekStart = ref(getMonday(new Date()))
const selectedSlots = ref<SelectedSlot[]>([])
const occupiedSlots = reactive<Record<string, OccupiedSlot[]>>({})

export function useCalendar() {

  const weekDays = computed(() => {
    return Array.from({ length: 7 }, (_, i) => addDays(currentWeekStart.value, i))
  })

  const weekLabel = computed(() => {
    const end = addDays(currentWeekStart.value, 6)
    const sm = currentWeekStart.value.getMonth() + 1
    const sd = currentWeekStart.value.getDate()
    const em = end.getMonth() + 1
    const ed = end.getDate()
    const sy = currentWeekStart.value.getFullYear()
    const ey = end.getFullYear()
    if (sy !== ey) {
      return `${sy}年${sm}月${sd}日 — ${ey}年${em}月${ed}日`
    }
    return `${sy}年${sm}月${sd}日 — ${em}月${ed}日`
  })

  const canGoPrev = computed(() => {
    const today = getMonday(new Date())
    return currentWeekStart.value > today
  })

  const canGoNext = computed(() => {
    const limit = addDays(new Date(), 13)
    return addDays(currentWeekStart.value, 6) < limit
  })

  function changeWeek(delta: number) {
    currentWeekStart.value = addDays(currentWeekStart.value, delta * 7)
    selectedSlots.value = []
    fetchOccupiedSlots()
  }

  async function fetchOccupiedSlots() {
    const dates = weekDays.value.map(d => formatDate(d))
    const promises = dates.map(async (ds) => {
      if (occupiedSlots[ds] !== undefined) return
      try {
        const data = await api.getOccupiedSlots(ds)
        occupiedSlots[ds] = data || []
      } catch {
        occupiedSlots[ds] = []
      }
    })
    await Promise.all(promises.map(p => p.catch(() => {})))
  }

  function getSlotStatus(dateStr: string, startTime: string, endTime: string): string | null {
    const slots = occupiedSlots[dateStr]
    if (!slots) return null

    for (const s of slots) {
      const sStart = extractTime(s.start_time)
      const sEnd = extractTime(s.end_time)
      if (startTime < sEnd && endTime > sStart) {
        return s.status
      }
    }
    return null
  }

  function isSlotSelected(dateStr: string, startTime: string, endTime: string): boolean {
    return selectedSlots.value.some(
      s => s.date === dateStr && s.startTime === startTime && s.endTime === endTime
    )
  }

  function toggleSlot(dateStr: string, startTime: string, endTime: string) {
    const idx = selectedSlots.value.findIndex(
      s => s.date === dateStr && s.startTime === startTime && s.endTime === endTime
    )
    if (idx >= 0) {
      selectedSlots.value.splice(idx, 1)
    } else {
      if (selectedSlots.value.length >= MAX_SLOTS) {
        return false
      }
      selectedSlots.value.push({ date: dateStr, startTime, endTime })
    }
    return true
  }

  function getCellState(dateStr: string, startTime: string, endTime: string, date: Date): string {
    if (isPastDay(date) || isBeyondBookable(date)) return 'past'
    if (isSlotInPast(dateStr, endTime) && addDays(new Date(), 0).toDateString() === date.toDateString()) return 'past'
    const status = getSlotStatus(dateStr, startTime, endTime)
    if (status) return status
    if (isSlotSelected(dateStr, startTime, endTime)) return 'selected'
    return 'available'
  }

  function formatSlotDisplay(slot: SelectedSlot): string {
    const d = new Date(slot.date + 'T00:00:00')
    const wd = WEEKDAYS[(d.getDay() + 6) % 7]
    return `${d.getMonth() + 1}月${d.getDate()}日(周${wd}) ${slot.startTime}-${slot.endTime}`
  }

  function clearOccupiedCache() {
    Object.keys(occupiedSlots).forEach(k => delete occupiedSlots[k])
  }

  function resetSelection() {
    selectedSlots.value.length = 0
  }

  async function resetAndRefresh() {
    resetSelection()
    clearOccupiedCache()
    await fetchOccupiedSlots()
  }

  return {
    currentWeekStart,
    selectedSlots,
    occupiedSlots,
    weekDays,
    weekLabel,
    canGoPrev,
    canGoNext,
    changeWeek,
    fetchOccupiedSlots,
    getCellState,
    toggleSlot,
    isSlotSelected,
    formatSlotDisplay,
    clearOccupiedCache,
    resetSelection,
    resetAndRefresh,
    MAX_SLOTS,
    WEEKDAYS,
    TIME_SLOTS,
  }
}
