<script setup lang="ts">
defineProps<{
  date: string
  startTime: string
  endTime: string
  cellState: string
  label: string
  dayLabel: string
}>()

defineEmits<{
  toggle: []
}>()
</script>

<template>
  <button
    type="button"
    :aria-label="`${dayLabel} ${label}`"
    class="slot-cell flex items-center justify-center transition-colors select-none"
    :class="{
      'bg-primary-50 cursor-pointer hover:bg-primary-100 active:bg-primary-200 focus-visible:ring-2 focus-visible:ring-primary-300 focus-visible:ring-inset': cellState === 'available',
      'bg-primary-500 cursor-pointer focus-visible:ring-2 focus-visible:ring-primary-300 focus-visible:ring-inset': cellState === 'selected',
      'bg-red-100 cursor-not-allowed': cellState === 'approved',
      'bg-yellow-100 cursor-not-allowed': cellState === 'pending',
      'bg-gray-100 cursor-not-allowed': cellState === 'past',
    }"
    :disabled="cellState !== 'available' && cellState !== 'selected'"
    @click="cellState === 'available' || cellState === 'selected' ? $emit('toggle') : null"
  />
</template>

<style scoped>
.slot-cell {
  min-height: 36px;
  width: 100%;
  border: none;
  -webkit-tap-highlight-color: transparent;
  user-select: none;
}
</style>
