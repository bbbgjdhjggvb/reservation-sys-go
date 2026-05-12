# 什么是 RPC
remote procedure call，远程过程调用，是一种计算机通信协议。允许一台计算机上的程序调用另外一台计算机上的子程序(或者函数)

## RPC 的工作流程
它是一个 server-client 结构，
- 客户端调用: 客户端代码调用本地的 `stub` (存根)函数
- 序列化: Stub将函数名和参数打包成网络可传输的字节流
- 网络传输: 通过网络协议(TCP，HTTP等)将字节流发送给服务端
- 反序列化: 服务端的 `Stub` 接收到数据后，将其解析为函数名和参数
- 服务端执行: 服务端执行对应的真实函数，并将结果原路返回给客户端

---

# gRPC
go 官方推荐新项目使用 gRPC，标准化的 `net/rpc` 不再添加新的特性

## gRPC 的特点
- 基于 `HTTP/2` 协议
- 使用 `Protocol Buffers`，一种高效的二进制序列化协议

---

## 如何使用 gRPC
### 1. 编写 proto 文件
例如文件 `service/gateway/api/proto/notification.proto`里面定义了 `gateway` 服务支持的 rpc，在 `review` 服务中，可以调用 `gateway` 服务中的 `SendApprovalNotification` 服务，然后 `gateway` 服务将会自己运行这个函数，进行通知。

proto 文件编写要求:
```proto
// 说明使用的版本
syntax = "proto3";

// 定义 rpc 服务实现文件所属包名
package = notification;

// 定义导入包的路径
option go_package = "reservation-sys/api/gen/notification"

// 定义接受参数
mess Request {

}

// 定义返回响应
mess Response{

}

// 定义 rpc 函数
service NotificationService {
    rpc SendNotification(Request) returns(Response);
}
```

### 2. 使用 protobuf 工具生成代码
1. 安装工具

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

2. 运行命令生成代码

```sh
protoc \
  # 决定了 notification.pb.go 文件的根路径
  --go_out="$PROJECT_ROOT/service/gateway/api/gen" \
  # 决定了相对于根路径的子目录要和 proto 文件一致
  --go_opt=paths=source_relative \                      
  # 决定了 notification_grpc.pb.go 文件的根路径
  --go-grpc_out="$PROJECT_ROOT/service/gateway/api/gen" \
  # 决定了相对于根路径的子目录要和 proto 文件一致
  --go-grpc_opt=paths=source_relative \
  # 指定 proto 文件的根路径
  -I "$PROJECT_ROOT/service/gateway/api/proto" \
  # 指定要用到的文件
  "$PROJECT_ROOT/.../notification.proto"
```

### 3. 生成的代码与自己需要手动实现的代码
运行 `protoc` 生成命令后会生成 `notification.pb.go` 文件和 `notification_grpc.pb.go`文件。
这两个文件都不需要修改

`notification.pb.go` 文件定义了信息结构体和一些常用方法。

`notification_grpc.pb.go`文件定义了客户端调用方法，和服务端需要实现的接口。
接口的实现需要自己完成。
```go
type NotificationServiceServer interface {
	// 发送审核通过通知
	SendApprovalNotification(context.Context, *ApprovalNotificationReq) (*NotificationResp, error)
	// 发送审核驳回通知
	SendRejectionNotification(context.Context, *RejectionNotificationReq) (*NotificationResp, error)
	mustEmbedUnimplementedNotificationServiceServer()
}
```

服务端的简单实现实例
```sh
// 结构体定义
type GRPCServer struct {
    pb.UnimplementedNotificationServiceServer // 必须嵌入，目的看是有实现所有接口
    ... // 其他字段
}

// 构造函数
func NewGPCServer() *GRPCServer {
    return &GRPCServer{
        UnimplementedNotificationServiceServer: pb.UnimplementedNotificationServiceServer{},
    }
}

// 实现接口
func (s *GRPCServer) SendApprovalNotification(ctx context.Context, req *pb.ApprovalNotificationReq) (*pb.NotificationResp, error) {
	
}

func (s *GRPCServer) SendRejectionNotification(ctx context.Context, req *pb.RejectionNotificationReq) (*pb.NotificationResp, error) {
	
}
```

### 4. 运行程序时注册 RPC 服务
```go
	grpcLis, err := net.Listen("tcp", cfg.Server.GRPCPort)
	if err != nil {
		log.Fatalf("[error][grpc] 监听失败: %v", err)
	}
	grpcSrv := grpc.NewServer()
	pb.RegisterNotificationServiceServer(grpcSrv, notification.NewGRPCServer(oa, cfg.Wechat.TemplateID))
	go func() {
		log.Printf("[info][grpc] Notification gRPC service started on %s", cfg.Server.GRPCPort)
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Fatalf("[error][grpc] gRPC serve failed: %v", err)
		}
	}()
```

### 5. 其他服务如何调用
```go
type NotifyHandler struct {
	notifyCli pb.NotificationServiceClient
	// 通过 gRPC 调用 v1 发送通知
	resp, err := h.notifyCli.SendApprovalNotification(c.Request.Context(), &pb.ApprovalNotificationReq{
		Openid:            order.Openid,
		ApplicantName:     order.ApplicantName,
		AlumniAssociation: order.AlumniAssociation,
		OrderNo:           order.OrderNo,
		Slots:             orderSlotsToNotify(order.Slots),
	})
	if err != nil {
		log.Printf("[error][review/notify] 发送通知失败: order_id=%d err=%v", orderID, err)
		c.JSON(http.StatusInternalServerError, auth.AdminResp{Code: 500, Msg: fmt.Sprintf("通知发送失败: %v", err)})
		return
	}
}
```