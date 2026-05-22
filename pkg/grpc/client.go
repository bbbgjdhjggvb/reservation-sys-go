package grpc

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Connect 建立到 gRPC 服务的连接
func Connect(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("[error][grpc] 连接 %s 失败: %w", addr, err)
	}
	fmt.Printf("[info][grpc] 已连接到 %s\n", addr)
	return conn, nil
}
