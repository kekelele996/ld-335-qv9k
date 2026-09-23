package model

import "time"

// InsuredPerson 参保人。
type InsuredPerson struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	IDCardNo       string    `gorm:"size:18;uniqueIndex;not null" json:"id_card_no"`
	MedicalCardNo  string    `gorm:"size:32;uniqueIndex;not null" json:"medical_card_no"`
	Name           string    `gorm:"size:50;not null" json:"name"`
	InsuranceType  string    `gorm:"size:20;not null" json:"insurance_type"`
	InsuranceStatus string   `gorm:"size:20;not null" json:"insurance_status"`
	InsurancePlace string    `gorm:"size:100" json:"insurance_place"`
	PersonalBalance float64  `gorm:"default:0" json:"personal_balance"`
	CreatedAt      time.Time `json:"created_at"`
}
