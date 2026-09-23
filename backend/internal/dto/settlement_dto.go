package dto

// CalculatePresettlementRequest 预结算请求。
type CalculatePresettlementRequest struct {
	BatchID uint `json:"batch_id" binding:"required"`
}

// SubmitSettlementRequest 正式结算请求。
type SubmitSettlementRequest struct {
	PresettlementID uint `json:"presettlement_id" binding:"required"`
}
