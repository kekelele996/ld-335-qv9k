package dto

// CreateClientRequest 创建调用方请求。
type CreateClientRequest struct {
	Name         string `json:"name" binding:"required,max=100"`
	ClientType   string `json:"client_type" binding:"required,oneof=HIS THIRD_PARTY"`
	Role         string `json:"role" binding:"required"`
	RateLimitQPS int    `json:"rate_limit_qps" binding:"gte=1,lte=1000"`
}

// UpdateClientStatusRequest 更新调用方状态请求。
type UpdateClientStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active disabled"`
}

// ClientCreatedResponse 创建调用方响应（明文 API Key 仅此一次）。
type ClientCreatedResponse struct {
	ClientID uint   `json:"client_id"`
	Name     string `json:"name"`
	APIKey   string `json:"api_key"`
	Role     string `json:"role"`
}
