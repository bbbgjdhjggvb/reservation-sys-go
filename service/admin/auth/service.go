package auth

import (
	"context"
	"fmt"
	"log"

	pb "reservation-sys/service/gateway/api/gen/account"

	"reservation-sys/pkg/jwt"
)

// AdminAuthService 管理员认证服务（通过 gRPC 调用 Gateway 验证凭证）。
// Admin 服务不再直接访问 admins 表，所有凭证验证通过 Gateway 的 AccountGRPCServer 完成。
type AdminAuthService struct {
	accountClient pb.AccountServiceClient
}

// NewAdminAuthService 创建管理员认证服务实例。
//
// 参数:
//   - accountClient: Gateway 的 AccountService gRPC 客户端
//
// 返回值:
//   - *AdminAuthService: 认证服务实例
func NewAdminAuthService(accountClient pb.AccountServiceClient) *AdminAuthService {
	return &AdminAuthService{accountClient: accountClient}
}

// Login 管理员登录（通过 gRPC 调用 Gateway 验证凭证，本地生成 JWT）。
//
// 流程:
//  1. 调用 Gateway gRPC VerifyAdmin 接口验证用户名密码
//  2. 验证通过后，本地签发 Admin JWT Token
//
// 参数:
//   - username: 管理员用户名
//   - password: 管理员密码（明文）
//
// 返回值:
//   - *AdminInfo: 管理员信息（ID, Username, RealName, Role）
//   - string: JWT Token 字符串
//   - error: gRPC 调用失败、凭证错误、Token 生成失败时返回错误
func (s *AdminAuthService) Login(username, password string) (*AdminInfo, string, error) {
	resp, err := s.accountClient.VerifyAdmin(context.Background(), &pb.VerifyAdminReq{
		Username: username,
		Password: password,
	})
	if err != nil {
		log.Printf("[error][auth/admin/Login] gRPC 调用失败: %v", err)
		return nil, "", fmt.Errorf("账号验证服务不可用")
	}
	if !resp.Success {
		return nil, "", fmt.Errorf("%s", resp.Message)
	}

	admin := &AdminInfo{
		ID:       uint(resp.AdminId),
		Username: resp.Username,
		RealName: resp.RealName,
		Role:     int(resp.Role),
	}

	// 本地生成 JWT Token
	token, err := jwt.GenerateAdminToken(admin.ID, admin.Username, admin.Role)
	if err != nil {
		return nil, "", fmt.Errorf("生成token失败")
	}

	return admin, token, nil
}

// AdminInfo 管理员信息（从 gRPC 响应构造，不直接访问数据库）。
//
// 字段说明:
//   - ID: 管理员主键ID
//   - Username: 登录用户名
//   - RealName: 真实姓名
//   - Role: 角色等级（1:一级管理员, 2:二级管理员）
type AdminInfo struct {
	ID       uint
	Username string
	RealName string
	Role     int
}
