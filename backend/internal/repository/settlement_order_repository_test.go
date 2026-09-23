package repository

import (
	"testing"

	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newOrderRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SettlementOrder{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestSettlementOrderRepository_DuplicateRequestNo 验证并发首提落库时的幂等唯一索引冲突识别。
func TestSettlementOrderRepository_DuplicateRequestNo(t *testing.T) {
	db := newOrderRepoTestDB(t)
	repo := NewSettlementOrderRepository(db)

	first := &model.SettlementOrder{SettlementNo: "GB0001", ClientID: 1, RequestNo: "REQ-1", Status: "settled"}
	if err := repo.Create(first); err != nil {
		t.Fatalf("create first: %v", err)
	}
	dup := &model.SettlementOrder{SettlementNo: "GB0002", ClientID: 1, RequestNo: "REQ-1", Status: "settled"}
	err := repo.Create(dup)
	if err == nil {
		t.Fatal("expected unique constraint violation on (client_id, request_no)")
	}
	if !IsDuplicateRequestNoErr(err) {
		t.Fatalf("IsDuplicateRequestNoErr() = false for err: %v", err)
	}
	// 不同调用方复用同一 request_no 不冲突
	other := &model.SettlementOrder{SettlementNo: "GB0003", ClientID: 2, RequestNo: "REQ-1", Status: "settled"}
	if err := repo.Create(other); err != nil {
		t.Fatalf("create for other client: %v", err)
	}
}
