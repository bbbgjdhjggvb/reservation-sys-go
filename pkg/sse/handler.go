package sse

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler SSE HTTP Handler 工厂函数。
// 返回一个 gin.HandlerFunc，处理 SSE 长连接请求。
// 各服务在 main.go 中注册路由时调用此函数：
//
//	router.GET("/api/reservation/events", sse.Handler(hub))  // Reservation 服务
//	router.GET("/api/admin/events", sse.Handler(hub))        // Admin 服务
//
// 该端点不需要认证（参见文档第七章 7.2.2 SSE 认证困难分析）。
// SSE 推送的内容只有事件类型和 ID，不含业务敏感数据。
//
// 参数:
//   - hub: SSEHub 实例（通过 NewSSEHub() 创建）
//
// 返回值:
//   - gin.HandlerFunc: SSE 长连接处理函数
func Handler(hub *SSEHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		handleSSE(c, hub)
	}
}

// handleSSE SSE 长连接的核心处理逻辑。
//
// 流程:
//  1. 设置 SSE 响应头
//  2. 向 Hub 注册客户端通道
//  3. 进入流式写入循环（事件推送 + 30s 心跳）
//  4. 循环退出后注销客户端
//
// 参数:
//   - c: Gin 上下文
//   - hub: SSEHub 实例
func handleSSE(c *gin.Context, hub *SSEHub) {
	// ===== 第一步：设置 SSE 响应头 =====
	// 必须在写入任何 body 之前设置，否则 Gin 会自动发送 200 + 默认头
	//
	// 各响应头说明:
	//   Content-Type: text/event-stream    — SSE 协议必须
	//   Cache-Control: no-cache            — 禁止浏览器缓存事件流
	//   Connection: keep-alive             — 保持长连接
	//   X-Accel-Buffering: no              — 告知 Nginx 关闭代理缓冲（关键！）
	//     若不设置，Nginx 会将 SSE 流缓冲到磁盘再转发，导致事件延迟
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	// ===== 第二步：注册客户端 =====
	// Register() 返回该客户端专属的事件通道（缓冲 64）
	// Handler 从此通道读取事件并写入 HTTP 响应流
	ch := hub.Register()
	defer hub.Unregister(ch)

	// ===== 第三步：获取 Flusher =====
	// SSE 要求每个事件写入后立即刷新到客户端
	// Gin 的 c.Writer 实现了 http.Flusher 接口
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		// 不支持 Flush 的 ResponseWriter（极少见），返回 500
		c.String(http.StatusInternalServerError, "streaming not supported")
		return
	}

	// 立即刷新响应头，让客户端收到 200 + 正确的 Content-Type
	// 否则客户端会一直等待直到第一个事件写入
	flusher.Flush()

	// ===== 第四步：流式写入循环 =====
	// 使用 c.Stream() 进入循环，每次 step 返回 true 表示继续
	// 循环退出条件：客户端断开（ctx.Done()）或 channel 关闭
	//
	// 两种写入源:
	//   1. 客户端通道有新事件 → 格式化并写入
	//   2. 30s 心跳超时 → 写入注释行 ": ping\n\n" 保持连接
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				// channel 已关闭（Hub 已 Shutdown），退出循环
				return false
			}
			// 写入 SSE 格式事件:
			//   id: <自增ID>\n
			//   event: <事件类型>\n
			//   data: <JSON字符串>\n
			//   \n
			//
			// 注意：每个字段以 \n 结尾，事件之间以空行 \n\n 分隔
			if _, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n",
				event.ID, event.Type, event.Message); err != nil {
				log.Printf("[sse/handler] 写入事件失败: %v", err)
				return false
			}
			flusher.Flush()
			return true

		case <-heartbeat.C:
			// 心跳：每 30s 发送一次注释行，保持连接不被代理超时断开
			// SSE 协议规定以 ":" 开头的行是注释，客户端会忽略
			// 但这行数据会触发 Nginx/代理的 keepalive，防止连接被切断
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				// 心跳写入失败（客户端已断但 Context 未及时检测到）
				log.Printf("[sse/handler] 心跳写入失败，客户端可能已断开")
				return false
			}
			flusher.Flush()
			return true

		case <-c.Request.Context().Done():
			// 客户端主动断开连接（浏览器关闭标签页、网络断开等）
			// c.Stream() 检测到返回 false 并退出循环
			return false
		}
	})
	// c.Stream() 退出后，defer 的 Unregister(ch) 会自动执行
}
