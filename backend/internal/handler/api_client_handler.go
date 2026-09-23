package handler

import (
	"errors"
	"log/slog"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/dto"
	"github.com/blueship581/gbinsureapi/internal/service"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/gin-gonic/gin"
)

// ApiClientHandler 调用方管理接口。
type ApiClientHandler struct {
	svc *service.ApiClientService
	log *slog.Logger
}

// NewApiClientHandler 构造调用方接口。
func NewApiClientHandler(svc *service.ApiClientService, log *slog.Logger) *ApiClientHandler {
	return &ApiClientHandler{svc: svc, log: log}
}

// Create 创建调用方（管理员）。
// @Summary 创建调用方
// @Description 注册 HIS/第三方调用方，返回明文 API Key（仅此一次）
// @Tags clients
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.CreateClientRequest true "调用方参数"
// @Success 201 {object} util.Response
// @Router /api/v1/clients [post]
func (h *ApiClientHandler) Create(c *gin.Context) {
	var req dto.CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.BadRequest("调用方（ApiClient）参数不合法", err))
		return
	}
	client, apiKey, err := h.svc.CreateClient(c.Request.Context(), service.CreateClientInput{
		Name: req.Name, ClientType: req.ClientType, Role: req.Role, RateLimitQPS: req.RateLimitQPS,
	})
	if err != nil {
		c.Error(err)
		return
	}
	util.Created(c, dto.ClientCreatedResponse{ClientID: client.ID, Name: client.Name, APIKey: apiKey, Role: client.Role})
}

// List 调用方列表（管理员）。
// @Summary 调用方列表
// @Tags clients
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} util.Response
// @Router /api/v1/clients [get]
func (h *ApiClientHandler) List(c *gin.Context) {
	page := parseQueryInt(c.Query("page"), 1)
	pageSize := parseQueryInt(c.Query("page_size"), 20)
	clients, total, err := h.svc.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, util.PageData{List: clients, Total: total, Page: page, Size: pageSize})
}

// UpdateStatus 启用/停用调用方（管理员）。
// @Summary 更新调用方状态
// @Tags clients
// @Security BearerAuth
// @Param id path int true "调用方 ID"
// @Param body body dto.UpdateClientStatusRequest true "状态"
// @Success 200 {object} util.Response
// @Router /api/v1/clients/{id}/status [put]
func (h *ApiClientHandler) UpdateStatus(c *gin.Context) {
	id := parseUint(c.Param("id"))
	var req dto.UpdateClientStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.BadRequest("调用方（ApiClient.status）参数不合法", err))
		return
	}
	if err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		c.Error(err)
		return
	}
	util.OK(c, gin.H{"client_id": id, "status": req.Status})
}

// IssueToken 通过 API Key 换取服务 JWT。
// @Summary 换取服务令牌
// @Description 调用方使用 X-API-Key 换取服务 JWT（业务接口需 API Key + JWT 双认证）
// @Tags clients
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Success 200 {object} util.Response
// @Router /api/v1/clients/token [post]
func (h *ApiClientHandler) IssueToken(c *gin.Context) {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		c.Error(util.UnauthorizedError(constants.MsgApiKeyInvalid, errors.New("missing api key")))
		return
	}
	client, err := h.svc.VerifyAPIKey(c.Request.Context(), apiKey)
	if err != nil {
		c.Error(err)
		return
	}
	token, err := h.svc.IssueServiceToken(c.Request.Context(), client)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, gin.H{"token": token, "token_type": "service", "client_id": client.ID})
}
