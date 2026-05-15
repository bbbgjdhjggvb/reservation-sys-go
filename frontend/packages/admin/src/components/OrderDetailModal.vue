<script setup lang="ts">
import { ref, watch } from 'vue'
import { ORDER_STATUS_MAP } from '@reservation/shared'
import { adminApi } from '@/api/client'
import type { OrderDetailResp, ReviewRecordResp } from '../types'
import type { OrderResp } from '@reservation/shared'

const props = defineProps<{
  orderId: number | null
  adminRole: number
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

const loading = ref(false)
const error = ref('')
const detail = ref<OrderDetailResp | null>(null)
const passwordInputs = ref<Record<number, string>>({})
const savingPassword = ref<Record<number, boolean>>({})
const notifying = ref(false)

watch(() => props.orderId, async (id) => {
  if (id === null) return
  loading.value = true
  error.value = ''
  detail.value = null
  passwordInputs.value = {}
  try {
    const data = await adminApi.getOrderDetail(id)
    detail.value = data
    // Pre-populate password inputs
    if (data.order.slots) {
      for (const s of data.order.slots) {
        passwordInputs.value[s.id] = s.password || ''
      }
    }
  } catch (e: any) {
    error.value = e.message || '加载失败'
  } finally {
    loading.value = false
  }
})

function hasPassword(order: OrderResp): boolean {
  return order.slots?.some(s => !!s.password) ?? false
}

function showPasswordForm(order: OrderResp, role: number): boolean {
  return order.status === 1 && role === 1
}

function showNotifyBtn(order: OrderResp, role: number): boolean {
  return order.status === 1 && role === 1 && hasPassword(order)
}

const STATUS_BADGE: Record<number, string> = {
  0: 'bg-yellow-100 text-yellow-700',
  1: 'bg-green-100 text-green-600',
  5: 'bg-blue-100 text-blue-700',
  6: 'bg-red-100 text-red-600',
  7: 'bg-red-100 text-red-600',
}

async function savePassword(orderId: number, slotId: number) {
  const pwd = (passwordInputs.value[slotId] || '').trim()
  if (!pwd) return
  savingPassword.value[slotId] = true
  try {
    await adminApi.setPassword(orderId, slotId, pwd)
    // Show toast via emit
  } catch (e: any) {
    alert(e.message || '保存失败')
  } finally {
    savingPassword.value[slotId] = false
  }
}

async function sendNotify(orderId: number) {
  notifying.value = true
  try {
    await adminApi.notify(orderId)
    emit('updated')
  } catch (e: any) {
    alert(e.message || '发送失败')
  } finally {
    notifying.value = false
  }
}

function close() {
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="orderId !== null" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black bg-opacity-50 overscroll-contain" @click.self="close">
        <div class="bg-white rounded-xl shadow-xl w-full max-w-lg max-h-[85vh] overflow-y-auto">
          <!-- Loading -->
          <div v-if="loading" class="p-8 text-center text-gray-400">加载中…</div>

          <!-- Error -->
          <div v-else-if="error" class="p-8 text-center">
            <p class="text-gray-500 mb-2">{{ error }}</p>
            <button class="text-primary-500 text-sm" @click="close">关闭</button>
          </div>

          <!-- Detail Content -->
          <template v-else-if="detail">
            <div class="sticky top-0 bg-white px-6 py-4 border-b border-gray-100 flex items-center justify-between rounded-t-xl">
              <h3 class="text-lg font-semibold text-gray-800">订单详情</h3>
              <button class="text-gray-400 hover:text-gray-600 transition" aria-label="关闭详情" @click="close">
                <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div class="px-6 py-4 space-y-4">
              <!-- Order Info -->
              <label class="text-sm text-gray-500">订单号</label>
              <p class="font-mono text-sm text-gray-800">{{ detail.order.order_no }}</p>

              <div class="grid grid-cols-2 gap-3 text-sm">
                <div><span class="text-gray-500">申请人：</span>{{ detail.order.applicant_name }}</div>
                <div><span class="text-gray-500">手机：</span>{{ detail.order.phone }}</div>
                <div><span class="text-gray-500">校友会：</span>{{ detail.order.alumni_association }}</div>
                <div><span class="text-gray-500">专业：</span>{{ detail.order.major }} ({{ detail.order.year }}级)</div>
              </div>

              <div>
                <span class="text-sm text-gray-500">事由：</span>
                <p class="text-sm text-gray-800 mt-1 bg-gray-50 p-2 rounded">{{ detail.order.reason }}</p>
              </div>

              <div>
                <span
                  class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                  :class="STATUS_BADGE[detail.order.status] || 'bg-gray-100 text-gray-500'"
                >
                  {{ ORDER_STATUS_MAP[detail.order.status] || '未知' }}
                </span>
              </div>

              <!-- Slots Table -->
              <div>
                <h4 class="text-sm font-medium text-gray-700 mb-2">预约时段</h4>
                <table class="w-full text-left text-xs">
                  <thead>
                    <tr class="text-gray-400">
                      <th class="pb-1 font-medium">时间</th>
                      <th class="pb-1 font-medium">状态</th>
                      <th class="pb-1 font-medium">门锁密码</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="s in detail.order.slots" :key="s.id" class="border-t border-gray-50">
                      <td class="py-2 text-gray-700">{{ s.start_time }} ~ {{ s.end_time }}</td>
                      <td class="py-2">
                        <span
                          class="inline-flex items-center px-2 py-0.5 rounded-full text-xs"
                          :class="STATUS_BADGE[s.status] || 'bg-gray-100 text-gray-500'"
                        >
                          {{ s.status_text }}
                        </span>
                      </td>
                      <td class="py-2 text-gray-500 font-mono text-xs">
                        {{ s.password || '-' }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <!-- Password Form (conditional) -->
              <div v-if="showPasswordForm(detail.order, adminRole)" class="bg-blue-50 p-4 rounded-lg">
                <h4 class="text-sm font-medium text-blue-700 mb-3">设置门锁密码</h4>
                <div v-for="s in detail.order.slots" :key="'pwd_' + s.id" class="flex items-center gap-2 mb-2">
                  <span class="text-xs text-gray-600 w-36">{{ s.start_time.slice(5) }}~{{ s.end_time.slice(11) }}</span>
                  <input
                    v-model="passwordInputs[s.id]"
                    type="text"
                    maxlength="20"
                    class="flex-1 px-2 py-1 text-sm border border-gray-200 rounded focus:border-red-400 outline-none"
                    placeholder="输入密码…"
                  />
                  <button
                    :disabled="savingPassword[s.id]"
                    class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-60 transition"
                    @click="savePassword(detail.order.id, s.id)"
                  >
                    {{ savingPassword[s.id] ? '…' : '保存' }}
                  </button>
                </div>
              </div>

              <!-- Notify Button -->
              <button
                v-if="showNotifyBtn(detail.order, adminRole)"
                :disabled="notifying"
                class="w-full py-2.5 rounded-lg bg-primary-500 text-white font-medium hover:bg-primary-600 disabled:opacity-60 transition"
                @click="sendNotify(detail.order.id)"
              >
                {{ notifying ? '发送中…' : '发送通知给用户' }}
              </button>

              <!-- Review Records -->
              <div v-if="detail.review_records?.length">
                <h4 class="text-sm font-medium text-gray-700 mb-2">审核记录</h4>
                <div class="space-y-2">
                  <div v-for="r in detail.review_records" :key="r.id" class="border-b border-gray-100 pb-2 mb-2">
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
