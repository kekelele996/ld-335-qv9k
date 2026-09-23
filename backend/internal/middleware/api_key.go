package middleware

import (
	"net/http"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/service"
	"github.com/gin-gonic/gin"
)

// ApiKeyRequired 校验请求头 X-API-Key，并将调用方注入 gin.Context。
func ApiKeyRequired(clientSvc *service.ApiClientService) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			abortJSON(c, http.StatusUnauthorized, constants.CodeApiKeyInvalid, constants.MsgApiKeyInvalid)
			return
		}
		client, err := clientSvc.VerifyAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		c.Set("apiClient", client)
		c.Set(ClientIDKey, client.ID)
		c.Next()
	}
}
