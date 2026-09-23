package service

import (
	"context"
	"testing"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/repository"
)

func TestFeeService_Upload(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	insurance := NewInsuranceService(repository.NewInsuredPersonRepository(db), testLogger())
	batchRepo := repository.NewUploadBatchRepository(db)
	feeRepo := repository.NewFeeItemRepository(db)
	svc := NewFeeService(batchRepo, feeRepo, insurance, testLogger())

	person := model.InsuredPerson{
		IDCardNo: "110101199001011234", MedicalCardNo: "M110101199001011234",
		Name: "张三", InsuranceType: constants.InsuranceTypeEmployee,
		InsuranceStatus: constants.InsuranceStatusActive, InsurancePlace: "北京市", PersonalBalance: 3000,
	}
	if err := db.Create(&person).Error; err != nil {
		t.Fatal(err)
	}

	result, err := svc.Upload(ctx, UploadInput{
		ClientID:        1,
		InsuredPersonID: person.ID,
		Items: []FeeItemInput{
			{ItemCode: "DRUG01", ItemName: "阿莫西林", ItemType: constants.FeeItemDrug, UnitPrice: 50, Quantity: 10, MedicalCategory: constants.MedicalCategoryClassA},
			{ItemCode: "EXAM01", ItemName: "血常规", ItemType: constants.FeeItemExam, UnitPrice: 40, Quantity: 1, MedicalCategory: constants.MedicalCategoryClassA},
		},
	})
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if result.BatchID == 0 || result.UploadStatus != constants.UploadValidated {
		t.Fatalf("upload result invalid: %+v", result)
	}
	if result.TotalAmount != 540 {
		t.Fatalf("TotalAmount = %v, want 540", result.TotalAmount)
	}
	if result.ItemCount != 2 {
		t.Fatalf("ItemCount = %d, want 2", result.ItemCount)
	}

	items, err := feeRepo.ListByBatch(result.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("persisted items = %d, want 2", len(items))
	}

	batch, err := batchRepo.FindByID(result.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	if batch.ItemCount != 2 || batch.TotalAmount != 540 {
		t.Fatalf("batch summary invalid: count=%d amount=%v", batch.ItemCount, batch.TotalAmount)
	}
}

func TestFeeService_UploadInvalidItemType(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	insurance := NewInsuranceService(repository.NewInsuredPersonRepository(db), testLogger())
	batchRepo := repository.NewUploadBatchRepository(db)
	feeRepo := repository.NewFeeItemRepository(db)
	svc := NewFeeService(batchRepo, feeRepo, insurance, testLogger())

	person := model.InsuredPerson{
		IDCardNo: "110101199001011234", MedicalCardNo: "M110101199001011234",
		Name: "张三", InsuranceType: constants.InsuranceTypeEmployee,
		InsuranceStatus: constants.InsuranceStatusActive, InsurancePlace: "北京市", PersonalBalance: 3000,
	}
	if err := db.Create(&person).Error; err != nil {
		t.Fatal(err)
	}

	result, err := svc.Upload(ctx, UploadInput{
		ClientID:        1,
		InsuredPersonID: person.ID,
		Items:           []FeeItemInput{{ItemCode: "BAD", ItemName: "非法项目", ItemType: "invalid", UnitPrice: 1, Quantity: 1, MedicalCategory: constants.MedicalCategoryClassA}},
	})
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if result.UploadStatus != constants.UploadFailed || len(result.Errors) == 0 {
		t.Fatalf("expected failed upload with errors, got %+v", result)
	}
}
