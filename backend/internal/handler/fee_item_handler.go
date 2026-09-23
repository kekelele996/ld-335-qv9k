package handler

import (
	"log/slog"

	"github.com/blueship581/gbinsureapi/internal/dto"
	"github.com/blueship581/gbinsureapi/internal/service"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/gin-gonic/gin"
)

// FeeItemHandler 费用明细接口。
type FeeItemHandler struct {
	svc *service.FeeService
	log *slog.Logger
}

// NewFeeItemHandler 构造费用明细接口。
func NewFeeItemHandler(svc *service.FeeService, log *slog.Logger) *FeeItemHandler {
	return &FeeItemHandler{svc: svc, log: log}
}

// Upload 费用明细上传。
// @Summary 费用明细上传
// @Description 接收就诊费用明细，校验格式与重复性，生成上传批次号
// @Tags fees
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param body body dto.UploadFeesRequest true "费用明细"
// @Success 201 {object} util.Response
// @Router /api/v1/fees/upload [post]
func (h *FeeItemHandler) Upload(c *gin.Context) {
	var req dto.UploadFeesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.BadRequest("费用明细（FeeItem）参数不合法", err))
		return
	}
	items := make([]service.FeeItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, service.FeeItemInput{
			ItemCode: it.ItemCode, ItemName: it.ItemName, ItemType: it.ItemType,
			UnitPrice: it.UnitPrice, Quantity: it.Quantity,
			SelfPayRatio: it.SelfPayRatio, MedicalCategory: it.MedicalCategory,
		})
	}
	result, err := h.svc.Upload(c.Request.Context(), service.UploadInput{
		ClientID: req.ClientID, InsuredPersonID: req.InsuredPersonID, Items: items,
	})
	if err != nil {
		c.Error(err)
		return
	}
	if result.UploadStatus == "failed" {
		util.OK(c, result)
		return
	}
	util.Created(c, result)
}

// ListItems 批次明细。
// @Summary 批次费用明细
// @Tags fees
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param batch_id query int true "批次 ID"
// @Success 200 {object} util.Response
// @Router /api/v1/fees/items [get]
func (h *FeeItemHandler) ListItems(c *gin.Context) {
	batchID := parseUint(c.Query("batch_id"))
	items, err := h.svc.ListItemsByBatch(c.Request.Context(), batchID)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, items)
}
