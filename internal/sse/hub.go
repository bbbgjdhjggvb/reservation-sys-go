// Package sse 提供基于进程内事件总线的 SSE 实时推送。
// Hub 从 event.Bus.Subscribe() 读取事件并广播给所有已连接的 SSE 客户端。
package sse

import (
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"

	"reservation-sys/internal/event"
)

// ========== SSEEvent ==========

// SSEEvent 推送给客户端的 SSE 事件
type SSEEvent struct {
	Type    string // 事件类型，对应 event.OrderEventType
	Message string // 事件数据的 JSON 序列化字符串
	ID      int64  // 事件自增序号
}

// ========== Hub ==========

// Hub SSE 连接管理中心。
// 维护所有已连接的客户端通道，从事件总线读取事件并广播给所有客户端。
type Hub struct {
	mu      sync.RWMutex
	clients map[chan SSEEvent]bool
	bus     *event.Bus
	eventID int64
}

// NewHub 创建 SSE Hub 实例并启动事件转发协程。
// 创建后立即开始从 bus.Subscribe() 读取事件并向所有客户端广播。
func NewHub(bus *event.Bus) *Hub {
	hub := &Hub{
		clients: make(map[chan SSEEvent]bool),
		bus:     bus,
	}
	go hub.forwardEvents()
	return hub
}

// Register 注册一个新的 SSE 客户端连接，返回该客户端专属的事件通道（缓冲 64）
func (h *Hub) Register() chan SSEEvent {
	ch := make(chan SSEEvent, 64)
	h.mu.Lock()
	h.clients[ch] = true
	count := len(h.clients)
	h.mu.Unlock()
	log.Printf("[sse/hub] 客户端已连接，当前连接数: %d", count)
	return ch
}

// Unregister 注销客户端连接，关闭其通道
func (h *Hub) Unregister(ch chan SSEEvent) {
	h.mu.Lock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
	count := len(h.clients)
	h.mu.Unlock()
	log.Printf("[sse/hub] 客户端已断开，当前连接数: %d", count)
}

// Broadcast 向所有已注册客户端广播事件（非阻塞写入）
func (h *Hub) Broadcast(event SSEEvent) {
	event.ID = atomic.AddInt64(&h.eventID, 1)
	h.mu.RLock()
	skipped := 0
	for ch := range h.clients {
		select {
		case ch <- event:
		default:
			skipped++
		}
	}
	h.mu.RUnlock()
	if skipped > 0 {
		log.Printf("[sse/hub] 广播时跳过 %d 个慢客户端，eventID=%d type=%s", skipped, event.ID, event.Type)
	}
}

// Shutdown 优雅关闭 Hub，通知所有客户端断开
func (h *Hub) Shutdown() {
	shutdownEvent := SSEEvent{
		Type:    "shutdown",
		Message: `{"message":"server is shutting down"}`,
		ID:      atomic.AddInt64(&h.eventID, 1),
	}
	h.mu.Lock()
	for ch := range h.clients {
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

// ClientCount 返回当前已连接的客户端数量
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// forwardEvents 后台协程：从事件总线读取事件并广播
func (h *Hub) forwardEvents() {
	log.Printf("[sse/hub] 开始监听进程内事件总线")
	for evt := range h.bus.Subscribe() {
		payloadJSON, err := json.Marshal(evt)
		if err != nil {
			log.Printf("[sse/hub] 事件序列化失败: %v", err)
			continue
		}
		sseEvent := SSEEvent{
			Type:    string(evt.Type),
			Message: string(payloadJSON),
		}
		h.Broadcast(sseEvent)
	}
	log.Printf("[sse/hub] 事件总线通道已关闭，停止监听")
}
