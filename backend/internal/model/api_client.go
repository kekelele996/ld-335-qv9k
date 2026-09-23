package model

import "time"

// ApiClient 调用方：HIS 或第三方，持 API Key 与接口权限。
type ApiClient struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	ClientType   string    `gorm:"size:20;not null" json:"client_type"`
	APIKeyHash   string    `gorm:"size:128;not null" json:"-"`
	Role         string    `gorm:"size:50;not null" json:"role"`
	Status       string    `gorm:"size:20;default:active" json:"status"`
	RateLimitQPS int       `gorm:"default:10" json:"rate_limit_qps"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
