package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/repository"
	"github.com/blueship581/gbinsureapi/internal/util"
)

// SettlementService 结算服务：预结算计算、正式结算、当日冲正（复用 SettlementOrderRepository）。
type SettlementService struct {
	presetRepo  *repository.PresettlementRepository
	orderRepo   *repository.SettlementOrderRepository
	feeRepo     *repository.FeeItemRepository
	batchRepo   *repository.UploadBatchRepository
	insurance   *InsuranceService
	calculator  *util.SettlementCalculator
	log         *slog.Logger
}

// NewSettlementService 构造结算服务。
func NewSettlementService(presetRepo *repository.PresettlementRepository, orderRepo *repository.SettlementOrderRepository, feeRepo *repository.FeeItemRepository, batchRepo *repository.UploadBatchRepository, insurance *InsuranceService, calculator *util.SettlementCalculator, log *slog.Logger) *SettlementService {
	return &SettlementService{presetRepo: presetRepo, orderRepo: orderRepo, feeRepo: feeRepo, batchRepo: batchRepo, insurance: insurance, calculator: calculator, log: log}
}

// CalculatePresettlement 预结算计算（支持多次比对，不落库状态机）。
func (s *SettlementService) CalculatePresettlement(ctx context.Context, batchID uint) (*model.Presettlement, error) {
	batch, err := s.batchRepo.FindByID(batchID)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return nil, util.NotFoundError("费用批次（UploadBatch）不存在", err)
		}
		return nil, err
	}
	person, err := s.insurance.GetByID(ctx, batch.InsuredPersonID)
	if err != nil {
		return nil, err
	}
	items, err := s.feeRepo.ListByBatch(batchID)
	if err != nil {
		return nil, util.LogError(s.log, constants.LOG_PRESETTLEMENT_FAILED, fmt.Errorf("list fee items: %w", err))
	}
	inputs := make([]util.FeeInput, 0, len(items))
	for _, it := range items {
		inputs = append(inputs, util.FeeInput{
			ItemCode: it.ItemCode, ItemName: it.ItemName,
			MedicalCategory: it.MedicalCategory, Amount: it.Amount,
		})
	}
	result, err := s.calculator.Calculate(person.InsuranceType, person.PersonalBalance, inputs)
	if err != nil {
		return nil, util.LogError(s.log, constants.LOG_PRESETTLEMENT_FAILED, fmt.Errorf("calculate: %w", err))
	}
	preset := &model.Presettlement{
		BatchID: batch.ID, InsuredPersonID: person.ID,
		TotalAmount: result.TotalAmount, InsurancePayAmount: result.InsurancePayAmount,
		PersonalAccountAmount: result.PersonalAccountAmount, SelfPayAmount: result.SelfPayAmount,
		Deductible: result.Deductible, ReimbursementRatio: result.ReimbursementRatio,
		ResultPayload: MarshalResult(result),
	}
	if err := s.presetRepo.Create(preset); err != nil {
		return nil, util.LogError(s.log, constants.LOG_PRESETTLEMENT_FAILED, fmt.Errorf("create presettlement: %w", err))
	}
	s.log.InfoContext(ctx, constants.LOG_PRESETTLEMENT_CALCULATED, "preset_id", preset.ID, "batch_no", batch.BatchNo)
	return preset, nil
}

// SubmitSettlement 正式结算：确认预结算后生成唯一结算单号。
func (s *SettlementService) SubmitSettlement(ctx context.Context, clientID, presetID uint) (*model.SettlementOrder, error) {
	preset, err := s.presetRepo.FindByID(presetID)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return nil, util.NotFoundError(constants.MsgPresettleNotFound, err)
		}
		return nil, err
	}
	// 唯一结算单号（冲突重试）
	no := ""
	for i := 0; i < 5; i++ {
		seq, _ := s.orderRepo.Count()
		candidate := util.SettlementNo(seq + 1 + int64(i))
		exists, err := s.orderRepo.ExistsByNo(candidate)
		if err != nil {
			return nil, util.LogError(s.log, constants.LOG_SETTLEMENT_FAILED, fmt.Errorf("check settlement no: %w", err))
		}
		if !exists {
			no = candidate
			break
		}
	}
	if no == "" {
		return nil, util.InternalError(constants.MsgSettlementNoUnique, errors.New("settlement no conflict"))
	}
	now := time.Now()
	order := &model.SettlementOrder{
		SettlementNo: no, BatchID: preset.BatchID, InsuredPersonID: preset.InsuredPersonID,
		PresettlementID: preset.ID, ClientID: clientID, Status: constants.SettlementSettled,
		TotalAmount: preset.TotalAmount, InsurancePayAmount: preset.InsurancePayAmount,
		SettledAt: &now,
	}
	if err := s.orderRepo.Create(order); err != nil {
		return nil, util.LogError(s.log, constants.LOG_SETTLEMENT_FAILED, fmt.Errorf("create settlement order: %w", err))
	}
	s.log.InfoContext(ctx, constants.LOG_SETTLEMENT_SUBMITTED, "settlement_no", no, "preset_id", presetID)
	return order, nil
}

// ReverseSettlement 当日冲正（全额回退）。
func (s *SettlementService) ReverseSettlement(ctx context.Context, settlementNo string) (*model.SettlementOrder, error) {
	order, err := s.orderRepo.FindByNo(settlementNo)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return nil, util.NotFoundError(constants.MsgSettlementNotFound, err)
		}
		return nil, err
	}
	if order.Status == constants.SettlementReversed {
		return nil, util.ConflictError(constants.MsgReverseAlready, errors.New("already reversed"))
	}
	if order.SettledAt == nil || time.Since(*order.SettledAt) > 24*time.Hour {
		return nil, util.NewAppError(constants.CodeReverseNotToday, 409, constants.MsgReverseNotToday, errors.New("not same day"))
	}
	now := time.Now()
	order.Status = constants.SettlementReversed
	order.ReversedAt = &now
	if err := s.orderRepo.Update(order); err != nil {
		return nil, util.LogError(s.log, constants.LOG_SETTLEMENT_REVERSE_FAILED, fmt.Errorf("update settlement order: %w", err))
	}
	s.log.InfoContext(ctx, constants.LOG_SETTLEMENT_REVERSED, "settlement_no", settlementNo)
	return order, nil
}

// ListOrders 分页查询结算单。
func (s *SettlementService) ListOrders(ctx context.Context, clientID uint, status string, page, pageSize int) ([]model.SettlementOrder, int64, error) {
	return s.orderRepo.List(clientID, status, page, pageSize)
}

// GetOrder 查询结算单详情。
func (s *SettlementService) GetOrder(ctx context.Context, settlementNo string) (*model.SettlementOrder, error) {
	order, err := s.orderRepo.FindByNo(settlementNo)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return nil, util.NotFoundError(constants.MsgSettlementNotFound, err)
		}
		return nil, err
	}
	return order, nil
}

// ListPresettlements 查询批次预结算记录（多次比对）。
func (s *SettlementService) ListPresettlements(ctx context.Context, batchID uint) ([]model.Presettlement, error) {
	return s.presetRepo.ListByBatch(batchID)
}
