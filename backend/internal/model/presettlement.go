package model

import "time"

// Presettlement 预结算。
type Presettlement struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	BatchID              uint      `gorm:"index;not null" json:"batch_id"`
	InsuredPersonID      uint      `gorm:"index;not null" json:"insured_person_id"`
	TotalAmount          float64   `json:"total_amount"`
	InsurancePayAmount   float64   `json:"insurance_pay_amount"`
	PersonalAccountAmount float64  `json:"personal_account_amount"`
	SelfPayAmount        float64   `json:"self_pay_amount"`
	Deductible           float64   `json:"deductible"`
	ReimbursementRatio   float64   `json:"reimbursement_ratio"`
	ResultPayload        string    `gorm:"type:text" json:"result_payload"`
	CreatedAt            time.Time `json:"created_at"`
}
