<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { useAdminOrders } from '@/composables/useAdminOrders'
import { adminApi } from '@/api/client'
import AdminHeader from '@/components/AdminHeader.vue'
import StatusTabs from '@/components/StatusTabs.vue'
import AdminOrderCard from '@/components/AdminOrderCard.vue'
import Pagination from '@/components/Pagination.vue'
import OrderDetailModal from '@/components/OrderDetailModal.vue'
import ReviewModal from '@/components/ReviewModal.vue'

const router = useRouter()
const admin = useAdminStore()
const store = useAdminOrders()
const { orders, total, currentPage, currentStatus, loading } = store

const detailOrderId = ref<number | null>(null)
const showReviewModal = ref(false)
const reviewAction = ref(1)
const reviewActionLabel = ref('')
const reviewOrderId = ref(0)

const toastMsg = ref('')
const toastError = ref(false)
let toastTimer: ReturnType<typeof setTimeout> | null = null

function showToast(msg: string, isError = false) {
  toastMsg.value = msg
  toastError.value = isError
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toastMsg.value = '' }, 2500)
}

onMounted(async () => {
  if (!admin.isAuthenticated) {
    router.replace('/admin/login')
    return
  }
  try {
    await admin.fetchInfo()
    store.fetchOrders()
  } catch {
    router.replace('/admin/login')
  }
})

function openDetail(id: number) {
  detailOrderId.value = id
}

function closeDetail() {
  detailOrderId.value = null
}

function openReview(id: number, action: number) {
  reviewOrderId.value = id
  reviewAction.value = action
  reviewActionLabel.value = action === 1 ? '通过' : '拒绝'
  showReviewModal.value = true
}

async function confirmReview(comment: string) {
  showReviewModal.value = false
  try {
    if (admin.info?.role === 1) {
      await adminApi.reviewLevel1(reviewOrderId.value, reviewAction.value, comment)
    } else {
      await adminApi.reviewLevel2(reviewOrderId.value, reviewAction.value, comment)
    }
    showToast(`审核${reviewActionLabel.value}成功`)
    await store.fetchOrders()
  } catch (e: any) {
    showToast(e.message || '审核失败', true)
  }
}

async function handleNotify(orderId: number) {
  try {
    await adminApi.notify(orderId)
    showToast('通知已发送')
    await store.fetchOrders()
  } catch (e: any) {
    showToast(e.message || '发送失败', true)
  }
}

function handleDetailUpdated() {
  closeDetail()
  store.fetchOrders()
}
</script>

<template>
  <div class="min-h-screen bg-gray-50">
    <AdminHeader />

    <main class="max-w-6xl mx-auto px-4 py-5">
      <StatusTabs
        :admin-role="admin.info?.role || 0"
        :current-status="currentStatus"
        @update:current-status="store.setStatus"
      />

      <!-- Loading -->
      <div v-if="loading" class="mt-4 text-center py-12 text-gray-400">
        加载中…
      </div>

      <!-- Empty -->
      <div v-else-if="orders.length === 0" class="mt-4 text-center py-12 text-gray-400">
        暂无数据
      </div>

      <!-- Orders -->
      <div v-else class="mt-4 space-y-3">
        <AdminOrderCard
          v-for="order in orders"
          :key="order.id"
          :order="order"
          :admin-role="admin.info?.role || 0"
          @show-detail="openDetail"
          @review="(id, action) => openReview(id, action)"
          @notify-user="handleNotify"
        />
      </div>

      <Pagination
        :current-page="currentPage"
        :total="total"
        :page-size="store.pageSize"
        @page-change="store.goToPage"
      />
    </main>

    <OrderDetailModal
      :order-id="detailOrderId"
      :admin-role="admin.info?.role || 0"
      @close="closeDetail"
      @updated="handleDetailUpdated"
    />

    <ReviewModal
      :visible="showReviewModal"
      :action="reviewAction"
      :action-label="reviewActionLabel"
      @close="showReviewModal = false"
      @confirm="confirmReview"
    />

    <!-- Toast -->
    <div
      v-if="toastMsg"
      class="fixed top-4 left-1/2 -translate-x-1/2 z-50 px-6 py-3 rounded-lg shadow-lg text-sm font-medium text-white transition-all"
      :class="toastError ? 'bg-red-500' : 'bg-primary-500'"
      style="animation: slideDown 0.3s ease;"
    >
      {{ toastMsg }}
    </div>
  </div>
</template>

<style scoped>
@keyframes slideDown {
  from { opacity: 0; transform: translate(-50%, -10px); }
  to   { opacity: 1; transform: translate(-50%, 0); }
}
</style>
