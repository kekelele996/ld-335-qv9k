package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/blueship581/gbinsureapi/internal/constants"
	"github.com/blueship581/gbinsureapi/internal/model"
	"github.com/gin-gonic/gin"
)

// bucket 令牌桶。
type bucket struct {
	tokens float64
	last   time.Time
}

// RateLimiter 按调用方 QPS 的令牌桶限流。
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[uint]*bucket
}

// NewRateLimiter 构造限流器。
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: map[uint]*bucket{}}
}

func (rl *RateLimiter) allow(clientID uint, qps int) bool {
	if qps <= 0 {
		qps = 10
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.buckets[clientID]
	if !ok {
		rl.buckets[clientID] = &bucket{tokens: float64(qps), last: now}
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * float64(qps)
	if b.tokens > float64(qps) {
		b.tokens = float64(qps)
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Limit 返回限流中间件（基于调用方 QPS）。
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := uint(0)
		qps := 10
		if v, ok := c.Get("apiClient"); ok {
			if client, ok := v.(*model.ApiClient); ok {
				clientID = client.ID
				qps = client.RateLimitQPS
			}
		}
		if id, ok := c.Get(ClientIDKey); ok {
			if v, ok := id.(uint); ok {
				clientID = v
			}
		}
		if !rl.allow(clientID, qps) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": constants.CodeRateLimited, "message": constants.MsgRateLimited, "data": nil,
			})
			return
		}
		c.Next()
	}
}
