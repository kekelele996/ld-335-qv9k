package dto

// CalculatePresettlementRequest 预结算请求。
type CalculatePresettlementRequest struct {
	BatchID uint `json:"batch_id" binding:"required"`
}

// SubmitSettlementRequest 正式结算请求。
// RequestNo 为调用方请求流水号（HIS 超时重试必须保持不变），按调用方维度做结算幂等。
type SubmitSettlementRequest struct {
	PresettlementID uint   `json:"presettlement_id" binding:"required"`
	RequestNo       string `json:"request_no" binding:"required,max=64"`
}
