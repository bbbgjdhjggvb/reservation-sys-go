# 前端测试规范

## 总体要求
1. 说明测试的是哪个文件、哪个组件或哪个函数
2. 说明该功能或业务目标是什么
3. 说明测试场景和预期结果
4. 使用清晰的 `describe` / `it` 或 `test` 语义

## 测试类别

### 单元测试
- 组件内部逻辑
- composable / hooks
- store action / getter
- 工具函数、状态映射

### 组件测试
- 渲染内容是否符合 props / state
- 交互行为是否正确
- 异常提示是否显示
- loading / 空状态是否正确展示

### 集成测试
- 组件与 store、router、API client 的协同行为
- 业务流程是否完整
- 页面跳转、状态刷新、表单提交链路

## 常见测试目标

1. `ReservationForm.vue` 是否校验时段数量、必填字段
2. `useReservationForm` 是否正确合并连续时段并触发提交
3. `useReservationStore` 是否调用 API 并更新 `orders`
4. `AdminReviewView` 是否根据不同状态显示正确按钮
5. API 客户端请求失败时，页面是否显示错误提示
6. 登录失效后，是否触发路由重定向

## 场景说明

### 表单校验
- slots 为空时，显示“请至少选择一个时段”
- slots 超过 4 个时，显示“最多选择4个时段”
- 必填字段缺失时，禁用提交或显示校验错误

### API 成功
- 提交预约时，`submitReservation` 成功后调用成功提示
- 管理员审核通过后，列表状态更新

### API 失败
- 服务器返回 400/401/500 时，组件展示错误提示
- 网络异常时，显示统一错误提示

### 路由权限
- 未登录访问受限页面时，跳转登录页
- 普通用户访问管理员页面时，显示无权限或路由重定向

### 状态映射
- 订单状态码 `1-7` 是否映射到正确文本
- 列表显示的状态标签是否与后端一致

## 测试注释示例

```ts
// 测试 src/packages/reservation/src/composables/useReservationForm.ts
// 功能：校验时段数量并处理提交逻辑
describe('useReservationForm', () => {
  it('should show validation error when more than 4 slots selected', async () => {
    ...
  })
})
```

```ts
// 测试 src/packages/admin/src/views/AdminReviewView.vue
// 功能：根据订单状态显示审核按钮
describe('AdminReviewView', () => {
  it('shows approve button when status is pending level 1', () => {
    ...
  })
})
```

## 测试工具与约定

- 推荐使用 `Vitest`
- 组件测试推荐使用 `@testing-library/vue`
- Store 测试可使用 `pinia` 的 `setActivePinia`
- API mock 可使用 `vi.mock` 或 `msw`
- 不建议把快照测试作为唯一手段，关键逻辑应依赖行为断言

## 额外说明

- 组件测试应关注业务行为，不仅是 DOM 结构
- composable/store 测试应侧重状态变化与副作用
- API 失败场景要覆盖常见错误码和用户提示
