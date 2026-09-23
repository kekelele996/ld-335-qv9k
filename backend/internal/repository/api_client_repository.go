package repository

import (
	"errors"

	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/util"
	"gorm.io/gorm"
)

// ApiClientRepository 调用方仓储。
type ApiClientRepository struct{ db *gorm.DB }

// NewApiClientRepository 构造调用方仓储。
func NewApiClientRepository(db *gorm.DB) *ApiClientRepository { return &ApiClientRepository{db: db} }

// Create 创建调用方。
func (r *ApiClientRepository) Create(client *model.ApiClient) error { return r.db.Create(client).Error }

// FindByID 按 ID 查询。
func (r *ApiClientRepository) FindByID(id uint) (*model.ApiClient, error) {
	var client model.ApiClient
	if err := r.db.First(&client, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.ErrNotFound
		}
		return nil, err
	}
	return &client, nil
}

// List 分页查询。
func (r *ApiClientRepository) List(page, pageSize int) ([]model.ApiClient, int64, error) {
	var total int64
	if err := r.db.Model(&model.ApiClient{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var clients []model.ApiClient
	err := r.db.Order("id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&clients).Error
	return clients, total, err
}

// Update 更新调用方。
func (r *ApiClientRepository) Update(client *model.ApiClient) error { return r.db.Save(client).Error }

// UpdateStatus 更新状态。
func (r *ApiClientRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.ApiClient{}).Where("id = ?", id).Update("status", status).Error
}

// Count 统计数量。
func (r *ApiClientRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.ApiClient{}).Count(&count).Error
	return count, err
}
