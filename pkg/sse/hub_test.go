package sse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 测试文件: hub.go
// 测试对象: SSEHub 的 Register / Unregister / Broadcast / ClientCount / Shutdown

// newTestHub 创建一个不含 EventSubscriber 的 SSEHub（用于单元测试），
// 不启动 forwardRedisMessages 协程，避免依赖 Redis。
func newTestHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan SSEEvent]bool),
		sub:     nil, // 单元测试不需要 Redis 订阅
	}
}

// ========== Register 测试 ==========

func TestSSEHub_Register(t *testing.T) {
	// 测试 hub.go 文件中的 func (h *SSEHub) Register() chan SSEEvent
	// 函数功能：注册新的 SSE 客户端连接，返回专属事件通道

	t.Run("注册客户端_返回非nil通道且连接数增加", func(t *testing.T) {
		hub := newTestHub()
		assert.Equal(t, 0, hub.ClientCount())

		ch := hub.Register()
		assert.NotNil(t, ch)
		assert.Equal(t, 1, hub.ClientCount())
	})

	t.Run("注册多个客户端_连接数正确累加", func(t *testing.T) {
		hub := newTestHub()

		ch1 := hub.Register()
		ch2 := hub.Register()
		ch3 := hub.Register()

		assert.NotNil(t, ch1)
		assert.NotNil(t, ch2)
		assert.NotNil(t, ch3)
		assert.Equal(t, 3, hub.ClientCount())
	})
}

// ========== Unregister 测试 ==========

func TestSSEHub_Unregister(t *testing.T) {
	// 测试 hub.go 文件中的 func (h *SSEHub) Unregister(ch chan SSEEvent)
	// 函数功能：注销客户端连接，关闭通道

	t.Run("注销客户端_channel被关闭且连接数减少", func(t *testing.T) {
		hub := newTestHub()
		ch := hub.Register()
		assert.Equal(t, 1, hub.ClientCount())

		hub.Unregister(ch)
		assert.Equal(t, 0, hub.ClientCount())

		// channel 应被关闭（从已关闭的 channel 读取不会阻塞）
		_, ok := <-ch
		assert.False(t, ok)
	})

	t.Run("注销不存在的channel_不panic", func(t *testing.T) {
		hub := newTestHub()
		ch := make(chan SSEEvent, 1)

		// 不应 panic
		assert.NotPanics(t, func() {
			hub.Unregister(ch)
		})
		assert.Equal(t, 0, hub.ClientCount())
	})

	t.Run("重复注销同一个channel_不panic", func(t *testing.T) {
		hub := newTestHub()
		ch := hub.Register()

		hub.Unregister(ch)
		// 第二次 Unregister 不应 panic
		assert.NotPanics(t, func() {
			hub.Unregister(ch)
		})
	})
}

// ========== Broadcast 测试 ==========

func TestSSEHub_Broadcast(t *testing.T) {
	// 测试 hub.go 文件中的 func (h *SSEHub) Broadcast(event SSEEvent)
	// 函数功能：向所有已注册客户端广播事件

	t.Run("广播事件_所有已注册客户端收到", func(t *testing.T) {
		hub := newTestHub()
		ch1 := hub.Register()
		ch2 := hub.Register()
		ch3 := hub.Register()

		event := SSEEvent{
			Type:    "order_created",
			Message: `{"order_id":42}`,
		}

		hub.Broadcast(event)

		// 所有三个客户端都应收到事件
		select {
		case received := <-ch1:
			assert.Equal(t, "order_created", received.Type)
			assert.Equal(t, `{"order_id":42}`, received.Message)
			assert.Greater(t, received.ID, int64(0))
		default:
			t.Error("ch1 未收到事件")
		}

		select {
		case received := <-ch2:
			assert.Equal(t, "order_created", received.Type)
		default:
			t.Error("ch2 未收到事件")
		}

		select {
		case received := <-ch3:
			assert.Equal(t, "order_created", received.Type)
		default:
			t.Error("ch3 未收到事件")
		}
	})

	t.Run("广播时通道满_跳过不阻塞", func(t *testing.T) {
		hub := newTestHub()

		// 创建只有 1 个缓冲的通道，先填满
		ch := make(chan SSEEvent, 1)
		hub.mu.Lock()
		hub.clients[ch] = true
		hub.mu.Unlock()

		// 先写入一个事件填满通道
		ch <- SSEEvent{Type: "fill"}
		assert.Equal(t, 1, len(ch)) // 通道已满

		// 广播新事件：应跳过此通道而不阻塞
		event := SSEEvent{Type: "order_created", Message: "test"}
		hub.Broadcast(event)

		// 通道仍为满状态（新事件被跳过）
		assert.Equal(t, 1, len(ch))

		// 清理
		<-ch
		hub.Unregister(ch)
	})

	t.Run("广播时eventID递增", func(t *testing.T) {
		hub := newTestHub()
		ch := hub.Register()

		hub.Broadcast(SSEEvent{Type: "event1", Message: ""})
		id1 := (<-ch).ID

		hub.Broadcast(SSEEvent{Type: "event2", Message: ""})
		id2 := (<-ch).ID

		assert.Greater(t, id2, id1)
	})
}

// ========== Shutdown 测试 ==========

func TestSSEHub_Shutdown(t *testing.T) {
	// 测试 hub.go 文件中的 func (h *SSEHub) Shutdown()
	// 函数功能：优雅关闭 Hub，通知所有客户端后关闭所有通道

	t.Run("关闭Hub_所有channel被关闭", func(t *testing.T) {
		hub := newTestHub()
		ch1 := hub.Register()
		ch2 := hub.Register()

		hub.Shutdown()
		assert.Equal(t, 0, hub.ClientCount())

		// 先消费掉 shutdown 事件（Shutdown 会尝试非阻塞写入）
		select {
		case <-ch1:
		default:
		}
		select {
		case <-ch2:
		default:
		}

		// 通道应被关闭
		_, ok1 := <-ch1
		assert.False(t, ok1)

		_, ok2 := <-ch2
		assert.False(t, ok2)
	})
}
