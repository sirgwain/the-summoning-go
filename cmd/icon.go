package cmd

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/icon.png
var iconPNG []byte

// setWindowIcon sets the icon before opening a game window.
func setWindowIcon() {
	icon, _, err := image.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		panic(err) // The embedded image is part of the executable.
	}
	ebiten.SetWindowIcon([]image.Image{icon})
	setPlatformIcon(iconPNG)
}
