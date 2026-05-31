// ========== API Response Wrapper ==========
export interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data: T
}

// ========== Time Slots ==========
export interface TimeSlotReq {
  start_time: string // "2026-01-01 08:00:00"
  end_time: string // "2026-01-01 10:00:00"
}

export interface OccupiedSlot {
  start_time: string
  end_time: string
  status: 'pending' | 'approved'
}

// ========== Order ==========
export interface SlotResp {
  id: number
  start_time: string
  end_time: string
  status: number
  status_text: string
  password?: string
}

export interface OrderResp {
  id: number
  order_no: string
  applicant_name: string
  alumni_association: string
  year: number
  major: string
  reason: string
  phone: string
  total_slots: number
  status: number
  status_text: string
  created_at: string
  slots: SlotResp[]
}

export interface SubmitReq {
  applicant_name: string
  alumni_association: string
  year: number
  major: string
  reason: string
  phone: string
  slots: TimeSlotReq[]
}

// ========== Selected Slot (frontend-only) ==========
export interface SelectedSlot {
  date: string // "2026-01-15"
  startTime: string // "08:00"
  endTime: string // "10:00"
}

// ========== Order Status ==========
export const ORDER_STATUS_MAP: Record<number, string> = {
  1: '等待一级审核',
  2: '等待二级审核',
  3: '一级审核拒绝',
  4: '二级审核拒绝',
  5: '审核通过',
  6: '订单已经取消',
  7: '订单已经完成',
}

// ========== Time Slot Definitions ==========
export const TIME_SLOTS = [
  { label: '8:00-10:00', start: '08:00', end: '10:00' },
  { label: '10:00-12:00', start: '10:00', end: '12:00' },
  { label: '13:00-15:00', start: '13:00', end: '15:00' },
  { label: '15:00-17:00', start: '15:00', end: '17:00' },
  { label: '18:00-20:00', start: '18:00', end: '20:00' },
] as const

export const WEEKDAYS = ['一', '二', '三', '四', '五', '六', '日'] as const

export const MAX_SLOTS = 4

// ========== Alumni Options ==========
export const ALUMNI_GROUPS: Record<'domestic' | 'foreign' | 'industry', readonly string[]> = {
  domestic: [
    '北京地区校友分会',
    '长三角校友分会',
    '广州校友分会',
    '吴川校友分会',
    '普宁校友分会',
    '大潮阳校友分会',
    '饶平校友分会',
    '紫金校友分会',
    '兴宁校友分会',
    '惠来校友分会',
    '四川校友分会',
    '西藏校友分会',
    '香港校友联谊会',
    '东莞校友分会',
    '河南校友分会',
    '喀什校友联谊会',
  ],
  foreign: [
    '美国加州校友分会',
    '澳洲校友联谊会',
    '新西兰校友联谊会',
    '加拿大校友联谊会',
    '多伦多校友联谊会',
  ],
  industry: [
    '深圳大学MBA校友分会',
    '校友高尔夫俱乐部',
    '校友记者协会',
    '创业与投资联谊会',
    '校友房地产联谊会',
  ],
} as const

export const ALUMNI_OPTIONS: readonly string[] = [
  ...ALUMNI_GROUPS.domestic,
  ...ALUMNI_GROUPS.foreign,
  ...ALUMNI_GROUPS.industry,
]
