package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/repository"
	"github.com/blueship581/gbinsureapi/internal/util"
	"gorm.io/gorm"
)

// FeeService 费用服务：接收费用明细、格式校验、重复性检查、生成批次号。
type FeeService struct {
	batchRepo *repository.UploadBatchRepository
	feeRepo   *repository.FeeItemRepository
	insurance *InsuranceService
	log       *slog.Logger
}

// NewFeeService 构造费用服务。
func NewFeeService(batchRepo *repository.UploadBatchRepository, feeRepo *repository.FeeItemRepository, insurance *InsuranceService, log *slog.Logger) *FeeService {
	return &FeeService{batchRepo: batchRepo, feeRepo: feeRepo, insurance: insurance, log: log}
}

// FeeItemInput 费用明细入参。
type FeeItemInput struct {
	ItemCode       string  `json:"item_code" binding:"required,max=32"`
	ItemName       string  `json:"item_name" binding:"required,max=100"`
	ItemType       string  `json:"item_type" binding:"required,oneof=drug treatment consumable exam"`
	UnitPrice      float64 `json:"unit_price" binding:"gte=0"`
	Quantity       float64 `json:"quantity" binding:"gt=0"`
	SelfPayRatio   float64 `json:"self_pay_ratio" binding:"gte=0,lte=1"`
	MedicalCategory string `json:"medical_category" binding:"required,oneof=class_a class_b class_c"`
}

// UploadInput 费用上传入参。
type UploadInput struct {
	ClientID       uint           `json:"client_id" binding:"required"`
	InsuredPersonID uint          `json:"insured_person_id" binding:"required"`
	Items          []FeeItemInput `json:"items" binding:"required,min=1,dive"`
}

// UploadResult 上传结果。
type UploadResult struct {
	BatchNo      string            `json:"batch_no"`
	BatchID      uint              `json:"batch_id"`
	UploadStatus string            `json:"upload_status"`
	TotalAmount  float64           `json:"total_amount"`
	ItemCount    int               `json:"item_count"`
	FeeItems     []model.FeeItem   `json:"fee_items"`
	Errors       []string          `json:"errors,omitempty"`
}

// Upload 上传费用明细：格式校验 → 重复检查 → 生成批次号。
func (s *FeeService) Upload(ctx context.Context, input UploadInput) (*UploadResult, error) {
	s.log.InfoContext(ctx, constants.LOG_FEE_UPLOAD_START, "client_id", input.ClientID, "items", len(input.Items))
	if _, err := s.insurance.GetByID(ctx, input.InsuredPersonID); err != nil {
		return nil, err
	}
	// 重复性检查：同一调用方/参保人当日已上传则标记 Duplicate
	duplicate, err := s.batchRepo.ExistsByClientInsuredDate(input.ClientID, input.InsuredPersonID)
	if err != nil {
		return nil, util.LogError(s.log, constants.LOG_FEE_UPLOAD_FAILED, fmt.Errorf("check duplicate batch: %w", err))
	}
	if duplicate {
		return nil, util.ConflictError(constants.MsgBatchDuplicate, errors.New("batch duplicate for today"))
	}
	seq, _ := s.batchRepo.Count()
	batch := &model.UploadBatch{
		BatchNo: util.BatchNo(seq + 1), ClientID: input.ClientID,
		InsuredPersonID: input.InsuredPersonID, UploadStatus: constants.UploadValidating,
	}

	items := make([]model.FeeItem, 0, len(input.Items))
	errorsList := make([]string, 0)
	total := 0.0
	for _, it := range input.Items {
		if !containsType(constants.FeeItemTypes, it.ItemType) {
			errorsList = append(errorsList, fmt.Sprintf("FeeItem[item_code=%s] invalid item_type=%s", it.ItemCode, it.ItemType))
			continue
		}
		amount := round2(it.UnitPrice * it.Quantity)
		items = append(items, model.FeeItem{
			ItemCode: it.ItemCode, ItemName: it.ItemName,
			ItemType: it.ItemType, UnitPrice: it.UnitPrice, Quantity: it.Quantity,
			Amount: amount, SelfPayRatio: it.SelfPayRatio, MedicalCategory: it.MedicalCategory,
		})
		total += amount
	}
	if len(errorsList) > 0 {
		batch.UploadStatus = constants.UploadFailed
		if err := s.batchRepo.Create(batch); err != nil {
			return nil, util.LogError(s.log, constants.LOG_FEE_UPLOAD_FAILED, fmt.Errorf("create failed batch: %w", err))
		}
		s.log.InfoContext(ctx, constants.LOG_FEE_UPLOAD_FAILED, "batch_no", batch.BatchNo, "errors", len(errorsList))
		return &UploadResult{BatchNo: batch.BatchNo, BatchID: batch.ID, UploadStatus: constants.UploadFailed, Errors: errorsList}, nil
	}

	batch.TotalAmount = round2(total)
	batch.ItemCount = len(items)
	batch.UploadStatus = constants.UploadValidated
	err = s.batchRepo.Transaction(func(tx *gorm.DB) error {
		if err := s.batchRepo.WithTx(tx).Create(batch); err != nil {
			return fmt.Errorf("create batch: %w", err)
		}
		for i := range items {
			items[i].BatchID = batch.ID
		}
		if err := s.feeRepo.WithTx(tx).CreateBatch(items); err != nil {
			return fmt.Errorf("create fee items: %w", err)
		}
		if err := s.batchRepo.WithTx(tx).Update(batch); err != nil {
			return fmt.Errorf("update batch summary: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, util.LogError(s.log, constants.LOG_FEE_UPLOAD_FAILED, err)
	}
	s.log.InfoContext(ctx, constants.LOG_FEE_UPLOAD_VALIDATED, "batch_no", batch.BatchNo, "amount", total)
	return &UploadResult{
		BatchNo: batch.BatchNo, BatchID: batch.ID, UploadStatus: constants.UploadValidated,
		TotalAmount: round2(total), ItemCount: len(items), FeeItems: items,
	}, nil
}

// ListItemsByBatch 查询批次明细（结算单明细导出复用）。
func (s *FeeService) ListItemsByBatch(ctx context.Context, batchID uint) ([]model.FeeItem, error) {
	return s.feeRepo.ListByBatch(batchID)
}

// GetBatch 查询批次。
func (s *FeeService) GetBatch(ctx context.Context, batchID uint) (*model.UploadBatch, error) {
	batch, err := s.batchRepo.FindByID(batchID)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return nil, util.NotFoundError("费用批次（UploadBatch）不存在", err)
		}
		return nil, err
	}
	return batch, nil
}

// ListBatches 分页查询批次。
func (s *FeeService) ListBatches(ctx context.Context, clientID uint, page, pageSize int) ([]model.UploadBatch, int64, error) {
	return s.batchRepo.ListByClient(clientID, page, pageSize)
}

// MarshalResult 序列化结果负载。
func MarshalResult(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func containsType(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
