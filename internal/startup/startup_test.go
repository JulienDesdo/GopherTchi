package startup_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		t.Fatal(err)
	}

	inApp := strings.Contains(exe, ".app/Contents/MacOS/")
	if got := startup.Supported(); got != inApp {
		t.Fatalf("Supported() = %v, inAppBundle = %v (exe=%s)", got, inApp, exe)
	}
}
