package dto

// AdminLoginRequest 网关管理员登录请求。
type AdminLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse 令牌响应。
type TokenResponse struct {
	Token string      `json:"token"`
	Data  interface{} `json:"data,omitempty"`
}
