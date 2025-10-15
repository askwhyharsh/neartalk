package ratelimit

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Middleware struct {
	limiter *Limiter
}

func NewMiddleware(limiter *Limiter) *Middleware {
	return &Middleware{
		limiter: limiter,
	}
}

// IPRateLimit middleware for general IP-based rate limiting
func (m *Middleware) IPRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		allowed, err := m.limiter.AllowIPRequest(c.Request.Context(), ip)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check rate limit",
			})
			c.Abort()
			return
		}
		
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
				"code":  "RATE_LIMIT_IP",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// SessionRateLimit middleware for session-based rate limiting
func (m *Middleware) SessionRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.GetHeader("X-Session-ID")
		if sessionID == "" {
			sessionID = c.Query("session_id")
		}
		
		if sessionID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Session ID required",
			})
			c.Abort()
			return
		}
		
		// Store session ID in context for handlers
		c.Set("session_id", sessionID)
		c.Next()
	}
}