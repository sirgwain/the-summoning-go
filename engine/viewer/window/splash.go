package window

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sirgwain/the-summoning-go/engine/font"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
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

// drawSplash draws the splash: the artwork fading in, its title filling in from the left, and
// then how to go on fading in (docs/spec/scenes.md, The frontend).
func (s *Window) drawSplash(screen *ebiten.Image, shown int) {
	screen.Fill(color.Black)
	ramp := func(from, length int) float64 {
		return min(1, max(0, float64(shown-from)/float64(max(1, length))))
	}
	r := s.drawSplashArt(screen, float32(ramp(0, viewer.SplashArtUpdates)))
	s.drawSplashTitle(screen, r, ramp(viewer.SplashArtUpdates, viewer.SplashTitleUpdates))
	if a := ramp(viewer.SplashPromptUpdate, viewer.SplashPromptFade); a > 0 {
		w, h := float64(r.Dx()), float64(r.Dy())
		c := color.RGBA{uint8(224 * a), uint8(208 * a), uint8(179 * a), uint8(255 * a)}
		s.splashText(screen, "PRESS A KEY OR CLICK TO CONTINUE", float64(r.Min.X)+w/2, float64(r.Min.Y)+h*0.91, w/720, c)
	}
}

// titleFillSoftness is how much of the title's width the edge of its fill fades across;
// titleFillStrip is how wide each strip of the fill is, in screen pixels.
const (
	titleFillSoftness = 0.3
	titleFillStrip    = 2
)

// drawSplashTitle letters the title over the artwork at r, filled in from the left as far as
// fill (0..1): each column shows as the fill's soft edge passes it.
func (s *Window) drawSplashTitle(screen *ebiten.Image, r image.Rectangle, fill float64) {
	if fill <= 0 {
		return
	}
	dst := screen
	if fill < 1 {
		if s.title == nil || s.title.Bounds().Size() != screen.Bounds().Size() {
			s.title = ebiten.NewImage(screen.Bounds().Dx(), screen.Bounds().Dy())
		}
		s.title.Clear()
		dst = s.title
	}
	x, y, w, h := float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy())
	b := s.splashText(dst, "THE", x+w/2, y+h*0.125, w/320, color.RGBA{196, 180, 157, 255})
	b = b.Union(s.splashText(dst, "S U M M O N I N G", x+w/2, y+h*0.205, w/190, color.RGBA{195, 74, 57, 255}))
	if fill >= 1 || b.Empty() {
		return
	}
	edge := fill * (1 + titleFillSoftness)
	op := &ebiten.DrawImageOptions{}
	for sx := b.Min.X; sx < b.Max.X; sx += titleFillStrip {
		at := float64(sx-b.Min.X) / float64(b.Dx())
		a := min(1, max(0, (edge-at)/titleFillSoftness))
		if a <= 0 {
			break
		}
		strip := image.Rect(sx, b.Min.Y, min(sx+titleFillStrip, b.Max.X), b.Max.Y)
		op.GeoM.Reset()
		op.GeoM.Translate(float64(strip.Min.X), float64(strip.Min.Y))
		op.ColorScale.Reset()
		op.ColorScale.ScaleAlpha(float32(a))
		screen.DrawImage(s.title.SubImage(strip).(*ebiten.Image), op)
	}
}

// splashText letters text centered on x with its top at y, at the whole scale nearest to scale,
// so the lettering's pixels stay square and crisp, and returns where it went.
func (s *Window) splashText(screen *ebiten.Image, text string, x, y, scale float64, tint color.RGBA) image.Rectangle {
	k := max(1, int(scale+0.5))
	w := font.Small.Measure(text) * k
	left := int(x) - w/2
	s.printFace(screen, font.Small, text, left, int(y), k, tint)
	return image.Rect(left, int(y), left+w, int(y)+font.Small.LineHeight*k)
}
