package constants

// SettlementStatus 结算状态枚举（README 枚举出现位置清单必列）。
const (
	SettlementPresettled  = "presettled"   // 已预结算
	SettlementSettled     = "settled"      // 已正式结算
	SettlementReversed    = "reversed"     // 已冲正
	SettlementFailed      = "failed"       // 失败
	SettlementPendingManual = "pending_manual" // 待人工处理
)

// SettlementStatuses 全部结算状态。
var SettlementStatuses = []string{
	SettlementPresettled, SettlementSettled, SettlementReversed,
	SettlementFailed, SettlementPendingManual,
}

// UploadStatus 上传批次状态。
const (
	UploadValidating = "validating" // 校验中
	UploadValidated  = "validated"  // 校验通过
	UploadDuplicate  = "duplicate"  // 重复
	UploadFailed     = "failed"     // 校验失败
)

// ClientStatus 调用方状态。
const (
	ClientActive   = "active"
	ClientDisabled = "disabled"
)
