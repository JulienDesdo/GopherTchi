# Developing GopherTchi

This file is for people who want to modify GopherTchi rather than just run it.

The goal is to explain the parts that are not immediately obvious from the repository tree: how the application is split, why `systray` library is forked, what happened with menu click handling, how Gopher Packs are loaded and reloaded, where macOS-specific code enters the project, and a few implementation choices that were deliberately kept simple.

## Architecture

`main.go` embeds the built-in assets, creates the application and starts systray.

The main runtime path is roughly:

```text
gopsutil
   ↓
internal/metrics
   ↓
metrics.Snapshot
   ↓
internal/mood
   ↓
current Mood
   ↓
internal/packs
   ↓
resolved icon / sprite frames
   ↓
internal/animation
   ↓
systray
   ↓
macOS menu bar
```

`internal/app` sits around that flow and coordinates the menu items, current settings, active pack, metric updates and icon refreshes.

The Go packages used by the executable live under `internal/`, which also prevents them from becoming an accidental public Go API.

## Where to start when changing something

| If you want to change...                         | Start here                                 |
| ------------------------------------------------ | ------------------------------------------ |
| Mood thresholds, priorities, hysteresis or dwell | `internal/mood/`                           |
| CPU, memory or disk-space collection             | `internal/metrics/`                        |
| Gopher Pack discovery, validation or fallback    | `internal/packs/`                          |
| Sprite playback                                  | `internal/animation/`                      |
| Menu behaviour and application coordination      | `internal/app/`                            |
| Persisted settings                               | `internal/config/`                         |
| Launch at Login                                  | `internal/startup/`                        |
| Menu-bar PNG preparation                         | `internal/iconutil/`                       |
| macOS systray behaviour                          | `third_party/systray/`                     |
| `.app` packaging                                 | `packaging/macos/`, `scripts/build-app.sh` |

## The local systray fork

GopherTchi depends on:

```text
github.com/getlantern/systray
```

but `go.mod` deliberately replaces it with:

```text
./third_party/systray
```

So `third_party/systray/` is not disposable dependency output. It is part of GopherTchi's source tree and needs to stay version-controlled.

The fork exists for two practical reasons.

### Menu-bar icon size

Upstream `systray` sets the macOS image size to `16x16` points.

That made the Gopher noticeably smaller than intended.

GopherTchi first prepares each PNG in `internal/iconutil`:

```text
source image
→ transparent padding removed
→ aspect ratio preserved
→ fitted into a 44×44 pixel canvas
```

The 44×44 image is the Retina representation of a roughly 22pt menu-bar slot.

The local systray fork then keeps the image aspect ratio and displays it at a maximum of 22 points instead of forcing it to 16×16.

This split is intentional:

```text
internal/iconutil
→ prepares the pixels

third_party/systray
→ tells AppKit how large the image should appear
```

Removing the local `replace` directive and going back to the untouched upstream module therefore changes the visible menu-bar size even though `iconutil` itself still outputs 44×44 images.

### Gopher Pack submenu separator and rebuild

The `Gopher Pack` submenu has two groups:

```text
✓ Default
  CustomPack
  AnotherPack
  BrokenPack (invalid)
-------------------------
  Open Packs Folder
  Reload Packs
```

The separator is there to distinguish pack entries from actions on the pack system.

Upstream systray provides a root-level separator, but GopherTchi needs one inside a submenu. The fork therefore adds:

```text
AddSubMenuSeparator()
```

On macOS this maps to a real:

```objective-c
[NSMenuItem separatorItem]
```

rather than a disabled text item made to look like a separator.

`Reload Packs` also recreates the dynamic contents of this submenu, so the fork adds:

```text
ClearSubMenu()
```

The native menu entries are removed and the corresponding child items are also dropped from systray's Go-side registry before new ones are created.

## Menu click handlers and concurrency

The first v2 implementation handled menu clicks by periodically checking `ClickedCh` with non-blocking reads.

That was a bad match for the way systray sends events.

Inside systray, a click is sent with a non-blocking `select`:

```text
try to send click
→ receiver ready: send it
→ nobody ready: drop it
```

If GopherTchi was also only checking the channel from time to time, both sides had to meet at the right moment.

A click could therefore be valid from the user's point of view and still never reach the handler.

The fixed menu items now use blocking listeners instead:

```go
for range item.ClickedCh {
    handler()
}
```

There is always a goroutine waiting for the next event.

The pack submenu needs a slightly different treatment because its items are recreated by `Reload Packs`.

Those listeners use cancellable contexts. A reload currently follows this sequence:

```text
cancel handlers attached to old pack/action items
→ clear the old submenu
→ rescan and validate the pack directory
→ create the new menu items
→ start handlers for the new items
→ refresh the current icon
```

This is why `internal/app` contains separate handling for fixed menu items and dynamically rebuilt pack items.

The distinction is useful when adding another dynamic menu later: creating the visual `MenuItem` is only half of the job; the event listener attached to it also needs to be replaced when that item disappears.

## Gopher Pack loading and validation

There are two types of packs.

The **Default** pack comes from the application's embedded `assets/` tree.

User packs come from:

```text
~/Library/Application Support/GopherTchi/packs/
```

At startup or after `Reload Packs`, the catalog first loads and validates Default, then scans the user pack directories.

The built-in Default pack must provide a representation for every mood. User packs do not.

This is what allows a pack such as:

```text
MyPack/
└── icons/
    └── Tired.png
```

to be valid. 

For `Tired`, the selected pack wins. Missing moods are resolved through Default (as detailed in README.md). 

### Why layout validation exists

*(slight rewrite of README.md on a different angle)*


An early easy mistake is this:

```text
Colors/
├── Happy.png
├── Tired.png
└── Sad.png
```

The files look like a Gopher Pack, but they are not under `icons/`.

Without validation, the directory could appear selectable while every mood silently falls back to Default. From the user's side, that looks like pack selection is broken.

The validator therefore checks the layout before the pack is loaded.

The main rules are:

| Pack content                                                     | Result                           |
| ---------------------------------------------------------------- | -------------------------------- |
| At least one recognized icon or sprite in the expected location  | Valid                            |
| Only some moods implemented                                      | Valid, missing moods use Default |
| No recognized Gopher asset                                       | Empty                            |
| Only unknown files or directories                                | Empty                            |
| Mood PNG directly at pack root                                   | Invalid                          |
| Mood sprite directory directly at pack root                      | Invalid                          |
| Sprite PNG directly under `sprites/` instead of a mood directory | Invalid                          |
| Hidden metadata such as `.DS_Store`                              | Ignored                          |

An empty pack is shown as:

```text
MyPack (empty)
```

while a malformed pack is shown as: 

```text
BrokenPack (invalid)
```

Unknown files and directories are deliberately tolerated. They do not make a pack valid, but they also do not make it malformed. This keeps the format open to additional pack content without requiring the current loader to understand it.

### Asset preparation

PNG files are decoded and prepared when the Default pack or a user pack is loaded.

The animation loop does not reopen and resize PNG files on every frame.

Once a pack has been loaded, the normal animation path works with already prepared PNG bytes:

```text
load pack
→ decode / crop / resize frames
→ keep prepared frames in memory

animation tick
→ select next prepared frame
→ systray.SetIcon(...)
```

This keeps filesystem and image-processing work out of the animation tick itself.

## Why Reload Packs is manual

GopherTchi does not run a filesystem watcher on the pack directory.
The normal workflow is:

```text
Open Packs Folder
→ add / remove / modify files
→ Reload Packs
```

Filesystem notifications can produce several events for one operation, while a Gopher Pack may itself contain several files. An explicit reload gives GopherTchi one clear point at which the complete directory is rescanned and validated.

For the size of this application, that is simpler than maintaining another asynchronous subsystem just to make pack changes appear automatically.

## Metrics, moods and polling choices

`internal/metrics` currently collects three values:

```text
CPU usage
memory usage
root filesystem usage
```

The default reader polls on a 3-second interval. CPU sampling uses `gopsutil`, as do memory and disk-space usage.

The resulting snapshot is passed to `internal/mood`.

The default mood priority is:

```text
Sad      → high disk-space usage
Hungry   → high CPU usage
Tired    → high memory usage
Sleeping → low CPU / memory / disk-space usage
Happy    → otherwise
```

### Hysteresis

Each threshold has separate enter and exit values.

For example, entering a high-CPU state and leaving it do not happen at exactly the same percentage.

Without this gap, a value moving around one threshold could repeatedly switch the mood on consecutive polls.

### Dwell time

Hysteresis prevents a mood from changing back and forth around a threshold, but it does not protect against a short resource spike that clearly crosses that threshold.

Dwell time handles that second case.

When the evaluator detects a different mood, it first keeps it as a candidate. The mood only changes if that candidate remains valid for the configured duration.

For example, a short CPU spike should not immediately turn the Gopher `Hungry` if the load disappears a few seconds later.

The default dwell time is 6 seconds.

### Runtime overrides

The default polling interval and dwell time can be changed through environment variables:

```zsh
GOPHERTCHI_POLL_INTERVAL=5s ./gophertchi
GOPHERTCHI_DWELL_TIME=10s ./gophertchi
```

`GOPHERTCHI_POLL_INTERVAL` changes how frequently CPU, memory and disk-space metrics are sampled.

`GOPHERTCHI_DWELL_TIME` is useful when testing or tuning how quickly the Gopher reacts to a sustained change in system load.

These are development/runtime overrides rather than normal application settings. GopherTchi reads them from the process environment at startup; they are not persisted in `config.json` or exposed in the macOS Settings menu.


### Why disk usage does not have its own timer

Root filesystem usage normally changes much more slowly than CPU usage, so it could technically be sampled less frequently and cached between the other polls.

I considered splitting it out during development.

The gain for GopherTchi would be small, while the implementation would gain another timer, another cached value and another update path.

There was no need in the current application to add that extra state, so CPU, memory and disk-space usage remain part of the same metric poll.

The same idea explains some other deliberately simple parts of the project: *an optimization is useful when its practical benefit is larger than the code needed to maintain it.*

## Launch at Login: Go, CGO and Objective-C

Launch at Login needs a macOS API that is not part of Go's standard library.

On macOS 13 and later, GopherTchi uses Apple's ServiceManagement framework and:

```text
SMAppService.mainAppService
```

The platform-specific code lives in:

```text
internal/startup/
```

The Go side exposes a small API to the rest of the application:

```text
Supported()
CurrentStatus()
Enabled()
SetEnabled(...)
OpenSystemSettingsLoginItems()
```

The Darwin implementation then crosses into Objective-C through CGO:

```text
Go
 ↓
CGO declarations
 ↓
smapp_darwin.m
 ↓
ServiceManagement.framework
 ↓
SMAppService
```

The Objective-C bridge is intentionally small. The rest of the application does not need to know how `NSBundle`, `NSError` or `SMAppService` are called.

### Native state is authoritative

Launch at Login can also be changed directly from macOS System Settings.

For that reason, GopherTchi does not register or unregister the application at startup based on the value previously written to `config.json`.

It reads `SMAppService.status` and uses the operating-system state for the menu.

An explicit click on `Launch at Login` is what changes the native registration.

If macOS reports that approval is required, GopherTchi opens:

```text
System Settings
→ General
→ Login Items
```

through `SMAppService.openSystemSettingsLoginItems()` instead of trying to register the already-pending service again.

### Why it is disabled with `go run .`

`SMAppService.mainAppService` represents the application bundle itself.

The startup bridge therefore checks that GopherTchi is running from a `.app` bundle.

With:

```zsh
go run .
```

or a loose:

```zsh
./gophertchi
```

there is no normal `GopherTchi.app` context, so the menu item is disabled.

### Migration from the old LaunchAgent implementation

An earlier implementation tested was :

```text
~/Library/LaunchAgents/com.gophertchi.app.plist
```

The current startup package performs a one-time best-effort cleanup of that old GopherTchi LaunchAgent before using `SMAppService`.

It only targets GopherTchi's previous plist and does not touch unrelated Login Items or LaunchAgents.

### A CGO build issue encountered during the migration

The Objective-C implementation already lives in a `.m` file.

An earlier build attempt also forced:

```text
-x objective-c
```

through Go's C compiler flags.

That made the CGO build fail because the flag also affected inputs that were not meant to be compiled as Objective-C.

The final version leaves language detection to the `.m` file and only uses CGO to link the required frameworks:

```text
Foundation
ServiceManagement
```

It is a small detail, but useful if this bridge is modified later: the Objective-C source does not need a global `-x objective-c` override.

## Building the macOS application

The `.app` is generated from source rather than committed to Git.

The packaging inputs live in:

```text
packaging/macos/
├── Info.plist
└── AppIcon.icns
```

The build command is:

```zsh
bash scripts/build-app.sh
```

and the result is:

```text
dist/
└── GopherTchi.app/
    └── Contents/
        ├── Info.plist
        ├── MacOS/
        │   └── GopherTchi
        └── Resources/
            └── AppIcon.icns
```

So there are two different directories with different purposes:

```text
packaging/
→ version-controlled inputs used to create the application

dist/
→ generated application output
```

`dist/` is intentionally ignored by Git.

The bundle's `Info.plist` also sets `LSUIElement`, which is appropriate for a menu-bar utility that does not need a normal Dock application interface.

The build script removes any existing `dist/GopherTchi.app` before recreating the bundle, preventing stale generated files or resources from surviving between builds.

## Tests

The project uses normal Go tests and the full suite can be run with:

```zsh
go test ./...
```

The current tests cover:

- mood transitions, priorities, hysteresis, dwell time and candidate reset;
- animation frame cycling and static fallback;
- configuration defaults;
- Gopher Pack resolution and fallback;
- filesystem-level pack validation, including partial packs, empty packs, malformed layouts and ignored metadata;
- the distinction between empty and invalid pack entries;
- startup behaviour outside a macOS .app bundle.

I focused on these areas because they contain most of the state transitions, fallback rules and filesystem edge cases in the application. They are easy to break without producing an obvious compilation error, and reproducing every case manually would quickly become tedious.

Native behaviour that depends on a real macOS session or application bundle is still checked manually, particularly menu-bar rendering, Finder integration and the real SMAppService registration and approval flow.