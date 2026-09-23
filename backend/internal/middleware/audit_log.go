package middleware

import (
	"log/slog"
	"time"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/blueship581/gbinsureapi/internal/repository"
	"github.com/blueship581/gbinsureapi/internal/util"
	"github.com/gin-gonic/gin"
)

// AuditLog 审计日志中间件：记录调用方、方法、路径、状态码、耗时。
func AuditLog(repo *repository.AuditLogRepository, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		clientID := uint(0)
		if v, ok := c.Get(ClientIDKey); ok {
			if id, ok := v.(uint); ok {
				clientID = id
			}
		}
		entry := &model.AuditLog{
			ClientID: clientID, Method: c.Request.Method, Path: c.Request.URL.Path,
			StatusCode: c.Writer.Status(), LatencyMs: time.Since(start).Milliseconds(),
		}
		if err := repo.Create(entry); err != nil {
			util.LogError(log, constants.LOG_AUDIT_WRITTEN, err, "client_id", clientID)
		}
	}
}
