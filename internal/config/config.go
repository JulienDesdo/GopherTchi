package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	DefaultPackName = "Default"
	fileName        = "config.json"
)

// Settings holds user preferences persisted across restarts.
type Settings struct {
	Animations    bool   `json:"animations"`
	LaunchAtLogin bool   `json:"launch_at_login"`
	SelectedPack  string `json:"selected_pack"`
}

// DefaultSettings returns factory defaults.
func DefaultSettings() Settings {
	return Settings{
		Animations:    true,
		LaunchAtLogin: false,
		SelectedPack:  DefaultPackName,
	}
}

// Store loads and saves settings under ~/Library/Application Support/GopherTchi/.
type Store struct {
	mu   sync.Mutex
	dir  string
	path string
}

// NewStore opens or creates the application support directory.
func NewStore() (*Store, error) {
	dir, err := appSupportDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create app support dir: %w", err)
	}
	packsDir := filepath.Join(dir, "packs")
	if err := os.MkdirAll(packsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create packs dir: %w", err)
	}
	return &Store{
		dir:  dir,
		path: filepath.Join(dir, fileName),
	}, nil
}

// Dir returns the application support root (…/GopherTchi).
func (s *Store) Dir() string {
	return s.dir
}

// PacksDir returns …/GopherTchi/packs.
func (s *Store) PacksDir() string {
	return filepath.Join(s.dir, "packs")
}

// Load reads settings from disk, returning defaults when the file is absent.
func (s *Store) Load() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultSettings(), nil
		}
		return Settings{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Settings{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.SelectedPack == "" {
		cfg.SelectedPack = DefaultPackName
	}
	return cfg, nil
}

// Save writes settings to disk.
func (s *Store) Save(cfg Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cfg.SelectedPack == "" {
		cfg.SelectedPack = DefaultPackName
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("commit config: %w", err)
	}
	return nil
}

func appSupportDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "GopherTchi"), nil
}
