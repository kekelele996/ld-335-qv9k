package model

// FeeItem 费用明细。
type FeeItem struct {
	ID             uint    `gorm:"primaryKey" json:"id"`
	BatchID        uint    `gorm:"index;not null" json:"batch_id"`
	ItemCode       string  `gorm:"size:32;not null" json:"item_code"`
	ItemName       string  `gorm:"size:100;not null" json:"item_name"`
	ItemType       string  `gorm:"size:20;not null" json:"item_type"`
	UnitPrice      float64 `json:"unit_price"`
	Quantity       float64 `json:"quantity"`
	Amount         float64 `json:"amount"`
	SelfPayRatio   float64 `json:"self_pay_ratio"`
	MedicalCategory string `gorm:"size:20;not null" json:"medical_category"`
}
