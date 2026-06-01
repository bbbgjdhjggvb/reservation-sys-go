import { useAuthStore } from '@/stores/auth'
import type { ApiResponse, OccupiedSlot, OrderResp, SubmitReq } from '@reservation/shared'
import { RATE_LIMIT_CODE } from '@reservation/shared'

const BASE = '/api/reservation'

/** 限流提示文案 */
function getRateLimitHint(): string {
  return '请勿频繁操作'
}

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

  // HTTP 429 限流 → 给出友好提示
  if (res.status === RATE_LIMIT_CODE) {
    throw new Error(getRateLimitHint())
  }

  const data: ApiResponse<T> = await res.json()

  if (data.code !== RATE_LIMIT_CODE && data.code !== 200) {
    throw new Error(data.msg || '请求失败')
  }

  // 兜底：业务 code 为 429（部分中间件可能返回此 code 但 HTTP 状态码非 429）
  if (data.code === RATE_LIMIT_CODE) {
    throw new Error(getRateLimitHint())
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
