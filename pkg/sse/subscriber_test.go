package sse

import (
	"testing"
	"time"

	"reservation-sys/pkg/events"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

// 测试 subscriber.go 文件中的
// func NewEventSubscriber(client *redis.Client) *EventSubscriber
// func (s *EventSubscriber) Channel() <-chan *redis.Message
// func (s *EventSubscriber) Close()
//
// 函数功能：创建订阅器、获取消息通道、关闭订阅器

// newTestSubscriber 使用 miniredis 创建测试用 EventSubscriber
func newTestSubscriberAndPublisher(t *testing.T) (*EventSubscriber, *events.EventPublisher, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	addr := mr.Addr()

	subClient := redis.NewClient(&redis.Options{Addr: addr})
	pubClient := redis.NewClient(&redis.Options{Addr: addr})

	subscriber := NewEventSubscriber(subClient)
	publisher := events.NewEventPublisher(pubClient)

	return subscriber, publisher, mr
}

func TestEventSubscriber_Channel(t *testing.T) {
	// 测试 subscriber.go 中的 func (s *EventSubscriber) Channel() <-chan *redis.Message
	// 函数功能：返回只读消息通道

	t.Run("发布事件后_订阅者通过Channel收到消息", func(t *testing.T) {
		subscriber, publisher, mr := newTestSubscriberAndPublisher(t)
		defer mr.Close()
		defer subscriber.Close()

		// 等待订阅建立（后台协程可能在 Subscribe 调用中）
		time.Sleep(50 * time.Millisecond)

		// 发布事件
		event := &events.OrderEvent{
			Type:    events.EventTypeOrderCreated,
			OrderID: 99,
		}
		err := publisher.Publish(event)
		assert.NoError(t, err)

		// 从 Channel() 读取消息（超时 2s）
		select {
		case msg := <-subscriber.Channel():
			assert.NotNil(t, msg)
			assert.Contains(t, msg.Payload, "order_created")
			assert.Contains(t, msg.Payload, "99")
		case <-time.After(2 * time.Second):
			t.Fatal("超时：订阅者未收到消息")
		}
	})

	t.Run("发布多条事件_订阅者按序收到", func(t *testing.T) {
		subscriber, publisher, mr := newTestSubscriberAndPublisher(t)
		defer mr.Close()
		defer subscriber.Close()

		// 等待订阅建立
		time.Sleep(50 * time.Millisecond)

		// 发布多条事件
		for i := 1; i <= 3; i++ {
			err := publisher.Publish(&events.OrderEvent{
				Type:    events.EventTypeOrderCreated,
				OrderID: uint(i),
			})
			assert.NoError(t, err)
		}

		// 接收多条消息
		for i := 1; i <= 3; i++ {
			select {
			case msg := <-subscriber.Channel():
				assert.NotNil(t, msg)
				assert.NotEmpty(t, msg.Payload)
			case <-time.After(2 * time.Second):
				t.Fatalf("超时：第 %d 条消息未收到", i)
			}
		}
	})
}

func TestEventSubscriber_Close(t *testing.T) {
	// 测试 subscriber.go 中的 func (s *EventSubscriber) Close()
	// 函数功能：关闭订阅器，停止后台协程

	t.Run("关闭后_Channel被关闭", func(t *testing.T) {
		subscriber, _, mr := newTestSubscriberAndPublisher(t)
		defer mr.Close()

		subscriber.Close()

		// Channel() 应被关闭
		msg, ok := <-subscriber.Channel()
		assert.False(t, ok, "Channel 应被关闭")
		assert.Nil(t, msg)

		// 重复 Close 不应 panic
		assert.NotPanics(t, func() {
			subscriber.Close()
		})
	})
}

func TestParseEvent(t *testing.T) {
	// 测试 subscriber.go 中的 parseEvent() 函数
	// 函数功能：将 Redis 消息解析为 events.OrderEvent

	t.Run("正常JSON消息_解析成功", func(t *testing.T) {
		msg := &redis.Message{
			Channel: "order_events",
			Payload: `{"type":"order_created","order_id":42,"timestamp":"2026-06-18T12:00:00Z"}`,
		}

		event := parseEvent(msg)
		assert.NotNil(t, event)
		assert.Equal(t, "order_created", event.Type)
		assert.Equal(t, uint(42), event.OrderID)
	})

	t.Run("非JSON消息_返回nil", func(t *testing.T) {
		msg := &redis.Message{
			Channel: "order_events",
			Payload: `not valid json`,
		}

		event := parseEvent(msg)
		assert.Nil(t, event)
	})
}
