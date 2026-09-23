package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/repository"
	"github.com/blueship581/gbinsureapi/internal/util"
)

// InsuranceService 参保人服务：身份核验、参保状态与余额查询。
type InsuranceService struct {
	repo *repository.InsuredPersonRepository
	log  *slog.Logger
}

// NewInsuranceService 构造参保人服务。
func NewInsuranceService(repo *repository.InsuredPersonRepository, log *slog.Logger) *InsuranceService {
	return &InsuranceService{repo: repo, log: log}
}

// Verify 参保人身份核验。
func (s *InsuranceService) Verify(ctx context.Context, idCardNo, medicalCardNo string) (*model.InsuredPerson, error) {
	s.log.InfoContext(ctx, constants.LOG_INSURED_VERIFY_START, "id_card_masked", maskIDCard(idCardNo))
	p, err := s.repo.FindByIDCardAndCardNo(idCardNo, medicalCardNo)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return nil, util.NotFoundError(constants.MsgInsuredNotFound, err)
		}
		return nil, util.LogError(s.log, constants.LOG_INSURED_VERIFY_FAILED, fmt.Errorf("verify insured: %w", err))
	}
	s.log.InfoContext(ctx, constants.LOG_INSURED_VERIFY_SUCCESS, "insured_id", p.ID)
	return p, nil
}

// GetByID 查询参保人。
func (s *InsuranceService) GetByID(ctx context.Context, id uint) (*model.InsuredPerson, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return nil, util.NotFoundError("参保人（InsuredPerson）不存在", err)
		}
		return nil, err
	}
	return p, nil
}

// List 分页查询参保人。
func (s *InsuranceService) List(ctx context.Context, page, pageSize int) ([]model.InsuredPerson, int64, error) {
	return s.repo.List(page, pageSize)
}

func maskIDCard(idCard string) string {
	if len(idCard) < 8 {
		return "***"
	}
	return idCard[:4] + "**********" + idCard[len(idCard)-4:]
}
