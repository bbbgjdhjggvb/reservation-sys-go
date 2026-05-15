import { useAuthStore } from '@/stores/auth'
import type { ApiResponse, OccupiedSlot, OrderResp, SubmitReq } from '@reservation/shared'

const BASE = '/api/reservation'

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const auth = useAuthStore()
  if (!auth.token) {
    throw new Error('NoAuthToken')
  }

  const res = await fetch(`${BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${auth.token}`,
      ...(options.headers as Record<string, string> || {}),
    },
  })

  if (res.status === 401) {
    auth.logout()
    throw new Error('Unauthorized')
  }

  const data: ApiResponse<T> = await res.json()

  if (data.code !== 200) {
    throw new Error(data.msg || '请求失败')
  }

  return data.data as T
}

export const api = {
  getOccupiedSlots(date: string) {
    return request<OccupiedSlot[]>(`/reservation/occupied?date=${encodeURIComponent(date)}`)
  },

  submit(body: SubmitReq) {
    return request<OrderResp>('/reservation/submit', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },

  getMyOrders() {
    return request<OrderResp[]>('/reservation/my')
  },

  cancelOrder(id: number) {
    return request<null>(`/reservation/${id}`, { method: 'DELETE' })
  },
}
