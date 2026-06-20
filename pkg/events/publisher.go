package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// ========== EventPublisher ==========

// EventPublisher 事件发布器，将订单事件序列化为 JSON 后发布到 Redis Pub/Sub 频道。
// 各服务在业务操作（提交预约、审核、取消等）成功后调用 Publish() 通知其他服务。
//
// 设计要点:
//   - Publish() 失败只记录日志、返回 error，不阻断业务流程
//     （预约提交、审核操作仍正常完成，事件丢失仅影响实时性）
//   - 前端降级轮询（15s/10s）作为最终兜底，即使 Publish 失败也能同步数据
type EventPublisher struct {
	client  *redis.Client
	channel string
}

// NewEventPublisher 创建事件发布器实例。
//
// 参数:
//   - client: 已连接的 Redis 客户端（通过 platform.InitRedis() 获取）
//
// 返回值:
//   - *EventPublisher: 发布器实例
func NewEventPublisher(client *redis.Client) *EventPublisher {
	return &EventPublisher{
		client:  client,
		channel: RedisPubSubChannel,
	}
}

// Publish 将事件发布到 Redis Pub/Sub 频道。
//
// 参数:
//   - event: 订单事件实例（Type 和 OrderID 必填）
//
// 返回值:
//   - nil: 发布成功
//   - error: JSON 序列化失败或 Redis 发布失败时返回错误
//
// 流程:
//  1. 补全 Timestamp（若为零值，设为当前 UTC 时间）
//  2. json.Marshal(event) 序列化
//  3. client.Publish(ctx, channel, jsonBytes).Err() 发布到 Redis
//
// 注意:
//   - 调用方应在业务操作成功后才调用此方法
//   - 返回 error 时调用方应记录日志但不中断业务流程
func (p *EventPublisher) Publish(event *OrderEvent) error {
	// 补全 Timestamp 字段：若未设置，使用当前 UTC 时间
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// 序列化为 JSON
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败: %w", err)
	}

	// 发布到 Redis Pub/Sub 频道
	// 使用 5s 超时 context，防止 Redis 不可用时阻塞过久
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.client.Publish(ctx, p.channel, payload).Err(); err != nil {
		return fmt.Errorf("发布事件到 Redis 失败: %w", err)
	}

	log.Printf("[events] 事件已发布: type=%s orderID=%d channel=%s", event.Type, event.OrderID, p.channel)
	return nil
}

// ========== 全局单例 ==========

// 全局发布器单例。
// 各服务在 main.go 中调用 InitPublisher() 初始化，
// 业务层通过 GetPublisher() 获取。
var (
	globalPublisher *EventPublisher
	publisherOnce   sync.Once
)

// InitPublisher 初始化全局事件发布器单例。
// 应在服务 main.go 中 Redis 初始化之后调用，且只调用一次。
//
// 参数:
//   - client: 已连接的 Redis 客户端
//
// 注意:
//   - 重复调用会被 sync.Once 忽略（安全）
//   - 若服务不需要发布事件（如 Gateway），可不调用此函数
func InitPublisher(client *redis.Client) {
	publisherOnce.Do(func() {
		globalPublisher = NewEventPublisher(client)
		log.Printf("[events] 全局事件发布器已初始化，频道: %s", RedisPubSubChannel)
	})
}

// GetPublisher 获取全局事件发布器。
// 若未调用 InitPublisher() 初始化，会 panic（与 reservationdb.GetRepository() 行为一致）。
//
// 返回值:
//   - *EventPublisher: 全局发布器实例
func GetPublisher() *EventPublisher {
	if globalPublisher == nil {
		panic("events: 全局发布器未初始化，请先调用 InitPublisher()")
	}
	return globalPublisher
}

// getPublisherSafe 安全获取全局事件发布器。
// 若未初始化返回 nil，不 panic。仅供便捷发布函数内部使用。
// 便捷发布函数是可选增强功能，未初始化时应安全降级而非阻断业务。
func getPublisherSafe() *EventPublisher {
	return globalPublisher
}

// ========== 便捷发布函数 ==========

// PublishOrderCreated 发布"新预约提交"事件的便捷函数。
// 若 Publisher 未初始化（如测试环境），安全降级返回 nil，不阻断业务。
//
// 参数:
//   - orderID: 新创建的订单 ID
//
// 返回值:
//   - error: 发布失败时返回错误（调用方应记录日志但不中断业务）
func PublishOrderCreated(orderID uint) error {
	pub := getPublisherSafe()
	if pub == nil {
		return nil
	}
	return pub.Publish(&OrderEvent{
		Type:    EventTypeOrderCreated,
		OrderID: orderID,
	})
}

// PublishOrderCancelled 发布"预约取消"事件的便捷函数。
// 若 Publisher 未初始化（如测试环境），安全降级返回 nil，不阻断业务。
//
// 参数:
//   - orderID: 被取消的订单 ID
//
// 返回值:
//   - error: 发布失败时返回错误（调用方应记录日志但不中断业务）
func PublishOrderCancelled(orderID uint) error {
	pub := getPublisherSafe()
	if pub == nil {
		return nil
	}
	return pub.Publish(&OrderEvent{
		Type:    EventTypeOrderCancelled,
		OrderID: orderID,
	})
}

// PublishOrderReviewed 发布"审核操作"事件的便捷函数。
// 若 Publisher 未初始化（如测试环境），安全降级返回 nil，不阻断业务。
//
// 参数:
//   - orderID: 被审核的订单 ID
//
// 返回值:
//   - error: 发布失败时返回错误（调用方应记录日志但不中断业务）
func PublishOrderReviewed(orderID uint) error {
	pub := getPublisherSafe()
	if pub == nil {
		return nil
	}
	return pub.Publish(&OrderEvent{
		Type:    EventTypeOrderReviewed,
		OrderID: orderID,
	})
}

// PublishSlotUpdated 发布"时段更新"事件的便捷函数。
// 若 Publisher 未初始化（如测试环境），安全降级返回 nil，不阻断业务。
//
// 参数:
//   - orderID: 关联的订单 ID（时段更新时通过 orderID 关联）
//
// 返回值:
//   - error: 发布失败时返回错误（调用方应记录日志但不中断业务）
func PublishSlotUpdated(orderID uint) error {
	pub := getPublisherSafe()
	if pub == nil {
		return nil
	}
	return pub.Publish(&OrderEvent{
		Type:    EventTypeSlotUpdated,
		OrderID: orderID,
	})
}
