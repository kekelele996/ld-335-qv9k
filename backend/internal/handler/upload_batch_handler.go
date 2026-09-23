package handler

import (
	"log/slog"

	"github.com/blueship581/gbinsureapi/internal/service"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/gin-gonic/gin"
)

// UploadBatchHandler 上传批次接口。
type UploadBatchHandler struct {
	svc *service.FeeService
	log *slog.Logger
}

// NewUploadBatchHandler 构造上传批次接口。
func NewUploadBatchHandler(svc *service.FeeService, log *slog.Logger) *UploadBatchHandler {
	return &UploadBatchHandler{svc: svc, log: log}
}

// List 批次列表。
// @Summary 上传批次列表
// @Tags batches
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param client_id query int false "调用方 ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} util.Response
// @Router /api/v1/batches [get]
func (h *UploadBatchHandler) List(c *gin.Context) {
	clientID := parseUint(c.Query("client_id"))
	page := parseQueryInt(c.Query("page"), 1)
	pageSize := parseQueryInt(c.Query("page_size"), 20)
	batches, total, err := h.svc.ListBatches(c.Request.Context(), clientID, page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, util.PageData{List: batches, Total: total, Page: page, Size: pageSize})
}

// Detail 批次详情。
// @Summary 批次详情
// @Tags batches
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param id path int true "批次 ID"
// @Success 200 {object} util.Response
// @Router /api/v1/batches/{id} [get]
func (h *UploadBatchHandler) Detail(c *gin.Context) {
	id := parseUint(c.Param("id"))
	batch, err := h.svc.GetBatch(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, batch)
}
