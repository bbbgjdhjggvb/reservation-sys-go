package auth

import (
	"context"
	"fmt"

	pb "reservation-sys/service/gateway/api/gen/account"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AccountGRPCServer 账号验证 gRPC 服务端。
// 由 Gateway 服务提供，供 Admin 服务远程调用验证管理员登录凭证，
// 避免 Admin 直接访问 home_xy 数据库的 admins 表。
type AccountGRPCServer struct {
	pb.UnimplementedAccountServiceServer
	adminRepo AdminRepository
}

// NewAccountGRPCServer 创建账号验证 gRPC 服务端。
//
// 参数:
//   - adminRepo: 管理员数据仓库接口（用于查询 admins 表）
//
// 返回值:
//   - *AccountGRPCServer: gRPC 服务端实例
func NewAccountGRPCServer(adminRepo AdminRepository) *AccountGRPCServer {
	return &AccountGRPCServer{adminRepo: adminRepo}
}

// VerifyAdmin 验证管理员登录凭证（gRPC 方法）。
//
// 流程:
//  1. 校验用户名和密码非空
//  2. 根据用户名查询 admins 表（WHERE status = 1 仅查正常账号）
//  3. 使用 bcrypt 比对密码哈希
//  4. 验证通过返回管理员信息，失败返回统一错误提示（防止用户名枚举）
//
// 参数:
//   - ctx: gRPC 上下文
//   - req: 验证请求（username + password）
//
// 返回值:
//   - *pb.VerifyAdminResp: 验证结果（success + admin_id + username + real_name + role + message）
//   - error: gRPC 内部错误（业务错误通过 resp.Success 标识）
func (s *AccountGRPCServer) VerifyAdmin(ctx context.Context, req *pb.VerifyAdminReq) (*pb.VerifyAdminResp, error) {
	if req.Username == "" || req.Password == "" {
		return &pb.VerifyAdminResp{
			Success: false,
			Message: "用户名和密码不能为空",
		}, nil
	}

	admin, err := s.adminRepo.FindAdminByUsername(req.Username)
	if err != nil {
		return &pb.VerifyAdminResp{
			Success: false,
			Message: "用户名或密码错误",
		}, nil
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)); err != nil {
		return &pb.VerifyAdminResp{
			Success: false,
			Message: "用户名或密码错误",
		}, nil
	}

	return &pb.VerifyAdminResp{
		Success:  true,
		AdminId:  uint32(admin.ID),
		Username: admin.Username,
		RealName: admin.RealName,
		Role:     int32(admin.Role),
		Message:  "验证成功",
	}, nil
}

// VerifyAdminViaGRPC 通过 gRPC 调用 Gateway 验证管理员凭证（供 Admin 服务使用）。
//
// 参数:
//   - client: AccountService 的 gRPC 客户端
//   - username: 管理员用户名
//   - password: 管理员密码（明文，传输由 gRPC TLS 保护）
//
// 返回值:
//   - uint: 管理员ID
//   - string: 用户名
//   - string: 真实姓名
//   - int: 角色等级（1:一级管理员, 2:二级管理员）
//   - error: gRPC 调用失败或凭证验证失败时返回错误
func VerifyAdminViaGRPC(client pb.AccountServiceClient, username, password string) (uint, string, string, int, error) {
	resp, err := client.VerifyAdmin(context.Background(), &pb.VerifyAdminReq{
		Username: username,
		Password: password,
	})
	if err != nil {
		return 0, "", "", 0, status.Errorf(codes.Internal, "账号验证服务调用失败: %v", err)
	}
	if !resp.Success {
		return 0, "", "", 0, fmt.Errorf("%s", resp.Message)
	}
	return uint(resp.AdminId), resp.Username, resp.RealName, int(resp.Role), nil
}
