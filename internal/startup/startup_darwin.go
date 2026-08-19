//go:build darwin

package startup

/*
#cgo LDFLAGS: -framework Foundation -framework ServiceManagement
#include "smapp_darwin.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// ErrUnavailable is returned when launch-at-login cannot be used.
var ErrUnavailable = errors.New("launch at login unavailable outside GopherTchi.app on macOS 13+")

var migrateOnce sync.Once

// Status represents SMAppService.mainAppService registration state.
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

// Supported reports whether launch-at-login can be toggled (macOS 13+ .app only).
func Supported() bool {
	migrateLegacyLaunchAgent()
	return bool(C.gophertchi_login_supported())
}

// CurrentStatus returns the live SMAppService status.
func CurrentStatus() Status {
	migrateLegacyLaunchAgent()
	if !Supported() {
		return StatusUnsupported
	}
	switch C.gophertchi_login_status() {
	case C.GOPHER_LOGIN_ENABLED:
		return StatusEnabled
	case C.GOPHER_LOGIN_NOT_REGISTERED:
		return StatusNotRegistered
	case C.GOPHER_LOGIN_REQUIRES_APPROVAL:
		return StatusRequiresApproval
	case C.GOPHER_LOGIN_NOT_FOUND:
		return StatusNotFound
	case C.GOPHER_LOGIN_ERROR:
		return StatusError
	default:
		return StatusUnsupported
	}
}

// Enabled reports whether SMAppService currently has the app enabled at login.
func Enabled() bool {
	return CurrentStatus() == StatusEnabled
}

// SetEnabled registers or unregisters the app via SMAppService.mainAppService.
func SetEnabled(enable bool) error {
	migrateLegacyLaunchAgent()
	if !Supported() {
		return ErrUnavailable
	}
	var cErr *C.char
	ok := C.gophertchi_login_set(C.bool(enable), &cErr)
	if cErr != nil {
		msg := C.GoString(cErr)
		C.gophertchi_login_free(cErr)
		if !bool(ok) {
			return fmt.Errorf("SMAppService: %s", msg)
		}
	}
	if !bool(ok) {
		return errors.New("SMAppService update failed")
	}
	return nil
}

// OpenSystemSettingsLoginItems opens macOS Login Items settings.
func OpenSystemSettingsLoginItems() error {
	migrateLegacyLaunchAgent()
	if !Supported() {
		return ErrUnavailable
	}
	C.gophertchi_login_open_settings()
	return nil
}

// migrateLegacyLaunchAgent removes the old LaunchAgent used before SMAppService.
func migrateLegacyLaunchAgent() {
	migrateOnce.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		plist := filepath.Join(home, "Library", "LaunchAgents", "com.gophertchi.app.plist")
		if _, err := os.Stat(plist); err != nil {
			return
		}
		_ = exec.Command("launchctl", "unload", "-w", plist).Run()
		_ = os.Remove(plist)
	})
}
