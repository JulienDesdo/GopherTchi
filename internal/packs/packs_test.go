package packs_test

import (
	"testing"

	"github.com/jlnesc/gophertchi/internal/mood"
	"github.com/jlnesc/gophertchi/internal/packs"
)

func icon(label string) []byte { return []byte("icon:" + label) }
func frame(label string) []byte { return []byte("frame:" + label) }

func TestResolveAnimationsPreferUserSprites(t *testing.T) {
	user := &packs.Pack{
		Name: "User",
		Moods: map[mood.Mood]packs.MoodAssets{
			mood.Happy: {Sprite: [][]byte{frame("u0"), frame("u1")}, Icon: icon("u")},
		},
	}
	def := &packs.Pack{
		Name: "Default",
		Moods: map[mood.Mood]packs.MoodAssets{
			mood.Happy: {Sprite: [][]byte{frame("d0")}, Icon: icon("d")},
		},
	}

	got := packs.Resolve(true, user, def, mood.Happy)
	if len(got.Sprite) != 2 || string(got.Sprite[0]) != "frame:u0" {
		t.Fatalf("expected user sprites, got %+v", got)
	}
}

func TestResolveAnimationsFallbackChain(t *testing.T) {
	def := &packs.Pack{
		Name: "Default",
		Moods: map[mood.Mood]packs.MoodAssets{
			mood.Happy: {Icon: icon("d")},
			mood.Sad:   {Sprite: [][]byte{frame("d-sad")}},
		},
	}

	// No user pack → default sprites when present.
	got := packs.Resolve(true, nil, def, mood.Sad)
	if len(got.Sprite) != 1 {
		t.Fatalf("expected default sprites, got %+v", got)
	}

	// User icon only → static icon while animations on.
	user := &packs.Pack{
		Name: "User",
		Moods: map[mood.Mood]packs.MoodAssets{
			mood.Happy: {Icon: icon("u")},
		},
	}
	got = packs.Resolve(true, user, def, mood.Happy)
	if string(got.Icon) != "icon:u" || len(got.Sprite) != 0 {
		t.Fatalf("expected user icon, got %+v", got)
	}

	// Missing mood in user → default icon.
	got = packs.Resolve(true, user, def, mood.Tired)
	if string(got.Icon) != "" && !got.HasRepresentation() {
		t.Fatalf("expected empty or default, got %+v", got)
	}
	got = packs.Resolve(true, user, &packs.Pack{
		Name: "Default",
		Moods: map[mood.Mood]packs.MoodAssets{
			mood.Tired: {Icon: icon("d-tired")},
		},
	}, mood.Tired)
	if string(got.Icon) != "icon:d-tired" {
		t.Fatalf("expected default icon fallback, got %+v", got)
	}
}

func TestResolveAnimationsOffUsesFirstSprite(t *testing.T) {
	user := &packs.Pack{
		Name: "User",
		Moods: map[mood.Mood]packs.MoodAssets{
			mood.Happy: {Sprite: [][]byte{frame("u0"), frame("u1")}},
		},
	}
	def := &packs.Pack{
		Name: "Default",
		Moods: map[mood.Mood]packs.MoodAssets{
			mood.Happy: {Icon: icon("d")},
		},
	}

	got := packs.Resolve(false, user, def, mood.Happy)
	if string(got.Icon) != "frame:u0" {
		t.Fatalf("expected first user sprite as static icon, got %+v", got)
	}
}

func TestValidateComplete(t *testing.T) {
	incomplete := &packs.Pack{
		Name: "Default",
		Moods: map[mood.Mood]packs.MoodAssets{
			mood.Happy: {Icon: icon("h")},
		},
	}
	if err := packs.ValidateComplete(incomplete); err == nil {
		t.Fatal("expected validation error")
	}

	complete := &packs.Pack{Name: "Default", Moods: map[mood.Mood]packs.MoodAssets{}}
	for _, m := range packs.AllMoods() {
		complete.Moods[m] = packs.MoodAssets{Icon: icon(m.String())}
	}
	if err := packs.ValidateComplete(complete); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
