package constants

// messages.go 集中管理后端返回文案与日志文案（屎山耦合点）。

const (
	MsgOK                     = "ok"
	MsgClientCreated          = "调用方创建成功"
	MsgClientTokenIssued      = "服务令牌签发成功"
	MsgInsuredVerified        = "参保人身份核验通过"
	MsgFeesUploaded           = "费用明细上传成功"
	MsgPresettlementDone      = "预结算计算完成"
	MsgSettlementDone         = "正式结算完成"
	MsgSettlementReversed     = "结算已冲正"
	MsgReconciliationDone     = "日终对账完成"
	MsgApiKeyInvalid          = "API Key（ApiClient.api_key_hash）校验失败"
	MsgClientDisabled         = "调用方（ApiClient）已被停用"
	MsgTokenInvalid           = "JWT（ApiClient token）无效或已过期"
	MsgInsuredNotFound        = "参保人（InsuredPerson）身份核验失败"
	MsgBatchDuplicate         = "费用批次（UploadBatch）内容重复"
	MsgBatchInvalid           = "费用明细（FeeItem）格式校验失败"
	MsgPresettleNotFound      = "预结算记录（Presettlement）不存在"
	MsgSettlementNotFound     = "结算单（SettlementOrder）不存在"
	MsgSettlementNoUnique     = "结算单号（SettlementOrder.settlement_no）生成冲突"
	MsgReverseNotToday        = "仅支持当日结算冲正（SettlementOrder）"
	MsgReverseAlready         = "结算单（SettlementOrder）已冲正，禁止重复操作"
	MsgRateLimited            = "请求频率超出调用方（ApiClient.rate_limit_qps）限制"
	MsgInternalError          = "服务内部错误"
	MsgParamInvalid           = "请求参数校验失败"
	MsgUnauthorized           = "未认证或认证失败"
	MsgForbidden              = "无权执行该操作"
)
