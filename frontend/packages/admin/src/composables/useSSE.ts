// useSSE.ts — 审核端 SSE 实时推送 composable。
//
// 功能：
//   建立与 Admin 服务的 SSE 长连接，监听订单变更事件。
//   收到事件后自动刷新管理员订单列表（通过 useAdminOrders.fetchOrders）。
//   SSE 连接异常时自动降级为轮询模式（每 10s 拉取一次），保证功能可用。
//
// 依赖：
//   - useAdminOrders composable（调用 fetchOrders 刷新订单列表）
//
// 返回值：
//   - connected: Ref<boolean>     — SSE 连接是否处于活跃状态
//   - usePolling: Ref<boolean>    — 是否已降级为轮询模式
//
// 副作用：
//   - onMounted: 建立 EventSource 连接
//   - onUnmounted: 关闭 EventSource，停止轮询定时器
//
// 使用场景：
//   仅在 DashboardView（订单列表页）中使用。
//   审核操作本身由 REST API 完成，SSE 负责感知"其他管理员"的操作结果。
//
// SSE 端点（无认证）：GET /api/admin/events

import { ref, onMounted, onUnmounted } from 'vue'

// ========== 配置常量 ==========

// SSE 端点路径（通过 Nginx 代理到 Admin 服务）
const SSE_ENDPOINT = '/api/admin/events'

// 轮询间隔（毫秒）：SSE 不可用时每隔此时间拉取一次订单列表
// Admin 端使用 10s（比 Reservation 端的 15s 更短），因为审核操作的时效性要求更高
const POLLING_INTERVAL_MS = 10_000

// SSE 重连间隔（毫秒）：轮询模式下每隔此时间尝试重新建立 SSE 连接
const RECONNECT_INTERVAL_MS = 30_000

// ========== Composable ==========

/**
 * useAdminSSE — 审核端 SSE 实时推送 composable。
 *
 * 行为概述：
 *   1. 组件挂载时自动建立 EventSource 连接到 /api/admin/events
 *   2. 收到事件后调用 fetchOrders 刷新订单列表
 *   3. SSE 连接断开时自动降级为 10s 间隔轮询
 *   4. 轮询期间每 30s 尝试重新连接 SSE
 *   5. SSE 恢复后自动停止轮询
 *   6. 组件卸载时清理所有连接和定时器
 *
 * 参数:
 *   - fetchOrders: 来自 useAdminOrders() 实例的 fetchOrders 函数，
 *     传入引用而非内部调用 useAdminOrders() 是为了确保操作的是视图绑定的同一实例
 *
 * 返回:
 *   - connected: SSE 连接是否活跃
 *   - usePolling: 是否处于降级轮询模式
 */
export function useAdminSSE(fetchOrders: () => Promise<void>) {
  // SSE 连接状态
  const connected = ref(false)

  // 是否已降级为轮询模式
  const usePolling = ref(false)

  // EventSource 实例引用
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

    // onopen: 连接成功 → 标记 connected=true，停止轮询
    eventSource.onopen = () => {
      connected.value = true
      stopPolling()
    }

    // 监听 order_created 事件：新预约提交 → 刷新订单列表
    eventSource.addEventListener('order_created', handleSSEEvent)

    // 监听 order_cancelled 事件：预约取消 → 刷新订单列表
    eventSource.addEventListener('order_cancelled', handleSSEEvent)

    // 监听 order_reviewed 事件：其他管理员的审核操作 → 刷新订单列表
    eventSource.addEventListener('order_reviewed', handleSSEEvent)

    // 监听 slot_updated 事件：时段更新（密码设置等）→ 刷新订单列表
    eventSource.addEventListener('slot_updated', handleSSEEvent)

    // 监听 shutdown 事件：服务端主动关闭 → 启动降级轮询
    eventSource.addEventListener('shutdown', () => {
      if (eventSource) {
        eventSource.close()
        eventSource = null
      }
      connected.value = false
      startPolling()
    })

    // onerror: 连接失败或断开
    eventSource.onerror = () => {
      connected.value = false

      // CLOSED(2): 连接已永久关闭，手动启动轮询
      if (eventSource?.readyState === EventSource.CLOSED) {
        startPolling()
      }
    }
  }

  /**
   * handleSSEEvent 处理 SSE 事件的统一入口。
   * 所有事件类型的处理逻辑相同：刷新管理员订单列表。
   *
   * @param _event - MessageEvent（当前仅用于日志）
   */
  function handleSSEEvent(_event: MessageEvent): void {
    fetchOrders().catch(() => {
      // 刷新失败时静默处理（降级轮询会兜底）
    })
  }

  /**
   * startPolling 启动降级轮询。
   * SSE 不可用时，每 POLLING_INTERVAL_MS 毫秒拉取一次订单列表。
   * 同时每 RECONNECT_INTERVAL_MS 毫秒尝试重新建立 SSE 连接。
   */
  function startPolling(): void {
    if (usePolling.value) {
      return
    }

    usePolling.value = true

    // 启动轮询定时器：每 10s 调用 fetchOrders()
    if (!pollingTimer) {
      pollingTimer = setInterval(() => {
        fetchOrders().catch(() => {
          // 轮询失败静默处理
        })
      }, POLLING_INTERVAL_MS)
    }

    // 启动重连定时器：每 30s 尝试重新建立 SSE
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
   * 关闭旧的 EventSource，重新调用 connect()。
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
   * 在 onUnmounted 中调用。
   */
  function disconnect(): void {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }

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
