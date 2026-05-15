export function getMonday(d: Date): Date {
  const date = new Date(d)
  const day = date.getDay()
  const diff = day === 0 ? -6 : 1 - day
  date.setDate(date.getDate() + diff)
  date.setHours(0, 0, 0, 0)
  return date
}

export function addDays(d: Date, n: number): Date {
  const result = new Date(d)
  result.setDate(result.getDate() + n)
  return result
}

function pad2(n: number): string {
  return n < 10 ? '0' + n : '' + n
}

export function formatDate(d: Date): string {
  const y = d.getFullYear()
  const m = pad2(d.getMonth() + 1)
  const day = pad2(d.getDate())
  return `${y}-${m}-${day}`
}

export function formatDateShort(d: Date): string {
  return `${d.getMonth() + 1}/${d.getDate()}`
}

export function isSameDay(d1: Date, d2: Date): boolean {
  return d1.getFullYear() === d2.getFullYear() &&
    d1.getMonth() === d2.getMonth() &&
    d1.getDate() === d2.getDate()
}

export function isToday(d: Date): boolean {
  return isSameDay(d, new Date())
}

export function isPastDay(d: Date): boolean {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return d < today
}

export function isBeyondBookable(d: Date): boolean {
  const limit = addDays(new Date(), 13)
  limit.setHours(23, 59, 59, 999)
  return d > limit
}

export function isSlotInPast(dateStr: string, endTime: string): boolean {
  const now = new Date()
  const slotEnd = new Date(`${dateStr}T${endTime}:00`)
  return slotEnd < now
}

export function formatDateCN(d: Date): string {
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
}

export function extractTime(isoStr: string): string {
  return isoStr.slice(11, 16)
}
