package handler

import (
	"errors"
	"log/slog"

	"github.com/blueship581/gbinsureapi/internal/config"
	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/dto"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/gin-gonic/gin"
)

// AuthHandler 网关管理员登录。
type AuthHandler struct {
	cfg config.Config
	log *slog.Logger
}

// NewAuthHandler 构造认证接口。
func NewAuthHandler(cfg config.Config, log *slog.Logger) *AuthHandler {
	return &AuthHandler{cfg: cfg, log: log}
}

// AdminLogin 管理员登录签发管理 JWT。
// @Summary 管理员登录
// @Description 网关管理员登录，返回管理端 JWT（用于调用方管理接口）
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.AdminLoginRequest true "登录参数"
// @Success 200 {object} util.Response
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) AdminLogin(c *gin.Context) {
	var req dto.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.BadRequest("登录参数不合法", err))
		return
	}
	if req.Username != h.cfg.AdminUsername || req.Password != h.cfg.AdminPassword {
		c.Error(util.UnauthorizedError(constants.MsgUnauthorized, errors.New("admin credential mismatch")))
		return
	}
	token, err := util.GenerateToken(h.cfg.JWTSecret, h.cfg.JWTExpireHours, 0, "admin", "admin", "admin")
	if err != nil {
		c.Error(util.InternalError(constants.MsgInternalError, err))
		return
	}
	util.OK(c, gin.H{"token": token, "token_type": "admin"})
}
