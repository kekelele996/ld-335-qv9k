package repository

import (
	"errors"
	"strings"

	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/util"
	"gorm.io/gorm"
)

// SettlementOrderRepository 结算单仓储。
type SettlementOrderRepository struct{ db *gorm.DB }

// NewSettlementOrderRepository 构造结算单仓储。
func NewSettlementOrderRepository(db *gorm.DB) *SettlementOrderRepository {
	return &SettlementOrderRepository{db: db}
}

// Create 创建结算单。
func (r *SettlementOrderRepository) Create(order *model.SettlementOrder) error {
	return r.db.Create(order).Error
}

// FindByNo 按结算单号查询。
func (r *SettlementOrderRepository) FindByNo(no string) (*model.SettlementOrder, error) {
	var order model.SettlementOrder
	if err := r.db.Where("settlement_no = ?", no).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.ErrNotFound
		}
		return nil, err
	}
	return &order, nil
}

// FindByRequestNo 按调用方 + request_no 查询首次结算结果（正式结算幂等键）。
func (r *SettlementOrderRepository) FindByRequestNo(clientID uint, requestNo string) (*model.SettlementOrder, error) {
	var order model.SettlementOrder
	if err := r.db.Where("client_id = ? AND request_no = ?", clientID, requestNo).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.ErrNotFound
		}
		return nil, err
	}
	return &order, nil
}

// IsDuplicateRequestNoErr 判断是否为 (client_id, request_no) 幂等唯一索引冲突
// （并发首提时由数据库兜底）。Postgres 文案为 "duplicate key value ... uq_order_client_request"，
// SQLite 文案为 "UNIQUE constraint failed: ... request_no"。
func IsDuplicateRequestNoErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "uq_order_client_request") {
		return true
	}
	return strings.Contains(msg, "request_no") &&
		(strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint"))
}

// ExistsByNo 结算单号是否存在。
func (r *SettlementOrderRepository) ExistsByNo(no string) (bool, error) {
	var count int64
	err := r.db.Model(&model.SettlementOrder{}).Where("settlement_no = ?", no).Count(&count).Error
	return count > 0, err
}

// Update 更新结算单。
func (r *SettlementOrderRepository) Update(order *model.SettlementOrder) error {
	return r.db.Save(order).Error
}

// List 分页查询。
func (r *SettlementOrderRepository) List(clientID uint, status string, page, pageSize int) ([]model.SettlementOrder, int64, error) {
	q := r.db.Model(&model.SettlementOrder{})
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var orders []model.SettlementOrder
	err := r.db.Where("client_id = ?", clientID).Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders).Error
	return orders, total, err
}

// TodaySettled 当日已结算（对账，date 为 Asia/Shanghai 日期串）。
func (r *SettlementOrderRepository) TodaySettled(clientID uint, date string) ([]model.SettlementOrder, error) {
	var orders []model.SettlementOrder
	q := r.db.Where("settled_at::date = ?", date)
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	err := q.Find(&orders).Error
	return orders, err
}

// Count 统计。
func (r *SettlementOrderRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.SettlementOrder{}).Count(&count).Error
	return count, err
}
