// useSSE.ts — 预约端 SSE 实时推送 composable。
//
// 功能：
//   建立与 Reservation 服务的 SSE 长连接，监听订单变更事件。
//   收到事件后自动刷新日历已占用时段（通过 useCalendar.fetchOccupiedSlots）。
//   SSE 连接异常时自动降级为轮询模式（每 15s 拉取一次），保证功能可用。
//
// 依赖：
//   - useCalendar composable（调用 fetchOccupiedSlots 刷新数据）
//
// 返回值：
//   - connected: Ref<boolean>     — SSE 连接是否处于活跃状态
//   - usePolling: Ref<boolean>    — 是否已降级为轮询模式
//
// 副作用：
//   - onMounted: 建立 EventSource 连接
//   - onUnmounted: 关闭 EventSource，停止轮询定时器
//
// SSE 端点（无认证）：GET /api/reservation/events
// 推送内容不含敏感数据，客户端收到通知后仍需通过 REST API（带 Bearer Token）拉取完整数据。

import { ref, onMounted, onUnmounted } from 'vue'
import { useCalendar } from './useCalendar'

// ========== 配置常量 ==========

// SSE 端点路径（通过 Nginx 代理到 Reservation 服务）
const SSE_ENDPOINT = '/api/reservation/events'

// 轮询间隔（毫秒）：SSE 不可用时每隔此时间拉取一次已占用时段
const POLLING_INTERVAL_MS = 15_000

// SSE 重连间隔（毫秒）：轮询模式下每隔此时间尝试重新建立 SSE 连接
const RECONNECT_INTERVAL_MS = 30_000

// EventSource 错误码：重连超时
const ES_RECONNECTING = 0

// ========== Composable ==========

/**
 * useReservationSSE — 预约端 SSE 实时推送 composable。
 *
 * 行为概述：
 *   1. 组件挂载时自动建立 EventSource 连接到 /api/reservation/events
 *   2. 收到事件后调用 useCalendar().fetchOccupiedSlots() 刷新日历数据
 *   3. SSE 连接断开时自动降级为 15s 间隔轮询
 *   4. 轮询期间每 30s 尝试重新连接 SSE
 *   5. SSE 恢复后自动停止轮询
 *   6. 组件卸载时清理所有连接和定时器
 *
 * 返回:
 *   - connected: SSE 连接是否活跃
 *   - usePolling: 是否处于降级轮询模式
 */
export function useReservationSSE() {
  const { fetchOccupiedSlots, clearOccupiedCache } = useCalendar()

  // SSE 连接状态
  const connected = ref(false)

  // 是否已降级为轮询模式
  const usePolling = ref(false)

  // EventSource 实例引用（用于手动关闭和重建）
  let eventSource: EventSource | null = null

  // 轮询定时器 ID
  let pollingTimer: ReturnType<typeof setInterval> | null = null

  // SSE 重连定时器 ID
  let reconnectTimer: ReturnType<typeof setInterval> | null = null

  /**
   * connect 建立 SSE 连接。
   *
   * 流程:
   *  1. 创建 EventSource 实例，连接到 SSE_ENDPOINT
   *  2. 注册 onopen、onerror、各事件类型的监听器
   *  3. 连接成功时停止轮询，连接失败时启动轮询
   */
  function connect(): void {
    // 先关闭旧的 EventSource（若有）
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }

    eventSource = new EventSource(SSE_ENDPOINT)

    // onopen: 连接成功
    //   - 标记 connected=true, usePolling=false
    //   - 停止所有轮询/重连定时器
    eventSource.onopen = () => {
      connected.value = true
      stopPolling()
    }

    // 监听 order_created 事件：新预约提交 → 刷新日历已占用时段
    eventSource.addEventListener('order_created', handleSSEEvent)

    // 监听 order_cancelled 事件：预约取消 → 刷新日历（该时段变为可用）
    eventSource.addEventListener('order_cancelled', handleSSEEvent)

    // 监听 order_reviewed 事件：审核操作 → 刷新日历（状态可能变化）
    eventSource.addEventListener('order_reviewed', handleSSEEvent)

    // 监听 slot_updated 事件：时段更新 → 刷新日历（密码设置等）
    eventSource.addEventListener('slot_updated', handleSSEEvent)

    // 监听 shutdown 事件：服务端主动关闭 → 启动降级轮询
    eventSource.addEventListener('shutdown', () => {
      // clean up the current event source because server is going down
      if (eventSource) {
        eventSource.close()
        eventSource = null
      }
      connected.value = false
      startPolling()
    })

    // onerror: 连接失败或断开
    //   浏览器自身会在 onerror 后自动重连（EventSource 默认行为），
    //   但为了更快响应，我们也启动降级轮询
    eventSource.onerror = () => {
      connected.value = false

      // 根据 readyState 判断是否需要启动轮询
      //   CONNECTING(0): 正在重连，等待浏览器自动恢复
      //   CLOSED(2): 连接已关闭，手动启动轮询
      if (eventSource?.readyState === EventSource.CLOSED) {
        startPolling()
      } else if (eventSource?.readyState === ES_RECONNECTING) {
        // 浏览器正在重连，先等待；30s 后用 reconnect 定时器兜底
        if (!reconnectTimer) {
          reconnectTimer = setInterval(() => {
            if (!connected.value) {
              // 浏览器重连似乎也失效，手动启动轮询
              if (eventSource?.readyState !== EventSource.OPEN) {
                eventSource?.close()
                startPolling()
              }
            }
          }, RECONNECT_INTERVAL_MS)
        }
      }
    }
  }

  /**
   * handleSSEEvent 处理 SSE 事件的统一入口。
   * 所有事件类型的处理逻辑相同：先清除缓存，再刷新日历已占用时段。
   * 不清除缓存会导致 fetchOccupiedSlots 命中缓存跳过实际请求。
   *
   * @param _event - MessageEvent，data 字段为 JSON 字符串（当前仅用于日志）
   */
  function handleSSEEvent(_event: MessageEvent): void {
    clearOccupiedCache()
    fetchOccupiedSlots().catch(() => {
      // 刷新失败时静默处理（降级轮询会兜底）
    })
  }

  /**
   * startPolling 启动降级轮询。
   * SSE 不可用时，每 POLLING_INTERVAL_MS 毫秒拉取一次已占用时段。
   * 同时每 RECONNECT_INTERVAL_MS 毫秒尝试重新建立 SSE 连接。
   */
  function startPolling(): void {
    if (usePolling.value) {
      return // 已经在轮询模式，避免重复启动
    }

    usePolling.value = true

    // 启动轮询定时器：每 15s 调用 fetchOccupiedSlots()
    if (!pollingTimer) {
      pollingTimer = setInterval(() => {
        clearOccupiedCache()
        fetchOccupiedSlots().catch(() => {
          // 轮询失败静默处理，后续定时器会重试
        })
      }, POLLING_INTERVAL_MS)
    }

    // 启动重连定时器：每 30s 尝试重新建立 SSE 连接
    if (!reconnectTimer) {
      reconnectTimer = setInterval(() => {
        reconnect()
      }, RECONNECT_INTERVAL_MS)
    }
  }

  /**
   * stopPolling 停止降级轮询。
   * SSE 恢复连接后调用，清除所有轮询和重连定时器。
   */
  function stopPolling(): void {
    usePolling.value = false

    if (pollingTimer) {
      clearInterval(pollingTimer)
      pollingTimer = null
    }

    if (reconnectTimer) {
      clearInterval(reconnectTimer)
      reconnectTimer = null
    }
  }

  /**
   * reconnect 尝试重新建立 SSE 连接。
   * 关闭旧的 EventSource（若有），重新调用 connect()。
   * 若连接成功（onopen 触发），onopen 会自动调用 stopPolling()。
   */
  function reconnect(): void {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    connect()
  }

  /**
   * disconnect 完全断开 SSE 连接并停止所有定时器。
   * 在 onUnmounted 中调用，确保组件销毁后没有残留的定时器或连接。
   */
  function disconnect(): void {
    // 关闭 EventSource
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }

    // 停止所有定时器
    stopPolling()

    connected.value = false
  }

  // ===== 生命周期 =====
  onMounted(() => {
    connect()
  })

  onUnmounted(() => {
    disconnect()
  })

  return {
    connected,
    usePolling,
  }
}
