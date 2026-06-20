package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

// 测试 publisher.go 文件中的
// func (p *EventPublisher) Publish(event *OrderEvent) error
//
// 函数功能：将订单事件序列化为 JSON 后发布到 Redis Pub/Sub 频道

// newTestPublisher 使用 miniredis 创建测试用的 EventPublisher
func newTestPublisher(t *testing.T) (*EventPublisher, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)

	// 从 miniredis 获取实际地址并连接
	addr := mr.Addr()
	client := redis.NewClient(&redis.Options{Addr: addr})

	publisher := NewEventPublisher(client)
	return publisher, mr
}

func TestEventPublisher_Publish(t *testing.T) {
	t.Run("正常发布事件_Timestamp被自动填充", func(t *testing.T) {
		publisher, mr := newTestPublisher(t)
		defer mr.Close()

		// 先订阅，再发布，验证消息到达
		addr := mr.Addr()
		subClient := redis.NewClient(&redis.Options{Addr: addr})
		defer subClient.Close()

		ctx := subClient.Context()
		pubsub := subClient.Subscribe(ctx, RedisPubSubChannel)
		defer pubsub.Close()

		event := &OrderEvent{
			Type:    EventTypeOrderCreated,
			OrderID: 42,
		}

		err := publisher.Publish(event)
		assert.NoError(t, err)

		// Timestamp 应被自动填充
		assert.False(t, event.Timestamp.IsZero())
		assert.WithinDuration(t, time.Now().UTC(), event.Timestamp, 1*time.Second)

		// 订阅者应能收到消息
		msg, err := pubsub.ReceiveMessage(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg.Payload)

		// 消息应可反序列化为 OrderEvent
		var received OrderEvent
		err = json.Unmarshal([]byte(msg.Payload), &received)
		assert.NoError(t, err)
		assert.Equal(t, EventTypeOrderCreated, received.Type)
		assert.Equal(t, uint(42), received.OrderID)
	})

	t.Run("Redis不可用_返回错误不panic", func(t *testing.T) {
		mr := miniredis.RunT(t)

		addr := mr.Addr()
		client := redis.NewClient(&redis.Options{Addr: addr})
		publisher := NewEventPublisher(client)

		// 关闭 Redis 模拟不可用
		mr.Close()

		event := &OrderEvent{
			Type:    EventTypeOrderCreated,
			OrderID: 1,
		}

		err := publisher.Publish(event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "发布事件到 Redis 失败")
	})

	t.Run("序列化后的事件JSON可正常反序列化", func(t *testing.T) {
		event := &OrderEvent{
			Type:      EventTypeOrderCreated,
			OrderID:   42,
			Timestamp: time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC),
		}

		payload, err := json.Marshal(event)
		assert.NoError(t, err)

		var decoded OrderEvent
		err = json.Unmarshal(payload, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, EventTypeOrderCreated, decoded.Type)
		assert.Equal(t, uint(42), decoded.OrderID)
		assert.Equal(t, "2026-06-18T12:00:00Z", decoded.Timestamp.Format(time.RFC3339))
	})
}
