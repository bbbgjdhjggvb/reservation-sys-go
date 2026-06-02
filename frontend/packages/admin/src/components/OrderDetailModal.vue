<script setup lang="ts">
import { ref, watch } from 'vue'
import { ORDER_STATUS_MAP } from '@reservation/shared'
import { adminApi } from '@/api/client'
import type { OrderDetailResp } from '../types'

const props = defineProps<{
  orderId: number | null
}>()

const emit = defineEmits<{
  close: []
}>()

const loading = ref(false)
const error = ref('')
const detail = ref<OrderDetailResp | null>(null)

watch(() => props.orderId, async (id) => {
  if (id === null) return
  loading.value = true
  error.value = ''
  detail.value = null
  try {
    detail.value = await adminApi.getOrderDetail(id)
  } catch (e: any) {
    error.value = e.message || '加载失败'
  } finally {
    loading.value = false
  }
})

const STATUS_BADGE: Record<number, string> = {
  1: 'bg-yellow-100 text-yellow-700',
  2: 'bg-blue-100 text-blue-700',
  3: 'bg-red-100 text-red-600',
  4: 'bg-red-100 text-red-600',
  5: 'bg-green-100 text-green-600',
  6: 'bg-red-100 text-red-600',
  7: 'bg-red-100 text-red-600',
}

function close() {
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="orderId !== null" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black bg-opacity-50 overscroll-contain" @click.self="close">
        <div class="bg-white rounded-xl shadow-xl w-full max-w-md max-h-[80vh] overflow-y-auto">
          <!-- Loading -->
          <div v-if="loading" class="p-8 text-center text-gray-400">加载中…</div>

          <!-- Error -->
          <div v-else-if="error" class="p-8 text-center">
            <p class="text-gray-500 mb-2">{{ error }}</p>
            <button class="text-primary-500 text-sm" @click="close">关闭</button>
          </div>

          <!-- Content -->
          <template v-else-if="detail">
            <div class="sticky top-0 bg-white px-5 py-3 border-b border-gray-100 flex items-center justify-between rounded-t-xl">
              <h3 class="text-base font-semibold text-gray-800">订单详情</h3>
              <button class="text-gray-400 hover:text-gray-600 transition" aria-label="关闭" @click="close">
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div class="px-5 py-4 space-y-3">
              <!-- Order No & Status -->
              <div class="flex items-center justify-between">
                <p class="font-mono text-xs text-gray-400">{{ detail.order.order_no }}</p>
                <span
                  class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                  :class="STATUS_BADGE[detail.order.status] || 'bg-gray-100 text-gray-500'"
                >
                  {{ ORDER_STATUS_MAP[detail.order.status] || '未知' }}
                </span>
              </div>

              <!-- Applicant Info -->
              <div class="grid grid-cols-2 gap-x-4 gap-y-1.5 text-sm">
                <div><span class="text-gray-400">申请人</span> <span class="text-gray-700 ml-1">{{ detail.order.applicant_name }}</span></div>
                <div><span class="text-gray-400">手机</span> <span class="text-gray-700 ml-1">{{ detail.order.phone }}</span></div>
                <div><span class="text-gray-400">会议人数</span> <span class="text-gray-700 ml-1">{{ detail.order.attendee_count }} 人</span></div>
                <div><span class="text-gray-400">校友会</span> <span class="text-gray-700 ml-1">{{ detail.order.alumni_association }}</span></div>
                <div><span class="text-gray-400">专业</span> <span class="text-gray-700 ml-1">{{ detail.order.major }} ({{ detail.order.year }}级)</span></div>
              </div>

              <!-- Reason -->
              <div>
                <span class="text-xs text-gray-400">事由</span>
                <p class="text-sm text-gray-700 mt-0.5 bg-gray-50 p-2 rounded">{{ detail.order.reason }}</p>
              </div>

              <!-- Slots Table -->
              <div>
                <h4 class="text-xs font-medium text-gray-500 mb-1.5">预约时段</h4>
                <table class="w-full text-left text-xs">
                  <thead>
                    <tr class="text-gray-400">
                      <th class="pb-1 font-medium">时间</th>
                      <th class="pb-1 font-medium">状态</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="s in detail.order.slots" :key="s.id" class="border-t border-gray-50">
                      <td class="py-1.5 text-gray-700">{{ s.start_time }} ~ {{ s.end_time }}</td>
                      <td class="py-1.5">
                        <span
                          class="inline-flex items-center px-2 py-0.5 rounded-full text-xs"
                          :class="STATUS_BADGE[s.status] || 'bg-gray-100 text-gray-500'"
                        >
                          {{ s.status_text }}
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </template>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active { animation: modalIn 0.2s ease; }
.modal-leave-active { animation: modalIn 0.2s ease reverse; }
@keyframes modalIn {
  from { opacity: 0; transform: scale(1) translateY(10px); }
  to   { opacity: 1; transform: scale(1) translateY(0); }
}
</style>
