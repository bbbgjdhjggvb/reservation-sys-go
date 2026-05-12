# 审核模块功能介绍

一共有两类管理员工：1级管理员、2级管理员。
在浏览器输入 `http://domain/admin` 将会
**进入账号登录界面**。
然后输入账号密码，登录进入管理员面板界面。

管理员认证通过 gRPC 调用 Gateway 服务完成，Admin 服务不直接访问 `admins` 表。

## 1级管理员
1. 登录注册账号（首次注册后自动登录）
2. 可以查看所有预约申请
3. 可以分类查看预约申请，分为 **待一级审核，待二级审核，已通过，已驳回**，其中：
    - 待一级审核（status=5）：新提交的预约，等待一级审核
    - 待二级审核（status=6）：一级审核已通过，等待二级审核
    - 已通过（status=1/9）：二级审核通过，终审通过
    - 一级驳回（status=7）：一级管理员拒绝
    - 二级驳回（status=8）：二级管理员拒绝
4. 点击申请条目，会弹出信息展示页面。
    1. 订单号: R20260504103000a1b2
    2. 申请人: 张三
    3. 学院: 计算机与软件学院
    4. 年级/专业: 2022级/软件工程
    5. 电话: 13829096726
    6. 会议内容: 跨境电商交流会
    7. 状态: (待一级审核，待二级审核，一级驳回，二级驳回，已通过)
    8. 时间段明细
    9. 审核记录: 记录审核人、通过/拒绝、原因、时间
5. 在申请条目处可以点击通过、拒绝按钮，并输入原因（最多500字）
6. 在二级审核完成后，预约请求通过（状态变为终审通过），1级管理员将可以给每个时间段设置密码（最多20位），并点击保存密码
7. 密码设置完成后，才可以点击"通知用户"，将会把时间段和密码从微信服务号中推送到用户身上
8. 如果一级或二级审核驳回，可以点击"通知用户（驳回）"，输入驳回原因，将驳回信息推送给用户

## 2级管理员
### 预约申请2级管理
1. 密码和账号已经预设好
2. 可以查看所有申请
3. 可以分类查看预约申请，分为**待二级审核，已驳回，已通过**，其中已驳回指的是已二级驳回，已通过指的是已二级通过
    1. 申请的详情如上
4. 在申请条目处可以点击通过、拒绝按钮，并输入原因

### 1级管理员管理（规划中）
1. 2级管理员可以查看所有1级管理员的信息
    1. 账号
    2. 姓名
    3. 电话
2. 2级管理员可以设置管理1级管理员账号
    1. 冻结
    2. 解冻

---

# 审核状态流转

新预约提交后状态为"待一级审核(5)"，经过两级审核：

```
待一级审核(5) ──一级通过──> 待二级审核(6) ──二级通过──> 终审通过(1)
     │                            │
     └──一级驳回──> 一级驳回(7)    └──二级驳回──> 二级驳回(8)
```

审核操作使用乐观锁（`WHERE status = fromStatus`），防止并发审核冲突。

---

# 数据库设计

账号数据库由 Gateway 管理（`home_xy.admins`），审核数据库 `home_res` 由 Admin 和 Reservation 共享。

## 表格设计

admins（位于 `home_xy`，由 Gateway 服务管理）
- id
- username
- password（bcrypt 哈希）
- real_name
- role: 1（1级管理员），2（2级管理员）
- status: 0（冻结），1（正常）
- last_login_at: 最后登录时间
- created_at
- updated_at

reservation_orders（位于 `home_res`）
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
- status: 0-待审核, 1-通过, 2-拒绝, 3-完成, 4-取消, 5-待一级审核, 6-待二级审核, 7-一级驳回, 8-二级驳回
- created_at
- updated_at

reservation_slots（位于 `home_res`）
- id
- order_id: 关联订单ID（FK → reservation_orders.id，级联删除）
- start_time: 开始时间
- end_time: 结束时间
- status: 时段状态（与订单状态同步）
- password: 门锁动态密码（审核通过后由1级管理员设置）
- created_at
- updated_at

review_records（位于 `home_res`）
- id
- order_id: 关联订单ID
- reviewer_id: 审核人ID
- reviewer_role: 审核人角色（1-一级，2-二级）
- action: 操作（1-通过，2-拒绝）
- comment: 审核意见
- created_at

# API 设计

所有接口路径前缀 `/api/admin`，除登录外均需携带 Bearer token（管理员 JWT）。

## 管理员登录
1. 请求: POST /api/admin/auth/login
2. 请求体:
```json
{
    "username": "admin1",
    "password": "admin123"
}
```
3. 成功响应
```json
{
    "code": 200,
    "msg": "登录成功",
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIs...",
        "username": "admin1",
        "real_name": "张三",
        "role": 1,
        "role_text": "一级管理员"
    }
}
```
4. 失败响应
```json
{
    "code": 401,
    "msg": "账号或密码错误"
}
```

说明：登录时 Admin 服务通过 gRPC 调用 Gateway 的 `AccountService.VerifyAdmin` 验证账号密码，验证通过后在本地签发管理员 JWT。

---

## 获取当前管理员信息
1. 请求: GET /api/admin/admin/info
2. 请求头: Authorization: Bearer {token}
3. 成功响应
```json
{
    "code": 200,
    "msg": "success",
    "data": {
        "id": 1,
        "username": "admin1",
        "role": 1,
        "role_text": "一级管理员"
    }
}
```
4. 失败响应
```json
{
    "code": 401,
    "msg": "未登录"
}
```

---

## 获取预约订单列表（分页）
1. 请求: GET /api/admin/orders?page=1&page_size=20&status=5&status=6
2. 请求头: Authorization: Bearer {token}
3. 查询参数:
    - page: 页码，默认 1
    - page_size: 每页条数，默认 20，最大 50
    - status: 筛选状态，可重复传多个（如 `?status=5&status=6`）
4. 成功响应
```json
{
    "code": 200,
    "msg": "success",
    "data": {
        "list": [
            {
                "id": 1,
                "order_no": "R20260504103000a1b2",
                "openid": "oXyz123...",
                "applicant_name": "张三",
                "alumni_association": "计算机与软件学院校友会",
                "year": 2022,
                "major": "软件工程",
                "reason": "跨境电商交流会",
                "phone": "13829096726",
                "total_slots": 1,
                "status": 5,
                "status_text": "待一级审核",
                "created_at": "2026-05-04 10:30:00",
                "slots": [
                    {
                        "id": 1,
                        "start_time": "2026-05-06 08:00",
                        "end_time": "2026-05-06 12:00",
                        "status": 5,
                        "status_text": "待一级审核",
                        "password": ""
                    }
                ]
            }
        ],
        "total": 42,
        "page": 1,
        "page_size": 20
    }
}
```
5. 失败响应
```json
{
    "code": 500,
    "msg": "查询失败"
}
```

---

## 获取订单详情（含审核记录）
1. 请求: GET /api/admin/orders/:id
2. 请求头: Authorization: Bearer {token}
3. 成功响应
```json
{
    "code": 200,
    "msg": "success",
    "data": {
        "order": {
            "id": 1,
            "order_no": "R20260504103000a1b2",
            "openid": "oXyz123...",
            "applicant_name": "张三",
            "alumni_association": "计算机与软件学院校友会",
            "year": 2022,
            "major": "软件工程",
            "reason": "跨境电商交流会",
            "phone": "13829096726",
            "total_slots": 1,
            "status": 5,
            "status_text": "待一级审核",
            "created_at": "2026-05-04 10:30:00",
            "slots": [
                {
                    "id": 1,
                    "start_time": "2026-05-06 08:00",
                    "end_time": "2026-05-06 12:00",
                    "status": 5,
                    "status_text": "待一级审核",
                    "password": ""
                }
            ]
        },
        "review_records": [
            {
                "id": 1,
                "reviewer_name": "管理员1",
                "reviewer_role": 1,
                "role_text": "一级管理员",
                "action": 1,
                "action_text": "通过",
                "comment": "材料齐全，同意",
                "created_at": "2026-05-05 09:00:00"
            }
        ]
    }
}
```
4. 失败响应
```json
{
    "code": 400,
    "msg": "订单不存在"
}
```

---

## 一级审核
1. 请求: POST /api/admin/review/level1/:id
2. 请求头: Authorization: Bearer {token}（需要一级管理员角色）
3. 请求体:
```json
{
    "action": 1,
    "comment": "校友身份已验证，通过一级审核"
}
```
    - action: 1-通过，2-拒绝
    - comment: 审核意见，最多500字
4. 成功响应
```json
{
    "code": 200,
    "msg": "一级审核通过成功"
}
```
```json
{
    "code": 200,
    "msg": "一级审核拒绝成功"
}
```
5. 失败响应
```json
{
    "code": 400,
    "msg": "当前订单状态不允许一级审核（仅待一级审核(5)状态可操作）"
}
```
```json
{
    "code": 403,
    "msg": "仅一级管理员可进行一级审核"
}
```

---

## 二级审核
1. 请求: POST /api/admin/review/level2/:id
2. 请求头: Authorization: Bearer {token}（需要二级管理员角色）
3. 请求体:
```json
{
    "action": 2,
    "comment": "场地使用时间与已有活动冲突，不予通过"
}
```
    - action: 1-通过，2-拒绝
    - comment: 审核意见，最多500字
4. 成功响应
```json
{
    "code": 200,
    "msg": "二级审核通过成功"
}
```
```json
{
    "code": 200,
    "msg": "二级审核拒绝成功"
}
```
5. 失败响应
```json
{
    "code": 400,
    "msg": "当前订单状态不允许二级审核（需先经一级审核通过(6)才可操作）"
}
```
```json
{
    "code": 403,
    "msg": "仅二级管理员可进行二级审核"
}
```

---

## 设置门锁密码
1. 请求: PUT /api/admin/review/level1/:id/slots/:slotID/password
2. 请求头: Authorization: Bearer {token}（需要一级管理员角色）
3. 请求体:
```json
{
    "password": "123456"
}
```
    - password: 门锁密码，最多20位
4. 成功响应
```json
{
    "code": 200,
    "msg": "门锁密码设置成功"
}
```
5. 失败响应
```json
{
    "code": 400,
    "msg": "仅审核通过的订单可设置门锁密码"
}
```
```json
{
    "code": 400,
    "msg": "时段不存在或状态不允许设置密码"
}
```

说明：仅当订单状态为"终审通过(1)"且时段状态为"已通过(1)"时，才可以设置密码。需在发送通知之前完成。

---

## 发送审核通过通知
1. 请求: POST /api/admin/review/level1/:id/notify
2. 请求头: Authorization: Bearer {token}（需要一级管理员角色）
3. 请求体: 无
4. 成功响应
```json
{
    "code": 200,
    "msg": "通知已发送给用户（订单号: R20260504103000a1b2）",
    "data": "微信消息发送成功"
}
```
5. 失败响应
```json
{
    "code": 400,
    "msg": "仅审核通过的订单可发送通知"
}
```
```json
{
    "code": 400,
    "msg": "请先设置门锁密码后再发送通知"
}
```

说明：通过 gRPC 调用 Gateway 的 `NotificationService.SendApprovalNotification`，Gateway 再通过微信模板消息推送给用户。微信消息中包含时段、门锁密码等信息。

---

## 发送审核驳回通知
1. 请求: POST /api/admin/review/level1/:id/reject-notify
2. 请求头: Authorization: Bearer {token}（需要一级管理员角色）
3. 请求体:
```json
{
    "reason": "场地已被占用，请重新选择时间"
}
```
4. 成功响应
```json
{
    "code": 200,
    "msg": "驳回通知已发送给用户（订单号: R20260504103000a1b2）",
    "data": "微信消息发送成功"
}
```
5. 失败响应
```json
{
    "code": 400,
    "msg": "仅驳回状态的订单可发送驳回通知（一级驳回(7)或二级驳回(8)）"
}
```

---

## 1级管理员账号注册（规划中）
1. 请求: POST /api/admin/auth/register
2. 请求体:
```json
{
    "username": "admin_new",
    "password": "admin123",
    "real_name": "李四",
    "phone": "13800138000"
}
```
3. 成功响应
```json
{
    "code": 200,
    "msg": "注册成功",
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIs...",
        "username": "admin_new",
        "real_name": "李四",
        "role": 1,
        "role_text": "一级管理员"
    }
}
```
4. 失败响应
```json
{
    "code": 409,
    "msg": "该用户名或手机号已被注册"
}
```

---

## 获取1级管理员列表（规划中）
1. 请求: GET /api/admin/admin/admins?page=1&page_size=20&status=active&status=lock
2. 成功响应:
```json
{
    "code": 200,
    "data": {
        "list": [
            {
                "id": 1,
                "username": "admin1",
                "real_name": "张三",
                "phone": "13800138000"
            }
        ],
        "total": 1,
        "page": 1,
        "page_size": 20
    }
}
```

## 冻结/解冻1级管理员账号（规划中）
1. 冻结: POST /api/admin/admin/admins/:id/freeze
2. 解冻: POST /api/admin/admin/admins/:id/unfreeze

---

# 前端页面设计

1. 登录页面（`index.html`）：输入账号密码登录
2. 管理面板页面（`dashboard.html`）：
    - 顶部导航：按状态分类 Tab（全部、待一级审核、待二级审核、已通过、已驳回）
    - 订单列表：分页展示，每条显示订单号、申请人、学院、状态标签、创建时间
    - 订单详情弹窗：展示完整信息 + 审核记录 + 操作按钮（通过/拒绝/设置密码/通知）
    - 每 10 分钟自动刷新一次页面，更新前端数据

# 遇到的重要问题与解决方式

### 乐观锁防止并发审核
两个管理员可能同时对同一订单进行审核，导致状态错乱。
- **解决**: 审核操作使用乐观锁，SQL 为 `UPDATE reservation_orders SET status = {to} WHERE id = {id} AND status = {from}`。如果 `RowsAffected == 0` 说明订单状态已被其他操作改变，返回错误提示"订单状态不匹配，无法执行此操作"。

### Admin 服务不直接访问账号数据库
Admin 服务需要验证管理员登录，但不希望直接访问 Gateway 管理的 `admins` 表。
- **解决**: Admin 服务通过 gRPC 调用 Gateway 的 `AccountService.VerifyAdmin` 完成认证，Gateway 是唯一直接操作 `admins` 表的服务，保持数据归属清晰。

### 通知发送与管理员工号绑定
通知需要通过微信服务号模板消息推送，Admin 服务不应直接持有微信 SDK 配置。
- **解决**: 通知通过 gRPC 调用 Gateway 的 `NotificationService` 完成，Gateway 统一管理微信 access_token 和模板消息发送。
