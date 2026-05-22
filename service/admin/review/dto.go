package review

// ========== 请求结构 ==========

// ReviewActionReq 审核操作请求。
//
// 字段说明:
//   - Action: 审核动作（1=通过, 2=拒绝）
//   - Comment: 审核意见，最大500字
type ReviewActionReq struct {
	Action  int    `json:"action" binding:"required,oneof=1 2"`      // 1:通过, 2:拒绝
	Comment string `json:"comment" binding:"max=500" example:"审核通过"` // 审核意见
}

// SetPasswordReq 设置门锁密码请求。
//
// 字段说明:
//   - Password: 门锁密码，最大20字符
type SetPasswordReq struct {
	Password string `json:"password" binding:"required,max=20" example:"123456"` // 门锁密码
}

// RejectionNotifyReq 驳回通知请求。
//
// 字段说明:
//   - Reason: 驳回原因，最大500字
type RejectionNotifyReq struct {
	Reason string `json:"reason" binding:"max=500" example:"场地已被占用"` // 驳回原因
}

// ========== 响应结构 ==========

// ReviewRecordResp 审核记录响应
type ReviewRecordResp struct {
	ID           uint   `json:"id"`
	ReviewerName string `json:"reviewer_name"`
	ReviewerRole int    `json:"reviewer_role"`
	RoleText     string `json:"role_text"`
	Action       int    `json:"action"`
	ActionText   string `json:"action_text"`
	Comment      string `json:"comment"`
	CreatedAt    string `json:"created_at"`
}

// Response 统一响应结构
type Response struct {
	Code int    `json:"code" example:"200"`
	Msg  string `json:"msg" example:"success"`
	Data any    `json:"data"`
}

// ========== 工具函数 ==========

// ActionText 返回审核操作的中文描述。
//
// 参数:
//   - action: 审核动作（1=通过, 其他=拒绝）
//
// 返回值:
//   - string: "通过" 或 "拒绝"
func ActionText(action int) string {
	if action == 1 {
		return "通过"
	}
	return "拒绝"
}
