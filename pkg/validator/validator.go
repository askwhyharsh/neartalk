package validator

import (
	"regexp"
	"strings"

	apperrors "github.com/askwhyharsh/peoplearoundme/pkg/errors"
)

type Validator interface {
	ValidateUsername(username string) error
	ValidateMessage(content string) error
	ValidateCoordinates(lat, lon float64) error
	ValidateRadius(radius int) error
}

type validator struct {
	usernameRegex *regexp.Regexp
}

func NewValidator() Validator {
	return &validator{
		usernameRegex: regexp.MustCompile(`^[a-zA-Z0-9_ ]+$`),
	}
}

func (v *validator) ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 20 {
		return apperrors.ErrInvalidUsernameLength
	}

	if !v.usernameRegex.MatchString(username) {
		return apperrors.ErrInvalidUsernameChars
	}

	return nil
}

func (v *validator) ValidateMessage(content string) error {
	trimmed := strings.TrimSpace(content)
	
	if len(trimmed) == 0 {
		return apperrors.ErrEmptyMessage
	}

	if len(content) < 1 || len(content) > 500 {
		return apperrors.ErrInvalidMessageLength
	}

	return nil
}

func (v *validator) ValidateCoordinates(lat, lon float64) error {
	if lat < -90 || lat > 90 {
		return apperrors.ErrInvalidLatitude
	}

	if lon < -180 || lon > 180 {
		return apperrors.ErrInvalidLongitude
	}

	return nil
}

func (v *validator) ValidateRadius(radius int) error {
	if radius < 100 || radius > 2000 {
		return apperrors.ErrInvalidRadius
	}

	return nil
}
