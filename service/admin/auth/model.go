package auth

// Admin 模型已移至 Gateway 服务管理（账号数据库 home_xy）
// 本模块不再直接访问 admins 表，管理员验证通过 Gateway gRPC 调用完成

// 管理员角色常量（保持向后兼容，推荐使用 pkg/constants 包）
const (
	RoleLevel1 = 1 // 一级管理员
	RoleLevel2 = 2 // 二级管理员
)

// RoleText 返回管理员角色对应的中文描述。
//
// 参数:
//   - role: 管理员角色ID（1=一级管理员, 2=二级管理员）
//
// 返回值:
//   - string: 角色中文描述（未知角色返回空字符串）
func RoleText(role int) string {
	switch role {
	case RoleLevel1:
		return "一级管理员"
	case RoleLevel2:
		return "二级管理员"
	default:
		return ""
	}
}
