package animation_test

import (
	"testing"

	"github.com/jlnesc/gophertchi/internal/animation"
	"github.com/jlnesc/gophertchi/internal/mood"
	"github.com/jlnesc/gophertchi/internal/packs"
)

func TestPlayerCyclesPreloadedFrames(t *testing.T) {
	p := animation.NewPlayer()
	frames := [][]byte{[]byte("a"), []byte("b"), []byte("c")}
	p.SetMood(mood.Happy, packs.MoodAssets{Sprite: frames}, true)

	if got := string(p.Current()); got != "a" {
		t.Fatalf("current = %q, want a", got)
	}
	if got := string(p.Advance()); got != "b" {
		t.Fatalf("advance = %q, want b", got)
	}
	if got := string(p.Advance()); got != "c" {
		t.Fatalf("advance = %q, want c", got)
	}
	if got := string(p.Advance()); got != "a" {
		t.Fatalf("wrap = %q, want a", got)
	}
}

func TestPlayerStaticWhenAnimationsOff(t *testing.T) {
	p := animation.NewPlayer()
	p.SetMood(mood.Hungry, packs.MoodAssets{
		Icon:   []byte("icon"),
		Sprite: [][]byte{[]byte("f0"), []byte("f1")},
	}, false)

	if p.IsAnimated() {
		t.Fatal("expected non-animated player")
	}
	if got := string(p.Current()); got != "icon" {
		t.Fatalf("got %q, want icon", got)
	}
	if got := string(p.Advance()); got != "icon" {
		t.Fatalf("advance should stay on icon, got %q", got)
	}
}

func TestPlayerUsesFirstSpriteAsStaticFallback(t *testing.T) {
	p := animation.NewPlayer()
	p.SetMood(mood.Sad, packs.MoodAssets{
		Sprite: [][]byte{[]byte("only")},
	}, false)

	if got := string(p.Current()); got != "only" {
		t.Fatalf("got %q, want only", got)
	}
}
