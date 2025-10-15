package api

import (
	"time"

	"github.com/askwhyharsh/neartalk/internal/location"
)

// Request types
type UpdateUsernameRequest struct {
	Username string `json:"username" binding:"required"`
}

type UpdateLocationRequest struct {
	Lat    float64 `json:"lat" binding:"required"`
	Lon    float64 `json:"lon" binding:"required"`
	Radius int     `json:"radius" binding:"required"`
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error details
type ErrorInfo struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// SuccessResponse creates a successful response
func SuccessResponse(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

// ErrorResponse creates an error response
func ErrorResponse(message, code string) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Message: message,
			Code:    code,
		},
	}
}

type CreateSessionResponse struct {
	SessionID   string `json:"session_id"`
	Username    string `json:"username"`
	ChangesLeft int    `json:"changes_left"`
	MaxChanges  int    `json:"max_changes"`
}

type UpdateUsernameResponse struct {
	Username    string `json:"username"`
	ChangesLeft int    `json:"changes_left"`
}

type NearbyUsersResponse struct {
	Count int                   `json:"count"`
	Users []location.NearbyUser `json:"users"`
}

type HealthResponse struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}
