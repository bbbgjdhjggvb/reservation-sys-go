// Package constants 提供跨模块共享的常量定义。
// 订单状态常量定义在 pkg/reservationdb，业务代码应直接引用 reservationdb.StatusXxx。
package constants

// 管理员角色
const (
	RoleLevel1 = 1 // 一级管理员
	RoleLevel2 = 2 // 二级管理员
)

// RoleText 返回角色中文描述
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
