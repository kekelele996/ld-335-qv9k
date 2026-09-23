package model

import "time"

// AuditLog 审计日志。
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ClientID  uint      `gorm:"index" json:"client_id"`
	Method    string    `gorm:"size:10" json:"method"`
	Path      string    `gorm:"size:200" json:"path"`
	StatusCode int      `json:"status_code"`
	LatencyMs int64     `json:"latency_ms"`
	CreatedAt time.Time `json:"created_at"`
}
