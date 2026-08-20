package packs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sentinel validation results for unusable packs.
var (
	// ErrEmpty means the folder has no recognized usable Gopher assets.
	ErrEmpty = errors.New("no usable assets")
	// ErrMalformed means a recognized Gopher asset is in the wrong place.
	ErrMalformed = errors.New("malformed pack layout")
)

// Entry is a discovered user pack, valid or unusable.
type Entry struct {
	Name   string
	Pack   *Pack
	Err    error
	Reason string
}

// Valid reports whether the pack can be selected.
func (e Entry) Valid() bool {
	return e.Err == nil && e.Pack != nil
}

// Empty reports whether the pack has no recognized usable assets.
func (e Entry) Empty() bool {
	return errors.Is(e.Err, ErrEmpty)
}

// MenuLabel returns the submenu title for this entry.
func (e Entry) MenuLabel() string {
	if e.Valid() {
		return e.Name
	}
	if e.Empty() {
		return e.Name + " (empty)"
	}
	return e.Name + " (invalid)"
}

// MenuTooltip returns a short explanation for disabled entries.
func (e Entry) MenuTooltip() string {
	if e.Valid() {
		return "User Gopher pack"
	}
	if e.Reason != "" {
		return e.Reason
	}
	if e.Empty() {
		return "No usable Gopher assets found"
	}
	return "Invalid Gopher Pack layout"
}

// ValidateLayout checks that a pack directory uses icons/ and sprites/ correctly.
// Partial packs are valid when they contain at least one recognized icon or sprite.
// Folders with no recognized assets are empty (ErrEmpty). Recognized assets in the
// wrong place are malformed (ErrMalformed). Unknown files/directories are ignored.
func ValidateLayout(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read pack dir: %w", err)
	}

	for _, e := range entries {
		name := e.Name()
		if shouldIgnorePackFile(name) {
			continue
		}

		if e.IsDir() {
			switch strings.ToLower(name) {
			case "icons", "sprites":
				continue
			}
			if _, ok := moodByFolderName(name); ok {
				return fmt.Errorf("%w: mood folder %q must be under sprites/ (expected sprites/%s/), not at pack root", ErrMalformed, name, name)
			}
			// Unknown directories are tolerated for future pack extensions.
			continue
		}

		if strings.EqualFold(filepath.Ext(name), ".png") {
			if _, ok := moodByFileName(name); ok {
				return fmt.Errorf("%w: %s should be at icons/%s, not at pack root", ErrMalformed, name, name)
			}
			// Unknown PNGs at pack root are tolerated.
			continue
		}
	}

	if err := validateSpritesLayout(filepath.Join(dir, "sprites")); err != nil {
		return err
	}
	if err := validateIconsLayout(filepath.Join(dir, "icons")); err != nil {
		return err
	}
	if !hasRecognizedAssets(dir) {
		return fmt.Errorf("%w: pack needs at least one recognized icon or sprite", ErrEmpty)
	}
	return nil
}

func hasRecognizedAssets(dir string) bool {
	iconsDir := filepath.Join(dir, "icons")
	if files, err := listPNGFiles(iconsDir); err == nil {
		for _, fn := range files {
			if _, ok := moodByFileName(fn); ok {
				return true
			}
		}
	}

	spritesDir := filepath.Join(dir, "sprites")
	entries, err := os.ReadDir(spritesDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() || shouldIgnorePackFile(e.Name()) {
			continue
		}
		if _, ok := moodByFolderName(e.Name()); !ok {
			continue
		}
		files, err := listPNGFiles(filepath.Join(spritesDir, e.Name()))
		if err == nil && len(files) > 0 {
			return true
		}
	}
	return false
}

func validateIconsLayout(iconsDir string) error {
	entries, err := os.ReadDir(iconsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if shouldIgnorePackFile(name) {
			continue
		}
		if e.IsDir() {
			return fmt.Errorf("%w: unexpected directory icons/%s (icons must be PNG files)", ErrMalformed, name)
		}
	}
	return nil
}

func validateSpritesLayout(spritesDir string) error {
	entries, err := os.ReadDir(spritesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if shouldIgnorePackFile(name) {
			continue
		}
		if !e.IsDir() {
			if strings.EqualFold(filepath.Ext(name), ".png") {
				if _, ok := moodByFileName(name); ok {
					base := strings.TrimSuffix(name, filepath.Ext(name))
					return fmt.Errorf("%w: sprite frames for %s must be under sprites/%s/, not sprites/%s", ErrMalformed, base, base, name)
				}
			}
			continue
		}
		if _, ok := moodByFolderName(name); !ok {
			continue
		}
	}
	return nil
}

func shouldIgnorePackFile(name string) bool {
	return name == ".DS_Store" || strings.HasPrefix(name, ".")
}
