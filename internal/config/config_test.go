package config_test

import (
	"testing"

	"github.com/jlnesc/gophertchi/internal/config"
)

func TestDefaultAnimationsOff(t *testing.T) {
	cfg := config.DefaultSettings()
	if cfg.Animations {
		t.Fatal("factory default Animations must be false")
	}
	if cfg.SelectedPack != config.DefaultPackName {
		t.Fatalf("SelectedPack = %q, want %q", cfg.SelectedPack, config.DefaultPackName)
	}
}
