package main

import (
	"embed"
	"log"

	"github.com/getlantern/systray"
	"github.com/jlnesc/gophertchi/internal/app"
)

//go:embed assets/icons/*.png
//go:embed assets/sprites
var defaultAssets embed.FS

func main() {
	application, err := app.New(defaultAssets)
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	systray.Run(application.OnReady, application.OnExit)
}
