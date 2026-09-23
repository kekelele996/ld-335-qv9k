package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/repository"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.ApiClient{}, &model.InsuredPerson{}, &model.UploadBatch{}, &model.FeeItem{},
		&model.Presettlement{}, &model.SettlementOrder{}, &model.DailyReconciliation{}, &model.AuditLog{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// 内存 SQLite 按连接隔离，单连接保证各用例（含并发）共享同一库。
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
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

func TestSettlementService_SubmitAndReverse(t *testing.T) {
	db := newTestDB(t)
	clientID, personID := seedData(t, db)
	ctx := context.Background()
	insurance := NewInsuranceService(repository.NewInsuredPersonRepository(db), testLogger())
	batchRepo := repository.NewUploadBatchRepository(db)
	feeRepo := repository.NewFeeItemRepository(db)
	batch := model.UploadBatch{BatchNo: util.BatchNo(1), ClientID: clientID, InsuredPersonID: personID, UploadStatus: constants.UploadValidated, TotalAmount: 1000, ItemCount: 2}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	if err := feeRepo.CreateBatch([]model.FeeItem{
		{BatchID: batch.ID, ItemCode: "DRUG01", ItemName: "阿莫西林", ItemType: constants.FeeItemDrug, Amount: 600, MedicalCategory: constants.MedicalCategoryClassA},
		{BatchID: batch.ID, ItemCode: "EXAM01", ItemName: "血常规", ItemType: constants.FeeItemExam, Amount: 400, MedicalCategory: constants.MedicalCategoryClassA},
	}); err != nil {
		t.Fatal(err)
	}
	svc := NewSettlementService(
		repository.NewPresettlementRepository(db), repository.NewSettlementOrderRepository(db),
		feeRepo, batchRepo, insurance, util.NewSettlementCalculator(), testLogger(),
	)
	preset, err := svc.CalculatePresettlement(ctx, batch.ID)
	if err != nil {
		t.Fatalf("CalculatePresettlement() error = %v", err)
	}
	if preset.TotalAmount != 1000 {
		t.Fatalf("TotalAmount = %v, want 1000", preset.TotalAmount)
	}
	order, created, err := svc.SubmitSettlement(ctx, clientID, preset.ID, "REQ-001")
	if err != nil {
		t.Fatalf("SubmitSettlement() error = %v", err)
	}
	if !created || order.SettlementNo == "" || order.Status != constants.SettlementSettled || order.RequestNo != "REQ-001" {
		t.Fatalf("order invalid: %+v", order)
	}
	// 同一 request_no 重试：返回原单（幂等，不重复开单）
	order2, created2, err := svc.SubmitSettlement(ctx, clientID, preset.ID, "REQ-001")
	if err != nil {
		t.Fatalf("SubmitSettlement() replay error = %v", err)
	}
	if created2 {
		t.Fatal("replay must not create a new order")
	}
	if order2.SettlementNo != order.SettlementNo || order2.ID != order.ID {
		t.Fatalf("replay should return the original order: got no=%s id=%d, want no=%s id=%d",
			order2.SettlementNo, order2.ID, order.SettlementNo, order.ID)
	}
	// 不同 request_no 才会开出新单
	other, createdOther, err := svc.SubmitSettlement(ctx, clientID, preset.ID, "REQ-002")
	if err != nil || !createdOther {
		t.Fatalf("SubmitSettlement() other request_no error = %v created = %v", err, createdOther)
	}
	if other.SettlementNo == order.SettlementNo {
		t.Fatal("different request_no should generate a new settlement no")
	}
	// 冲正
	reversed, err := svc.ReverseSettlement(ctx, order.SettlementNo)
	if err != nil {
		t.Fatalf("ReverseSettlement() error = %v", err)
	}
	if reversed.Status != constants.SettlementReversed {
		t.Fatalf("status = %s, want reversed", reversed.Status)
	}
	// 冲正后用同一 request_no 重试：仍返回原单，状态为 reversed
	replay, created3, err := svc.SubmitSettlement(ctx, clientID, preset.ID, "REQ-001")
	if err != nil {
		t.Fatalf("SubmitSettlement() post-reverse replay error = %v", err)
	}
	if created3 || replay.SettlementNo != order.SettlementNo || replay.Status != constants.SettlementReversed {
		t.Fatalf("post-reverse replay should return original reversed order: %+v", replay)
	}
	// 旧 request_no 换到另一条预结算：冲突
	preset2, err := svc.CalculatePresettlement(ctx, batch.ID)
	if err != nil {
		t.Fatalf("CalculatePresettlement() 2nd error = %v", err)
	}
	if _, _, err := svc.SubmitSettlement(ctx, clientID, preset2.ID, "REQ-001"); err == nil {
		t.Fatal("expected conflict when old request_no switches to another presettlement")
	} else {
		var appErr *util.AppError
		if !errors.As(err, &appErr) || appErr.Code != constants.CodeSettleRequestConflict || appErr.HTTPStatus != 409 {
			t.Fatalf("expected 409/CodeSettleRequestConflict, got %v", err)
		}
	}
	// 重复冲正应冲突
	if _, err := svc.ReverseSettlement(ctx, order.SettlementNo); err == nil {
		t.Fatal("expected conflict on double reverse")
	}
}

// TestSettlementService_ConcurrentSameRequestNo 同一调用方同一 request_no 并发提交只生成一张结算单。
func TestSettlementService_ConcurrentSameRequestNo(t *testing.T) {
	db := newTestDB(t)
	clientID, personID := seedData(t, db)
	ctx := context.Background()
	insurance := NewInsuranceService(repository.NewInsuredPersonRepository(db), testLogger())
	batchRepo := repository.NewUploadBatchRepository(db)
	feeRepo := repository.NewFeeItemRepository(db)
	batch := model.UploadBatch{BatchNo: util.BatchNo(1), ClientID: clientID, InsuredPersonID: personID, UploadStatus: constants.UploadValidated, TotalAmount: 500, ItemCount: 1}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	if err := feeRepo.CreateBatch([]model.FeeItem{
		{BatchID: batch.ID, ItemCode: "DRUG01", ItemName: "阿莫西林", ItemType: constants.FeeItemDrug, Amount: 500, MedicalCategory: constants.MedicalCategoryClassA},
	}); err != nil {
		t.Fatal(err)
	}
	svc := NewSettlementService(
		repository.NewPresettlementRepository(db), repository.NewSettlementOrderRepository(db),
		feeRepo, batchRepo, insurance, util.NewSettlementCalculator(), testLogger(),
	)
	preset, err := svc.CalculatePresettlement(ctx, batch.ID)
	if err != nil {
		t.Fatalf("CalculatePresettlement() error = %v", err)
	}

	const n = 10
	var wg sync.WaitGroup
	errs := make(chan error, n)
	orders := make(chan *model.SettlementOrder, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			o, _, err := svc.SubmitSettlement(ctx, clientID, preset.ID, "REQ-CONCURRENT")
			if err != nil {
				errs <- err
				return
			}
			orders <- o
		}()
	}
	wg.Wait()
	close(errs)
	close(orders)
	for err := range errs {
		t.Fatalf("concurrent SubmitSettlement() error = %v", err)
	}
	var count int64
	if err := db.Model(&model.SettlementOrder{}).Where("client_id = ? AND request_no = ?", clientID, "REQ-CONCURRENT").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 settlement order, got %d", count)
	}
	first := <-orders
	for o := range orders {
		if o.SettlementNo != first.SettlementNo || o.ID != first.ID {
			t.Fatalf("all concurrent calls must return the same order: %s/%d vs %s/%d",
				o.SettlementNo, o.ID, first.SettlementNo, first.ID)
		}
	}
}

// TestSettlementService_DBUniqueIndexFallback 跨实例并发（绕过进程内锁）由唯一索引兜底，只保留一张单。
func TestSettlementService_DBUniqueIndexFallback(t *testing.T) {
	db := newTestDB(t)
	clientID, personID := seedData(t, db)
	batch := model.UploadBatch{BatchNo: util.BatchNo(1), ClientID: clientID, InsuredPersonID: personID, UploadStatus: constants.UploadValidated, TotalAmount: 100, ItemCount: 1}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	preset := model.Presettlement{BatchID: batch.ID, InsuredPersonID: personID, TotalAmount: 100, InsurancePayAmount: 80}
	if err := db.Create(&preset).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	orders := []*model.SettlementOrder{
		{SettlementNo: util.SettlementNo(1), RequestNo: "REQ-DB", BatchID: batch.ID, InsuredPersonID: personID, PresettlementID: preset.ID, ClientID: clientID, Status: constants.SettlementSettled, TotalAmount: 100, InsurancePayAmount: 80, SettledAt: &now},
		{SettlementNo: util.SettlementNo(2), RequestNo: "REQ-DB", BatchID: batch.ID, InsuredPersonID: personID, PresettlementID: preset.ID, ClientID: clientID, Status: constants.SettlementSettled, TotalAmount: 100, InsurancePayAmount: 80, SettledAt: &now},
	}
	repo := repository.NewSettlementOrderRepository(db)
	conflicts := 0
	for _, o := range orders {
		if err := repo.Create(o); err != nil {
			if !errors.Is(err, repository.ErrRequestNoConflict) {
				t.Fatalf("unexpected create error = %v", err)
			}
			conflicts++
		}
	}
	if conflicts != 1 {
		t.Fatalf("expected 1 unique conflict, got %d", conflicts)
	}
	var count int64
	if err := db.Model(&model.SettlementOrder{}).Where("client_id = ? AND request_no = ?", clientID, "REQ-DB").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 order, got %d", count)
	}
}
