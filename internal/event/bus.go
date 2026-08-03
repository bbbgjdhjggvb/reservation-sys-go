// Package event 提供进程内事件总线。
// 替代原先的 Redis Pub/Sub，在单进程内以 channel 方式传递订单变更事件。
// SSE Hub 通过 Subscribe() 获取事件通知，各模块 Service 通过 Publish() 发布事件。
package event

import "sync"

// ========== 事件类型常量 ==========

// OrderEventType 订单事件类型
type OrderEventType string

const (
	OrderCreated   OrderEventType = "order_created"
	OrderCancelled OrderEventType = "order_cancelled"
	OrderReviewed  OrderEventType = "order_reviewed"
	SlotUpdated    OrderEventType = "slot_updated"
)

// ========== 事件数据结构 ==========

// OrderEvent 订单事件，通过进程内 channel 在模块间传递。
// 仅携带事件类型和关联 ID，不含业务详情。
type OrderEvent struct {
	Type      OrderEventType `json:"type"`
	OrderID   uint           `json:"order_id"`
	Timestamp int64          `json:"timestamp"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// ========== 事件总线 ==========

// Bus 进程内事件总线。
// 发布者调用 Publish() 推送事件，订阅者通过 Subscribe() 获取只读 channel。
// 慢消费者会被跳过（非阻塞写入），避免阻塞发布者。
type Bus struct {
	subscribers []chan OrderEvent
	mu          sync.RWMutex
}

// NewBus 创建事件总线实例
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe 订阅事件，返回只读通道。
// 调用方从此通道读取事件，生命周期与总线相同。
// 通道缓冲为 64，避免偶发慢消费导致丢失。
func (b *Bus) Subscribe() <-chan OrderEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan OrderEvent, 64)
	b.subscribers = append(b.subscribers, ch)
	return ch
}

// Publish 向所有订阅者发布事件。
// 使用非阻塞写入（select + default），慢消费者会被跳过。
func (b *Bus) Publish(event OrderEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			// 慢消费者跳过，避免阻塞发布者
		}
	}
}
