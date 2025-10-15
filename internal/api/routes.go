package api

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, handler *Handler, wsHandler WebSocketHandler) {
	// Apply global middleware
	r.Use(CORSMiddleware())
	r.Use(RequestTimeMiddleware())
	r.Use(RecoveryMiddleware())

	// API routes
	api := r.Group("/api")
	{
		// Session routes
		session := api.Group("/session")
		{
			session.POST("/create", handler.CreateSession)
			session.PATCH("/username", handler.UpdateUsername)
		}

		// Location routes
		location := api.Group("/location")
		{
			location.POST("/update", handler.UpdateLocation)
		}

		// Nearby users
		api.GET("/nearby", handler.GetNearbyUsers)

		// Health check
		api.GET("/health", handler.Health)
	}

	// WebSocket route
	r.GET("/ws", wsHandler.HandleWebSocket)
}

type WebSocketHandler interface {
	HandleWebSocket(c *gin.Context)
}