package packs_test

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
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

func TestValidPartialPack(t *testing.T) {
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

	entry := packs.Entry{Name: "Partial", Pack: p}
	if !entry.Valid() || entry.MenuLabel() != "Partial" {
		t.Fatalf("menu label = %q", entry.MenuLabel())
	}
}

func TestRecognizedRootPNGIsInvalid(t *testing.T) {
	root := t.TempDir()
	packDir := filepath.Join(root, "Colors")
	writeTinyPNG(t, filepath.Join(packDir, "Hungry.png"))

	err := packs.ValidateLayout(packDir)
	if err == nil {
		t.Fatal("expected malformed layout error")
	}
	if !errors.Is(err, packs.ErrMalformed) {
		t.Fatalf("got %v, want ErrMalformed", err)
	}
	entry := packs.Entry{Name: "Colors", Err: err, Reason: err.Error()}
	if entry.Empty() || entry.MenuLabel() != "Colors (invalid)" {
		t.Fatalf("menu label = %q", entry.MenuLabel())
	}
}

func TestEmptyDirectoryIsEmpty(t *testing.T) {
	root := t.TempDir()
	packDir := filepath.Join(root, "Empty")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	err := packs.ValidateLayout(packDir)
	if err == nil {
		t.Fatal("empty pack should be ErrEmpty")
	}
	if !errors.Is(err, packs.ErrEmpty) {
		t.Fatalf("got %v, want ErrEmpty", err)
	}
	entry := packs.Entry{Name: "Empty", Err: err, Reason: err.Error()}
	if !entry.Empty() || entry.MenuLabel() != "Empty (empty)" {
		t.Fatalf("menu label = %q", entry.MenuLabel())
	}
}

func TestUnknownContentOnlyIsEmpty(t *testing.T) {
	root := t.TempDir()
	packDir := filepath.Join(root, "MyPack")
	if err := os.MkdirAll(filepath.Join(packDir, "icons"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(packDir, "nothing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "nothing", "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "readme.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTinyPNG(t, filepath.Join(packDir, "logo.png")) // unknown PNG name, tolerated

	err := packs.ValidateLayout(packDir)
	if err == nil {
		t.Fatal("expected ErrEmpty")
	}
	if !errors.Is(err, packs.ErrEmpty) {
		t.Fatalf("got %v, want ErrEmpty", err)
	}
	entry := packs.Entry{Name: "MyPack", Err: err}
	if entry.MenuLabel() != "MyPack (empty)" {
		t.Fatalf("menu label = %q", entry.MenuLabel())
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

func TestCatalogReportsEmptyAndInvalid(t *testing.T) {
	root := t.TempDir()
	writeTinyPNG(t, filepath.Join(root, "Bad", "Hungry.png"))
	writeTinyPNG(t, filepath.Join(root, "Good", "icons", "Happy.png"))
	if err := os.MkdirAll(filepath.Join(root, "Empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	c := packs.NewCatalog(root, identityPrepare)
	def := &packs.Pack{Name: "Default", Moods: map[mood.Mood]packs.MoodAssets{}}
	for _, m := range packs.AllMoods() {
		def.Moods[m] = packs.MoodAssets{Icon: []byte(m.String())}
	}
	if err := c.Reload(func() (*packs.Pack, error) { return def, nil }); err != nil {
		t.Fatal(err)
	}

	if len(c.Entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(c.Entries))
	}
	for _, e := range c.Entries {
		switch e.Name {
		case "Bad":
			if e.Valid() || e.Empty() || e.MenuLabel() != "Bad (invalid)" {
				t.Fatalf("Bad entry: valid=%v empty=%v label=%q err=%v", e.Valid(), e.Empty(), e.MenuLabel(), e.Err)
			}
		case "Empty":
			if e.Valid() || !e.Empty() || e.MenuLabel() != "Empty (empty)" {
				t.Fatalf("Empty entry: valid=%v empty=%v label=%q err=%v", e.Valid(), e.Empty(), e.MenuLabel(), e.Err)
			}
		case "Good":
			if !e.Valid() {
				t.Fatalf("Good pack invalid: %v", e.Err)
			}
		}
	}
	if c.Selected("Bad") != nil || c.Selected("Empty") != nil {
		t.Fatal("unusable packs must not be selectable")
	}
}
