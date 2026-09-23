package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
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

	// submitLocks 按 (clientID, requestNo) 串行化同幂等键的并发提交，
	// 保证同一调用方同一 request_no 的并发请求在单实例内只开一张单；
	// 跨实例并发由 (client_id, request_no) 唯一索引兜底。
	submitLocks sync.Map
}

// lockRequest 取出幂等键对应的互斥锁。
func (s *SettlementService) lockRequest(clientID uint, requestNo string) *sync.Mutex {
	key := fmt.Sprintf("%d:%s", clientID, requestNo)
	actual, _ := s.submitLocks.LoadOrStore(key, &sync.Mutex{})
	return actual.(*sync.Mutex)
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
//
// 幂等语义：按调用方（clientID）与请求流水号（requestNo）记住首次结果。
//   - 同一 clientID + requestNo 重试（含正式结算响应超时后的 HIS 重试、并发请求）：
//     直接返回首次生成的结算单（含其当前状态，冲正后返回原单与 reversed），不重复开单、不重复累计日终金额；
//   - 同一 requestNo 换到另一条预结算（presetID 不同）：返回 409 冲突；
//   - 首次请求才校验预结算并开单。
//
// 返回 order 与 created（false 表示命中幂等、返回的是历史单）。
func (s *SettlementService) SubmitSettlement(ctx context.Context, clientID, presetID uint, requestNo string) (order *model.SettlementOrder, created bool, err error) {
	// 同幂等键并发在进程内串行，避免并发请求同时开出两张单。
	mu := s.lockRequest(clientID, requestNo)
	mu.Lock()
	defer mu.Unlock()

	// 1. 幂等检查：该调用方该 request_no 是否已结算过。
	existing, err := s.orderRepo.FindByClientAndRequest(clientID, requestNo)
	if err != nil && !errors.Is(err, util.ErrNotFound) {
		return nil, false, util.LogError(s.log, constants.LOG_SETTLEMENT_FAILED, fmt.Errorf("find by request_no[%s]: %w", requestNo, err))
	}
	if existing != nil {
		// 旧 request_no 换到另一条预结算：冲突，拒绝复用。
		if existing.PresettlementID != presetID {
			return nil, false, util.NewAppError(
				constants.CodeSettleRequestConflict, 409,
				fmt.Sprintf("%s：SettlementOrder[no=%s,request_no=%s] 已绑定 Presettlement[id=%d]，拒绝改用 Presettlement[id=%d]",
					constants.MsgSettlementRequestConflict, existing.SettlementNo, requestNo, existing.PresettlementID, presetID),
				repository.ErrRequestNoConflict)
		}
		// 重试（含冲正后的重试）：原样返回首次结算单与当前状态（settled/reversed）。
		s.log.InfoContext(ctx, constants.LOG_SETTLEMENT_IDEMPOTENT_REPLAY,
			"settlement_no", existing.SettlementNo, "request_no", requestNo, "status", existing.Status)
		return existing, false, nil
	}

	// 2. 首次请求：校验预结算。
	preset, err := s.presetRepo.FindByID(presetID)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return nil, false, util.NotFoundError(constants.MsgPresettleNotFound, err)
		}
		return nil, false, err
	}
	// 唯一结算单号（冲突重试）
	no := ""
	for i := 0; i < 5; i++ {
		seq, _ := s.orderRepo.Count()
		candidate := util.SettlementNo(seq + 1 + int64(i))
		exists, err := s.orderRepo.ExistsByNo(candidate)
		if err != nil {
			return nil, false, util.LogError(s.log, constants.LOG_SETTLEMENT_FAILED, fmt.Errorf("check settlement no: %w", err))
		}
		if !exists {
			no = candidate
			break
		}
	}
	if no == "" {
		return nil, false, util.InternalError(constants.MsgSettlementNoUnique, errors.New("settlement no conflict"))
	}
	now := time.Now()
	newOrder := &model.SettlementOrder{
		SettlementNo: no, RequestNo: requestNo,
		BatchID: preset.BatchID, InsuredPersonID: preset.InsuredPersonID,
		PresettlementID: preset.ID, ClientID: clientID, Status: constants.SettlementSettled,
		TotalAmount: preset.TotalAmount, InsurancePayAmount: preset.InsurancePayAmount,
		SettledAt: &now,
	}
	if err := s.orderRepo.Create(newOrder); err != nil {
		// 跨实例并发：唯一索引兜底，重读首次单并按幂等语义返回。
		if errors.Is(err, repository.ErrRequestNoConflict) {
			return s.reloadAfterConflict(ctx, clientID, presetID, requestNo)
		}
		return nil, false, util.LogError(s.log, constants.LOG_SETTLEMENT_FAILED, fmt.Errorf("create settlement order: %w", err))
	}
	s.log.InfoContext(ctx, constants.LOG_SETTLEMENT_SUBMITTED,
		"settlement_no", no, "request_no", requestNo, "preset_id", presetID)
	return newOrder, true, nil
}

// reloadAfterConflict 创建时命中幂等唯一索引（跨实例并发抢跑），重读已落库的首次单。
func (s *SettlementService) reloadAfterConflict(ctx context.Context, clientID, presetID uint, requestNo string) (*model.SettlementOrder, bool, error) {
	existing, err := s.orderRepo.FindByClientAndRequest(clientID, requestNo)
	if err != nil {
		return nil, false, util.LogError(s.log, constants.LOG_SETTLEMENT_FAILED, fmt.Errorf("reload by request_no[%s]: %w", requestNo, err))
	}
	if existing.PresettlementID != presetID {
		return nil, false, util.NewAppError(
			constants.CodeSettleRequestConflict, 409,
			fmt.Sprintf("%s：SettlementOrder[no=%s,request_no=%s] 已绑定 Presettlement[id=%d]，拒绝改用 Presettlement[id=%d]",
				constants.MsgSettlementRequestConflict, existing.SettlementNo, requestNo, existing.PresettlementID, presetID),
			repository.ErrRequestNoConflict)
	}
	s.log.InfoContext(ctx, constants.LOG_SETTLEMENT_IDEMPOTENT_REPLAY,
		"settlement_no", existing.SettlementNo, "request_no", requestNo, "status", existing.Status, "mode", "db_unique")
	return existing, false, nil
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
