<script setup lang="ts">
import { ref, watch } from 'vue'
import { adminApi } from '@/api/client'
import type { ReviewRecordResp } from '../types'

const props = defineProps<{
  visible: boolean
  orderId: number | null
}>()

const emit = defineEmits<{
  close: []
}>()

const loading = ref(false)
const error = ref('')
const records = ref<ReviewRecordResp[]>([])

watch(() => props.orderId, async (id) => {
  if (id === null) return
  loading.value = true
  error.value = ''
  records.value = []
  try {
    const data = await adminApi.getOrderDetail(id)
    records.value = data.review_records || []
  } catch (e: any) {
    error.value = e.message || '加载失败'
  } finally {
    loading.value = false
  }
})

function close() {
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black bg-opacity-50 overscroll-contain" @click.self="close">
        <div class="bg-white rounded-xl shadow-xl w-full max-w-md max-h-[70vh] overflow-y-auto">
          <div class="sticky top-0 bg-white px-6 py-4 border-b border-gray-100 flex items-center justify-between rounded-t-xl">
            <h3 class="text-lg font-semibold text-gray-800">审核记录</h3>
            <button class="text-gray-400 hover:text-gray-600 transition" aria-label="关闭" @click="close">
              <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div v-if="loading" class="p-8 text-center text-gray-400">加载中…</div>

          <div v-else-if="error" class="p-8 text-center">
            <p class="text-gray-500 mb-2">{{ error }}</p>
            <button class="text-primary-500 text-sm" @click="close">关闭</button>
          </div>

          <div v-else-if="records.length === 0" class="p-8 text-center text-gray-400">
            暂无审核记录
          </div>

          <div v-else class="px-6 py-4 space-y-3">
            <div v-for="r in records" :key="r.id" class="border-b border-gray-100 pb-3 last:border-0">
              <div class="flex items-center justify-between text-sm">
                <span class="font-medium text-gray-700">{{ r.reviewer_name }}</span>
                <span class="text-xs text-gray-400">{{ r.created_at }}</span>
              </div>
              <div class="flex items-center gap-2 mt-1">
                <span class="text-xs text-gray-500">{{ r.role_text }}</span>
                <span
                  class="text-xs px-2 py-0.5 rounded-full"
                  :class="r.action === 1 ? 'bg-green-100 text-green-600' : 'bg-red-100 text-red-600'"
                >
                  {{ r.action_text }}
                </span>
              </div>
              <p v-if="r.comment" class="text-sm text-gray-600 mt-1">{{ r.comment }}</p>
            </div>
          </div>
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
