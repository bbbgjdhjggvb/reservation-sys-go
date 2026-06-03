<script setup lang="ts">
/**
 * 日历时段单元格卡片。根据 cellState 渲染不同的视觉样式：
 *   available      - 白色卡片，可点击选择
 *   selected       - 红色卡片，可点击取消
 *   pending        - 琥珀色，"待审核"标签（仅自己的待审核，他人的已合并到"已占用"）
 *   approved       - 红色条纹，"已占用"标签（他人的预约，含 pending + approved）
 *   approved-mine  - 绿色调，"已通过"标签
 *   past           - 灰色，"不可选"标签
 */
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
      'bg-emerald-50/70 border-emerald-200 text-emerald-600': cellState === 'approved-mine',
      'bg-gray-100 border-gray-200/50 text-gray-400': cellState === 'past',
    }"
    @click="handleClick"
  >
    <div class="flex flex-col">
      <span class="text-[10px] text-gray-400 font-medium mb-0.5">时段</span>
      <span class="text-sm font-bold tracking-wide">{{ timeLabel }}</span>
    </div>
    <div>
      <!-- Available -->
      <button
        v-if="cellState === 'available'"
        class="text-xs px-4 py-1.5 border border-szu-red text-szu-red hover:bg-szu-red-light font-bold rounded-lg transition-all pointer-events-none"
      >
        选择
      </button>
      <!-- Selected -->
      <button
        v-else-if="cellState === 'selected'"
        class="text-xs px-4 py-1.5 bg-szu-red text-white font-bold rounded-lg shadow-sm pointer-events-none"
      >
        已选
      </button>
      <!-- 我的待审核（仅自己的 pending 才显示） -->
      <span
        v-else-if="cellState === 'pending'"
        class="text-xs px-3 py-1 bg-amber-100 text-amber-700 font-bold rounded-lg"
      >
        待审核
      </span>
      <!-- 他人的已占用（含 pending + approved） -->
      <span
        v-else-if="cellState === 'approved'"
        class="text-xs px-3 py-1 bg-red-100 text-red-500 font-bold rounded-lg"
      >
        已占用
      </span>
      <!-- 我的已通过 -->
      <span
        v-else-if="cellState === 'approved-mine'"
        class="text-xs px-3 py-1 bg-emerald-100 text-emerald-700 font-bold rounded-lg flex items-center gap-1"
      >
        <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 inline-block"></span>
        已通过
      </span>
      <!-- Past -->
      <span
        v-else
        class="text-xs px-3 py-1 bg-gray-200 text-gray-400 font-bold rounded-lg"
      >
        不可选
      </span>
    </div>
  </div>
</template>
