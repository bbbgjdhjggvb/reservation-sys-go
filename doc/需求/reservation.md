# 预约模块功能介绍

普通用户通过微信服务号进入预约页面，完成场地预约。

用户在微信中点击服务号菜单，跳转到 `http://domain/reserve?token=<jwt>`，携带 JWT token 完成身份认证，**进入预约页面**。

也可以在浏览器中访问 `http://domain/myorders` 查看自己的预约记录。

## 预约流程

1. 用户从微信服务号进入预约页面，URL 携带 token 参数
2. 在日历上选择日期和时间段（8:00-10:00、10:00-12:00、13:00-15:00、15:00-17:00），最多可选 4 个时段
3. 点击"下一步"，填写预约信息
    1. 申请人姓名
    2. 年级（入学年份）
    3. 所属学院校友会（从下拉列表中选择）
    4. 专业
    5. 手机号码（11位）
    6. 会议内容（最多500字）
4. 点击"确定提交"，弹出确认弹窗核对信息
5. 确认后提交预约，等待管理员审核
6. 审核通过后，管理员设置门锁密码并通过微信服务号推送给用户

## 我的预约

1. 可以查看自己的所有预约记录
2. 每条预约显示：
    1. 订单号
    2. 申请人姓名
    3. 预约状态（待审核、已通过、已拒绝、已完成、已取消、待一级审核、待二级审核、一级驳回、二级驳回）
    4. 时间段明细（每个时段有独立的状态标识）
    5. 学院、专业、手机号
    6. 创建时间
    7. 会议内容
3. 对于状态为"待审核(0)"或"待一级审核(5)"的预约，可以点击"取消预约"按钮取消
4. 点击"刷新"按钮重新加载预约列表

---

# 数据库设计

预约数据库 `home_res`，存储预约订单、时段明细、审核记录。

## 表格设计

reservation_orders
- id
- order_no: 订单号，格式 R{14位时间戳}{4位hex}
- open_id: 微信用户标识
- applicant_name: 申请人姓名
- alumni_association: 所属学院校友会
- year: 入学年份
- major: 专业
- reason: 会议内容/预约理由
- phone: 联系电话
- total_slots: 预约时段数量（1~4）
- status: 0-待审核, 1-通过(终审通过), 2-拒绝, 3-完成, 4-取消, 5-待一级审核, 6-待二级审核, 7-一级驳回, 8-二级驳回, 9-终审通过
- created_at
- updated_at

reservation_slots
- id
- order_id: 关联订单ID（FK → reservation_orders.id，级联删除）
- start_time: 开始时间
- end_time: 结束时间
- status: 时段状态（与订单状态同步）
- password: 门锁动态密码（审核通过后由管理员设置）
- created_at
- updated_at

review_records
- id
- order_id: 关联订单ID
- reviewer_id: 审核人ID
- reviewer_role: 审核人角色（1-一级，2-二级）
- action: 操作（1-通过，2-拒绝）
- comment: 审核意见
- created_at

# API 设计

## 前端-后端的请求响应

所有接口路径前缀 `/api/reservation`，需携带 Bearer token（用户 JWT）。

### 提交预约
1. 请求: POST /api/reservation/reservation/submit
2. 请求头: Authorization: Bearer {token}
3. 请求体:
```json
{
    "applicant_name": "张三",
    "alumni_association": "计算机与软件学院校友会",
    "year": 2022,
    "major": "软件工程",
    "reason": "跨境电商交流会",
    "phone": "13829096726",
    "slots": [
        {
            "start_time": "2026-05-06 08:00:00",
            "end_time": "2026-05-06 10:00:00"
        },
        {
            "start_time": "2026-05-06 10:00:00",
            "end_time": "2026-05-06 12:00:00"
        }
    ]
}
```
4. 成功响应
```json
{
    "code": 200,
    "msg": "预约提交成功，共2个时段，请等待审核",
    "data": {
        "id": 1,
        "order_no": "R20260506103000a1b2",
        "applicant_name": "张三",
        "alumni_association": "计算机与软件学院校友会",
        "year": 2022,
        "major": "软件工程",
        "reason": "跨境电商交流会",
        "phone": "13829096726",
        "total_slots": 1,
        "status": 0,
        "status_text": "待审核",
        "created_at": "2026-05-06 10:30:00",
        "slots": [
            {
                "id": 1,
                "start_time": "2026-05-06 08:00",
                "end_time": "2026-05-06 12:00",
                "status": 0,
                "status_text": "待审核"
            }
        ]
    }
}
```
5. 失败响应
```json
{
    "code": 400,
    "msg": "请选择1-4个时间段"
}
```
```json
{
    "code": 400,
    "msg": "所选时段已被占用，请重新选择"
}
```
```json
{
    "code": 401,
    "msg": "未授权访问"
}
```

注意：如果提交的时段在时间上连续（如前一个时段的 end_time 等于后一个时段的 start_time 且在同一天内），后端会自动合并为一个时段。

---

### 获取我的预约
1. 请求: GET /api/reservation/reservation/my
2. 请求头: Authorization: Bearer {token}
3. 请求体: 无
4. 成功响应
```json
{
    "code": 200,
    "msg": "success",
    "data": [
        {
            "id": 1,
            "order_no": "R20260506103000a1b2",
            "applicant_name": "张三",
            "alumni_association": "计算机与软件学院校友会",
            "year": 2022,
            "major": "软件工程",
            "reason": "跨境电商交流会",
            "phone": "13829096726",
            "total_slots": 1,
            "status": 0,
            "status_text": "待审核",
            "created_at": "2026-05-06 10:30:00",
            "slots": [
                {
                    "id": 1,
                    "start_time": "2026-05-06 08:00",
                    "end_time": "2026-05-06 12:00",
                    "status": 0,
                    "status_text": "待审核"
                }
            ]
        }
    ]
}
```
5. 失败响应
```json
{
    "code": 401,
    "msg": "未授权访问"
}
```

---

### 获取已占用时段（日历展示）
1. 请求: GET /api/reservation/reservation/occupied?date=2026-05-06
2. 请求头: Authorization: Bearer {token}
3. 查询参数: date（可选，格式 YYYY-MM-DD，默认当天）
4. 成功响应
```json
{
    "code": 200,
    "msg": "success",
    "data": [
        {
            "start_time": "2026-05-06 08:00",
            "end_time": "2026-05-06 10:00",
            "status": "approved"
        },
        {
            "start_time": "2026-05-06 13:00",
            "end_time": "2026-05-06 15:00",
            "status": "pending"
        }
    ]
}
```
5. 说明
   - status 为 "approved" 表示该时段已被审核通过，显示为红色不可选（已占用）
   - status 为 "pending" 表示该时段有待审核的预约，显示为黄色不可选（待审核）
   - 未被占用的时段不返回

---

### 取消预约
1. 请求: DELETE /api/reservation/reservation/:id
2. 请求头: Authorization: Bearer {token}
3. 路径参数: id（预约订单ID）
4. 成功响应
```json
{
    "code": 200,
    "msg": "取消成功"
}
```
5. 失败响应
```json
{
    "code": 400,
    "msg": "该预约状态不允许取消"
}
```
```json
{
    "code": 400,
    "msg": "只能取消自己的预约"
}
```

说明：只有状态为"待审核(0)"或"已通过(1)"的预约可以取消，且只能取消自己的预约。

---

# 前端页面设计

1. 页面每 10 分钟自动检查 token 过期状态
2. 日历支持前后翻周，限制在当前周 ~ 未来2周范围内
3. 日历网格在手机上支持横向滚动（8列：1列时间标签 + 7列日期）
4. 校友会输入框支持模糊搜索和键盘上下键选择
5. 会议内容输入框实时显示字数统计
6. 提交前弹出确认弹窗，展示所有填写信息供核对

# 业务规则与约束

1. 每个预约最多选择 4 个时段
2. 连续时段自动合并（如前一个 end_time 等于后一个 start_time 且在同一天）
3. 提交时使用 SELECT ... FOR UPDATE 行锁防止重复预订
4. 只有待审核(0)和待一级审核(5)状态的预约可以取消（前端限制）
5. 只能取消自己的预约（校验 openid）
6. 已过期的时段和超出 14 天范围的日期不可选
7. 同一时段如果已有 approved 或 pending 状态的预约，则不可再选
8. 订单号格式: R + 14位时间戳(yyyyMMddHHmmss) + 4位十六进制随机数

# 遇到的重要问题与解决方式

### 跨手机排版错乱问题
不同手机打开预约页面排版不一致，日历网格在窄屏上严重错位。
- **原因**: 
  1. Tailwind CSS v2.2.19 CDN 不支持 `min-w-[560px]` 这种 JIT 任意值语法（该功能在 v3 才引入），导致日历网格在窄屏上没有最小宽度约束
  2. `<base href="/">` 在不同移动浏览器上处理方式不一致
  3. 缺少对刘海屏 safe-area、暗色模式强制覆盖、iOS 双击缩放等移动端特性的适配
- **解决**: 
  1. 在 CSS 中直接定义 `.calendar-grid { min-width: 560px }` 替代无效的 Tailwind class
  2. 移除 `<base href="/">`，资源路径改为绝对路径 `/static/...`
  3. 新增 5 层级响应式断点（≤374px / ≤639px / 640-768px / 高DPR / 暗色模式），添加 `touch-action: manipulation`、`safe-area-inset`、`100dvh` 等移动端适配

### 并发重复预订问题
多个用户同时提交相同时段的预约，可能导致重复预订。
- **解决**: 使用 MySQL `SELECT ... FOR UPDATE` 行锁，在事务中先锁定相同时段范围内状态为 pending/approved 的记录，确认无冲突后再插入新记录。

### Token 过期与跨页面传递
用户从预约页面跳转到"我的预约"页面时，token 需要在两个页面之间传递。
- **解决**: 使用 localStorage 存储 token，页面初始化时优先从 URL 参数读取，其次从 localStorage 读取。如果都没有，显示未授权提示。
