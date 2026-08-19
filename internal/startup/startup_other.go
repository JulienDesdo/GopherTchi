//go:build !darwin

package startup

import "errors"

// ErrUnavailable is returned when launch-at-login is not supported.
var ErrUnavailable = errors.New("launch at login is only supported on macOS")

// Status represents login-item registration state.
type Status int

const (
	StatusUnsupported Status = iota
	StatusEnabled
	StatusNotRegistered
	StatusRequiresApproval
	StatusNotFound
	StatusError
)

func (s Status) String() string {
	switch s {
	case StatusEnabled:
		return "enabled"
	case StatusNotRegistered:
		return "not_registered"
	case StatusRequiresApproval:
		return "requires_approval"
	case StatusNotFound:
		return "not_found"
	case StatusError:
		return "error"
	default:
		return "unsupported"
	}
}

// Supported reports whether launch-at-login can be toggled.
func Supported() bool { return false }

// CurrentStatus returns the live registration status.
func CurrentStatus() Status { return StatusUnsupported }

// Enabled reports whether launch-at-login is active.
func Enabled() bool { return false }

// SetEnabled is a no-op outside macOS.
func SetEnabled(enable bool) error { return ErrUnavailable }
