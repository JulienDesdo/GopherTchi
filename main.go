package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/jlnesc/gophertchi/internal/iconutil"
	"github.com/jlnesc/gophertchi/internal/metrics"
	"github.com/jlnesc/gophertchi/internal/mood"
)

//go:embed assets/icons/*.png
var iconFS embed.FS

const (
	defaultPollInterval = 3 * time.Second
)

var (
	mu          sync.RWMutex
	lastSnap    metrics.Snapshot
	currentMood mood.Mood = mood.Happy
	iconCache   map[mood.Mood][]byte
)

func main() {
	if err := loadIcons(); err != nil {
		log.Fatalf("load icons: %v", err)
	}

	systray.Run(onReady, onExit)
}

func loadIcons() error {
	iconCache = make(map[mood.Mood][]byte)
	moods := []mood.Mood{mood.Happy, mood.Hungry, mood.Tired, mood.Sad, mood.Sleeping}
	for _, m := range moods {
		data, err := iconFS.ReadFile("assets/icons/" + m.FileName())
		if err != nil {
			return fmt.Errorf("read %s: %w", m.FileName(), err)
		}
		prepared, err := iconutil.PrepareForMenuBar(data)
		if err != nil {
			return fmt.Errorf("prepare %s: %w", m.FileName(), err)
		}
		iconCache[m] = prepared
	}
	return nil
}

func onReady() {
	systray.SetTitle("")
	systray.SetTooltip("GopherTchi — system health Tamagotchi")

	titleItem := systray.AddMenuItem("GopherTchi", "GopherTchi — system health Tamagotchi")
	titleItem.Disable()
	systray.AddSeparator()

	cpuItem := systray.AddMenuItem("CPU: —", "Current CPU usage")
	cpuItem.Disable()
	ramItem := systray.AddMenuItem("RAM: —", "Current memory usage")
	ramItem.Disable()
	diskItem := systray.AddMenuItem("Disk: —", "Current disk usage")
	diskItem.Disable()
	systray.AddSeparator()
	moodItem := systray.AddMenuItem("Mood: Happy", "Current Gopher mood")
	moodItem.Disable()
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "Exit GopherTchi")

	go pollMetrics(cpuItem, ramItem, diskItem, moodItem)
	go func() {
		<-quitItem.ClickedCh
		systray.Quit()
		os.Exit(0)
	}()

	setIcon(mood.Happy)
}

func onExit() {}

func pollMetrics(cpuItem, ramItem, diskItem, moodItem *systray.MenuItem) {
	reader := metrics.DefaultReader()
	if iv := os.Getenv("GOPHERTCHI_POLL_INTERVAL"); iv != "" {
		if d, err := time.ParseDuration(iv); err == nil {
			reader.Interval = d
		}
	}

	evaluator := mood.DefaultEvaluator()
	if iv := os.Getenv("GOPHERTCHI_DWELL_TIME"); iv != "" {
		if d, err := time.ParseDuration(iv); err == nil {
			evaluator.DwellTime = d
		}
	}

	ctx := context.Background()
	ticker := time.NewTicker(reader.Interval)
	defer ticker.Stop()

	// Initial poll (blocks ~1s for CPU sampling).
	if snap, err := reader.Poll(ctx); err == nil {
		applyUpdate(snap, evaluator, cpuItem, ramItem, diskItem, moodItem)
	}

	for range ticker.C {
		snap, err := reader.Poll(ctx)
		if err != nil {
			log.Printf("metrics poll: %v", err)
			continue
		}
		applyUpdate(snap, evaluator, cpuItem, ramItem, diskItem, moodItem)
	}
}

func applyUpdate(
	snap metrics.Snapshot,
	evaluator *mood.Evaluator,
	cpuItem, ramItem, diskItem, moodItem *systray.MenuItem,
) {
	newMood := evaluator.Update(snap)

	mu.Lock()
	prevMood := currentMood
	lastSnap = snap
	currentMood = newMood
	mu.Unlock()

	cpuItem.SetTitle(fmt.Sprintf("CPU:  %.1f%%", snap.CPU))
	ramItem.SetTitle(fmt.Sprintf("RAM:  %.1f%%", snap.Memory))
	diskItem.SetTitle(fmt.Sprintf("Disk: %.1f%%", snap.Disk))
	moodItem.SetTitle(fmt.Sprintf("Mood: %s", newMood))

	if newMood != prevMood {
		setIcon(newMood)
	}
}

func setIcon(m mood.Mood) {
	mu.RLock()
	data, ok := iconCache[m]
	mu.RUnlock()
	if ok {
		systray.SetIcon(data)
	}
}
