package grpc

import (
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Connect 建立到 gRPC 服务的连接
func Connect(addr string) *grpc.ClientConn {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("[error][grpc] 连接 %s 失败: %v", addr, err)
	}
	fmt.Printf("[info][grpc] 已连接到 %s\n", addr)
	return conn
}
