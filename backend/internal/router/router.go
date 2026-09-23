package router

import (
	"log/slog"
	"net/http"

	"github.com/blueship581/gbinsureapi/internal/config"
	"github.com/blueship581/gbinsureapi/internal/handler"
	"github.com/blueship581/gbinsureapi/internal/middleware"
	"github.com/blueship581/gbinsureapi/internal/repository"
	"github.com/blueship581/gbinsureapi/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handlers 全部接口处理器集合。
type Handlers struct {
	Auth          *handler.AuthHandler
	ApiClient     *handler.ApiClientHandler
	Insured       *handler.InsuredPersonHandler
	UploadBatch   *handler.UploadBatchHandler
	FeeItem       *handler.FeeItemHandler
	Presettlement *handler.PresettlementHandler
	Settlement    *handler.SettlementOrderHandler
	Recon         *handler.DailyReconciliationHandler
}

// New 装配 Gin 路由。
func New(cfg config.Config, log *slog.Logger, h Handlers, clientSvc *service.ApiClientService, auditRepo *repository.AuditLogRepository, limiter *middleware.RateLimiter) *gin.Engine {
	allowOrigins := cfg.CORSOriginsList()
	if len(allowOrigins) == 0 {
		allowOrigins = []string{"http://localhost:19935"}
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger(log))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key", "X-Request-ID"},
		AllowCredentials: true,
	}))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": gin.H{"status": "up"}})
	})
	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	jwtAuth := middleware.JwtAuthRequired(cfg.JWTSecret)
	adminAuth := middleware.AdminRequired()
	apiKeyAuth := middleware.ApiKeyRequired(clientSvc)

	api := r.Group("/api/v1")
	api.POST("/auth/login", h.Auth.AdminLogin)

	// 调用方管理（管理员 JWT）
	admin := api.Group("/clients")
	admin.Use(jwtAuth, adminAuth)
	{
		admin.POST("", h.ApiClient.Create)
		admin.GET("", h.ApiClient.List)
		admin.PUT("/:id/status", h.ApiClient.UpdateStatus)
	}

	// 调用方换取服务令牌（仅 API Key）
	api.POST("/clients/token", apiKeyAuth, h.ApiClient.IssueToken)

	// 业务接口：API Key + JWT 双认证 + 限流 + 审计
	biz := api.Group("")
	biz.Use(apiKeyAuth, jwtAuth, limiter.Limit(), middleware.AuditLog(auditRepo, log))
	{
		biz.POST("/clients/verify", h.Insured.Verify)
		biz.GET("/insured-persons", h.Insured.List)
		biz.POST("/fees/upload", h.FeeItem.Upload)
		biz.GET("/fees/items", h.FeeItem.ListItems)
		biz.GET("/batches", h.UploadBatch.List)
		biz.GET("/batches/:id", h.UploadBatch.Detail)
		biz.POST("/presettlements/calculate", h.Presettlement.Calculate)
		biz.GET("/presettlements", h.Presettlement.ListByBatch)
		biz.POST("/settlements/submit", h.Settlement.Submit)
		biz.POST("/settlements/:settlement_no/reverse", h.Settlement.Reverse)
		biz.GET("/settlements", h.Settlement.List)
		biz.GET("/settlements/:settlement_no", h.Settlement.Detail)
		biz.GET("/reconciliations/daily", h.Recon.Daily)
		biz.GET("/reconciliations", h.Recon.List)
	}
	return r
}
