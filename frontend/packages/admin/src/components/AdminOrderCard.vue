<script setup lang="ts">
import { ORDER_STATUS_MAP } from '@reservation/shared'
import type { OrderResp } from '@reservation/shared'

const props = defineProps<{
  order: OrderResp
  adminRole: number
}>()

const emit = defineEmits<{
  showDetail: [id: number]
  review: [orderId: number, action: number]
  notifyUser: [orderId: number]
}>()

const STATUS_STYLE: Record<number, string> = {
  1: 'bg-yellow-100 text-yellow-700',
  2: 'bg-blue-100 text-blue-700',
  3: 'bg-red-100 text-red-600',
  4: 'bg-red-100 text-red-600',
  5: 'bg-green-100 text-green-600',
  6: 'bg-red-100 text-red-600',
  7: 'bg-red-100 text-red-600',
}

function showL1Actions(status: number, role: number): boolean {
  return status === 1 && role === 1
}

function showL2Actions(status: number, role: number): boolean {
  return status === 2 && role === 2
}

function hasPassword(order: OrderResp): boolean {
  return order.slots?.some(s => !!s.password) ?? false
}

function showNotify(status: number, role: number, order: OrderResp): boolean {
  return status === 5 && role === 1 && hasPassword(order)
}

function onReview(action: number) {
  emit('review', props.order.id, action)
}

function onNotify() {
  emit('notifyUser', props.order.id)
}
</script>

<template>
  <div
    class="bg-white rounded-xl border border-gray-100 p-4 cursor-pointer hover:border-primary-200 hover:shadow-md transition-all"
    @click="$emit('showDetail', order.id)"
  >
    <div class="flex items-start justify-between mb-2">
      <div class="flex-1 min-w-0">
        <p class="font-semibold text-gray-800 text-sm">{{ order.applicant_name }}</p>
        <p class="text-xs text-gray-400 font-mono mt-0.5">{{ order.order_no }}</p>
      </div>
      <span
        class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium flex-shrink-0 ml-2"
        :class="STATUS_STYLE[order.status] || 'bg-gray-100 text-gray-500'"
      >
        {{ ORDER_STATUS_MAP[order.status] || order.status_text }}
      </span>
    </div>

    <div class="text-xs text-gray-500 space-y-0.5 mb-3">
      <p>{{ order.alumni_association }} | {{ order.year }}级 | {{ order.major }}</p>
      <p>{{ order.phone }}</p>
      <p v-for="s in order.slots" :key="s.id" class="text-gray-600">{{ s.start_time }} ~ {{ s.end_time }}</p>
    </div>

    <div class="flex items-center gap-2" @click.stop>
      <button
        v-if="showL1Actions(order.status, adminRole)"
        class="px-3 py-1 text-xs font-medium rounded-lg bg-green-50 text-green-600 border border-green-200 hover:bg-green-100 transition"
        @click="onReview(1)"
      >
        通过
      </button>
      <button
        v-if="showL1Actions(order.status, adminRole)"
        class="px-3 py-1 text-xs font-medium rounded-lg bg-red-50 text-red-600 border border-red-200 hover:bg-red-100 transition"
        @click="onReview(2)"
      >
        拒绝
      </button>
      <button
        v-if="showL2Actions(order.status, adminRole)"
        class="px-3 py-1 text-xs font-medium rounded-lg bg-green-50 text-green-600 border border-green-200 hover:bg-green-100 transition"
        @click="onReview(1)"
      >
        通过
      </button>
      <button
        v-if="showL2Actions(order.status, adminRole)"
        class="px-3 py-1 text-xs font-medium rounded-lg bg-red-50 text-red-600 border border-red-200 hover:bg-red-100 transition"
        @click="onReview(2)"
      >
        拒绝
      </button>
      <button
        v-if="showNotify(order.status, adminRole, order)"
        class="ml-auto px-3 py-1 text-xs font-medium rounded-lg bg-primary-50 text-primary-600 border border-primary-200 hover:bg-primary-100 transition"
        @click="onNotify"
      >
        通知用户
      </button>
    </div>
  </div>
</template>
