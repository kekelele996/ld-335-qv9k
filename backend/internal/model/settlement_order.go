package model

import "time"

// SettlementOrder 结算单。
type SettlementOrder struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	SettlementNo     string     `gorm:"size:32;uniqueIndex;not null" json:"settlement_no"`
	BatchID          uint       `gorm:"index;not null" json:"batch_id"`
	InsuredPersonID  uint       `gorm:"index;not null" json:"insured_person_id"`
	PresettlementID  uint       `gorm:"index;not null" json:"presettlement_id"`
	ClientID         uint       `gorm:"index;not null" json:"client_id"`
	Status           string     `gorm:"size:20;default:presettled" json:"status"`
	TotalAmount      float64    `json:"total_amount"`
	InsurancePayAmount float64  `json:"insurance_pay_amount"`
	SettledAt        *time.Time `json:"settled_at"`
	ReversedAt       *time.Time `json:"reversed_at"`
	CreatedAt        time.Time  `json:"created_at"`
}
