package handler

import (
	"log/slog"
	"strconv"

	"github.com/blueship581/gbinsureapi/internal/dto"
	"github.com/blueship581/gbinsureapi/internal/service"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/gin-gonic/gin"
)

// InsuredPersonHandler 参保人接口。
type InsuredPersonHandler struct {
	svc *service.InsuranceService
	log *slog.Logger
}

// NewInsuredPersonHandler 构造参保人接口。
func NewInsuredPersonHandler(svc *service.InsuranceService, log *slog.Logger) *InsuredPersonHandler {
	return &InsuredPersonHandler{svc: svc, log: log}
}

// Verify 参保人身份核验。
// @Summary 参保人身份核验
// @Description 接收身份证号与医保卡号，返回参保状态、参保地、医保类型、个人账户余额
// @Tags insured
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param body body dto.VerifyRequest true "核验参数"
// @Success 200 {object} util.Response
// @Router /api/v1/clients/verify [post]
func (h *InsuredPersonHandler) Verify(c *gin.Context) {
	var req dto.VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.BadRequest("核验参数（InsuredPerson）不合法", err))
		return
	}
	person, err := h.svc.Verify(c.Request.Context(), req.IDCardNo, req.MedicalCardNo)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, gin.H{
		"name": person.Name, "insurance_type": person.InsuranceType,
		"insurance_status": person.InsuranceStatus, "insurance_place": person.InsurancePlace,
		"personal_balance": person.PersonalBalance,
	})
}

// List 参保人列表。
// @Summary 参保人列表
// @Tags insured
// @Security ApiKeyAuth
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} util.Response
// @Router /api/v1/insured-persons [get]
func (h *InsuredPersonHandler) List(c *gin.Context) {
	page := parseQueryInt(c.Query("page"), 1)
	pageSize := parseQueryInt(c.Query("page_size"), 20)
	persons, total, err := h.svc.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, util.PageData{List: persons, Total: total, Page: page, Size: pageSize})
}

func parseQueryInt(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil && v > 0 {
		return v
	}
	return def
}

func parseUint(s string) uint {
	v, _ := strconv.ParseUint(s, 10, 64)
	return uint(v)
}
