package util

import (
	"fmt"

	"github.com/blueship581/gbinsureapi/internal/constants"
)

// SettlementCalculator 医保结算计算：起付线、报销比例、个人账户支付、自费金额。
type SettlementCalculator struct{}

// NewSettlementCalculator 构造计算器。
func NewSettlementCalculator() *SettlementCalculator { return &SettlementCalculator{} }

// Policy 结算政策（按参保地/医保类型）。
type Policy struct {
	Deductible        float64 // 起付线
	ReimbursementRate float64 // 报销比例
	ClassBRatio       float64 // 乙类纳入比例
	PersonalCap       float64 // 个人账户支付上限
}

// GetPolicy 返回医保类型对应的政策。
func (c *SettlementCalculator) GetPolicy(insuranceType string) Policy {
	switch insuranceType {
	case constants.InsuranceTypeEmployee:
		return Policy{Deductible: 500, ReimbursementRate: 0.8, ClassBRatio: 0.9, PersonalCap: 2000}
	case constants.InsuranceTypeResident:
		return Policy{Deductible: 300, ReimbursementRate: 0.7, ClassBRatio: 0.9, PersonalCap: 1000}
	case constants.InsuranceTypeNewRural:
		return Policy{Deductible: 200, ReimbursementRate: 0.65, ClassBRatio: 0.9, PersonalCap: 800}
	default:
		return Policy{Deductible: 500, ReimbursementRate: 0.7, ClassBRatio: 0.9, PersonalCap: 1000}
	}
}

// ItemResult 单条明细结算结果。
type ItemResult struct {
	ItemCode       string  `json:"item_code"`
	ItemName       string  `json:"item_name"`
	MedicalCategory string `json:"medical_category"`
	Amount         float64 `json:"amount"`
	BaseAmount     float64 `json:"base_amount"`   // 纳入报销基数金额
	InsurancePay   float64 `json:"insurance_pay"` // 统筹支付
	SelfPay        float64 `json:"self_pay"`      // 自费金额
}

// CalculateResult 预结算结果。
type CalculateResult struct {
	TotalAmount        float64      `json:"total_amount"`
	BaseAmount         float64      `json:"base_amount"`
	InsurancePayAmount float64      `json:"insurance_pay_amount"`
	PersonalAccountAmount float64   `json:"personal_account_amount"`
	SelfPayAmount      float64      `json:"self_pay_amount"`
	Deductible         float64      `json:"deductible"`
	ReimbursementRatio float64      `json:"reimbursement_ratio"`
	Items              []ItemResult `json:"items"`
}

// Calculate 按医保目录与参保地政策计算（items 需含 category 与 amount）。
func (c *SettlementCalculator) Calculate(insuranceType string, personalBalance float64, items []FeeInput) (*CalculateResult, error) {
	policy := c.GetPolicy(insuranceType)
	result := &CalculateResult{
		Deductible: policy.Deductible, ReimbursementRatio: policy.ReimbursementRate,
		Items: make([]ItemResult, 0, len(items)),
	}
	for _, it := range items {
		base := 0.0
		switch it.MedicalCategory {
		case constants.MedicalCategoryClassA:
			base = it.Amount
		case constants.MedicalCategoryClassB:
			base = it.Amount * policy.ClassBRatio
		case constants.MedicalCategoryClassC:
			base = 0
		default:
			return nil, fmt.Errorf("invalid medical category %q", it.MedicalCategory)
		}
		result.TotalAmount += it.Amount
		result.BaseAmount += base
		result.Items = append(result.Items, ItemResult{
			ItemCode: it.ItemCode, ItemName: it.ItemName, MedicalCategory: it.MedicalCategory,
			Amount: it.Amount, BaseAmount: base,
		})
	}
	reimburseBase := result.BaseAmount - policy.Deductible
	if reimburseBase < 0 {
		reimburseBase = 0
	}
	insurancePay := reimburseBase * policy.ReimbursementRate
	personal := insurancePay
	if personal > personalBalance {
		personal = personalBalance
	}
	if personal > policy.PersonalCap {
		personal = policy.PersonalCap
	}
	// 个人账户支付不能超过统筹支付
	if personal > insurancePay {
		personal = insurancePay
	}
	result.InsurancePayAmount = round2(insurancePay)
	result.PersonalAccountAmount = round2(personal)
	result.SelfPayAmount = round2(result.TotalAmount - insurancePay)
	// 回填单条明细的统筹支付与自费
	ratio := 1.0
	if result.BaseAmount > 0 {
		ratio = insurancePay / result.BaseAmount
	}
	for i := range result.Items {
		result.Items[i].InsurancePay = round2(result.Items[i].BaseAmount * ratio)
		result.Items[i].SelfPay = round2(result.Items[i].Amount - result.Items[i].InsurancePay)
	}
	return result, nil
}

// FeeInput 明细输入。
type FeeInput struct {
	ItemCode       string
	ItemName       string
	MedicalCategory string
	Amount         float64
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
