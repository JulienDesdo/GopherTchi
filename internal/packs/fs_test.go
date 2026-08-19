package packs_test

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jlnesc/gophertchi/internal/mood"
	"github.com/jlnesc/gophertchi/internal/packs"
)

func writeTinyPNG(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{0, 172, 215, 255})
	img.Set(1, 1, color.RGBA{244, 208, 59, 255})
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func identityPrepare(b []byte) ([]byte, error) { return b, nil }

func TestLoadValidPartialPack(t *testing.T) {
	root := t.TempDir()
	packDir := filepath.Join(root, "Partial")
	writeTinyPNG(t, filepath.Join(packDir, "icons", "Happy.png"))
	writeTinyPNG(t, filepath.Join(packDir, "icons", "Sad.png"))

	if err := packs.ValidateLayout(packDir); err != nil {
		t.Fatalf("ValidateLayout: %v", err)
	}
	p, err := packs.LoadFromDir(packDir, "Partial", identityPrepare)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Moods[mood.Happy].HasRepresentation() || !p.Moods[mood.Sad].HasRepresentation() {
		t.Fatal("expected Happy and Sad icons")
	}
	if p.Moods[mood.Tired].HasRepresentation() {
		t.Fatal("Tired should be absent in partial pack")
	}
}

func TestMalformedRootPNG(t *testing.T) {
	root := t.TempDir()
	packDir := filepath.Join(root, "Colors")
	writeTinyPNG(t, filepath.Join(packDir, "Tired.png"))

	err := packs.ValidateLayout(packDir)
	if err == nil {
		t.Fatal("expected malformed layout error")
	}
	if got := err.Error(); got == "" || !containsAll(got, "Tired.png", "icons/") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEmptyPackIsInvalid(t *testing.T) {
	root := t.TempDir()
	packDir := filepath.Join(root, "Empty")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	err := packs.ValidateLayout(packDir)
	if err == nil {
		t.Fatal("empty pack should be invalid")
	}
	if !strings.Contains(err.Error(), "no usable assets") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidPackWithDSStore(t *testing.T) {
	root := t.TempDir()
	packDir := filepath.Join(root, "WithJunk")
	writeTinyPNG(t, filepath.Join(packDir, "icons", "Happy.png"))
	if err := os.WriteFile(filepath.Join(packDir, ".DS_Store"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "icons", ".DS_Store"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := packs.ValidateLayout(packDir); err != nil {
		t.Fatalf("ValidateLayout: %v", err)
	}
	p, err := packs.LoadFromDir(packDir, "WithJunk", identityPrepare)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Moods[mood.Happy].HasRepresentation() {
		t.Fatal("expected Happy icon")
	}
}

func TestCatalogReportsInvalidPack(t *testing.T) {
	root := t.TempDir()
	writeTinyPNG(t, filepath.Join(root, "Bad", "Hungry.png"))
	writeTinyPNG(t, filepath.Join(root, "Good", "icons", "Happy.png"))

	c := packs.NewCatalog(root, identityPrepare)
	def := &packs.Pack{Name: "Default", Moods: map[mood.Mood]packs.MoodAssets{}}
	for _, m := range packs.AllMoods() {
		def.Moods[m] = packs.MoodAssets{Icon: []byte(m.String())}
	}
	if err := c.Reload(func() (*packs.Pack, error) { return def, nil }); err != nil {
		t.Fatal(err)
	}

	if len(c.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(c.Entries))
	}
	var sawBad, sawGood bool
	for _, e := range c.Entries {
		switch e.Name {
		case "Bad":
			sawBad = true
			if e.Valid() {
				t.Fatal("Bad pack should be invalid")
			}
		case "Good":
			sawGood = true
			if !e.Valid() {
				t.Fatalf("Good pack invalid: %v", e.Err)
			}
		}
	}
	if !sawBad || !sawGood {
		t.Fatalf("missing entries: %+v", c.Entries)
	}
	if c.Selected("Bad") != nil {
		t.Fatal("invalid pack must not be selectable")
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
