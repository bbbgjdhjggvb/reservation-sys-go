<script setup lang="ts">
const props = defineProps<{
  date: string
  startTime: string
  endTime: string
  cellState: string
}>()

const emit = defineEmits<{
  toggle: []
}>()

function handleClick() {
  if (props.cellState === 'available' || props.cellState === 'selected') {
    emit('toggle')
  }
}

// Time slot label like "08:00 - 10:00"
const timeLabel = `${props.startTime.slice(0, 5)} - ${props.endTime.slice(0, 5)}`
</script>

<template>
  <div
    class="flex items-center justify-between p-4 border rounded-xl transition-all select-none"
    :class="{
      'bg-white border-gray-100 hover:border-szu-red/50 shadow-sm cursor-pointer': cellState === 'available',
      'border-szu-red bg-szu-red/5 text-szu-red ring-2 ring-szu-red/20 shadow-sm cursor-pointer': cellState === 'selected',
      'bg-amber-50/50 border-amber-100 text-amber-600': cellState === 'pending',
      'bg-red-50/70 border-red-100 text-red-400 stripe-bg': cellState === 'approved',
      'bg-gray-100 border-gray-200/50 text-gray-400': cellState === 'past',
    }"
    @click="handleClick"
  >
    <div class="flex flex-col">
      <span class="text-[10px] text-gray-400 font-medium mb-0.5">时段</span>
      <span class="text-sm font-bold tracking-wide">{{ timeLabel }}</span>
    </div>
    <div>
      <!-- Available: select button -->
      <button
        v-if="cellState === 'available'"
        class="text-xs px-4 py-1.5 border border-szu-red text-szu-red hover:bg-szu-red-light font-bold rounded-lg transition-all pointer-events-none"
      >
        选择
      </button>
      <!-- Selected: deselect button -->
      <button
        v-else-if="cellState === 'selected'"
        class="text-xs px-4 py-1.5 bg-szu-red text-white font-bold rounded-lg shadow-sm pointer-events-none"
      >
        已选
      </button>
      <!-- Pending: label -->
      <span
        v-else-if="cellState === 'pending'"
        class="text-xs px-3 py-1 bg-amber-100 text-amber-700 font-bold rounded-lg"
      >
        待审核
      </span>
      <!-- Approved/Occupied: label -->
      <span
        v-else-if="cellState === 'approved'"
        class="text-xs px-3 py-1 bg-red-100 text-red-500 font-bold rounded-lg"
      >
        已占用
      </span>
      <!-- Past / disabled: label -->
      <span
        v-else
        class="text-xs px-3 py-1 bg-gray-200 text-gray-400 font-bold rounded-lg"
      >
        不可选
      </span>
    </div>
  </div>
</template>
