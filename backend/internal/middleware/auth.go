package middleware

import (
	"net/http"
	"strings"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/gin-gonic/gin"
)

// ClientKey / ClientIDKey 注入 gin.Context 的键。
const (
	ClientKey   = "client"
	ClientIDKey = "clientID"
)

// JwtAuthRequired 校验 Bearer JWT 并注入调用方信息。
func JwtAuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			abortJSON(c, http.StatusUnauthorized, constants.CodeUnauthorized, constants.MsgTokenInvalid)
			return
		}
		claims, err := util.ParseToken(jwtSecret, strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			abortJSON(c, http.StatusUnauthorized, constants.CodeUnauthorized, constants.MsgTokenInvalid)
			return
		}
		c.Set(ClientIDKey, claims.ClientID)
		c.Set(ClientKey, claims)
		c.Next()
	}
}

// AdminRequired 仅允许管理员令牌（网关管理接口）。
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(ClientKey)
		if !ok {
			abortJSON(c, http.StatusForbidden, constants.CodeForbidden, constants.MsgForbidden)
			return
		}
		claims, ok := v.(*util.Claims)
		if !ok || claims.TokenType != "admin" {
			abortJSON(c, http.StatusForbidden, constants.CodeForbidden, constants.MsgForbidden)
			return
		}
		c.Next()
	}
}

func abortJSON(c *gin.Context, status, code int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"code": code, "message": message, "data": nil})
}
