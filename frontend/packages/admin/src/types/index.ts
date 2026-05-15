import type { OrderResp } from '@reservation/shared'

export interface AdminLoginResp {
  token: string
  username: string
  real_name: string
  role: number
  role_text: string
}

export interface AdminInfoResp {
  id: number
  username: string
  real_name: string
  role: number
  role_text: string
}

export interface ReviewRecordResp {
  id: number
  reviewer_name: string
  reviewer_role: number
  role_text: string
  action: number
  action_text: string
  comment: string
  created_at: string
}

export interface OrderDetailResp {
  order: OrderResp
  review_records: ReviewRecordResp[]
}

export interface OrderListResp {
  list: OrderResp[]
  total: number
  page: number
  page_size: number
}
