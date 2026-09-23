package model

import "time"

// DailyReconciliation 日终对账。
type DailyReconciliation struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ReconcileDate   string    `gorm:"size:10;uniqueIndex;not null" json:"reconcile_date"`
	TotalCount      int64     `json:"total_count"`
	TotalAmount     float64   `json:"total_amount"`
	SuccessCount    int64     `json:"success_count"`
	FailCount       int64     `json:"fail_count"`
	AbnormalOrders  int64     `json:"abnormal_orders"`
	CreatedAt       time.Time `json:"created_at"`
}
