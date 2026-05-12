package auth

// ========== 管理员相关 DTO ==========

// LoginReq 管理员登录请求
type LoginReq struct {
	Username string `json:"username" binding:"required" example:"admin1"`
	Password string `json:"password" binding:"required" example:"123456"`
}

// LoginResp 登录响应
type LoginResp struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	RealName string `json:"real_name"`
	Role     int    `json:"role"`
	RoleText string `json:"role_text"`
}

// AdminInfoResp 管理员信息响应
type AdminInfoResp struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	RealName  string `json:"real_name"`
	Role      int    `json:"role"`
	RoleText  string `json:"role_text"`
	LastLogin string `json:"last_login,omitempty"`
}

// AdminResp 管理员接口统一响应结构
type AdminResp struct {
	Code int    `json:"code" example:"200"`
	Msg  string `json:"msg" example:"success"`
	Data any    `json:"data"`
}
