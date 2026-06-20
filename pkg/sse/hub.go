package sse

import (
	"log"
	"sync"
	"sync/atomic"
)

// ========== SSEEvent ==========

// SSEEvent 推送给客户端的 SSE 事件。
// Handler 将其格式化为 SSE 协议文本: "id: N\nevent: type\ndata: json\n\n"
type SSEEvent struct {
	// Type 事件类型，对应 events.EventType 常量（如 "order_created"）
	Type string

	// Message 事件数据的 JSON 序列化字符串
	// Handler 直接写入 "data:" 字段，无需二次序列化
	Message string

	// ID 事件自增序号，用于 SSE Last-Event-ID 断线重连
	// 每次广播递增，客户端可通过此 ID 识别是否漏掉事件
	ID int64
}

// ========== SSEHub ==========

// SSEHub SSE 连接管理中心。
// 维护所有已连接的客户端通道，从 EventSubscriber 读取 Redis 消息并广播给所有客户端。
// 每个服务（Reservation、Admin）各自创建一个独立的 Hub 实例。
//
// 并发安全:
//   - clients map 由 sync.RWMutex 保护
//   - Broadcast() 使用读锁，Register()/Unregister() 使用写锁
//   - eventID 使用 atomic 操作递增
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan SSEEvent]bool
	sub     *EventSubscriber
	eventID int64
}

// NewSSEHub 创建 SSE Hub 实例并启动 Redis 消息转发协程。
// 创建后立即开始从 subscriber.Channel() 读取消息并广播。
//
// 参数:
//   - subscriber: 已启动的 EventSubscriber 实例（通过 NewEventSubscriber() 创建）
//
// 返回值:
//   - *SSEHub: Hub 实例
//
// 后台协程 forwardRedisMessages():
//   - 循环从 subscriber.Channel() 读取 Redis 消息
//   - 解析为 SSEEvent，调用 Broadcast() 推送给所有客户端
//   - subscriber.Channel() 关闭时退出（服务关闭时触发）
func NewSSEHub(subscriber *EventSubscriber) *SSEHub {
	hub := &SSEHub{
		clients: make(map[chan SSEEvent]bool),
		sub:     subscriber,
	}
	go hub.forwardRedisMessages()
	return hub
}

// Register 注册一个新的 SSE 客户端连接。
// Handler 在客户端连接时调用，返回该客户端专属的事件通道。
//
// 返回值:
//   - chan SSEEvent: 客户端事件通道（缓冲大小 64）
//     Handler 从此通道读取事件并写入 HTTP 响应流
//
// 流程:
//  1. 创建缓冲为 64 的 SSEEvent channel
//  2. 加写锁，clients[ch] = true
//  3. 记录日志: 当前连接数
//  4. 返回 channel
func (h *SSEHub) Register() chan SSEEvent {
	ch := make(chan SSEEvent, 64)

	h.mu.Lock()
	h.clients[ch] = true
	count := len(h.clients)
	h.mu.Unlock()

	log.Printf("[sse/hub] 客户端已连接，当前连接数: %d", count)
	return ch
}

// Unregister 注销客户端连接。
// Handler 在客户端断开（Context.Done()）或心跳写入失败时调用。
//
// 参数:
//   - ch: 客户端的 SSEEvent 通道（Register() 返回的通道）
//
// 流程:
//  1. 加写锁
//  2. 若 ch 存在于 clients map，删除并 close(ch)
//  3. 记录日志: 当前连接数
//
// 注意:
//   - close(ch) 后 Handler 的 for-range 循环会自然退出
//   - 多次调用 Unregister 同一个 ch 不会 panic（内部检查存在性）
func (h *SSEHub) Unregister(ch chan SSEEvent) {
	h.mu.Lock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
	count := len(h.clients)
	h.mu.Unlock()

	log.Printf("[sse/hub] 客户端已断开，当前连接数: %d", count)
}

// Broadcast 向所有已注册客户端广播事件。
// 使用非阻塞写入（select + default），避免慢客户端阻塞整个广播。
//
// 参数:
//   - event: 要广播的 SSE 事件（Type 和 Message 必填）
//
// 流程:
//  1. 原子递增 eventID，赋值给 event.ID
//  2. 加读锁
//  3. 遍历 clients，对每个 ch:
//     - select { case ch <- event: / default: }
//     - 写入失败（通道满）→ 跳过，记录警告日志
//  4. 解读锁
func (h *SSEHub) Broadcast(event SSEEvent) {
	// 原子递增事件 ID
	event.ID = atomic.AddInt64(&h.eventID, 1)

	h.mu.RLock()
	skipped := 0
	for ch := range h.clients {
		// 非阻塞写入：通道满时跳过，避免阻塞所有客户端
		// 跳过的客户端将通过降级轮询（15s/10s）同步数据
		select {
		case ch <- event:
			// 写入成功
		default:
			// 通道满，跳过（客户端消费太慢）
			skipped++
		}
	}
	h.mu.RUnlock()

	if skipped > 0 {
		log.Printf("[sse/hub] 广播时跳过 %d 个慢客户端（通道已满），eventID=%d type=%s",
			skipped, event.ID, event.Type)
	}
}

// Shutdown 优雅关闭 Hub，通知所有客户端断开。
// 向每个客户端发送一条 "shutdown" 事件后关闭所有通道。
// 客户端收到 shutdown 事件后应立即启动降级轮询。
//
// 参数:
//   - 无
//
// 流程:
//  1. 构造 shutdown SSEEvent（Type: "shutdown"）
//  2. 加写锁，遍历所有 clients:
//     a. 尝试写入 shutdown 事件（非阻塞）
//     b. close(ch)
//     c. 从 map 中删除
//  3. 记录日志: "Hub 已关闭，所有客户端已断开"
func (h *SSEHub) Shutdown() {
	shutdownEvent := SSEEvent{
		Type:    "shutdown",
		Message: `{"message":"server is shutting down"}`,
		ID:      atomic.AddInt64(&h.eventID, 1),
	}

	h.mu.Lock()
	for ch := range h.clients {
		// 非阻塞写入 shutdown 事件
		select {
		case ch <- shutdownEvent:
		default:
		}
		close(ch)
		delete(h.clients, ch)
	}
	h.mu.Unlock()

	log.Printf("[sse/hub] Hub 已关闭，所有客户端已断开")
}

// ClientCount 返回当前已连接的客户端数量。
// 用于监控和日志。
//
// 返回值:
//   - int: 当前连接数
func (h *SSEHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// forwardRedisMessages 后台协程：从 EventSubscriber 读取 Redis 消息并广播。
// 在 NewSSEHub() 中启动，生命周期与 Hub 相同。
//
// 流程:
//  1. 循环从 sub.Channel() 读取消息
//  2. 调用 parseEvent() 解析为 events.OrderEvent
//  3. 将 OrderEvent 的 Payload 序列化为 JSON 字符串
//  4. 构造 SSEEvent{Type, Message, ID}
//  5. 调用 Broadcast(sseEvent) 推送给所有客户端
//  6. sub.Channel() 关闭时退出循环（服务关闭时触发）
func (h *SSEHub) forwardRedisMessages() {
	log.Printf("[sse/hub] 开始监听 Redis 消息")

	for msg := range h.sub.Channel() {
		// 解析 Redis 消息为业务事件
		event := parseEvent(msg)
		if event == nil {
			// 解析失败，跳过（日志已在 parseEvent 中记录）
			continue
		}

		// 构造 SSEEvent 并广播
		sseEvent := SSEEvent{
			Type:    event.Type,
			Message: msg.Payload, // 直接使用 Redis 消息的 JSON 字符串
		}
		h.Broadcast(sseEvent)
	}

	log.Printf("[sse/hub] Redis 消息通道已关闭，停止监听")
}
