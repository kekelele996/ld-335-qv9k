// GbInsureAPI 医保智能结算 API 网关
// @title 医保智能结算 API 网关
// @version 1.0
// @description 为医院信息系统提供标准化的医保结算接口服务：参保人身份核验、费用上传、预结算、正式结算、结算单查询与日终对账。业务接口需 API Key（X-API-Key）与 JWT（Authorization: Bearer）双认证。
// @host localhost:19935
// @BasePath /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/blueship581/gbinsureapi/docs"
	"github.com/blueship581/gbinsureapi/internal/config"
	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/handler"
	"github.com/blueship581/gbinsureapi/internal/middleware"
	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/repository"
	"github.com/blueship581/gbinsureapi/internal/router"
	"github.com/blueship581/gbinsureapi/internal/service"
	"github.com/blueship581/gbinsureapi/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}
	log := util.NewLogger()

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		panic(fmt.Errorf("open database: %w", err))
	}
	if err := migrateAndSeed(db, cfg, log); err != nil {
		panic(fmt.Errorf("migrate database: %w", err))
	}
	log.Info(constants.LOG_DB_INITIALIZED)

	clientRepo := repository.NewApiClientRepository(db)
	insuredRepo := repository.NewInsuredPersonRepository(db)
	batchRepo := repository.NewUploadBatchRepository(db)
	feeRepo := repository.NewFeeItemRepository(db)
	presetRepo := repository.NewPresettlementRepository(db)
	orderRepo := repository.NewSettlementOrderRepository(db)
	recRepo := repository.NewDailyReconciliationRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)

	calculator := util.NewSettlementCalculator()
	clientSvc := service.NewApiClientService(clientRepo, cfg.APIKeySecret, cfg.JWTSecret, cfg.JWTExpireHours, log)
	insuranceSvc := service.NewInsuranceService(insuredRepo, log)
	feeSvc := service.NewFeeService(batchRepo, feeRepo, insuranceSvc, log)
	settlementSvc := service.NewSettlementService(presetRepo, orderRepo, feeRepo, batchRepo, insuranceSvc, calculator, log)
	reconSvc := service.NewReconciliationService(orderRepo, recRepo, log)

	h := router.Handlers{
		Auth:          handler.NewAuthHandler(cfg, log),
		ApiClient:     handler.NewApiClientHandler(clientSvc, log),
		Insured:       handler.NewInsuredPersonHandler(insuranceSvc, log),
		UploadBatch:   handler.NewUploadBatchHandler(feeSvc, log),
		FeeItem:       handler.NewFeeItemHandler(feeSvc, log),
		Presettlement: handler.NewPresettlementHandler(settlementSvc, log),
		Settlement:    handler.NewSettlementOrderHandler(settlementSvc, log),
		Recon:         handler.NewDailyReconciliationHandler(reconSvc, log),
	}
	r := router.New(cfg, log, h, clientSvc, auditRepo, middleware.NewRateLimiter())

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Info(constants.LOG_SERVER_STARTED, "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(constants.LOG_SERVER_STARTED, "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error(constants.LOG_SERVER_SHUTDOWN, "error", err)
	}
}

// migrateAndSeed 自动迁移并注入种子数据；init.sql 已建表则跳过。
func migrateAndSeed(db *gorm.DB, cfg config.Config, log *slog.Logger) error {
	var tableCount int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='insured_persons'").Scan(&tableCount).Error; err != nil {
		return err
	}
	if tableCount > 0 {
		// init.sql 已建表：修复种子调用方的 API Key 哈希（与当前 API_KEY_SECRET 一致）
		return syncDemoClientHashes(db, cfg)
	}
	if err := db.AutoMigrate(
		&model.ApiClient{}, &model.InsuredPerson{}, &model.UploadBatch{}, &model.FeeItem{},
		&model.Presettlement{}, &model.SettlementOrder{}, &model.DailyReconciliation{}, &model.AuditLog{},
	); err != nil {
		return err
	}
	// 种子调用方（管理端演示）
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_ = hash
	clients := []model.ApiClient{
		{Name: "演示医院 HIS", ClientType: constants.ClientTypeHIS, APIKeyHash: util.HashAPIKey("ak_demo_his", cfg.APIKeySecret), Role: "settlement", Status: constants.ClientActive, RateLimitQPS: 20},
		{Name: "第三方药房", ClientType: constants.ClientTypeThirdParty, APIKeyHash: util.HashAPIKey("ak_demo_third", cfg.APIKeySecret), Role: "query", Status: constants.ClientActive, RateLimitQPS: 5},
	}
	if err := db.Create(&clients).Error; err != nil {
		return err
	}
	persons := []model.InsuredPerson{
		{IDCardNo: "110101199001011234", MedicalCardNo: "M110101199001011234", Name: "张三", InsuranceType: constants.InsuranceTypeEmployee, InsuranceStatus: constants.InsuranceStatusActive, InsurancePlace: "北京市", PersonalBalance: 3200},
		{IDCardNo: "310101198505052345", MedicalCardNo: "M310101198505052345", Name: "李四", InsuranceType: constants.InsuranceTypeResident, InsuranceStatus: constants.InsuranceStatusResident, InsurancePlace: "上海市", PersonalBalance: 1500},
		{IDCardNo: "440101197803036789", MedicalCardNo: "M440101197803036789", Name: "王五", InsuranceType: constants.InsuranceTypeNewRural, InsuranceStatus: constants.InsuranceStatusActive, InsurancePlace: "广州市", PersonalBalance: 600},
	}
	if err := db.Create(&persons).Error; err != nil {
		return err
	}
	log.Info(constants.LOG_DB_INITIALIZED, "seed", "ok")
	return syncDemoClientHashes(db, cfg)
}

// syncDemoClientHashes 确保演示调用方使用当前 API_KEY_SECRET 生成的哈希（init.sql 占位哈希不匹配）。
func syncDemoClientHashes(db *gorm.DB, cfg config.Config) error {
	demo := []struct {
		name  string
		key   string
	}{
		{name: "演示医院 HIS", key: "ak_demo_his"},
		{name: "第三方药房", key: "ak_demo_third"},
	}
	for _, d := range demo {
		if err := db.Model(&model.ApiClient{}).Where("name = ?", d.name).
			Update("api_key_hash", util.HashAPIKey(d.key, cfg.APIKeySecret)).Error; err != nil {
			return err
		}
	}
	return nil
}
