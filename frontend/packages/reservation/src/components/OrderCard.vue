<script setup lang="ts">
import { ORDER_STATUS_MAP } from '@reservation/shared'
import type { OrderResp } from '@reservation/shared'

defineProps<{
  order: OrderResp
  showCancel?: boolean
}>()

defineEmits<{
  cancel: [id: number]
}>()

const STATUS_STYLE: Record<number, string> = {
  0: 'bg-yellow-100 text-yellow-700',
  1: 'bg-green-100 text-green-600',
  2: 'bg-red-100 text-red-600',
  3: 'bg-gray-100 text-gray-500',
  4: 'bg-gray-100 text-gray-400',
  5: 'bg-blue-100 text-blue-700',
  6: 'bg-red-100 text-red-600',
  7: 'bg-red-100 text-red-600',
}

const SLOT_STATUS_DOT: Record<number, string> = {
  0: '#f59e0b',
  1: '#10b981',
  2: '#ef4444',
  3: '#9ca3af',
  4: '#d1d5db',
  5: '#f59e0b',
  6: '#ef4444',
  7: '#ef4444',
}

function canCancel(status: number): boolean {
  return status === 0 || status === 5
}
</script>

<template>
  <div class="order-card bg-white rounded-xl border border-gray-100 p-4 shadow-sm hover:shadow-md transition-shadow">
    <div class="flex items-start justify-between mb-2">
      <div class="flex-1 min-w-0">
        <p class="font-semibold text-gray-800 text-sm">{{ order.applicant_name }}</p>
        <p class="text-xs text-gray-400 font-mono mt-0.5">{{ order.order_no }}</p>
      </div>
      <span
        class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium flex-shrink-0 ml-2"
        :class="STATUS_STYLE[order.status] || 'bg-gray-100 text-gray-500'"
      >
        {{ ORDER_STATUS_MAP[order.status] || '未知' }}
      </span>
    </div>

    <div class="flex flex-wrap gap-1.5 mb-3">
      <span
        v-for="slot in order.slots"
        :key="slot.id"
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-gray-50 border border-gray-200"
      >
        <span class="w-1.5 h-1.5 rounded-full" :style="{ backgroundColor: SLOT_STATUS_DOT[slot.status] || '#9ca3af' }" />
        {{ slot.start_time.slice(5) }}~{{ slot.end_time.slice(11) }}
      </span>
    </div>

    <div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-gray-500">
      <p>校友会：{{ order.alumni_association }}</p>
      <p>专业：{{ order.major }}</p>
      <p>手机：{{ order.phone }}</p>
      <p>提交：{{ order.created_at }}</p>
    </div>

    <div class="mt-3 pt-3 border-t border-gray-50 flex items-center justify-between">
      <p v-if="order.reason" class="text-xs text-gray-600 flex-1 min-w-0 truncate">
        {{ order.reason }}
      </p>
      <button
        v-if="showCancel && canCancel(order.status)"
        class="ml-auto px-3 py-1.5 text-xs font-medium text-red-600 bg-red-50 border border-red-200 rounded-lg hover:bg-red-100 transition flex-shrink-0"
        @click="$emit('cancel', order.id)"
      >
        取消预约
      </button>
    </div>
  </div>
</template>
