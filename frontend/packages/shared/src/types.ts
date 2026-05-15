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
  0: '待审核',
  1: '已通过',
  2: '已拒绝',
  3: '已完成',
  4: '已取消',
  5: '审核中',
  6: '一级审核驳回',
  7: '二级审核驳回',
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
export const ALUMNI_OPTIONS = [
  '计算机与软件学院校友会',
  '电子与信息工程学院校友会',
  '土木与交通工程学院校友会',
  '建筑与城市规划学院校友会',
  '机电与控制工程学院校友会',
  '材料学院校友会',
  '化学与环境工程学院校友会',
  '生命与海洋科学学院校友会',
  '物理与光电工程学院校友会',
  '数学科学学院校友会',
  '经济学院校友会',
  '管理学院校友会',
  '法学院校友会',
  '外国语学院校友会',
  '传播学院校友会',
  '艺术学部校友会',
  '医学部校友会',
  '体育学院校友会',
  '国际交流学院校友会',
  '高等研究院校友会',
  '金融科技学院校友会',
  '继续教育学院校友会',
  '国际交流与合作部校友会',
  '研究生院校友会',
]
