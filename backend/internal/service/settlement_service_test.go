package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/repository"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// 共享内存库 + 单连接，保证并发用例能观察到唯一索引串行化效果
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.ApiClient{}, &model.InsuredPerson{}, &model.UploadBatch{}, &model.FeeItem{},
		&model.Presettlement{}, &model.SettlementOrder{}, &model.DailyReconciliation{}, &model.AuditLog{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Migrator().DropTable("settlement_orders", "presettlements", "fee_items", "upload_batches", "insured_persons", "api_clients", "daily_reconciliations", "audit_logs")
	})
	return db
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func seedData(t *testing.T, db *gorm.DB) (uint, uint) {
	t.Helper()
	client := model.ApiClient{Name: "测试HIS", ClientType: constants.ClientTypeHIS, APIKeyHash: "h", Role: "settlement", Status: constants.ClientActive, RateLimitQPS: 10}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	person := model.InsuredPerson{IDCardNo: "110101199001011234", MedicalCardNo: "M110101199001011234", Name: "张三", InsuranceType: constants.InsuranceTypeEmployee, InsuranceStatus: constants.InsuranceStatusActive, InsurancePlace: "北京市", PersonalBalance: 3000}
	if err := db.Create(&person).Error; err != nil {
		t.Fatal(err)
	}
	return client.ID, person.ID
}

func newSettlementSvc(db *gorm.DB) *SettlementService {
	insurance := NewInsuranceService(repository.NewInsuredPersonRepository(db), testLogger())
	return NewSettlementService(
		repository.NewPresettlementRepository(db), repository.NewSettlementOrderRepository(db),
		repository.NewFeeItemRepository(db), repository.NewUploadBatchRepository(db),
		insurance, util.NewSettlementCalculator(), testLogger(),
	)
}

func createPreset(t *testing.T, db *gorm.DB, clientID, personID uint, batchSuffix string) (uint, *SettlementService) {
	t.Helper()
	svc := newSettlementSvc(db)
	batch := model.UploadBatch{
		BatchNo: util.BatchNo(int64(len(batchSuffix))+1) + batchSuffix, ClientID: clientID, InsuredPersonID: personID,
		UploadStatus: constants.UploadValidated, TotalAmount: 1000, ItemCount: 2,
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.NewFeeItemRepository(db).CreateBatch([]model.FeeItem{
		{BatchID: batch.ID, ItemCode: "DRUG01", ItemName: "阿莫西林", ItemType: constants.FeeItemDrug, Amount: 600, MedicalCategory: constants.MedicalCategoryClassA},
		{BatchID: batch.ID, ItemCode: "EXAM01", ItemName: "血常规", ItemType: constants.FeeItemExam, Amount: 400, MedicalCategory: constants.MedicalCategoryClassA},
	}); err != nil {
		t.Fatal(err)
	}
	preset, err := svc.CalculatePresettlement(context.Background(), batch.ID)
	if err != nil {
		t.Fatalf("CalculatePresettlement() error = %v", err)
	}
	return preset.ID, svc
}

func conflictCode(err error) int {
	var appErr *util.AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return -1
}

func TestSettlementService_SubmitIdempotentRetry(t *testing.T) {
	db := newTestDB(t)
	clientID, personID := seedData(t, db)
	ctx := context.Background()
	presetID, svc := createPreset(t, db, clientID, personID, "a")

	// 首提
	order, replayed, err := svc.SubmitSettlement(ctx, clientID, presetID, "REQ-1001")
	if err != nil {
		t.Fatalf("SubmitSettlement() error = %v", err)
	}
	if replayed || order.Status != constants.SettlementSettled || order.RequestNo != "REQ-1001" {
		t.Fatalf("first submit invalid: replayed=%v order=%+v", replayed, order)
	}
	// HIS 超时后用同一 request_no 重试：回放原单，不再开新单
	retry, replayed, err := svc.SubmitSettlement(ctx, clientID, presetID, "REQ-1001")
	if err != nil {
		t.Fatalf("SubmitSettlement() retry error = %v", err)
	}
	if !replayed {
		t.Fatal("retry should be marked replayed")
	}
	if retry.SettlementNo != order.SettlementNo || retry.Status != constants.SettlementSettled {
		t.Fatalf("retry should replay first order: first=%s retry=%+v", order.SettlementNo, retry)
	}
	var count int64
	if err := db.Model(&model.SettlementOrder{}).Where("client_id = ? AND request_no = ?", clientID, "REQ-1001").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("idempotent key should map to exactly one order, got %d", count)
	}
}

func TestSettlementService_SubmitConcurrentSameRequestNo(t *testing.T) {
	db := newTestDB(t)
	clientID, personID := seedData(t, db)
	presetID, _ := createPreset(t, db, clientID, personID, "b")

	const n = 8
	var wg sync.WaitGroup
	type result struct {
		no  string
		err error
	}
	results := make([]result, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			svc := newSettlementSvc(db)
			<-start
			o, _, err := svc.SubmitSettlement(context.Background(), clientID, presetID, "REQ-CONC-1")
			if err == nil {
				results[idx] = result{no: o.SettlementNo}
			} else {
				results[idx] = result{err: err}
			}
		}(i)
	}
	close(start)
	wg.Wait()

	var firstNo string
	for i, r := range results {
		if r.err != nil {
			t.Fatalf("concurrent submit #%d error = %v", i, r.err)
		}
		if firstNo == "" {
			firstNo = r.no
		} else if r.no != firstNo {
			t.Fatalf("concurrent submits produced different settlement_no: %s vs %s", firstNo, r.no)
		}
	}
	var count int64
	if err := db.Model(&model.SettlementOrder{}).Where("client_id = ? AND request_no = ?", clientID, "REQ-CONC-1").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("concurrent same request_no created %d orders, want 1", count)
	}
}

func TestSettlementService_RequestNoReboundConflict(t *testing.T) {
	db := newTestDB(t)
	clientID, personID := seedData(t, db)
	ctx := context.Background()
	preset1, svc := createPreset(t, db, clientID, personID, "c")
	preset2, _ := createPreset(t, db, clientID, personID, "d")

	order, _, err := svc.SubmitSettlement(ctx, clientID, preset1, "REQ-2002")
	if err != nil {
		t.Fatalf("SubmitSettlement() error = %v", err)
	}
	// 旧 request_no 换到另一条预结算：必须报幂等冲突
	if _, _, err := svc.SubmitSettlement(ctx, clientID, preset2, "REQ-2002"); err == nil {
		t.Fatal("expected conflict when request_no rebound to another presettlement")
	} else if code := conflictCode(err); code != constants.CodeSettleIdempotentConflict {
		t.Fatalf("conflict code = %d, want %d (err=%v)", code, constants.CodeSettleIdempotentConflict, err)
	}
	// 原单不受影响
	if got, err := repository.NewSettlementOrderRepository(db).FindByNo(order.SettlementNo); err != nil || got.Status != constants.SettlementSettled {
		t.Fatalf("original order changed: %+v err=%v", got, err)
	}
}

func TestSettlementService_ReverseThenRetryReplaysReversed(t *testing.T) {
	db := newTestDB(t)
	clientID, personID := seedData(t, db)
	ctx := context.Background()
	presetID, svc := createPreset(t, db, clientID, personID, "e")

	order, _, err := svc.SubmitSettlement(ctx, clientID, presetID, "REQ-3003")
	if err != nil {
		t.Fatalf("SubmitSettlement() error = %v", err)
	}
	reversed, err := svc.ReverseSettlement(ctx, order.SettlementNo)
	if err != nil {
		t.Fatalf("ReverseSettlement() error = %v", err)
	}
	if reversed.Status != constants.SettlementReversed {
		t.Fatalf("status = %s, want reversed", reversed.Status)
	}
	// 冲正后 HIS 用同一 request_no 重试：仍返回原单与 reversed，不再开单
	retry, replayed, err := svc.SubmitSettlement(ctx, clientID, presetID, "REQ-3003")
	if err != nil {
		t.Fatalf("SubmitSettlement() post-reverse retry error = %v", err)
	}
	if !replayed || retry.SettlementNo != order.SettlementNo || retry.Status != constants.SettlementReversed {
		t.Fatalf("post-reverse retry should replay reversed order: replayed=%v retry=%+v", replayed, retry)
	}
	var count int64
	if err := db.Model(&model.SettlementOrder{}).Where("client_id = ? AND request_no = ?", clientID, "REQ-3003").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("post-reverse retry created extra orders, count=%d", count)
	}
}

func TestSettlementService_DifferentRequestNoSamePresetCreatesNewOrder(t *testing.T) {
	db := newTestDB(t)
	clientID, personID := seedData(t, db)
	ctx := context.Background()
	presetID, svc := createPreset(t, db, clientID, personID, "f")

	o1, _, err := svc.SubmitSettlement(ctx, clientID, presetID, "REQ-A")
	if err != nil {
		t.Fatalf("SubmitSettlement() #1 error = %v", err)
	}
	o2, _, err := svc.SubmitSettlement(ctx, clientID, presetID, "REQ-B")
	if err != nil {
		t.Fatalf("SubmitSettlement() #2 error = %v", err)
	}
	if o1.SettlementNo == o2.SettlementNo {
		t.Fatal("different request_no should produce different settlement orders")
	}
}
