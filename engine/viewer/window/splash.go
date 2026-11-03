package window

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sirgwain/the-summoning-go/engine/font"
)

//go:embed assets/splash.png
var splashPNG []byte

// splashArt returns the splash artwork, decoded once, or nil when it cannot be.
func (s *Window) splashArt() *ebiten.Image {
	if s.splash == nil {
		img, _, err := image.Decode(bytes.NewReader(splashPNG))
		if err != nil {
			if s.err == nil {
				s.err = err
			}
			return nil
		}
		s.splash = ebiten.NewImageFromImage(img)
	}
	return s.splash
}

// drawSplashArt draws the artwork as large as fits the screen, centered, at brightness (0..1),
// and returns where it went.
func (s *Window) drawSplashArt(screen *ebiten.Image, brightness float32) image.Rectangle {
	art := s.splashArt()
	if art == nil {
		return screen.Bounds()
	}
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	b := art.Bounds()
	scale := min(float64(w)/float64(b.Dx()), float64(h)/float64(b.Dy()))
	width, height := int(float64(b.Dx())*scale), int(float64(b.Dy())*scale)
	at := image.Pt((w-width)/2, (h-height)/2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	op.Filter = ebiten.FilterNearest
	op.ColorScale.Scale(brightness, brightness, brightness, 1)
	screen.DrawImage(art, op)
	return image.Rectangle{at, at.Add(image.Pt(width, height))}
}

// drawSplash draws the splash: the artwork, its title and how to go on.
func (s *Window) drawSplash(screen *ebiten.Image) {
	screen.Fill(color.Black)
	r := s.drawSplashArt(screen, 1)
	s.drawSplashTitle(screen, r)
	w, h := float64(r.Dx()), float64(r.Dy())
	s.splashText(screen, "PRESS A KEY OR CLICK TO CONTINUE", float64(r.Min.X)+w/2, float64(r.Min.Y)+h*0.91, w/720, color.RGBA{224, 208, 179, 255})
}

// drawSplashTitle letters the title over the artwork at r.
func (s *Window) drawSplashTitle(screen *ebiten.Image, r image.Rectangle) {
	x, y, w, h := float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy())
	s.splashText(screen, "THE", x+w/2, y+h*0.125, w/320, color.RGBA{196, 180, 157, 255})
	s.splashText(screen, "S U M M O N I N G", x+w/2, y+h*0.205, w/190, color.RGBA{195, 74, 57, 255})
}

// splashText letters text centered on x with its top at y, at the whole scale nearest to scale,
// so the lettering's pixels stay square and crisp.
func (s *Window) splashText(screen *ebiten.Image, text string, x, y, scale float64, tint color.RGBA) {
	k := max(1, int(scale+0.5))
	s.printFace(screen, font.Small, text, int(x)-font.Small.Measure(text)*k/2, int(y), k, tint)
}
