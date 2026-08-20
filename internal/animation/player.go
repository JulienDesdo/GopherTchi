package animation

import (
	"sync"

	"github.com/jlnesc/gophertchi/internal/mood"
	"github.com/jlnesc/gophertchi/internal/packs"
)

// Player cycles through preloaded sprite frames without decoding PNGs at runtime.
type Player struct {
	mu sync.Mutex

	enabled   bool
	mood      mood.Mood
	icon      []byte
	frames    [][]byte
	frameIdx  int
}

// NewPlayer returns a player starting on Happy with no frames.
func NewPlayer() *Player {
	return &Player{mood: mood.Happy}
}

// SetEnabled toggles animation without resetting the current mood assets.
func (p *Player) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
	if !enabled {
		p.frameIdx = 0
	}
}

// Enabled reports whether animation is active.
func (p *Player) Enabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enabled
}

// SetMood updates assets for a mood. Resets the frame index.
func (p *Player) SetMood(m mood.Mood, assets packs.MoodAssets, animations bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.mood = m
	p.enabled = animations
	p.frameIdx = 0
	p.frames = nil
	p.icon = nil

	if animations && len(assets.Sprite) > 0 {
		p.frames = assets.Sprite
		return
	}
	if len(assets.Icon) > 0 {
		p.icon = assets.Icon
		return
	}
	if len(assets.Sprite) > 0 {
		p.icon = assets.Sprite[0]
	}
}

// Current returns the PNG bytes that should be shown right now.
func (p *Player) Current() []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentLocked()
}

// Advance moves to the next animation frame when enabled and returns the new bytes.
// When animation is off or only a static icon is available, returns the same bytes.
func (p *Player) Advance() []byte {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.enabled && len(p.frames) > 1 {
		p.frameIdx = (p.frameIdx + 1) % len(p.frames)
	}
	return p.currentLocked()
}

func (p *Player) currentLocked() []byte {
	if p.enabled && len(p.frames) > 0 {
		return p.frames[p.frameIdx]
	}
	return p.icon
}

// Mood returns the mood currently loaded in the player.
func (p *Player) Mood() mood.Mood {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mood
}

// IsAnimated reports whether the player is cycling sprite frames.
func (p *Player) IsAnimated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enabled && len(p.frames) > 1
}
