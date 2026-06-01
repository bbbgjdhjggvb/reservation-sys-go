# 注释的总体要求
1. 说明函数参数和返回值
2. 容易犯错的注意点
3. 说明结构体

# repository 层
1. 函数的功能
2. 参数
3. 返回值
4. 并在函数体内添加 SQL 注释，流程、逻辑、算法注释

函数的注释如下: 
```go
// CreateOrderWithLock 原子化创建订单（事务内行锁，防止并发双重预约）。
//
// 参数:
//  - order: 订单结构体实例
//  - slots: 时间段结构体实例
//
// 返回值：
//  - nil：创建成功
//
// 运行过程:
//  1. 逐个检测时段冲突
//  2. 创建订单记录
//  3. 批量创建时段记录
func (r *repository) CreateOrderWithLock(order *ReservationOrder, slots []ReservationSlot) error {
    //	BEGIN;
	return r.db.Transaction(func(tx *gorm.DB) error {
        //	SELECT COUNT(*) FROM reservation_slots WHERE status IN (1,2,5) AND start_time < ? AND end_time > ? FOR UPDATE;

        //	INSERT INTO reservation_orders (...) VALUES (...);

        //	INSERT INTO reservation_slots (order_id, ...) VALUES (?, ...), (?, ...), ...;

		return nil
	})
    //	COMMIT;
}
```

---

# service 层
注释参考代码如下：
```go
// Submit 批量提交预约申请。
//
// 参数:
//   - openid: 微信用户唯一标识
//   - slots: 已解析的时段列表
//   - req: 提交请求（含申请人信息）
//
// 返回值:
//   - *reservationdb.ReservationOrder: 创建成功的订单实体（含 ID）
//   - error: 时段数量不合法、时段冲突、创建失败时返回错误
//
// 流程:
//  1. 校验时段数量（1~4个）
//  2. 合并同一天连续的时段
//  3. 生成订单号
//  4. 在事务内创建订单+时段（行锁防并发）
func (s *ReservationService) Submit(openid string, slots []ParsedSlot, req *SubmitReq) (*reservationdb.ReservationOrder, error) {
    return nil, nil
}
```

---

# handler 层
handler 层使用 swagger 文档注解，并在函数体里面编写好处理流程注释。
```go
// SubmitHandler 处理预约提交请求。
//
//	@Summary		提交预约申请
//	@Description	用户提交预约申请，包含申请人信息和时段列表（1~4个时段），同一天连续时段自动合并
//	@Tags			预约-用户端
//	@Accept			json
//	@Produce		json
//	@Param			body	body		SubmitReq	true	"预约申请信息"
//	@Success		200		{object}	Response{data=OrderResp}	"预约提交成功"
//	@Failure		400		{object}	Response					"参数错误/时段冲突"
//	@Failure		401		{object}	Response					"未授权"
//	@Security		BearerAuth
//	@Router			/api/reservation/reservation/submit [post]
func (h *ReservationHandler) SubmitHandler(c *gin.Context) {
    // 参数绑定
    // - 如果参数绑定失败，调用 badRequest(c, "表单填写有误，请检查") 并 return

    // 时间段个数检验
    // - 如果没有时间段，调用 badRequest(c, "请至少选择一个时间段") 并 return
    // - 如果选择时间段过多，调用 badRequest(c, "最多选择4个时间段") 并 return

    // 时间段格式检验
    //  - 解析时间段
    //      - 解析失败，调用 badRequest(c, fmt.Sprintf("第%d个时间段格式错误", i)) 并 return
    //  - 结束时间必须晚于开始时间
    //      - 结束时间早于开始时间，调用 badRequest(c, fmt.Sprintf("第%d个时间段的结束时间必须晚于开始时间", i))

    // 调用 getOpenID() 判断 openid 是否存在于 gin 的上下文环境中
    // - 如果不存在，调用 unauthorized(c, "未授权，请从服务号进入") 并 return

    // 调用 svc.Submit()
    // - 如果出现错误，调用 badRequest(c, err.Error())

    // 检查数据库中是否真的存入订单，调用 svc.GetOrderByID()
    // - 如果存在，就回出现错误，调用 okWithMsg(c, fmt.Sprintf("预约提交成功，共%d个时间段，请等待审核", slotCount), order)

    // 调用 svc.GetOrderByID 判断是否正常存入到数据集中
	// 如果已经存入就会有 err，调用 okWithMsg(c, fmt.Sprintf("预约提交成功，共%d个时段，请等待审核", slotCount), OrderToResp(fullOrder)) 并 return

    // 调用okWithMsg(c, fmt.Sprintf("预约提交成功，共%d个时段，请等待审核", slotCount), OrderToResp(fullOrder))

}
```