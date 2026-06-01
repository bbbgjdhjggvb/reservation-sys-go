<script setup lang="ts">
import type { OrderResp } from '@reservation/shared'

defineProps<{
  adminRole: number
  currentStatus: number | string
}>()

const emit = defineEmits<{
  'update:currentStatus': [status: number | string]
}>()

const L1_TABS = [
  { label: '全部', status: -1 as const },
  { label: '待一级审核', status: 1 as const },
  { label: '已通过', status: 5 as const },
  { label: '已驳回', status: '3,4' as const },
]

const L2_TABS = [
  { label: '全部', status: -1 as const },
  { label: '待二级审核', status: 2 as const },
  { label: '已通过', status: 5 as const },
  { label: '已驳回', status: '3,4' as const },
]
</script>

<template>
  <div class="bg-white rounded-t-xl border-b border-gray-100">
    <div class="flex gap-0 px-6">
      <button
        v-for="tab in (adminRole === 2 ? L2_TABS : L1_TABS)"
        :key="String(tab.status)"
        class="px-4 py-3 text-sm font-medium border-b-2 transition-colors -mb-px"
        :class="currentStatus === tab.status
          ? 'text-primary-500 border-primary-500'
          : 'text-gray-400 border-transparent hover:text-primary-400'"
        @click="$emit('update:currentStatus', tab.status)"
      >
        {{ tab.label }}
      </button>
    </div>
  </div>
</template>
