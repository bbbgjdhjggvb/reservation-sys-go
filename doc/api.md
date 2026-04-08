# API 接口文档

## 概述

本项目采用微服务架构设计，目前包含三个独立服务：

| 服务 | 端口 | 配置文件 | 说明 |
|------|------|----------|------|
| v1 - 微信服务 | :8080 | config_v1.yaml | 微信消息推送、OAuth 认证 |
| v2 - 预约服务 | :8081 | config_v2.yaml | 用户预约相关业务 |
| v3 - 管理服务 | :8082 | config_v3.yaml | 管理员审核功能 (待开发) |

---

## v1 - 微信服务 API

### 1. 微信消息推送

**接口**: `ANY /wx`

微信服务器消息入口，处理用户关注/取消关注等事件。

| 事件类型 | 说明 |
|----------|------|
| subscribe | 用户关注公众号，自动创建用户记录并返回欢迎消息 |
| unsubscribe | 用户取消关注，更新用户状态 |
| text | 普通文本消息，返回客服联系方式 |

**响应示例**:
```
欢迎关注场地预约系统！
点击下方菜单即可开始预约。
```

### 2. OAuth 认证回调

**接口**: `GET /api/v1/auth/callback`

**描述**: 微信 OAuth 授权回调，获取用户 OpenID 并签发 JWT Token

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | string | 是 | 微信授权码 |

**响应**:

| 状态码 | 说明 |
|--------|------|
| 302 | 重定向到预约前端页面，URL 携带 token 参数 |
| 400 | 缺少 code 参数 |
| 401 | 微信授权失效 |
| 500 | Token 生成失败 |

**重定向示例**:
```
Location: http://xxxx/reserve?token=eyJhbGciOiJIUzI1NiIs...
```

---

## v2 - 预约服务 API

> **认证说明**: 所有接口需要在 Header 中携带 JWT Token
> ```
> Authorization: Bearer <token>
> ```

### 1. 提交预约

**接口**: `POST /api/v2/reservation/submit`

**描述**: 用户提交场地预约申请

**请求体**:
```json
{
  "applicant_name": "张三",
  "alumni_association": "北京校友会",
  "reason": "校友聚会活动",
  "phone": "13800138000",
  "start_time": "2026-03-30 14:00:00",
  "end_time": "2026-03-30 18:00:00"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| applicant_name | string | 是 | 申请人姓名 |
| alumni_association | string | 是 | 所属校友会 |
| reason | string | 是 | 预约理由 (最大500字) |
| phone | string | 是 | 手机号码 (11位) |
| start_time | string | 是 | 开始时间 (格式: 2006-01-02 15:04:05) |
| end_time | string | 是 | 结束时间 |

**响应**:
```json
{
  "code": 200,
  "msg": "预约申请提交成功，请等待审核",
  "data": {
    "id": 1,
    "order_no": "R20260330140001",
    "applicant_name": "张三",
    "reason": "校友聚会活动",
    "phone": "13800138000",
    "start_time": "2026-03-30 14:00",
    "end_time": "2026-03-30 18:00",
    "status": 0,
    "status_text": "待审核",
    "created_at": "2026-03-30 10:00"
  }
}
```

### 2. 获取我的预约列表

**接口**: `GET /api/v2/reservation/my`

**描述**: 获取当前用户的所有预约记录

**响应**:
```json
{
  "code": 200,
  "data": [
    {
      "id": 1,
      "order_no": "R20260330140001",
      "applicant_name": "张三",
      "reason": "校友聚会活动",
      "phone": "13800138000",
      "start_time": "2026-03-30 14:00",
      "end_time": "2026-03-30 18:00",
      "status": 1,
      "status_text": "已通过",
      "created_at": "2026-03-30 10:00"
    }
  ]
}
```

### 3. 获取已占用时间段

**接口**: `GET /api/v2/reservation/occupied`

**描述**: 获取指定日期已被预约的时间段，供前端展示不可选时间

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| date | string | 否 | 查询日期 (格式: 2006-01-02)，默认今天 |

**响应**:
```json
{
  "code": 200,
  "data": [
    {
      "start_time": "2026-03-30 09:00",
      "end_time": "2026-03-30 12:00"
    },
    {
      "start_time": "2026-03-30 14:00",
      "end_time": "2026-03-30 18:00"
    }
  ]
}
```

### 4. 取消预约

**接口**: `DELETE /api/v2/reservation/:id`

**描述**: 用户取消自己的预约（仅限待审核状态）

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| id | uint | 预约记录 ID |

**响应**:
```json
{
  "code": 200,
  "msg": "取消成功"
}
```

---

## v3 - 管理服务 API (待开发)

> 管理员审核功能，计划包含：
> - 审核预约申请
> - 查看所有预约列表
> - 导出预约数据

---

## 错误码说明

| HTTP 状态码 | code | 说明 |
|-------------|------|------|
| 200 | 200 | 成功 |
| 400 | 400 | 请求参数错误 |
| 401 | 401 | 未授权/Token 无效 |
| 500 | 500 | 服务器内部错误 |

## 预约状态说明

| status | status_text | 说明 |
|--------|-------------|------|
| 0 | 待审核 | 刚提交，等待管理员审核 |
| 1 | 已通过 | 管理员审核通过 |
| 2 | 已拒绝 | 管理员审核拒绝 |
| 3 | 已完成 | 预约时间已过 |
| 4 | 已取消 | 用户主动取消 |

---

## 数据模型

### Reservation (预约表)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| order_no | string | 订单号 (唯一) |
| openid | string | 用户标识 (索引) |
| application_name | string | 申请人姓名 |
| alumni_association | string | 所属校友会 |
| reason | string | 预约理由 |
| phone | string | 手机号码 |
| num | int | 预约人数 |
| start_time | datetime | 开始时间 |
| end_time | datetime | 结束时间 |
| status | tinyint | 预约状态 |
| password | string | 门锁动态密码 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |
