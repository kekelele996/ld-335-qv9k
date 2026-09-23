package repository

import (
	"errors"

	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/util"
	"gorm.io/gorm"
)

// DailyReconciliationRepository 日终对账仓储。
type DailyReconciliationRepository struct{ db *gorm.DB }

// NewDailyReconciliationRepository 构造日终对账仓储。
func NewDailyReconciliationRepository(db *gorm.DB) *DailyReconciliationRepository {
	return &DailyReconciliationRepository{db: db}
}

// Create 创建对账记录。
func (r *DailyReconciliationRepository) Create(rec *model.DailyReconciliation) error {
	return r.db.Create(rec).Error
}

// FindByDate 按日期查询。
func (r *DailyReconciliationRepository) FindByDate(date string) (*model.DailyReconciliation, error) {
	var rec model.DailyReconciliation
	if err := r.db.Where("reconcile_date = ?", date).First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}

// List 分页查询。
func (r *DailyReconciliationRepository) List(page, pageSize int) ([]model.DailyReconciliation, int64, error) {
	var total int64
	if err := r.db.Model(&model.DailyReconciliation{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.DailyReconciliation
	err := r.db.Order("reconcile_date desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}

// Update 更新对账记录。
func (r *DailyReconciliationRepository) Update(rec *model.DailyReconciliation) error {
	return r.db.Save(rec).Error
}
