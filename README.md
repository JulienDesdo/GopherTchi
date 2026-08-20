# GopherTchi

## A tiny system-health Tamagotchi for your macOS menu bar.

GopherTchi is a lightweight macOS menu bar application written in Go. It displays a small Gopher whose mood changes according to the current resource usage of your computer.

![Software\_Bar](docs_images/bar.png)

Instead of constantly checking CPU, memory, and disk statistics yourself, you can simply look at your Gopher.


GopherTchi is also designed to be easy to experiment with: you can create your own **Gopher Packs**, use static icons or animated sprites, and modify the mood rules if you want to change how system health is interpreted.

## Features

- Periodic CPU, memory, and disk-space monitoring
- Five moods: **Happy, Hungry, Tired, Sad, and Sleeping**
- Hysteresis and dwell time to avoid rapid mood changes
- Static Gopher icons
- Optional sprite animations
- Custom Gopher Packs without rebuilding the application
- Partial packs with automatic fallback to the built-in Default pack
- Pack validation with visible feedback for malformed packs
- Manual pack reload from the menu bar
- Launch at Login on macOS
- macOS `.app` bundle

## Gopher Packs and customization

Custom artwork is organized into **Gopher Packs**.

User packs live in:

```text
~/Library/Application Support/GopherTchi/packs/
```

You can open this directory directly from the menu:

```text
Gopher Pack
└── Open Packs Folder
```

A pack may contain static icons, sprites, or both:

```text
MyPack/
├── icons/
│   ├── Happy.png
│   ├── Hungry.png
│   └── Tired.png
└── sprites/
    └── Sleeping/
        ├── 00.png
        ├── 01.png
        └── 02.png
```

Static mood images belong in:

```text
icons/<Mood>.png
```

Animated frames belong in:

```text
sprites/<Mood>/*.png
```

Sprite frames are loaded in filename order, so names such as `00.png`, `01.png`, `02.png`, ... are recommended.

### Partial packs and fallback

A pack does **not** need to implement every mood.

For example:

```text
MyPack/
└── icons/
    └── Tired.png
```

is a valid Gopher Pack.

When this pack is active, GopherTchi uses the custom image `Tired.png`. Other moods fall back to the built-in Default pack.

With animations enabled, artwork is resolved in this order if something missing:

```text
selected pack sprites
Then checks (and potentially use) → selected pack static icon
Then checks (and potentially use) → Default sprites
Then uses → Default static icon
```

With animations disabled:

```text
selected pack static icon
Then checks (and potentially uses) → first selected pack sprite
Then checks (and potentially uses) → Default static icon
Then uses → first Default sprite
```

The Default pack is embedded in the application and is always available as fallback.

### Pack structure validation

GopherTchi is tolerant about **how much** of a pack you implement, but stricter about **where recognized assets are placed**.

This is valid:

```text
Colors/
└── icons/
    └── Tired.png
```

This is not:

```text
Colors/
└── Tired.png
```

Mood PNGs placed directly at the pack root are reported as malformed instead of being silently ignored.

A pack containing no recognized Gopher assets is considered empty. This also applies if the directory contains unrelated files or folders that GopherTchi does not currently use.

Empty and malformed packs remain visible in the menu but cannot be selected:

```text
MyEmptyPack (empty)
BrokenPack (invalid)
```

Unknown files and directories are otherwise left alone, so the pack format can still be extended without making unrelated content an error.

If the currently selected pack disappears or becomes unusable after Reload Packs, GopherTchi falls back to `Default`.


After adding, renaming or modifying a pack, use:

```text
Gopher Pack
└── Reload Packs
```

to reload the pack directory without restarting the application.

For creating custom Gophers, [GopherKon](https://www.quasilyte.dev/gopherkon/) is an easy way to experiment with colors, expressions and visual styles.

If you are modifying GopherTchi itself rather than creating a user pack, the built-in Default artwork lives under:

```text
assets/icons/
```

The application already supports embedded sprites under `assets/sprites/` as well; built-in sprite sets are still being added.

## Configuration

Most user-facing settings are available directly from the menu bar:

```text
Settings
├── Animations
└── Launch at Login
```

Animations are disabled by default.

Additional runtime overrides are available for development and experimentation; see [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md#runtime-overrides).

### Custom mood behaviour

The mood evaluation logic lives in:

```text
internal/mood/
```

System metric collection lives in:

```text
internal/metrics/
```

You can modify the thresholds, priorities, hysteresis and dwell behaviour, or introduce additional system signals.

For example, you may want to interpret sustained CPU load differently, change the memory thresholds, or make another system metric influence the Gopher's mood.

---

## Architecture

The main runtime flow is:

```text
system metrics
      ↓
 mood evaluator
      ↓
  current mood
      ↓
  pack resolver
      ↓
static icon / animation
      ↓
    systray
      ↓
macOS menu bar
```

`internal/app` coordinates this flow and the menu-bar UI, while the macOS-specific startup code is kept separately under `internal/startup`.

**Interested in the implementation?**
See [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) for the architecture, the local systray fork, CGO/Objective-C integration, concurrency issues encountered during development, Gopher Pack internals, packaging, tests, and design choices.

---

## Install

### macOS application

Pre-built `.app` bundles are distributed through the GitHub Releases page.

Download `GopherTchi.app` and move it into your `/Applications` directory.

GopherTchi is a menu bar application and does not open a traditional application window.

### Build from source

GopherTchi requires Go 1.22 or newer.

Clone the repository:

```zsh
git clone https://github.com/JulienDesdo/GopherTchi.git
cd GopherTchi
```

Build the project:

```zsh
go build .
```

Then run the generated binary:

```zsh
./gophertchi
```

When GopherTchi is executed as a loose binary or through `go run .`, functionality that requires a real macOS application bundle — notably Launch at Login — is unavailable.

### Build the macOS application

To generate the `.app` bundle:

```zsh
bash scripts/build-app.sh
```

The generated application is placed in:

```text
dist/GopherTchi.app
```

The macOS packaging inputs live under:

```text
packaging/macos/
├── AppIcon.icns
└── Info.plist
```

The build script assembles these files with the Go binary into the normal macOS application bundle structure.

`dist/` contains generated build artifacts and is intentionally ignored by Git.

---

## Resources

### Artworks

> *The original Go Gopher was designed by Renée French. The official Go project distributes the Gopher design under the Creative Commons Attribution 4.0 License.*
> [Official Go Gopher README](https://go.dev/doc/gopher/README)

* **GopherKon** — an excellent Gopher constructor website I used to generate the Gopher moods, and an easy starting point for custom Gopher Packs: [quasilyte.dev/gopherkon](https://www.quasilyte.dev/gopherkon/)
* **Egon Elbre's Gophers** — a large collection of Gopher illustrations: [github.com/egonelbre/gophers](https://github.com/egonelbre/gophers)
* **Gopher Icons** — additional Gopher artwork and icons: [github.com/shalakhin/gophericons](https://github.com/shalakhin/gophericons)
* **Sprite AI** — a tool that may be useful when experimenting with sprite generation: [sprite-ai.art](https://www.sprite-ai.art/)

### macOS and Go resources

For developers interested in deeper macOS integration from Go, the following article is a useful introduction to bridging Go with macOS APIs:

* **Calling macOS APIs from Go / Core Location notes** — [Vladimir Varankin](https://vladimir.varank.in/notes/2020/03/go-osx-core-location/)

## License

GopherTchi is released under the MIT License.

Made with Go, macOS, and an unnecessarily emotional Gopher.
Possible next steps and ideas are kept in [`docs/FUTURE.md`](docs/FUTURE.md).

---
