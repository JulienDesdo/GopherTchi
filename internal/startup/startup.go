package startup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const launchAgentLabel = "com.gophertchi.app"

// ErrUnavailable is returned when launch-at-login cannot be used.
var ErrUnavailable = errors.New("launch at login unavailable outside GopherTchi.app")

// Supported reports whether launch-at-login can be toggled in this environment.
// It is only available when running from a macOS .app bundle.
func Supported() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, ok := appExecutable()
	return ok
}

// Enabled reports whether the LaunchAgent is currently installed.
func Enabled() bool {
	if !Supported() {
		return false
	}
	_, err := os.Stat(agentPlistPath())
	return err == nil
}

// SetEnabled registers or removes the LaunchAgent for the current .app.
// Calling SetEnabled(true) repeatedly overwrites the same plist (no duplicates).
func SetEnabled(enable bool) error {
	if !Supported() {
		return ErrUnavailable
	}

	exe, ok := appExecutable()
	if !ok {
		return ErrUnavailable
	}

	plistPath := agentPlistPath()
	if !enable {
		_ = unloadAgent()
		if err := os.Remove(plistPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove launch agent: %w", err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`, launchAgentLabel, xmlEscape(exe))

	tmp := plistPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write launch agent: %w", err)
	}
	if err := os.Rename(tmp, plistPath); err != nil {
		return fmt.Errorf("commit launch agent: %w", err)
	}

	_ = unloadAgent()
	return loadAgent()
}

func agentPlistPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
}

func loadAgent() error {
	return exec.Command("launchctl", "load", "-w", agentPlistPath()).Run()
}

func unloadAgent() error {
	return exec.Command("launchctl", "unload", "-w", agentPlistPath()).Run()
}

// appExecutable returns the absolute path to the binary when running inside a .app.
func appExecutable() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", false
	}
	if !strings.Contains(exe, ".app/Contents/MacOS/") {
		return "", false
	}
	return exe, true
}

func xmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(s)
}
