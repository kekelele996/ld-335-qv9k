package dto

// VerifyRequest 参保人身份核验请求。
type VerifyRequest struct {
	IDCardNo      string `json:"id_card_no" binding:"required"`
	MedicalCardNo string `json:"medical_card_no" binding:"required"`
}
