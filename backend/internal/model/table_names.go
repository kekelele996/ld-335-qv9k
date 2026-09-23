package model

// 显式指定表名，确保与 database/init.sql 中的建表名一致（GORM 对 InsuredPerson 的复数化不正确）。

func (ApiClient) TableName() string              { return "api_clients" }
func (InsuredPerson) TableName() string          { return "insured_persons" }
func (UploadBatch) TableName() string            { return "upload_batches" }
func (FeeItem) TableName() string                { return "fee_items" }
func (Presettlement) TableName() string          { return "presettlements" }
func (SettlementOrder) TableName() string        { return "settlement_orders" }
func (DailyReconciliation) TableName() string    { return "daily_reconciliations" }
func (AuditLog) TableName() string               { return "audit_logs" }
