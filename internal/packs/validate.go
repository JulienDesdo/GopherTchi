package packs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Entry is a discovered user pack, valid or invalid.
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

// ValidateLayout checks that a pack directory uses icons/ and sprites/ correctly.
// Partial packs (including empty ones) are valid. Root-level mood PNGs or mood
// folders are malformed.
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
				return fmt.Errorf("malformed pack layout: mood folder %q must be under sprites/ (expected sprites/%s/), not at pack root", name, name)
			}
			continue
		}

		if strings.EqualFold(filepath.Ext(name), ".png") {
			if _, ok := moodByFileName(name); ok {
				return fmt.Errorf("malformed pack layout: %s should be at icons/%s, not at pack root", name, name)
			}
			return fmt.Errorf("malformed pack layout: unexpected file %q at pack root (place icons in icons/)", name)
		}
	}

	if err := validateSpritesLayout(filepath.Join(dir, "sprites")); err != nil {
		return err
	}
	if err := validateIconsLayout(filepath.Join(dir, "icons")); err != nil {
		return err
	}
	return nil
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
			return fmt.Errorf("malformed pack layout: unexpected directory icons/%s (icons must be PNG files)", name)
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
					return fmt.Errorf("malformed pack layout: sprite frames for %s must be under sprites/%s/, not sprites/%s", base, base, name)
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
