package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/askwhyharsh/neartalk/internal/api"
	"github.com/askwhyharsh/neartalk/internal/config"
	"github.com/askwhyharsh/neartalk/internal/location"
	"github.com/askwhyharsh/neartalk/internal/message"
	"github.com/askwhyharsh/neartalk/internal/ratelimit"
	"github.com/askwhyharsh/neartalk/internal/session"
	"github.com/askwhyharsh/neartalk/internal/spam"
	"github.com/askwhyharsh/neartalk/internal/storage"
	"github.com/askwhyharsh/neartalk/internal/websocket"
	"github.com/askwhyharsh/neartalk/pkg/logger"
	"github.com/askwhyharsh/neartalk/pkg/validator"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize logger
	appLogger := logger.NewLogger(os.Getenv("LOG_LEVEL"))
	appLogger.Info("Starting PeopleAroundMe server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// TODO: make it FATAL err later
		appLogger.Error("Failed to load configuration", "error", err)
	}

	// Initialize Redis
	redisClient, err := storage.NewRedisClient(cfg)
	if err != nil {
		// TODO: make it FATAL err later
		appLogger.Error("Failed to connect to Redis", "error", err)
	}
	defer redisClient.Close()
	appLogger.Info("Connected to Redis", "address", cfg.RedisAddr())

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize services
	sessionService := session.NewService(redisClient, cfg.Session.TTL, cfg.RateLimit.MaxUsernameChanges)

	sessionManager := session.NewManager(sessionService, appLogger)

	locationService := location.NewService(
		redisClient,
		cfg.Location.GeohashPrecision,
		cfg.Location.MinRadiusMeters,
		cfg.Location.MaxRadiusMeters,
	)

	messageStore := message.NewStore(redisClient, cfg.Session.MessageTTL)
	// messageRouter := message.NewRouter(redisClient, messageStore)
	ttlManager := message.NewTTLManager(messageStore, appLogger)

	spamDetector := spam.NewDetector(
		redisClient,
		cfg.Spam.ProfanityEnabled,
		cfg.Spam.DuplicateWindowSeconds,
		cfg.Spam.MaxURLsPerMessage,
	)

	rateLimiter := ratelimit.NewLimiter(redisClient, cfg.RateLimit)
	rateLimitMiddleware := ratelimit.NewMiddleware(rateLimiter)

	// Initialize validator
	val := validator.NewValidator()

	// Initialize WebSocket hub
	// hub := websocket.NewHub(appLogger, messageRouter, locationService, sessionService)
	hub := websocket.NewHub(ctx, redisClient)
	go hub.Run()

	// Initialize WebSocket handler
	wsHandler := websocket.NewHandler(
		hub,
		redisClient,
		sessionService,
		locationService,
		spamDetector,
		rateLimiter,
		cfg.Session.MessageTTL,
	)

	// Initialize API handler
	apiHandler := api.NewHandler(
		sessionService,
		locationService,
		rateLimiter,
		val,
	)

	// Start background services
	go sessionManager.Start(ctx)
	go ttlManager.Start(ctx)

	// Setup Gin router
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Add logging middleware
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		appLogger.Info("Request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", duration,
			"ip", c.ClientIP(),
		)
	})

	// Setup routes
	api.SetupRoutes(router, apiHandler, wsHandler, rateLimitMiddleware)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		appLogger.Info("Server starting", "address", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Failed to start server", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Cancel context to stop background services
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
	}

	appLogger.Info("Server stopped")
}
