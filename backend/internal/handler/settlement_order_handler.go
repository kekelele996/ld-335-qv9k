package handler

import (
	"log/slog"

	"github.com/blueship581/gbinsureapi/internal/dto"
	"github.com/blueship581/gbinsureapi/internal/middleware"
	"github.com/blueship581/gbinsureapi/internal/service"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/gin-gonic/gin"
)

// SettlementOrderHandler 结算单接口。
type SettlementOrderHandler struct {
	svc *service.SettlementService
	log *slog.Logger
}

// NewSettlementOrderHandler 构造结算单接口。
func NewSettlementOrderHandler(svc *service.SettlementService, log *slog.Logger) *SettlementOrderHandler {
	return &SettlementOrderHandler{svc: svc, log: log}
}

// Submit 正式结算。
// @Summary 正式结算
// @Description 确认预结算后生成唯一结算单号并返回凭证信息
// @Tags settlements
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param body body dto.SubmitSettlementRequest true "预结算 ID"
// @Success 201 {object} util.Response
// @Router /api/v1/settlements/submit [post]
func (h *SettlementOrderHandler) Submit(c *gin.Context) {
	var req dto.SubmitSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.BadRequest("结算参数（SettlementOrder）不合法", err))
		return
	}
	clientID, _ := c.Get(middleware.ClientIDKey)
	order, err := h.svc.SubmitSettlement(c.Request.Context(), clientID.(uint), req.PresettlementID)
	if err != nil {
		c.Error(err)
		return
	}
	util.Created(c, order)
}

// Reverse 结算冲正。
// @Summary 结算冲正
// @Description 结算当日全额回退
// @Tags settlements
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param settlement_no path string true "结算单号"
// @Success 200 {object} util.Response
// @Router /api/v1/settlements/{settlement_no}/reverse [post]
func (h *SettlementOrderHandler) Reverse(c *gin.Context) {
	order, err := h.svc.ReverseSettlement(c.Request.Context(), c.Param("settlement_no"))
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, order)
}

// List 历史结算查询。
// @Summary 结算单列表
// @Tags settlements
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param status query string false "状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} util.Response
// @Router /api/v1/settlements [get]
func (h *SettlementOrderHandler) List(c *gin.Context) {
	clientID := parseUint(c.Query("client_id"))
	page := parseQueryInt(c.Query("page"), 1)
	pageSize := parseQueryInt(c.Query("page_size"), 20)
	orders, total, err := h.svc.ListOrders(c.Request.Context(), clientID, c.Query("status"), page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, util.PageData{List: orders, Total: total, Page: page, Size: pageSize})
}

// Detail 结算单详情。
// @Summary 结算单详情
// @Tags settlements
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param settlement_no path string true "结算单号"
// @Success 200 {object} util.Response
// @Router /api/v1/settlements/{settlement_no} [get]
func (h *SettlementOrderHandler) Detail(c *gin.Context) {
	order, err := h.svc.GetOrder(c.Request.Context(), c.Param("settlement_no"))
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, order)
}
