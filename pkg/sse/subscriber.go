package sse

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"reservation-sys/pkg/events"

	"github.com/go-redis/redis/v8"
)

// ========== EventSubscriber ==========

// EventSubscriber Redis Pub/Sub 订阅器。
// 负责监听 Redis 频道中的事件消息，并通过内部 channel 转发给 SSEHub。
// SSEHub 的 forwardRedisMessages() 协程从 Channel() 读取消息并广播给客户端。
//
// 设计要点:
//   - 内部协程在 Subscribe 断连后持续重试，Redis 恢复后自动重建订阅
//   - 使用 background context，生命周期由 Close() 控制
//   - msgCh 缓冲为 256，避免 Redis 消息积压导致 Subscribe 超时
type EventSubscriber struct {
	client  *redis.Client
	channel string
	msgCh   chan *redis.Message
	cancel  context.CancelFunc
	closed  bool // 防止重复 Close 导致 panic
}

// NewEventSubscriber 创建并启动 Redis 订阅器。
// 创建后立即启动后台协程开始订阅，无需手动调用 Start()。
//
// 参数:
//   - client: 已连接的 Redis 客户端
//
// 返回值:
//   - *EventSubscriber: 已启动的订阅器实例
//
// 后台协程行为:
//  1. 调用 client.Subscribe(ctx, channel) 建立订阅
//  2. 循环读取 pubsub.Channel() 中的消息，写入 msgCh
//  3. 若 Redis 断连（Channel 关闭），等待 3s 后重新订阅
//  4. 收到 ctx.Done() 信号时退出协程
func NewEventSubscriber(client *redis.Client) *EventSubscriber {
	ctx, cancel := context.WithCancel(context.Background())

	sub := &EventSubscriber{
		client:  client,
		channel: "order_events",
		msgCh:   make(chan *redis.Message, 256),
		cancel:  cancel,
	}

	go sub.run(ctx)
	log.Printf("[sse/subscriber] 已启动 Redis 订阅，频道: %s", sub.channel)
	return sub
}

// Channel 返回只读消息通道，供 SSEHub 读取。
// SSEHub 的 forwardRedisMessages() 从此通道接收 Redis 消息并广播。
//
// 返回值:
//   - <-chan *redis.Message: 只读消息通道
func (s *EventSubscriber) Channel() <-chan *redis.Message {
	return s.msgCh
}

// Close 关闭订阅器，停止后台协程。
// 应在服务优雅关闭时调用（main.go 中的 defer）。
//
// 流程:
//  1. 调用 cancel() 取消 context，触发协程退出
//  2. 关闭 msgCh（通知 SSEHub 停止读取）
func (s *EventSubscriber) Close() {
	// 防止重复关闭（panic: close of closed channel）
	if s.closed {
		return
	}
	s.closed = true

	log.Printf("[sse/subscriber] 关闭订阅器，频道: %s", s.channel)
	// 先取消 context 停止订阅协程
	s.cancel()
	// 关闭消息通道，通知下游（SSEHub.forwardRedisMessages）退出
	close(s.msgCh)
}

// run 后台协程的主循环，负责从 Redis 持续订阅消息。
// 断连后自动重连，每次重试间隔 3s。
//
// 流程:
//  1. 循环（直到 ctx 被取消）:
//     a. client.Subscribe(ctx, channel) 建立订阅
//     b. 循环读取 pubsub.Channel()，将消息写入 msgCh
//     c. 若 pubsub.Channel() 被关闭（Redis 断连），log 并 sleep 3s 后重连
//  2. ctx.Done() 触发时退出
func (s *EventSubscriber) run(ctx context.Context) {
	const reconnectDelay = 3 * time.Second

	for {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			log.Printf("[sse/subscriber] 订阅协程退出")
			return
		default:
		}

		// 建立 Redis Pub/Sub 订阅
		pubsub := s.client.Subscribe(ctx, s.channel)
		ch := pubsub.Channel()

		// 从 Redis channel 读取消息，写入内部 msgCh
		// 循环直到 channel 关闭（Redis 断连）或 context 取消
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					// Redis Pub/Sub channel 关闭，跳出内层循环进入重连逻辑
					log.Printf("[sse/subscriber] Redis 订阅通道关闭，%v 后重连...", reconnectDelay)
					goto reconnect
				}
				// 将消息写入内部通道（阻塞写入，防止超速）
				// 注意：若 msgCh 写满会导致 Redis 消息积压，
				// 但这也是限流的一种形式，防止下游处理不过来
				select {
				case s.msgCh <- msg:
				case <-ctx.Done():
					_ = pubsub.Close()
					return
				}
			case <-ctx.Done():
				_ = pubsub.Close()
				log.Printf("[sse/subscriber] 订阅协程退出")
				return
			}
		}

	reconnect:
		// 关闭旧的 pubsub 连接
		_ = pubsub.Close()

		// 等待 3s 后重试
		// 使用 select 支持 context 取消时提前退出
		select {
		case <-ctx.Done():
			log.Printf("[sse/subscriber] 订阅协程退出")
			return
		case <-time.After(reconnectDelay):
			// 继续下一次循环，重新订阅
		}
	}
}

// parseEvent 将 Redis 消息解析为 events.OrderEvent。
// 解析失败时返回 nil 并记录日志（不中断订阅流程）。
//
// 参数:
//   - msg: Redis Pub/Sub 消息
//
// 返回值:
//   - *events.OrderEvent: 解析成功返回事件实例，失败返回 nil
func parseEvent(msg *redis.Message) *events.OrderEvent {
	var event events.OrderEvent
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		log.Printf("[sse/subscriber] 解析 Redis 消息失败: %v, payload: %s", err, msg.Payload)
		return nil
	}
	return &event
}
