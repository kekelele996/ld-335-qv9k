package constants

// MedicalCategory 医保目录分类枚举（README 枚举出现位置清单必列）。
const (
	MedicalCategoryClassA = "class_a" // 甲类：全额纳入报销
	MedicalCategoryClassB = "class_b" // 乙类：按比例纳入报销
	MedicalCategoryClassC = "class_c" // 丙类：完全自费
)

// MedicalCategories 全部医保目录分类。
var MedicalCategories = []string{MedicalCategoryClassA, MedicalCategoryClassB, MedicalCategoryClassC}

// InsuranceType 医保类型枚举。
const (
	InsuranceTypeEmployee  = "employee"  // 职工
	InsuranceTypeResident  = "resident"  // 居民
	InsuranceTypeNewRural  = "new_rural" // 新农合
)

// InsuranceTypes 全部医保类型。
var InsuranceTypes = []string{InsuranceTypeEmployee, InsuranceTypeResident, InsuranceTypeNewRural}

// InsuranceStatus 参保状态。
const (
	InsuranceStatusActive    = "active"    // 在职
	InsuranceStatusRetired   = "retired"   // 退休
	InsuranceStatusResident  = "resident"  // 居民医保
	InsuranceStatusInactive  = "inactive"  // 停保
)

// FeeItemType 费用明细类型。
const (
	FeeItemDrug        = "drug"       // 药品
	FeeItemTreatment   = "treatment"  // 诊疗
	FeeItemConsumable  = "consumable" // 耗材
	FeeItemExam        = "exam"       // 检查
)

// FeeItemTypes 全部费用明细类型。
var FeeItemTypes = []string{FeeItemDrug, FeeItemTreatment, FeeItemConsumable, FeeItemExam}

// ClientType 调用方类型。
const (
	ClientTypeHIS         = "HIS"          // 医院信息系统
	ClientTypeThirdParty  = "THIRD_PARTY"  // 第三方
)
