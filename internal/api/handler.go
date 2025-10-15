package api

import (
	"net/http"

	"github.com/askwhyharsh/neartalk/internal/location"
	"github.com/askwhyharsh/neartalk/internal/ratelimit"
	"github.com/askwhyharsh/neartalk/internal/session"
	"github.com/askwhyharsh/neartalk/pkg/validator"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	sessionService  session.SessionService
	locationService location.LocationService
	rateLimiter     ratelimit.RateLimiter
	validator       validator.Validator
}

type SessionResponse struct {
	SessionID   string `json:"session_id"`
	Username    string `json:"username"`
	ChangesLeft int    `json:"changes_left"`
	MaxChanges  int    `json:"max_changes"`
	CreatedAt   string `json:"created_at"`
}

type NearbyUser struct {
	Username string `json:"username"`
	Distance string `json:"distance"`
}

func NewHandler(sessionService session.SessionService, locationService location.LocationService, rateLimiter ratelimit.RateLimiter, validator validator.Validator) *Handler {
	return &Handler{
		sessionService:  sessionService,
		locationService: locationService,
		rateLimiter:     rateLimiter,
		validator:       validator,
	}
}

// POST /api/session/create
func (h *Handler) CreateSession(c *gin.Context) {
	ip := c.ClientIP()

	// Check rate limit
	allowed, err := h.rateLimiter.AllowSessionCreation(c, ip)
	if err != nil || !allowed {
		c.JSON(http.StatusTooManyRequests, ErrorResponse("Rate limit exceeded", "RATE_LIMIT"))
		return
	}

	// Create session
	session, err := h.sessionService.Create(c, ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("Failed to create session", "INTERNAL_ERROR"))
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse(session))
}

// PATCH /api/session/username
func (h *Handler) UpdateUsername(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
		Username  string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("Invalid request", "INVALID_REQUEST"))
		return
	}

	// Validate username
	if err := h.validator.ValidateUsername(req.Username); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err.Error(), "INVALID_USERNAME"))
		return
	}

	// Check rate limit
	allowed, _, err := h.rateLimiter.AllowUsernameChange(c, req.SessionID)
	if err != nil || !allowed {
		c.JSON(http.StatusTooManyRequests, ErrorResponse("Username change limit reached", "RATE_LIMIT"))
		return
	}

	// Update username
	if err := h.sessionService.UpdateUsername(c, req.SessionID, req.Username); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err.Error(), "UPDATE_FAILED"))
		return
	}

	// Get remaining changes
	remaining, _ := h.sessionService.GetRemainingChanges(c, req.SessionID)

	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"username":     req.Username,
		"changes_left": remaining,
	}))
}

// POST /api/location/update
func (h *Handler) UpdateLocation(c *gin.Context) {
	var req struct {
		SessionID string  `json:"session_id" binding:"required"`
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
		Radius    int     `json:"radius" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("Invalid request", "INVALID_REQUEST"))
		return
	}

	// Validate coordinates
	if err := h.validator.ValidateCoordinates(req.Latitude, req.Longitude); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err.Error(), "INVALID_COORDINATES"))
		return
	}

	// Validate radius
	if err := h.validator.ValidateRadius(req.Radius); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse(err.Error(), "INVALID_RADIUS"))
		return
	}

	// Check rate limit
	allowed, err := h.rateLimiter.AllowLocationUpdate(c, req.SessionID)
	if err != nil || !allowed {
		c.JSON(http.StatusTooManyRequests, ErrorResponse("Location update rate limit exceeded", "RATE_LIMIT"))
		return
	}

	// Update location
	if err := h.locationService.UpdateLocation(c, req.SessionID, req.Latitude, req.Longitude, req.Radius); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("Failed to update location", "INTERNAL_ERROR"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"message": "Location updated successfully",
	}))
}

// GET /api/nearby
func (h *Handler) GetNearbyUsers(c *gin.Context) {
	sessionID := c.Query("session_id")
	ctx := c.Request.Context()
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse("session_id required", "INVALID_REQUEST"))
		return
	}

	// Get nearby users
	users, err := h.locationService.GetNearbyUsers(ctx, sessionID, func(sid string) string {
		session, err := h.sessionService.Get(ctx, sid)
		if err != nil {
			return "Unknown"
		}
		return session.Username
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("Failed to get nearby users", "INTERNAL_ERROR"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"count": len(users),
		"users": users,
	}))
}

// GET /api/health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   c.GetTime("request_time"),
	})
}
