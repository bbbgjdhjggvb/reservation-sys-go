# 前端代码注释规范

## 总体要求
1. 说明组件/函数的功能和业务目标
2. 说明参数、返回值、事件、state、props
3. 说明依赖关系、外部接口、权限边界
4. 说明可重复使用的 composable/store 的行为和副作用

## 模块职责

### shared
- 定义共享类型、状态码映射、公共工具函数
- 负责前后端状态同步，避免硬编码状态值
- 核心对象：`ORDER_STATUS_MAP`、API 类型、公共 DTO

### reservation
- 负责用户侧预约流程
- 主要页面：预约页、我的预约、订单详情、登录/授权
- 业务职责：表单校验、预约提交、订单查询、取消逻辑

### admin
- 负责管理员审核流程
- 主要页面：审核列表、审核详情、登录/权限管理
- 业务职责：审核操作、状态过滤、列表刷新、通知交互

## 组件层注释

### 页面组件
- 说明页面的业务场景、展示目的、关键交互
- 说明是否依赖 store 或 composable
- 说明是否包含权限保护

示例：
```ts
// ReservationView 用户预约页面。
//
// 功能：展示预约表单、可选时间段、历史订单入口
// 依赖：useReservationForm、auth store、API client
// 行为：表单提交时触发预约请求，结果成功后刷新订单列表
export default defineComponent({ ... })
```

### 业务组件
- 说明 props 和 emits
- 说明展示条件、loading 状态、错误信息处理
- 说明是否支持复用、是否仅在某个场景使用

示例：
```ts
// TimeSlotPicker 可选择单个时段或多个时段。
//
// props:
//   - slots: Slot[]
//   - disabled: boolean
// emits:
//   - update:selectedSlots
// 行为：选择时段时自动合并连续时间段，禁用已预约时段
const TimeSlotPicker = defineComponent({ ... })
```

## Composable / Hooks 说明

### 说明内容
- Composable 目的
- 输入参数
- 返回值结构
- 副作用，例如请求、订阅、路由跳转
- 是否依赖外部 store 或 router

### 典型注释模板
```ts
// useReservationForm 提供预约表单数据和提交逻辑。
//
// 参数:
//   - initialSlots: Slot[]
// 返回:
//   - form: Ref<ReservationForm>
//   - errors: Ref<Record<string, string>>
//   - submit: () => Promise<void>
//
// 流程:
//   1. 校验时段数量和必填字段
//   2. 将同一天连续时段合并为一个请求项
//   3. 调用 API client 提交预约
//   4. 处理接口错误并设置提示
export function useReservationForm(initialSlots: Slot[]) {
  ...
}
```

## Store 说明

### Pinia Store
- 说明 store 的职责和作用域
- 说明 state / getters / actions 的业务意义
- 说明 action 的输入、输出、异步请求行为

### 示例
```ts
// useReservationStore 管理用户预约数据。
//
// state:
//   - orders: ReservationOrder[]
//   - loading: boolean
// actions:
//   - fetchOrders(page: number)
//   - cancelOrder(orderId: string)
//
// 业务说明: fetchOrders 仅负责加载列表，不负责页面展示逻辑
export const useReservationStore = defineStore('reservation', {
  state: () => ({ ... }),
  actions: { ... }
})
```

## API 接口定义

### 请求函数注释
- 说明请求目的、输入 DTO、输出 DTO
- 说明错误处理策略
- 说明 token / auth header 的要求

### 示例
```ts
// fetchUserOrders 获取当前用户的预约列表。
//
// 参数:
//   - page: number
// 返回:
//   - Promise<ReservationOrderResponse>
//
// 说明: 该接口调用时需要携带用户 Bearer token。
export function fetchUserOrders(page: number) {
  return apiClient.get('/reservation/orders', { params: { page } })
}
```

## 路由与权限

- 说明前端路由分层：用户端 `/reservation/`、管理员端 `/admin/`
- 说明路由守卫职责：登录校验、权限校验、拦截未授权访问
- 说明登录状态如何存储（cookie / localStorage / pinia）

## 共享类型与状态映射

- 明确 `shared` 中类型定义为前端 DTO 权威来源
- 说明状态码映射必须与后端一致
- 说明常量、枚举、字段名和后端字段映射关系

### 说明示例
```ts
// ORDER_STATUS_MAP 定义订单状态中文显示。
// 该映射必须与后端 `pkg/reservationdb/model.go` 的状态码保持一致。
export const ORDER_STATUS_MAP = {
  1: '待一级审核',
  2: '待二级审核',
  3: '一级审核拒绝',
  4: '二级审核拒绝',
  5: '审核通过',
  6: '已取消',
  7: '已完成'
}
```

## 注释书写规范建议

- 组件注释应语义清晰、避免实现细节
- composable/store 注释应强调副作用和异步逻辑
- API 注释应说明身份验证和异常场景
- 推荐在导出函数/组件前添加注释，保持与后端说明风格一致
