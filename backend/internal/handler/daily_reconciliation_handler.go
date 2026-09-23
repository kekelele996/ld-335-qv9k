package handler

import (
	"log/slog"

	"github.com/blueship581/gbinsureapi/internal/service"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/gin-gonic/gin"
)

// DailyReconciliationHandler 日终对账接口。
type DailyReconciliationHandler struct {
	svc *service.ReconciliationService
	log *slog.Logger
}

// NewDailyReconciliationHandler 构造日终对账接口。
func NewDailyReconciliationHandler(svc *service.ReconciliationService, log *slog.Logger) *DailyReconciliationHandler {
	return &DailyReconciliationHandler{svc: svc, log: log}
}

// Daily 日终对账。
// @Summary 日终对账
// @Description 返回当日总笔数、总金额、成功/失败笔数
// @Tags reconciliations
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param client_id query int false "调用方 ID"
// @Success 200 {object} util.Response
// @Router /api/v1/reconciliations/daily [get]
func (h *DailyReconciliationHandler) Daily(c *gin.Context) {
	clientID := parseUint(c.Query("client_id"))
	rec, err := h.svc.Daily(c.Request.Context(), clientID)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, rec)
}

// List 对账记录列表。
// @Summary 对账记录列表
// @Tags reconciliations
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} util.Response
// @Router /api/v1/reconciliations [get]
func (h *DailyReconciliationHandler) List(c *gin.Context) {
	page := parseQueryInt(c.Query("page"), 1)
	pageSize := parseQueryInt(c.Query("page_size"), 20)
	items, total, err := h.svc.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, util.PageData{List: items, Total: total, Page: page, Size: pageSize})
}
