package util

import (
	"testing"

	"github.com/blueship581/gbinsureapi/internal/constants"
)

func TestSettlementCalculator_Calculate(t *testing.T) {
	calc := NewSettlementCalculator()
	tests := []struct {
		name          string
		insuranceType string
		balance       float64
		items         []FeeInput
		wantInsurance float64
		wantSelfPay   float64
		wantErr       bool
	}{
		{
			name: "employee class A full reimburse",
			insuranceType: constants.InsuranceTypeEmployee,
			balance:  3000,
			items:    []FeeInput{{ItemCode: "DRUG01", ItemName: "阿莫西林", MedicalCategory: constants.MedicalCategoryClassA, Amount: 1000}},
			wantInsurance: 400, // (1000-500)*0.8
			wantSelfPay:   600,
		},
		{
			name: "class C fully self pay",
			insuranceType: constants.InsuranceTypeResident,
			balance:  1000,
			items:    []FeeInput{{ItemCode: "COS01", ItemName: "进口耗材", MedicalCategory: constants.MedicalCategoryClassC, Amount: 800}},
			wantInsurance: 0,
			wantSelfPay:   800,
		},
		{
			name: "invalid category",
			insuranceType: constants.InsuranceTypeEmployee,
			balance:  1000,
			items:    []FeeInput{{ItemCode: "X", ItemName: "x", MedicalCategory: "class_d", Amount: 100}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.Calculate(tt.insuranceType, tt.balance, tt.items)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Calculate() error = %v", err)
			}
			if result.InsurancePayAmount != tt.wantInsurance {
				t.Fatalf("InsurancePayAmount = %v, want %v", result.InsurancePayAmount, tt.wantInsurance)
			}
			if result.SelfPayAmount != tt.wantSelfPay {
				t.Fatalf("SelfPayAmount = %v, want %v", result.SelfPayAmount, tt.wantSelfPay)
			}
		})
	}
}
