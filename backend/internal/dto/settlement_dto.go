package dto

// CalculatePresettlementRequest 预结算请求。
type CalculatePresettlementRequest struct {
	BatchID uint `json:"batch_id" binding:"required"`
}

// SubmitSettlementRequest 正式结算请求。
type SubmitSettlementRequest struct {
	// RequestNo 调用方请求流水号（HIS 超时重试时保持不变），按调用方做幂等。
	PresettlementID uint   `json:"presettlement_id" binding:"required"`
	RequestNo       string `json:"request_no" binding:"required,max=64"`
}
