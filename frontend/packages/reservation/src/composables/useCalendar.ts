import { ref, reactive, computed } from 'vue'
import { addDays, formatDate, isPastDay, isBeyondBookable, isSlotInPast, extractTime } from '@/utils/date'
import { api } from '@/api/client'
import { MAX_SLOTS, WEEKDAYS, TIME_SLOTS } from '@reservation/shared'
import type { SelectedSlot, OccupiedSlot } from '@reservation/shared'

// Number of days visible in the date axis (today + 13 = 14 days)
const VISIBLE_DAY_COUNT = 14

// Module-level state — shared across all components that call useCalendar()
const activeDayIndex = ref(0)
const selectedSlots = ref<SelectedSlot[]>([])
const occupiedSlots = reactive<Record<string, OccupiedSlot[]>>({})

export function useCalendar() {

  // 14 days starting from today
  const visibleDays = computed(() => {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    return Array.from({ length: VISIBLE_DAY_COUNT }, (_, i) => addDays(today, i))
  })

  // Human-readable label for the active day (e.g. "今天 6月1日", "明天 6月2日", "周三 6月3日")
  const activeDayLabel = computed(() => {
    const day = visibleDays.value[activeDayIndex.value]
    if (!day) return ''
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    const diff = Math.round((day.getTime() - today.getTime()) / 86400000)
    if (diff === 0) return `今天 ${day.getMonth() + 1}月${day.getDate()}日`
    if (diff === 1) return `明天 ${day.getMonth() + 1}月${day.getDate()}日`
    // Map Sunday=0..Saturday=6 to Monday=0..Sunday=6
    const dayOfWeek = day.getDay()
    const adjustedDow = dayOfWeek === 0 ? 6 : dayOfWeek - 1
    return `周${WEEKDAYS[adjustedDow]} ${day.getMonth() + 1}月${day.getDate()}日`
  })

  // Whether the active day falls in "next week" relative to today
  const isNextWeek = computed(() => {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    const dayOfWeek = today.getDay() || 7 // Mon=1..Sun=7
    const daysUntilNextMonday = 8 - dayOfWeek
    const nextMonday = addDays(today, daysUntilNextMonday)
    const active = visibleDays.value[activeDayIndex.value]
    return active ? active >= nextMonday : false
  })

  // Formatted date string for the active day (YYYY-MM-DD)
  const activeDateStr = computed(() => {
    const day = visibleDays.value[activeDayIndex.value]
    return day ? formatDate(day) : ''
  })

  async function fetchOccupiedSlots() {
    const dates = visibleDays.value.map(d => formatDate(d))
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

  /**
   * 查询指定时段的占用状态及归属信息。
   * 遍历 occupiedSlots 中该日期的所有已占用时段，使用时间重叠算法匹配。
   *
   * @param dateStr - 日期字符串 "YYYY-MM-DD"
   * @param startTime - 时段开始时间 "HH:MM"
   * @param endTime - 时段结束时间 "HH:MM"
   * @returns 匹配到的占用信息，未匹配时返回 null
   */
  function getSlotInfo(dateStr: string, startTime: string, endTime: string): { status: string; isMine: boolean } | null {
    const slots = occupiedSlots[dateStr]
    if (!slots) return null

    for (const s of slots) {
      const sStart = extractTime(s.start_time)
      const sEnd = extractTime(s.end_time)
      // 时间范围有交集即视为占用
      if (startTime < sEnd && endTime > sStart) {
        return { status: s.status, isMine: s.is_mine }
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

  /**
   * 计算日历单元格的显示状态。
   * 优先级：past > 自己的占用 > 他人的占用 > selected > available
   *
   * 状态映射：
   *   自己的 pending  → pending   （待审核）
   *   自己的 approved → approved-mine（我的预约）
   *   他人的 pending  → approved  （已占用，合并到他人的已占用）
   *   他人的 approved → approved  （已占用）
   */
  function getCellState(dateStr: string, startTime: string, endTime: string, date: Date): string {
    if (isPastDay(date) || isBeyondBookable(date)) return 'past'
    if (isSlotInPast(dateStr, endTime) && addDays(new Date(), 0).toDateString() === date.toDateString()) return 'past'

    const info = getSlotInfo(dateStr, startTime, endTime)
    if (info) {
      if (info.isMine) {
        // 自己的 pending → "待审核"，自己的 approved → "我的预约"
        return info.status === 'pending' ? 'pending' : 'approved-mine'
      }
      // 他人的：pending 和 approved 统一显示为 "已占用"
      return 'approved'
    }

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
    activeDayIndex.value = 0
    clearOccupiedCache()
    await fetchOccupiedSlots()
  }

  return {
    // New: 14-day daily model
    visibleDays,
    activeDayIndex,
    activeDayLabel,
    activeDateStr,
    isNextWeek,

    // Unchanged
    selectedSlots,
    occupiedSlots,
    fetchOccupiedSlots,
    getCellState,
    toggleSlot,
    isSlotSelected,
    formatSlotDisplay,
    clearOccupiedCache,
    resetSelection,
    resetAndRefresh,
    MAX_SLOTS,
    TIME_SLOTS,
  }
}
