package constants

// 统一错误码：0 成功；1000 起业务错误。
const (
	CodeOK               = 0
	CodeBadRequest       = 1000
	CodeUnauthorized     = 1001
	CodeForbidden        = 1002
	CodeNotFound         = 1003
	CodeConflict         = 1004
	CodeValidationError  = 1005
	CodeRateLimited      = 1006
	CodeInternalError    = 1007
	CodeApiKeyInvalid    = 1101
	CodeClientDisabled   = 1102
	CodeInsuredNotFound  = 1201
	CodeBatchDuplicate   = 1301
	CodeBatchFailed      = 1302
	CodePresettleInvalid = 1401
	CodeSettleInvalid    = 1402
	CodeReverseNotToday  = 1403
	CodeReverseAlready   = 1404
)
