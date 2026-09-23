package dto

// FeeItemDTO 费用明细。
type FeeItemDTO struct {
	ItemCode       string  `json:"item_code" binding:"required,max=32"`
	ItemName       string  `json:"item_name" binding:"required,max=100"`
	ItemType       string  `json:"item_type" binding:"required,oneof=drug treatment consumable exam"`
	UnitPrice      float64 `json:"unit_price" binding:"gte=0"`
	Quantity       float64 `json:"quantity" binding:"gt=0"`
	SelfPayRatio   float64 `json:"self_pay_ratio" binding:"gte=0,lte=1"`
	MedicalCategory string `json:"medical_category" binding:"required,oneof=class_a class_b class_c"`
}

// UploadFeesRequest 费用上传请求。
type UploadFeesRequest struct {
	ClientID        uint         `json:"client_id" binding:"required"`
	InsuredPersonID uint         `json:"insured_person_id" binding:"required"`
	Items           []FeeItemDTO `json:"items" binding:"required,min=1,dive"`
}
