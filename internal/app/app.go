package app

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/jlnesc/gophertchi/internal/animation"
	"github.com/jlnesc/gophertchi/internal/config"
	"github.com/jlnesc/gophertchi/internal/iconutil"
	"github.com/jlnesc/gophertchi/internal/metrics"
	"github.com/jlnesc/gophertchi/internal/mood"
	"github.com/jlnesc/gophertchi/internal/packs"
	"github.com/jlnesc/gophertchi/internal/startup"
)

const animationInterval = 180 * time.Millisecond

// App wires systray UI, metrics, mood, packs, and animation together.
type App struct {
	defaultAssets embed.FS

	store     *config.Store
	catalog   *packs.Catalog
	player    *animation.Player
	evaluator *mood.Evaluator

	mu          sync.RWMutex
	settings    config.Settings
	currentMood mood.Mood
	lastSnap    metrics.Snapshot

	packMenu      *systray.MenuItem
	packItems     map[string]*systray.MenuItem
	invalidItems  []*systray.MenuItem
	openPacksItem *systray.MenuItem
	reloadItem    *systray.MenuItem
	animItem      *systray.MenuItem
	loginItem     *systray.MenuItem

	packCancel context.CancelFunc
	actionCancel context.CancelFunc
}

// New prepares configuration, packs, and runtime state.
func New(defaultAssets embed.FS) (*App, error) {
	store, err := config.NewStore()
	if err != nil {
		return nil, err
	}
	settings, err := store.Load()
	if err != nil {
		return nil, err
	}

	catalog := packs.NewCatalog(store.PacksDir(), iconutil.PrepareForMenuBar)
	a := &App{
		defaultAssets: defaultAssets,
		store:         store,
		settings:      settings,
		catalog:       catalog,
		player:        animation.NewPlayer(),
		evaluator:     mood.DefaultEvaluator(),
		currentMood:   mood.Happy,
		packItems:     make(map[string]*systray.MenuItem),
	}
	if err := a.reloadCatalog(); err != nil {
		return nil, err
	}
	a.syncLaunchAtLogin()
	return a, nil
}

// OnReady builds the systray menu and starts background workers.
func (a *App) OnReady() {
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

	a.packMenu = systray.AddMenuItem("Gopher Pack", "Choose a Gopher Pack")
	a.rebuildPackMenu()

	settingsMenu := systray.AddMenuItem("Settings", "GopherTchi settings")
	cfg := a.getSettings()
	a.animItem = settingsMenu.AddSubMenuItemCheckbox("Animations", "Animate Gopher sprites when available", cfg.Animations)
	a.loginItem = settingsMenu.AddSubMenuItemCheckbox("Launch at Login", "Start GopherTchi when you log in", false)
	a.applyLoginMenuState()
	systray.AddSeparator()

	quitItem := systray.AddMenuItem("Quit", "Exit GopherTchi")

	a.refreshIcon(a.getMood())
	a.startFixedMenuHandlers()

	go a.runAnimationLoop()
	go a.pollMetrics(cpuItem, ramItem, diskItem, moodItem)
	go func() {
		<-quitItem.ClickedCh
		systray.Quit()
		os.Exit(0)
	}()
}

// OnExit is called when systray shuts down.
func (a *App) OnExit() {}

func (a *App) getSettings() config.Settings {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.settings
}

func (a *App) setSettings(cfg config.Settings) {
	a.mu.Lock()
	a.settings = cfg
	a.mu.Unlock()
}

func (a *App) getMood() mood.Mood {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentMood
}

func (a *App) reloadCatalog() error {
	return a.catalog.Reload(func() (*packs.Pack, error) {
		return packs.LoadFromEmbed(config.DefaultPackName, a.defaultAssets, "assets", iconutil.PrepareForMenuBar)
	})
}

func (a *App) rebuildPackMenu() {
	a.mu.Lock()
	if a.packCancel != nil {
		a.packCancel()
		a.packCancel = nil
	}
	if a.actionCancel != nil {
		a.actionCancel()
		a.actionCancel = nil
	}
	a.packMenu.ClearSubMenu()
	a.packItems = make(map[string]*systray.MenuItem)
	a.invalidItems = nil

	cfg := a.settings
	defaultItem := a.packMenu.AddSubMenuItemCheckbox("Default", "Built-in Gopher pack", cfg.SelectedPack == config.DefaultPackName)
	a.packItems[config.DefaultPackName] = defaultItem

	for _, entry := range a.catalog.Entries {
		if entry.Valid() {
			item := a.packMenu.AddSubMenuItemCheckbox(entry.Name, "User Gopher pack", cfg.SelectedPack == entry.Name)
			a.packItems[entry.Name] = item
			continue
		}
		label := entry.Name + " (invalid)"
		tip := entry.Reason
		if tip == "" {
			tip = "Invalid Gopher Pack layout"
		}
		item := a.packMenu.AddSubMenuItem(label, tip)
		item.Disable()
		a.invalidItems = append(a.invalidItems, item)
	}

	a.packMenu.AddSubMenuSeparator()
	a.openPacksItem = a.packMenu.AddSubMenuItem("Open Packs Folder", "Open ~/Library/Application Support/GopherTchi/packs")
	a.reloadItem = a.packMenu.AddSubMenuItem("Reload Packs", "Rescan and reload pack assets")
	a.mu.Unlock()

	a.startPackWatchers()
	a.startActionWatchers()
}

func (a *App) startFixedMenuHandlers() {
	go a.watchClicks(a.animItem, a.toggleAnimations)
	go a.watchClicks(a.loginItem, a.toggleLaunchAtLogin)
}

func (a *App) watchClicks(item *systray.MenuItem, handler func()) {
	if item == nil {
		return
	}
	for range item.ClickedCh {
		handler()
	}
}

func (a *App) startPackWatchers() {
	a.mu.Lock()
	if a.packCancel != nil {
		a.packCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.packCancel = cancel
	items := make(map[string]*systray.MenuItem, len(a.packItems))
	for name, item := range a.packItems {
		items[name] = item
	}
	a.mu.Unlock()

	for name, item := range items {
		name, item := name, item
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-item.ClickedCh:
					a.selectPack(name)
				}
			}
		}()
	}
}

func (a *App) startActionWatchers() {
	a.mu.Lock()
	if a.actionCancel != nil {
		a.actionCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.actionCancel = cancel
	openItem := a.openPacksItem
	reloadItem := a.reloadItem
	a.mu.Unlock()

	if openItem != nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-openItem.ClickedCh:
					a.openPacksFolder()
				}
			}
		}()
	}
	if reloadItem != nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-reloadItem.ClickedCh:
					a.handleReloadPacks()
				}
			}
		}()
	}
}

func (a *App) openPacksFolder() {
	if err := packs.OpenPacksFolder(a.store.PacksDir()); err != nil {
		log.Printf("open packs folder: %v", err)
	}
}

func (a *App) selectPack(name string) {
	if name != config.DefaultPackName && a.catalog.Selected(name) == nil {
		a.updatePackChecks()
		return
	}
	cfg := a.getSettings()
	if cfg.SelectedPack == name {
		a.updatePackChecks()
		return
	}
	cfg.SelectedPack = name
	a.setSettings(cfg)
	if err := a.store.Save(cfg); err != nil {
		log.Printf("save config: %v", err)
	}
	a.updatePackChecks()
	a.refreshIcon(a.getMood())
}

func (a *App) handleReloadPacks() {
	if err := a.reloadCatalog(); err != nil {
		log.Printf("reload packs: %v", err)
		return
	}
	cfg := a.getSettings()
	if cfg.SelectedPack != config.DefaultPackName && a.catalog.Selected(cfg.SelectedPack) == nil {
		cfg.SelectedPack = config.DefaultPackName
		a.setSettings(cfg)
		_ = a.store.Save(cfg)
	}
	a.rebuildPackMenu()
	a.refreshIcon(a.getMood())
}

func (a *App) updatePackChecks() {
	cfg := a.getSettings()
	a.mu.RLock()
	defer a.mu.RUnlock()
	for name, item := range a.packItems {
		if name == cfg.SelectedPack {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

func (a *App) toggleAnimations() {
	cfg := a.getSettings()
	cfg.Animations = !cfg.Animations
	a.setSettings(cfg)
	if cfg.Animations {
		a.animItem.Check()
	} else {
		a.animItem.Uncheck()
	}
	if err := a.store.Save(cfg); err != nil {
		log.Printf("save config: %v", err)
	}
	a.refreshIcon(a.getMood())
}

func (a *App) toggleLaunchAtLogin() {
	if !startup.Supported() {
		a.applyLoginMenuState()
		return
	}

	desired := startup.CurrentStatus() != startup.StatusEnabled
	if err := startup.SetEnabled(desired); err != nil {
		log.Printf("launch at login: %v", err)
	}

	cfg := a.getSettings()
	cfg.LaunchAtLogin = startup.Enabled()
	a.setSettings(cfg)
	_ = a.store.Save(cfg)
	a.applyLoginMenuState()
}

func (a *App) syncLaunchAtLogin() {
	if !startup.Supported() {
		cfg := a.getSettings()
		cfg.LaunchAtLogin = false
		a.setSettings(cfg)
		return
	}

	cfg := a.getSettings()
	st := startup.CurrentStatus()
	if cfg.LaunchAtLogin && st == startup.StatusNotRegistered {
		if err := startup.SetEnabled(true); err != nil {
			log.Printf("restore launch at login: %v", err)
			cfg.LaunchAtLogin = false
			a.setSettings(cfg)
			_ = a.store.Save(cfg)
		}
	}
	if !cfg.LaunchAtLogin && st == startup.StatusEnabled {
		if err := startup.SetEnabled(false); err != nil {
			log.Printf("clear launch at login: %v", err)
		}
	}

	cfg = a.getSettings()
	cfg.LaunchAtLogin = startup.Enabled()
	a.setSettings(cfg)
}

func (a *App) applyLoginMenuState() {
	if a.loginItem == nil {
		return
	}
	if !startup.Supported() {
		a.loginItem.Disable()
		a.loginItem.Uncheck()
		a.loginItem.SetTooltip("Available only when running GopherTchi.app")
		return
	}

	a.loginItem.Enable()
	switch startup.CurrentStatus() {
	case startup.StatusEnabled:
		a.loginItem.Check()
		a.loginItem.SetTooltip("GopherTchi launches at login")
	case startup.StatusRequiresApproval:
		a.loginItem.Uncheck()
		a.loginItem.SetTooltip("Requires approval in System Settings → General → Login Items")
	case startup.StatusNotFound, startup.StatusError:
		a.loginItem.Uncheck()
		a.loginItem.SetTooltip("Launch at Login is temporarily unavailable")
	default:
		a.loginItem.Uncheck()
		a.loginItem.SetTooltip("Start GopherTchi when you log in")
	}
}

func (a *App) runAnimationLoop() {
	ticker := time.NewTicker(animationInterval)
	defer ticker.Stop()
	for range ticker.C {
		cfg := a.getSettings()
		if !cfg.Animations || !a.player.IsAnimated() {
			continue
		}
		if frame := a.player.Advance(); len(frame) > 0 {
			systray.SetIcon(frame)
		}
	}
}

func (a *App) pollMetrics(cpuItem, ramItem, diskItem, moodItem *systray.MenuItem) {
	reader := metrics.DefaultReader()
	if iv := os.Getenv("GOPHERTCHI_POLL_INTERVAL"); iv != "" {
		if d, err := time.ParseDuration(iv); err == nil {
			reader.Interval = d
		}
	}
	if iv := os.Getenv("GOPHERTCHI_DWELL_TIME"); iv != "" {
		if d, err := time.ParseDuration(iv); err == nil {
			a.evaluator.DwellTime = d
		}
	}

	ctx := context.Background()
	ticker := time.NewTicker(reader.Interval)
	defer ticker.Stop()

	poll := func() {
		snap, err := reader.Poll(ctx)
		if err != nil {
			log.Printf("metrics poll: %v", err)
			return
		}
		a.applyUpdate(snap, cpuItem, ramItem, diskItem, moodItem)
	}

	poll()
	for range ticker.C {
		poll()
	}
}

func (a *App) applyUpdate(snap metrics.Snapshot, cpuItem, ramItem, diskItem, moodItem *systray.MenuItem) {
	newMood := a.evaluator.Update(snap)

	a.mu.Lock()
	prevMood := a.currentMood
	a.lastSnap = snap
	a.currentMood = newMood
	a.mu.Unlock()

	cpuItem.SetTitle(fmt.Sprintf("CPU:  %.1f%%", snap.CPU))
	ramItem.SetTitle(fmt.Sprintf("RAM:  %.1f%%", snap.Memory))
	diskItem.SetTitle(fmt.Sprintf("Disk: %.1f%%", snap.Disk))
	moodItem.SetTitle(fmt.Sprintf("Mood: %s", newMood))

	if newMood != prevMood {
		a.refreshIcon(newMood)
	}
}

func (a *App) refreshIcon(m mood.Mood) {
	cfg := a.getSettings()
	assets := a.catalog.ResolveMood(cfg.Animations, cfg.SelectedPack, m)
	if !assets.HasRepresentation() {
		log.Printf("no assets for mood %s", m)
		return
	}
	a.player.SetMood(m, assets, cfg.Animations)
	if frame := a.player.Current(); len(frame) > 0 {
		systray.SetIcon(frame)
	}
}
