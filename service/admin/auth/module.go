package auth

import (
	pb "reservation-sys/service/gateway/api/gen/account"
)

var (
	adminService *AdminAuthService
	adminHandler *AdminAuthHandler
)

// InitModule 初始化管理员认证模块（通过 gRPC 调用 Gateway 验证凭证）。
//
// 参数:
//   - accountClient: Gateway 账号服务 gRPC 客户端
func InitModule(accountClient pb.AccountServiceClient) {
	adminService = NewAdminAuthService(accountClient)
	adminHandler = NewAdminAuthHandler(adminService)
}

// GetAdminAuthHandler 获取管理员认证处理器实例。
//
// 返回值:
//   - *AdminAuthHandler: 管理员认证处理器实例（未初始化时 panic）
func GetAdminAuthHandler() *AdminAuthHandler {
	if adminHandler == nil {
		panic("admin auth module not initialized")
	}
	return adminHandler
}
