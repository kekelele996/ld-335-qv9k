package model

import "time"

// UploadBatch 费用上传批次。
type UploadBatch struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	BatchNo         string    `gorm:"size:32;uniqueIndex;not null" json:"batch_no"`
	ClientID        uint      `gorm:"index;not null" json:"client_id"`
	InsuredPersonID uint      `gorm:"index;not null" json:"insured_person_id"`
	TotalAmount     float64   `json:"total_amount"`
	ItemCount       int       `json:"item_count"`
	UploadStatus    string    `gorm:"size:20;default:validating" json:"upload_status"`
	CreatedAt       time.Time `json:"created_at"`
}
