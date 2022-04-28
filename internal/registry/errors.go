package registry

import "errors"

var (
	// ErrUnconfiguredProvider is returned when the image registry provider is
	// not configured for login.
	ErrUnconfiguredProvider = errors.New("registry provider not configured for login")
)
