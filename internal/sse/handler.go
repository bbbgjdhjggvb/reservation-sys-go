package sse

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler 返回一个 gin.HandlerFunc，处理 SSE 长连接请求。
// 该端点不需要认证，SSE 推送的内容只有事件类型和 ID，不含业务敏感数据。
func Handler(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		handleSSE(c, hub)
	}
}

// handleSSE SSE 长连接的核心处理逻辑
func handleSSE(c *gin.Context, hub *Hub) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	// 注册客户端
	ch := hub.Register()
	defer hub.Unregister(ch)

	// 获取 Flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "streaming not supported")
		return
	}
	flusher.Flush()

	// 流式写入循环
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}
			if _, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n",
				event.ID, event.Type, event.Message); err != nil {
				log.Printf("[sse/handler] 写入事件失败: %v", err)
				return false
			}
			flusher.Flush()
			return true

		case <-heartbeat.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				log.Printf("[sse/handler] 心跳写入失败，客户端可能已断开")
				return false
			}
			flusher.Flush()
			return true

		case <-c.Request.Context().Done():
			return false
		}
	})
}
