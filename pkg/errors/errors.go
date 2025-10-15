package errors

import "errors"

var (
	// Session errors
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionExpired       = errors.New("session expired")
	ErrInvalidSessionID     = errors.New("invalid session ID")
	ErrMaxUsernameChanges   = errors.New("maximum username changes reached")

	// Validation errors
	ErrInvalidUsername      = errors.New("invalid username")
	ErrInvalidUsernameLength = errors.New("username must be 3-20 characters")
	ErrInvalidUsernameChars = errors.New("username can only contain letters, numbers, spaces and underscores")
	ErrInvalidMessage       = errors.New("invalid message")
	ErrInvalidMessageLength = errors.New("message must be 1-500 characters")
	ErrEmptyMessage         = errors.New("message cannot be empty")
	ErrInvalidCoordinates   = errors.New("invalid coordinates")
	ErrInvalidLatitude      = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude     = errors.New("longitude must be between -180 and 180")
	ErrInvalidRadius        = errors.New("radius must be between 100 and 2000 meters")

	// Rate limit errors
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrTooManyRequests      = errors.New("too many requests")

	// Spam errors
	ErrProfanityDetected    = errors.New("profanity detected")
	ErrSpamDetected         = errors.New("spam detected")
	ErrDuplicateMessage     = errors.New("duplicate message")
	ErrURLSpam              = errors.New("too many URLs in message")

	// WebSocket errors
	ErrWebSocketClosed      = errors.New("websocket connection closed")
	ErrInvalidMessageType   = errors.New("invalid message type")

	// Storage errors
	ErrStorageUnavailable   = errors.New("storage unavailable")
	ErrDataNotFound         = errors.New("data not found")
)

type AppError struct {
	Err        error
	Message    string
	StatusCode int
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(err error, message string, statusCode int) *AppError {
	return &AppError{
		Err:        err,
		Message:    message,
		StatusCode: statusCode,
	}
}