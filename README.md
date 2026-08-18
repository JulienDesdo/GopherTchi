PAS TOUCHE AU README POUR L instant ! 

# GopherTchi

## A tiny system-health Tamagotchi for your macOS menu bar.

GopherTchi is a lightweight macOS menu bar application written in Go. It displays a small Gopher whose mood changes according to the current resource usage of your computer.

![Software_Bar](docs_images/bar.png)

Instead of constantly checking CPU, memory, and disk statistics yourself, you can simply look at your Gopher. Depending on your preferences, you can personnalize the Gopher pictures (in assets/icons by using [gopherkon website](https://www.quasilyte.dev/gopherkon/) to change color, look, facial expression...) or even the criteria of moods (mood/mood.go, metrics/metrics.go). 

## Features 

gopher packs, launch, sprites vs static; 


## Config 

Few more details on config possibilies; 

### Poll interval

Controls how frequently system metrics are sampled:

```
GOPHERTCHI_POLL_INTERVAL=5s ./gophertchi
```

### Mood dwell time

Controls how long a new mood must remain valid before GopherTchi switches to it:
```
GOPHERTCHI_DWELL_TIME=10s ./gophertchi
```
These mechanisms help prevent rapid mood changes caused by short-lived resource spikes.

### Idea for custom

GopherTchi is intentionally easy to customize.
You can replace the images in assets/icons/ with your own Gopher artwork and adjust the mood thresholds and evaluation rules in internal/mood/mood.go.

This makes it possible to experiment with different visual styles as well as different interpretations of system health — for example, using sustained CPU load rather than short spikes, changing memory thresholds, or introducing additional system signals.

See the Resources section below for tools and Gopher artwork that can help when creating custom icons.

## Install

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
### macOS application

Pre-built .app bundles may also be distributed through the GitHub Releases page.

Download the application and move GopherTchi.app into your /Applications directory.

## Ressources 

### Artworks 

> *The original Go Gopher was designed by Renée French. The official Go project distributes the Gopher design under the Creative Commons Attribution 4.0 License.* *https://go.dev/doc/gopher/README* 

- **GopherKon** — an excellent Gopher constructor website i used to generate the gopher moods, head over here for easy custom: https://www.quasilyte.dev/gopherkon/
- **Egon Elbre's Gophers** — a large collection of Gopher illustrations: https://github.com/egonelbre/gophers
- **Gopher Icons** — additional Gopher artwork and icons: https://github.com/shalakhin/gophericons
- **Sprite AI** — a tool that may be useful when experimenting with sprite generation: https://www.sprite-ai.art/


### macOS and Go ressources 

For developper interested in deeper MacOS intergration from Go, the following article I met is a useful introduction to bridging Go with macOS APIs: 
- **Calling macOS APIs from Go / Core Location notes** : https://vladimir.varank.in/notes/2020/03/go-osx-core-location/

## License

GopherTchi is released under the MIT License.
Made with Go, macOS, and an unnecessarily emotional Gopher.

---

**To finish :** 
Add option to launch it as the computer starts; 
Add Readme (OK) + clean build 
units tests of mood.Evaluation 
release .app 


