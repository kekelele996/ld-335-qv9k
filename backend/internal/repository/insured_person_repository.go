package repository

import (
	"errors"

	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/util"
	"gorm.io/gorm"
)

// InsuredPersonRepository 参保人仓储。
type InsuredPersonRepository struct{ db *gorm.DB }

// NewInsuredPersonRepository 构造参保人仓储。
func NewInsuredPersonRepository(db *gorm.DB) *InsuredPersonRepository {
	return &InsuredPersonRepository{db: db}
}

// Create 创建参保人。
func (r *InsuredPersonRepository) Create(p *model.InsuredPerson) error { return r.db.Create(p).Error }

// FindByIDCardAndCardNo 按身份证号与医保卡号核验。
func (r *InsuredPersonRepository) FindByIDCardAndCardNo(idCard, medicalCard string) (*model.InsuredPerson, error) {
	var p model.InsuredPerson
	err := r.db.Where("id_card_no = ? AND medical_card_no = ?", idCard, medicalCard).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// FindByID 按 ID 查询。
func (r *InsuredPersonRepository) FindByID(id uint) (*model.InsuredPerson, error) {
	var p model.InsuredPerson
	if err := r.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// List 分页查询。
func (r *InsuredPersonRepository) List(page, pageSize int) ([]model.InsuredPerson, int64, error) {
	var total int64
	if err := r.db.Model(&model.InsuredPerson{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var persons []model.InsuredPerson
	err := r.db.Order("id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&persons).Error
	return persons, total, err
}
