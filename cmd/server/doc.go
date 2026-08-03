// 校友之家预约场地系统
//
// 这个项目是要面向只懂 go 编程语法，没有实际项目的小白同学的。
//
// 下面将简单介绍这个项目，以及应用开发的一些概念。
//
// # 前端、后端、api
//
// 前端，后端是怎么工作的，前端，后端是什么：
//   - 前端: 负责从后端中拿数据，展示数据，获取用户输入的数据。
//   - 后端: 负责处理前端发送的请求，组织数据，存储数据，返回数据。
//   - api: 前端和后端通信的接口，前端通过调用 api 来获取数据，发送数据，后端通过处理 api 来接受数据，存储数据，返回数据。
//
// 后端其实就是写一个个 api 接口，前端就是调一个个的 api 接口。
// 后端在整个应用系统中，还扮演着网络安全，数据安全，响应优化的角色。
//
// # 包组织形式
//
//   - cmd/server/ : 程序入口
//
//   - internal/ : 私有应用代码。整个系统的核心业务代码都放在里面。根据功能划分为三个模块。auth、reservation、review。后续有其他独立功能都往里面加。
//
//   - 每个模块内部又分为三层：handler、service、store。
//     1. handler 层：用来对请求的参数进行校验。代码为 routes.go
//     2. service 层：用来实现业务逻辑。其他根据小功能命名的文件。
//     3. store 层：用来对数据库的表格进行 CRUD（增删改查）。store 文件夹下面。
//
//   - router/ : 对 Gin 框架的二次封装，实现函数式权限设置。
//
//   - permission/ : 权限定义
//
//   - middleware/ : 中间件
//
//   - platform/ : 其他应用依赖，比如 mysql、redis
//
//   - sse/ ： 服务器信息推送
//
//   - model/ : 数据库表格定义
//
// # 核心设计原则
//
//  1. 构造函数注入: 每个模块通过 NewXxx(db, cfg, deps...) 创建
//  2. 消费者定义接口: review 模块定义 Notifier、SlotStore 接口，auth/reservation 提供实现
//  3. 路由自治注册: 每个模块通过 RegisterRoutes 向 Registrar 注册路由，main.go 只负责模块组装
//  4. 进程内事件总线: event.Bus 替代 Redis Pub/Sub，Subscribe channel 驱动 SSE Hub 广播
//  5. 模块内三层分离: store（数据访问） → service（业务逻辑） → handler（HTTP） + routes（路由注册）
//  6. doc.go: 注释即文档，一些重要模块会为 doc.go 作为模块的介绍
package main
