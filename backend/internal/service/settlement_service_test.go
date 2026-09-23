package service

import (
	"context"
	"log/slog"
	"os"
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
	order, err := svc.SubmitSettlement(ctx, clientID, preset.ID)
	if err != nil {
		t.Fatalf("SubmitSettlement() error = %v", err)
	}
	if order.SettlementNo == "" || order.Status != constants.SettlementSettled {
		t.Fatalf("order invalid: %+v", order)
	}
	// 重复结算同一预结算应再次成功生成新单（幂等性由单号唯一保证）
	order2, err := svc.SubmitSettlement(ctx, clientID, preset.ID)
	if err != nil {
		t.Fatalf("SubmitSettlement() 2nd error = %v", err)
	}
	if order2.SettlementNo == order.SettlementNo {
		t.Fatal("settlement no should be unique")
	}
	// 冲正
	reversed, err := svc.ReverseSettlement(ctx, order.SettlementNo)
	if err != nil {
		t.Fatalf("ReverseSettlement() error = %v", err)
	}
	if reversed.Status != constants.SettlementReversed {
		t.Fatalf("status = %s, want reversed", reversed.Status)
	}
	// 重复冲正应冲突
	if _, err := svc.ReverseSettlement(ctx, order.SettlementNo); err == nil {
		t.Fatal("expected conflict on double reverse")
	}
}
