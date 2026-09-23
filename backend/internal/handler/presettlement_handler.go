package handler

import (
	"log/slog"

	"github.com/blueship581/gbinsureapi/internal/dto"
	"github.com/blueship581/gbinsureapi/internal/service"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/gin-gonic/gin"
)

// PresettlementHandler 预结算接口。
type PresettlementHandler struct {
	svc *service.SettlementService
	log *slog.Logger
}

// NewPresettlementHandler 构造预结算接口。
func NewPresettlementHandler(svc *service.SettlementService, log *slog.Logger) *PresettlementHandler {
	return &PresettlementHandler{svc: svc, log: log}
}

// Calculate 预结算计算。
// @Summary 预结算计算
// @Description 按医保目录与参保地政策返回分项明细，支持多次比对
// @Tags presettlements
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param body body dto.CalculatePresettlementRequest true "批次 ID"
// @Success 200 {object} util.Response
// @Router /api/v1/presettlements/calculate [post]
func (h *PresettlementHandler) Calculate(c *gin.Context) {
	var req dto.CalculatePresettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.BadRequest("预结算参数（Presettlement）不合法", err))
		return
	}
	preset, err := h.svc.CalculatePresettlement(c.Request.Context(), req.BatchID)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, preset)
}

// ListByBatch 批次多次预结算比对。
// @Summary 批次预结算记录
// @Tags presettlements
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param batch_id query int true "批次 ID"
// @Success 200 {object} util.Response
// @Router /api/v1/presettlements [get]
func (h *PresettlementHandler) ListByBatch(c *gin.Context) {
	batchID := parseUint(c.Query("batch_id"))
	items, err := h.svc.ListPresettlements(c.Request.Context(), batchID)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, items)
}
