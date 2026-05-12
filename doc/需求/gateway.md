# Gateway 模块功能介绍

Gateway 模块是系统的微信服务号接入层，负责与微信服务器的所有交互，同时为内部微服务提供 gRPC 远程调用服务。

主要职责：
1. 全局接口调用凭证（access_token）的获取和刷新
2. 网页授权回调（OAuth 2.0），将微信用户转为系统用户 JWT
3. 处理用户关注、取消关注、文本消息等微信事件
4. 提供身份验证（管理员登录验证）和信息推送（模板消息）的 gRPC 远程调用服务
5. 管理账号数据库（`home_xy`），包括用户表和管理员表

# 数据库设计

Gateway 独立负责账号数据库 `home_xy`，管理用户和管理员信息。

## 表格设计

users
- id
- openid: 微信唯一标识（UNIQUE）
- nickname: 微信昵称
- status: 1-正常, 0-已取消关注
- created_at
- updated_at
- last_login: 最后登录时间

admins
- id
- username: 登录账号（UNIQUE）
- password: bcrypt 哈希密码
- real_name: 真实姓名
- role: 1-一级管理员, 2-二级管理员
- status: 1-正常, 0-禁用
- last_login_at: 最后登录时间
- created_at
- updated_at

默认管理员账号（密码均为 `admin123` 的 bcrypt 哈希）：
- admin1：一级管理员
- admin2：二级管理员

# API 设计

## 微信服务号接入

### 全局 access_token 管理
调用微信 API 时需要 access_token 作为凭证，有效期 7200 秒。Gateway 使用第三方 SDK（`silenceper/wechat/v2`）自动管理 token 的获取和刷新。

1. 向微信服务器请求 token
    - GET https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=APPID&secret=APPSECRET
2. 微信服务器返回
```json
{
    "access_token": "ACCESS_TOKEN",
    "expires_in": 7200
}
```

### 服务器配置的初次验证
微信需要确认服务器归属。在微信服务号配置页面填写 token 和 URL（如 `http://106.52.23.213/wx`）后，微信服务器会立即向该 URL 发送 GET 请求验证。

[使用第三方 SDK 包完成验证](../../service/gateway/cmd/main.go#L114)。

### 网页授权回调（OAuth 2.0）
用户在微信服务号点击菜单按钮后，跳转到微信授权页面，授权完成后回调到 Gateway。

**跳转链接格式：**
```
https://open.weixin.qq.com/connect/oauth2/authorize?appid=APPID&redirect_uri=REDIRECT_URI&response_type=code&scope=SCOPE&state=STATE#wechat_redirect
```

- `redirect_uri`: 授权后回调链接，必须 URL Encode，域名必须与微信服务号配置的网页授权域名一致
- `scope`: `snsapi_base`（静默授权，只获取 openid）
- `state`: 区分不同跳转来源，Gateway 根据 state 映射不同的前端重定向地址

**回调处理流程：**

1. 请求: GET /api/gateway/auth/callback?code=CODE&state=STATE
2. Gateway 处理步骤：
    a. 提取 code 参数
    b. 调用微信 API 用 code 换取 openid（通过 `sns/oauth2/access_token` 接口）
    c. 调用 `jwt.GenerateUserToken(openid)` 签发用户 JWT（HS256，默认24小时过期）
    d. 根据 state 参数查找对应的前端重定向地址（从配置文件 `redirect_urls` 映射）
    e. HTTP 302 重定向到前端地址，携带 `?token={jwt}`
3. 前端收到 token 后存入 localStorage，后续请求携带 Bearer token
4. 失败响应（code 缺失时）
```json
{
    "code": 400,
    "msg": "缺少 code 参数，从微信服务号进入预约界面"
}
```

**state 与重定向地址映射（在配置文件中定义）：**
```yaml
wechat:
  default_redirect: "http://domain/reserve"
  redirect_urls:
    "reserve": "http://domain/reserve"
    "myorders": "http://domain/myorders"
```

## 微信事件处理

### 服务器消息入口
请求: ANY /wx

**GET 请求**: 微信服务器验证（返回 echostr）

**POST 请求**: 接收微信事件推送，根据事件类型分发处理：

| 事件类型 | 处理方式 | 回复内容 |
|---------|---------|---------|
| 关注（subscribe） | 调用 `FindOrCreate(openid)` 创建/更新用户记录 | "欢迎关注场地预约系统！\n点击下方菜单即可开始预约。" |
| 取消关注（unsubscribe） | 调用 `SetStatus(openid, false)` 更新用户状态 | 不回复 |
| 文本消息（text） | 自动回复客服信息 | "如有疑问，请咨询客服：1234567" |

### 用户信息同步
用户关注时，Gateway 通过微信 API 获取用户昵称，执行 upsert 操作：
```sql
INSERT INTO users (openid, nickname, status, last_login)
VALUES (?, ?, 1, NOW())
ON DUPLICATE KEY UPDATE
    nickname = VALUES(nickname),
    status = VALUES(status),
    last_login = VALUES(last_login),
    updated_at = VALUES(updated_at)
```

## gRPC 远程调用服务

Gateway 同时启动 HTTP 服务（端口 8080）和 gRPC 服务（端口 9080）。

### AccountService - 管理员身份验证

定义文件: `service/gateway/api/proto/account/account.proto`

```
service AccountService {
    rpc VerifyAdmin(VerifyAdminReq) returns (VerifyAdminResp);
}
```

**请求：**
```protobuf
message VerifyAdminReq {
    string username = 1;
    string password = 2;
}
```

**响应：**
```protobuf
message VerifyAdminResp {
    bool success = 1;
    uint32 admin_id = 2;
    string username = 3;
    string real_name = 4;
    int32 role = 5;
    string message = 6;
}
```

业务逻辑：查询 `admins` 表（条件 `status=1`），用 bcrypt 验证密码，返回管理员信息。

调用方：Admin 服务在管理员登录时通过 gRPC 调用此接口。

### NotificationService - 模板消息推送

定义文件: `service/gateway/api/proto/notification/notification.proto`

```
service NotificationService {
    rpc SendApprovalNotification(ApprovalNotificationReq) returns (NotificationResp);
    rpc SendRejectionNotification(RejectionNotificationReq) returns (NotificationResp);
}
```

**审核通过通知请求：**
```protobuf
message ApprovalNotificationReq {
    string openid = 1;
    string applicant_name = 2;
    string alumni_association = 3;
    string order_no = 4;
    repeated SlotInfo slots = 5;
}

message SlotInfo {
    string start_time = 1;
    string end_time = 2;
    string password = 3;
}
```

**审核驳回通知请求（额外包含驳回原因）：**
```protobuf
message RejectionNotificationReq {
    string openid = 1;
    string applicant_name = 2;
    string alumni_association = 3;
    string order_no = 4;
    repeated SlotInfo slots = 5;
    string reason = 6;
}
```

**响应：**
```protobuf
message NotificationResp {
    bool success = 1;
    string message = 2;
}
```

### 模板消息格式

**审核通过消息：**
```
您的场地预约已审核通过！
申请人：张三
预约时间：05-06 08:00~10:00（密码：123456）
所属校友会：计算机与软件学院校友会

订单号: R20260504103000a1b2
请凭门锁密码在预约时间段内使用场地。
```

**审核驳回消息：**
```
您的场地预约未通过审核。
申请人：张三
预约时间：05-06 08:00~10:00
所属校友会：计算机与软件学院校友会

驳回原因：场地已被占用，请重新选择时间
如有疑问请联系管理员。
```

调用方：Admin 服务在审核完成后通过 gRPC 调用此接口。

## JWT 设计

Gateway 使用两套独立的 JWT 体系，均采用 HMAC-SHA256 签名。

### 用户 JWT
| 项目 | 说明 |
|-----|------|
| 载荷 | `openid`（微信用户唯一标识）|
| 签发者 | `reservation-sys-user` |
| 签名算法 | HS256 |
| 过期时间 | 默认 24 小时 |
| 使用场景 | 前端调用 Reservation 服务 API 时的身份凭证 |

### 管理员 JWT
| 项目 | 说明 |
|-----|------|
| 载荷 | `admin_id`、`username`、`role`（1或2）|
| 签发者 | `reservation-sys-admin` |
| 签名算法 | HS256 |
| 过期时间 | 默认 24 小时 |
| 使用场景 | Admin 服务登录后管理员的身份凭证 |

两者使用不同的 secret 密钥，通过 `sync.Once` 单例模式初始化，确保密钥只加载一次。

# 前端页面

Gateway 本身不提供前端页面。OAuth 回调完成后，Gateway 通过 HTTP 302 重定向将用户引导至 Reservation 服务或 Admin 服务的前端页面，并在 URL 中携带 JWT token。

# 遇到的重要问题与解决方式

### access_token 全局唯一与并发刷新
微信的 access_token 每天有调用次数限制（2000次），且每次获取新 token 会使旧 token 失效。多个请求并发时可能导致重复刷新。
- **解决**: 使用第三方 SDK（`silenceper/wechat/v2`）内置的 token 管理机制，自动处理 token 的缓存和刷新，避免并发问题。

### 微信 OAuth 回调域名限制
微信要求回调域名必须与配置的网页授权域名一致，且只能配置一个域名。
- **解决**: 生产环境统一使用 Nginx 反向代理后的域名 `http://domain/api/gateway/auth/callback`，由 Nginx 将请求转发到 Gateway 服务。

### state 参数的安全使用
OAuth 的 state 参数用于防止 CSRF 攻击和区分跳转来源。
- **解决**: Gateway 在配置文件中维护 `redirect_urls` 映射表，根据 state 值匹配对应的前端重定向地址，不匹配时使用 `default_redirect`。

### 模板消息推送失败
微信模板消息可能因 access_token 过期、用户取消关注、模板 ID 不匹配等原因推送失败。
- **解决**: gRPC 响应中包含 `success` 和 `message` 字段，Admin 服务可以根据返回结果判断推送是否成功并向管理员展示结果。
