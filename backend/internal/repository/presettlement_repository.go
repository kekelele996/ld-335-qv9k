package repository

import (
	"errors"

	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/util"
	"gorm.io/gorm"
)

// PresettlementRepository 预结算仓储。
type PresettlementRepository struct{ db *gorm.DB }

// NewPresettlementRepository 构造预结算仓储。
func NewPresettlementRepository(db *gorm.DB) *PresettlementRepository {
	return &PresettlementRepository{db: db}
}

// Create 创建预结算。
func (r *PresettlementRepository) Create(p *model.Presettlement) error { return r.db.Create(p).Error }

// FindByID 按 ID 查询。
func (r *PresettlementRepository) FindByID(id uint) (*model.Presettlement, error) {
	var p model.Presettlement
	if err := r.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// ListByBatch 按批次查询（支持多次预结算比对）。
func (r *PresettlementRepository) ListByBatch(batchID uint) ([]model.Presettlement, error) {
	var items []model.Presettlement
	err := r.db.Where("batch_id = ?", batchID).Order("id desc").Find(&items).Error
	return items, err
}

// Count 统计。
func (r *PresettlementRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.Presettlement{}).Count(&count).Error
	return count, err
}
