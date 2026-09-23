package repository

import (
	"github.com/blueship581/gbinsureapi/internal/model"
	"gorm.io/gorm"
)

// AuditLogRepository 审计日志仓储。
type AuditLogRepository struct{ db *gorm.DB }

// NewAuditLogRepository 构造审计日志仓储。
func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository { return &AuditLogRepository{db: db} }

// Create 写入审计日志。
func (r *AuditLogRepository) Create(log *model.AuditLog) error { return r.db.Create(log).Error }

// ListByClient 按调用方查询审计日志。
func (r *AuditLogRepository) ListByClient(clientID uint, page, pageSize int) ([]model.AuditLog, int64, error) {
	q := r.db.Model(&model.AuditLog{}).Where("client_id = ?", clientID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []model.AuditLog
	err := r.db.Where("client_id = ?", clientID).Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}
