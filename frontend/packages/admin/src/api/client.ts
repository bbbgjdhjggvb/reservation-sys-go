import { useAdminStore } from '@/stores/admin'
import type { ApiResponse } from '@reservation/shared'
import type { AdminLoginResp, AdminInfoResp, OrderListResp, OrderDetailResp } from '../types'

const BASE = '/api/admin'

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const admin = useAdminStore()

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  }

  if (admin.token) {
    headers['Authorization'] = `Bearer ${admin.token}`
  }

  const res = await fetch(`${BASE}${endpoint}`, { ...options, headers })

  if (res.status === 401) {
    admin.logout()
    throw new Error('Unauthorized')
  }

  const data: ApiResponse<T> = await res.json()

  if (data.code !== 200) {
    throw new Error(data.msg || '请求失败')
  }

  return data.data as T
}

export const adminApi = {
  login(username: string, password: string) {
    return request<AdminLoginResp>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
  },

  getAdminInfo() {
    return request<AdminInfoResp>('/admin/info')
  },

  getOrders(params: { page: number; page_size?: number; status?: number | string }) {
    const searchParams = new URLSearchParams()
    searchParams.set('page', String(params.page))
    searchParams.set('page_size', String(params.page_size || 20))
    if (params.status !== undefined && params.status !== -1) {
      const statuses = String(params.status).split(',')
      statuses.forEach(s => searchParams.append('status', s))
    }
    return request<OrderListResp>(`/orders?${searchParams.toString()}`)
  },

  getOrderDetail(id: number) {
    return request<OrderDetailResp>(`/orders/${id}`)
  },

  reviewLevel1(orderId: number, action: number, comment?: string) {
    return request<null>(`/review/level1/${orderId}`, {
      method: 'POST',
      body: JSON.stringify({ action, comment: comment || '' }),
    })
  },

  reviewLevel2(orderId: number, action: number, comment?: string) {
    return request<null>(`/review/level2/${orderId}`, {
      method: 'POST',
      body: JSON.stringify({ action, comment: comment || '' }),
    })
  },

  setPassword(orderId: number, slotId: number, password: string) {
    return request<null>(`/review/level1/${orderId}/slots/${slotId}/password`, {
      method: 'PUT',
      body: JSON.stringify({ password }),
    })
  },

  notify(orderId: number) {
    return request<null>(`/review/level1/${orderId}/notify`, { method: 'POST' })
  },

  rejectNotify(orderId: number, reason: string) {
    return request<null>(`/review/level1/${orderId}/reject-notify`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    })
  },
}
