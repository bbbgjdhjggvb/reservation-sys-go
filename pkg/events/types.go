// Package events 提供跨服务的事件定义与发布能力。
// 基于 Redis Pub/Sub 实现 Reservation 和 Admin 服务之间的事件通知，
// 使 SSE 实时推送能够感知到另一个服务产生的订单变更。
//
// 架构分层:
//   - 本包只定义事件类型和发布器，不包含订阅逻辑
//   - 订阅逻辑在 pkg/sse/subscriber.go 中实现
package events

import "time"

// ========== 事件类型常量 ==========

const (
	// EventTypeOrderCreated 新预约提交事件。
	// 触发时机：ReservationService.Submit() 成功创建订单后
	// 消费方：
	//   - Reservation SSE → 刷新其他用户的日历已占用时段
	//   - Admin SSE → 刷新管理员订单列表
	EventTypeOrderCreated = "order_created"

	// EventTypeOrderCancelled 预约取消事件。
	// 触发时机：ReservationService.Cancel() 成功取消订单后
	// 消费方：
	//   - Reservation SSE → 刷新其他用户的日历（该时段变为可用）
	//   - Admin SSE → 刷新管理员订单列表（状态变为已取消）
	EventTypeOrderCancelled = "order_cancelled"

	// EventTypeOrderReviewed 审核操作事件（通过或拒绝）。
	// 触发时机：ReviewService.Level1Review() 或 Level2Review() 执行后
	// 消费方：
	//   - Reservation SSE → 刷新用户的"我的预约"状态
	//   - Admin SSE → 刷新其他管理员的订单列表
	EventTypeOrderReviewed = "order_reviewed"

	// EventTypeSlotUpdated 时段信息更新事件（如设置门锁密码）。
	// 触发时机：ReviewService.SetPassword() 执行后
	// 消费方：
	//   - Admin SSE → 刷新其他管理员的订单详情
	EventTypeSlotUpdated = "slot_updated"
)

// ========== 事件数据结构 ==========

// OrderEvent 订单事件，通过 Redis Pub/Sub 在服务间传递。
// 仅携带事件类型和关联 ID，不含业务详情（安全性考量：SSE 推送不含敏感数据）。
type OrderEvent struct {
	// Type 事件类型，取值为上方 EventType 常量之一
	Type string `json:"type"`

	// OrderID 关联的订单主键 ID
	// 客户端收到后可作为日志参考，实际数据仍需通过 REST API 拉取
	OrderID uint `json:"order_id"`

	// Timestamp 事件发生时间（服务端生成，UTC）
	Timestamp time.Time `json:"timestamp"`

	// Payload 可选的附加数据
	// 预留字段，当前不使用。未来可扩展为 map[string]any
	// 注意：推送内容不含用户隐私数据
	Payload any `json:"payload,omitempty"`
}

// RedisPubSubChannel Redis Pub/Sub 频道名称常量。
// Reservation 和 Admin 服务均订阅此频道，事件发布者也向此频道发布。
const RedisPubSubChannel = "order_events"
