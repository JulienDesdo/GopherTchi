package startup_test

import (
	"runtime"
	"testing"

	"github.com/jlnesc/gophertchi/internal/startup"
)

func TestSupportedOnlyInsideAppBundle(t *testing.T) {
	if runtime.GOOS != "darwin" {
		if startup.Supported() {
			t.Fatal("Supported must be false on non-darwin")
		}
		return
	}

	// go test binaries are not inside a .app bundle.
	if startup.Supported() {
		t.Fatal("Supported must be false under go test / go run")
	}
	if startup.CurrentStatus() != startup.StatusUnsupported {
		t.Fatalf("CurrentStatus = %s, want unsupported", startup.CurrentStatus())
	}
}
