import { ref } from 'vue'
import { adminApi } from '@/api/client'
import type { OrderResp } from '@reservation/shared'

export function useAdminOrders() {
  const orders = ref<OrderResp[]>([])
  const total = ref(0)
  const currentPage = ref(1)
  const currentStatus = ref<number | string>(-1)
  const loading = ref(false)
  const pageSize = 20

  async function fetchOrders() {
    loading.value = true
    try {
      const data = await adminApi.getOrders({
        page: currentPage.value,
        page_size: pageSize,
        status: currentStatus.value,
      })
      orders.value = data.list || []
      total.value = data.total
    } catch {
      orders.value = []
      total.value = 0
    } finally {
      loading.value = false
    }
  }

  function setStatus(status: number | string) {
    currentStatus.value = status
    currentPage.value = 1
    fetchOrders()
  }

  function goToPage(page: number) {
    currentPage.value = page
    fetchOrders()
  }

  return {
    orders, total, currentPage, currentStatus, loading, pageSize,
    fetchOrders, setStatus, goToPage,
  }
}
