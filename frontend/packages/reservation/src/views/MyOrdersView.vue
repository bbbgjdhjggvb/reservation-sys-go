<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import { api } from '@/api/client'
import ToastNotification from '@/components/ToastNotification.vue'
import OrderCard from '@/components/OrderCard.vue'
import type { OrderResp } from '@reservation/shared'

const auth = useAuthStore()
const { showToast } = useToast()
const orders = ref<OrderResp[]>([])
const loading = ref(true)
const error = ref('')

const showCancelModal = ref(false)
const cancelOrderId = ref<number | null>(null)
const cancelOrderNo = ref('')
const cancelling = ref(false)

async function loadOrders() {
  loading.value = true
  error.value = ''
  try {
    const data = await api.getMyOrders()
    orders.value = (data || []).sort((a, b) => b.created_at.localeCompare(a.created_at))
  } catch {
    error.value = '加载失败，请重试'
  } finally {
    loading.value = false
  }
}

function openCancelModal(id: number) {
  const order = orders.value.find(o => o.id === id)
  if (!order) return
  cancelOrderId.value = id
  cancelOrderNo.value = order.order_no
  showCancelModal.value = true
}

function closeCancelModal() {
  showCancelModal.value = false
  cancelOrderId.value = null
  cancelOrderNo.value = ''
}

async function doCancel() {
  if (!cancelOrderId.value) return
  cancelling.value = true
  try {
    await api.cancelOrder(cancelOrderId.value)
    showToast('取消成功', 'success')
  } catch (e: any) {
    showToast(e.message || '取消失败', 'error')
  } finally {
    cancelling.value = false
    closeCancelModal()
    await loadOrders()
  }
}

onMounted(loadOrders)
</script>

<template>
  <div class="min-h-screen">
    <header class="sticky top-0 z-40 bg-white shadow-sm">
      <div class="max-w-4xl mx-auto px-4 py-3 flex items-center justify-between">
        <h1 class="text-lg font-bold text-primary-500">我的预约</h1>
      </div>
    </header>

    <main class="max-w-4xl mx-auto px-4 py-6">
      <!-- Token Error -->
      <div v-if="!auth.isAuthenticated" class="bg-red-50 border border-red-200 rounded-lg p-4 text-center text-red-600">
        未授权访问，请从微信服务号进入预约界面
      </div>

      <template v-else>
        <!-- Toolbar -->
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-base font-medium text-gray-700">预约记录</h2>
          <button class="flex items-center gap-1 text-sm text-primary-500 hover:text-primary-600 transition" aria-label="刷新预约列表" @click="loadOrders">
            <svg class="w-4 h-4" :class="{ 'animate-spin': loading }" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            刷新
          </button>
        </div>

        <!-- Loading -->
        <div v-if="loading" class="space-y-3">
          <div v-for="i in 2" :key="i" class="bg-white rounded-xl p-4 animate-pulse">
            <div class="h-4 bg-gray-200 rounded w-1/3 mb-3" />
            <div class="h-3 bg-gray-200 rounded w-1/2 mb-2" />
            <div class="h-3 bg-gray-200 rounded w-2/3" />
          </div>
        </div>

        <!-- Error -->
        <div v-else-if="error" class="bg-white rounded-lg shadow p-6 text-center">
          <p class="text-gray-500 mb-2">{{ error }}</p>
          <button class="text-primary-500 text-sm font-medium" @click="loadOrders">点击重试</button>
        </div>

        <!-- Empty -->
        <div v-else-if="orders.length === 0" class="bg-white rounded-lg shadow p-12 text-center">
          <svg class="w-16 h-16 text-gray-300 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
          </svg>
          <p class="text-gray-400">暂无预约记录</p>
        </div>

        <!-- Order List -->
        <div v-else class="space-y-3">
          <OrderCard
            v-for="order in orders"
            :key="order.id"
            :order="order"
            :show-cancel="true"
            @cancel="openCancelModal"
          />
        </div>
      </template>

      <!-- Cancel Modal -->
      <Teleport to="body">
        <Transition name="modal">
          <div v-if="showCancelModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black bg-opacity-50 overscroll-contain" @click.self="closeCancelModal">
            <div class="bg-white rounded-xl shadow-xl w-full max-w-sm p-6">
              <h3 class="text-lg font-semibold text-gray-800 mb-2">确认取消预约</h3>
              <p class="text-sm text-gray-500 mb-4">
                确定要取消预约 <span class="font-mono text-gray-700">{{ cancelOrderNo }}</span> 吗？取消后不可恢复。
              </p>
              <div class="flex gap-3">
                <button class="flex-1 px-4 py-2.5 rounded-lg border border-gray-200 text-gray-600 font-medium hover:bg-gray-50 transition" @click="closeCancelModal">
                  再想想
                </button>
                <button
                  :disabled="cancelling"
                  class="flex-1 px-4 py-2.5 rounded-lg bg-red-500 text-white font-medium hover:bg-red-600 disabled:opacity-60 transition flex items-center justify-center gap-2"
                  @click="doCancel"
                >
                  <svg v-if="cancelling" class="animate-spin w-4 h-4" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  {{ cancelling ? '取消中…' : '确认取消' }}
                </button>
              </div>
              <p class="mt-3 text-xs text-gray-400 text-center">
                请勿频繁操作
              </p>
            </div>
          </div>
        </Transition>
      </Teleport>
    </main>

    <ToastNotification />
  </div>
</template>

<style scoped>
.modal-enter-active { animation: modalIn 0.2s ease; }
.modal-leave-active { animation: modalIn 0.2s ease reverse; }
@keyframes modalIn {
  from { opacity: 0; transform: scale(1) translateY(10px); }
  to   { opacity: 1; transform: scale(1) translateY(0); }
}
</style>
