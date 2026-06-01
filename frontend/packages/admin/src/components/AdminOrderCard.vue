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
  reviewInfo: [orderId: number]
  setPassword: [order: OrderResp]
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

// 待审核状态，且当前管理员角色匹配 → 显示 通过/拒绝
function showReviewActions(status: number, role: number): boolean {
  return (status === 1 && role === 1) || (status === 2 && role === 2)
}

// 审核通过 + 一级管理员 → 显示 审核信息/设置密码/通知用户
function showApprovedActions(status: number, role: number): boolean {
  return status === 5 && role === 1
}

// 仅显示审核信息（已通过但非一级管理员，或已驳回/已取消/已完成）
function showReviewInfoOnly(status: number, role: number): boolean {
  return (status === 5 && role !== 1) || status === 3 || status === 4 || status === 6 || status === 7
}

function onReview(action: number) {
  emit('review', props.order.id, action)
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
      <!-- 待审核：通过 + 拒绝 -->
      <template v-if="showReviewActions(order.status, adminRole)">
        <button
          class="px-3 py-1 text-xs font-medium rounded-lg bg-green-50 text-green-600 border border-green-200 hover:bg-green-100 transition"
          @click="onReview(1)"
        >
          通过
        </button>
        <button
          class="px-3 py-1 text-xs font-medium rounded-lg bg-red-50 text-red-600 border border-red-200 hover:bg-red-100 transition"
          @click="onReview(2)"
        >
          拒绝
        </button>
      </template>

      <!-- 审核通过（一级管理员）：审核信息 + 设置密码 + 通知用户 -->
      <template v-if="showApprovedActions(order.status, adminRole)">
        <button
          class="px-3 py-1 text-xs font-medium rounded-lg bg-gray-50 text-gray-600 border border-gray-200 hover:bg-gray-100 transition"
          @click="$emit('reviewInfo', order.id)"
        >
          审核信息
        </button>
        <button
          class="px-3 py-1 text-xs font-medium rounded-lg bg-blue-50 text-blue-600 border border-blue-200 hover:bg-blue-100 transition"
          @click="$emit('setPassword', order)"
        >
          设置密码
        </button>
        <button
          class="px-3 py-1 text-xs font-medium rounded-lg bg-primary-50 text-primary-600 border border-primary-200 hover:bg-primary-100 transition"
          @click="$emit('notifyUser', order.id)"
        >
          通知用户
        </button>
      </template>

      <!-- 其他终态：仅审核信息 -->
      <button
        v-if="showReviewInfoOnly(order.status, adminRole)"
        class="px-3 py-1 text-xs font-medium rounded-lg bg-gray-50 text-gray-600 border border-gray-200 hover:bg-gray-100 transition"
        @click="$emit('reviewInfo', order.id)"
      >
        审核信息
      </button>
    </div>
  </div>
</template>
