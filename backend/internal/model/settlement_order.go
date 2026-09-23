package model

import "time"

// SettlementOrder 结算单。
type SettlementOrder struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	SettlementNo     string     `gorm:"size:32;uniqueIndex;not null" json:"settlement_no"`
	// RequestNo 调用方（HIS）正式结算请求流水号，与 ClientID 构成幂等键。
	// 正式结算提交按 (client_id, request_no) 记住首次结果：重试返回原单，不重复开单。
	RequestNo        string     `gorm:"size:64;uniqueIndex:idx_order_client_request,priority:2" json:"request_no"`
	BatchID          uint       `gorm:"index;not null" json:"batch_id"`
	InsuredPersonID  uint       `gorm:"index;not null" json:"insured_person_id"`
	PresettlementID  uint       `gorm:"index;not null" json:"presettlement_id"`
	ClientID         uint       `gorm:"index;uniqueIndex:idx_order_client_request,priority:1;not null" json:"client_id"`
	Status           string     `gorm:"size:20;default:presettled" json:"status"`
	TotalAmount      float64    `json:"total_amount"`
	InsurancePayAmount float64  `json:"insurance_pay_amount"`
	SettledAt        *time.Time `json:"settled_at"`
	ReversedAt       *time.Time `json:"reversed_at"`
	CreatedAt        time.Time  `json:"created_at"`
}
