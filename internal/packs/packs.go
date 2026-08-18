package packs

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jlnesc/gophertchi/internal/mood"
)

// MoodAssets holds prepared menu-bar PNG bytes ready for systray.
type MoodAssets struct {
	Icon   []byte
	Sprite [][]byte
}

// HasRepresentation reports whether at least one icon or sprite frame exists.
func (a MoodAssets) HasRepresentation() bool {
	return len(a.Icon) > 0 || len(a.Sprite) > 0
}

// Pack is a named collection of mood assets.
type Pack struct {
	Name  string
	Moods map[mood.Mood]MoodAssets
}

// AllMoods returns supported moods in stable order.
func AllMoods() []mood.Mood {
	return []mood.Mood{mood.Happy, mood.Hungry, mood.Tired, mood.Sad, mood.Sleeping}
}

// ValidateComplete returns an error when any mood lacks assets.
func ValidateComplete(p *Pack) error {
	if p == nil {
		return fmt.Errorf("pack is nil")
	}
	var missing []string
	for _, m := range AllMoods() {
		if !p.Moods[m].HasRepresentation() {
			missing = append(missing, m.String())
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("pack %q missing moods: %s", p.Name, strings.Join(missing, ", "))
	}
	return nil
}

// Resolve picks assets for a mood with user-pack → Default fallback rules.
func Resolve(animations bool, selected, fallback *Pack, m mood.Mood) MoodAssets {
	sel := moodFromPack(selected, m)
	def := moodFromPack(fallback, m)

	if animations {
		if len(sel.Sprite) > 0 {
			return MoodAssets{Sprite: sel.Sprite}
		}
		if len(sel.Icon) > 0 {
			return MoodAssets{Icon: sel.Icon}
		}
		if len(def.Sprite) > 0 {
			return MoodAssets{Sprite: def.Sprite}
		}
		if len(def.Icon) > 0 {
			return MoodAssets{Icon: def.Icon}
		}
		return MoodAssets{}
	}

	if len(sel.Icon) > 0 {
		return MoodAssets{Icon: sel.Icon}
	}
	if len(sel.Sprite) > 0 {
		return MoodAssets{Icon: sel.Sprite[0]}
	}
	if len(def.Icon) > 0 {
		return MoodAssets{Icon: def.Icon}
	}
	if len(def.Sprite) > 0 {
		return MoodAssets{Icon: def.Sprite[0]}
	}
	return MoodAssets{}
}

func moodFromPack(p *Pack, m mood.Mood) MoodAssets {
	if p == nil || p.Moods == nil {
		return MoodAssets{}
	}
	return p.Moods[m]
}

// Catalog holds the Default pack and discovered user packs.
type Catalog struct {
	Default   *Pack
	User      map[string]*Pack
	PacksDir  string
	loadIcons func([]byte) ([]byte, error)
}

// NewCatalog creates an empty catalog; call Reload to populate.
func NewCatalog(packsDir string, prepare func([]byte) ([]byte, error)) *Catalog {
	return &Catalog{
		User:      make(map[string]*Pack),
		PacksDir:  packsDir,
		loadIcons: prepare,
	}
}

// Reload rescans user packs and validates Default is complete.
func (c *Catalog) Reload(defaultLoader func() (*Pack, error)) error {
	def, err := defaultLoader()
	if err != nil {
		return fmt.Errorf("load Default pack: %w", err)
	}
	if err := ValidateComplete(def); err != nil {
		return fmt.Errorf("Default pack invalid: %w", err)
	}
	c.Default = def

	user := make(map[string]*Pack)
	names, err := ListUserPackNames(c.PacksDir)
	if err != nil {
		return err
	}
	for _, name := range names {
		p, err := LoadFromDir(filepath.Join(c.PacksDir, name), name, c.loadIcons)
		if err != nil {
			continue // skip malformed user packs
		}
		user[name] = p
	}
	c.User = user
	return nil
}

// Selected returns the named pack, or nil for Default-only selection.
func (c *Catalog) Selected(name string) *Pack {
	if name == "" || name == "Default" {
		return nil
	}
	return c.User[name]
}

// UserNames returns sorted user pack folder names.
func (c *Catalog) UserNames() []string {
	names := make([]string, 0, len(c.User))
	for n := range c.User {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ResolveMood resolves assets for the active selection.
func (c *Catalog) ResolveMood(animations bool, selectedName string, m mood.Mood) MoodAssets {
	return Resolve(animations, c.Selected(selectedName), c.Default, m)
}

// ListUserPackNames returns folder names under packsDir.
func ListUserPackNames(packsDir string) ([]string, error) {
	if err := os.MkdirAll(packsDir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		return nil, fmt.Errorf("read packs dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// OpenPacksFolder opens packsDir in Finder (macOS) or default file manager.
func OpenPacksFolder(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return exec.Command("open", dir).Start()
}

// moodByFolderName maps directory names to moods.
func moodByFolderName(name string) (mood.Mood, bool) {
	switch strings.ToLower(name) {
	case "happy":
		return mood.Happy, true
	case "hungry":
		return mood.Hungry, true
	case "tired":
		return mood.Tired, true
	case "sad":
		return mood.Sad, true
	case "sleeping":
		return mood.Sleeping, true
	default:
		return mood.Happy, false
	}
}

func moodByFileName(name string) (mood.Mood, bool) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	switch strings.ToLower(base) {
	case "happy":
		return mood.Happy, true
	case "hungry":
		return mood.Hungry, true
	case "tired":
		return mood.Tired, true
	case "sad":
		return mood.Sad, true
	case "sleeping":
		return mood.Sleeping, true
	default:
		return mood.Happy, false
	}
}

func readAndPrepare(path string, prepare func([]byte) ([]byte, error)) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return prepare(raw)
}

func listPNGFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".png") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

func walkEmbeddedPNG(root fs.FS, rootPath string, prepare func([]byte) ([]byte, error)) (map[mood.Mood]MoodAssets, error) {
	moods := make(map[mood.Mood]MoodAssets)
	for _, m := range AllMoods() {
		moods[m] = MoodAssets{}
	}

	// embed.FS always uses forward slashes.
	iconsDir := path.Join(rootPath, "icons")
	if entries, err := fs.ReadDir(root, iconsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			m, ok := moodByFileName(e.Name())
			if !ok {
				continue
			}
			raw, err := fs.ReadFile(root, path.Join(iconsDir, e.Name()))
			if err != nil {
				return nil, err
			}
			prepared, err := prepare(raw)
			if err != nil {
				return nil, fmt.Errorf("prepare icon %s: %w", e.Name(), err)
			}
			a := moods[m]
			a.Icon = prepared
			moods[m] = a
		}
	}

	spritesDir := path.Join(rootPath, "sprites")
	if entries, err := fs.ReadDir(root, spritesDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			m, ok := moodByFolderName(e.Name())
			if !ok {
				continue
			}
			frameDir := path.Join(spritesDir, e.Name())
			frameEntries, err := fs.ReadDir(root, frameDir)
			if err != nil {
				continue
			}
			var frames [][]byte
			var frameNames []string
			for _, fe := range frameEntries {
				if fe.IsDir() {
					continue
				}
				if strings.EqualFold(filepath.Ext(fe.Name()), ".png") {
					frameNames = append(frameNames, fe.Name())
				}
			}
			sort.Strings(frameNames)
			for _, fn := range frameNames {
				raw, err := fs.ReadFile(root, path.Join(frameDir, fn))
				if err != nil {
					return nil, err
				}
				prepared, err := prepare(raw)
				if err != nil {
					return nil, fmt.Errorf("prepare sprite %s/%s: %w", e.Name(), fn, err)
				}
				frames = append(frames, prepared)
			}
			if len(frames) > 0 {
				a := moods[m]
				a.Sprite = frames
				moods[m] = a
			}
		}
	}

	return moods, nil
}

// LoadFromEmbed loads the built-in Default pack from an embedded filesystem.
func LoadFromEmbed(name string, root fs.FS, rootPath string, prepare func([]byte) ([]byte, error)) (*Pack, error) {
	moods, err := walkEmbeddedPNG(root, rootPath, prepare)
	if err != nil {
		return nil, err
	}
	return &Pack{Name: name, Moods: moods}, nil
}

// LoadFromDir loads a user pack from disk.
func LoadFromDir(dir, name string, prepare func([]byte) ([]byte, error)) (*Pack, error) {
	moods := make(map[mood.Mood]MoodAssets)
	for _, m := range AllMoods() {
		moods[m] = MoodAssets{}
	}

	iconsDir := filepath.Join(dir, "icons")
	if files, err := listPNGFiles(iconsDir); err == nil {
		for _, fn := range files {
			m, ok := moodByFileName(fn)
			if !ok {
				continue
			}
			prepared, err := readAndPrepare(filepath.Join(iconsDir, fn), prepare)
			if err != nil {
				return nil, fmt.Errorf("icon %s: %w", fn, err)
			}
			a := moods[m]
			a.Icon = prepared
			moods[m] = a
		}
	}

	spritesDir := filepath.Join(dir, "sprites")
	spriteEntries, err := os.ReadDir(spritesDir)
	if err == nil {
		for _, e := range spriteEntries {
			if !e.IsDir() {
				continue
			}
			m, ok := moodByFolderName(e.Name())
			if !ok {
				continue
			}
			files, err := listPNGFiles(filepath.Join(spritesDir, e.Name()))
			if err != nil || len(files) == 0 {
				continue
			}
			var frames [][]byte
			for _, fn := range files {
				prepared, err := readAndPrepare(filepath.Join(spritesDir, e.Name(), fn), prepare)
				if err != nil {
					return nil, fmt.Errorf("sprite %s/%s: %w", e.Name(), fn, err)
				}
				frames = append(frames, prepared)
			}
			a := moods[m]
			a.Sprite = frames
			moods[m] = a
		}
	}

	return &Pack{Name: name, Moods: moods}, nil
}
