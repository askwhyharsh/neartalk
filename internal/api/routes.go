package api

import (
	"github.com/askwhyharsh/neartalk/internal/ratelimit"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, handler *Handler, wsHandler WebSocketHandler, rlMiddleware *ratelimit.Middleware) {
	// Apply global middleware
	r.Use(CORSMiddleware())
	r.Use(RequestTimeMiddleware())
	r.Use(RecoveryMiddleware())
	r.Use(rlMiddleware.IPRateLimit()) // IP-based rate limiting

	// API routes
	api := r.Group("/api")
	{
		// Session routes
		session := api.Group("/session")
		{
			session.POST("/create", handler.CreateSession)
			session.PATCH("/username", rlMiddleware.SessionRateLimit(), handler.UpdateUsername)
		}

		// Location routes
		location := api.Group("/location")
		{
			location.POST("/update", rlMiddleware.SessionRateLimit(), handler.UpdateLocation)
		}

		// Nearby users
		api.GET("/nearby", rlMiddleware.SessionRateLimit(), handler.GetNearbyUsers)

		// Nearby users
		api.GET("/recent-messages", rlMiddleware.SessionRateLimit(), handler.GetRecentMessages)

		// Health check (no rate limit)
		api.GET("/health", handler.Health)
	}

	// WebSocket route
	r.GET("/ws", wsHandler.HandleWebSocket)
}

type WebSocketHandler interface {
	HandleWebSocket(c *gin.Context)
}
