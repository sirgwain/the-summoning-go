package window

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/frame"
)

// framePics is a frame drawn at one size: its base and its groups of lights.
type framePics struct {
	base *ebiten.Image
	glow [frame.GlowGroups]*ebiten.Image
}

type frameKey struct {
	style frame.Style
	size  image.Point
}

// maxFrames is how many frames are kept before they are all drawn anew.
const maxFrames = 16

// drawFrame draws a frame of style filling outer, a panel's Frame on the screen, at scale screen
// pixels per panel pixel, its inside filled at opacity (0..1) of the style's own. Its lights
// breathe with the window's own clock, so they go on while the world is held.
func (s *Window) drawFrame(screen *ebiten.Image, style frame.Style, outer image.Rectangle, scale int, opacity float32) {
	key := frameKey{style, outer.Size().Div(scale)}
	f, ok := s.frames[key]
	if !ok {
		if len(s.frames) >= maxFrames {
			for k, old := range s.frames {
				old.deallocate()
				delete(s.frames, k)
			}
		}
		pic := frame.Draw(style, key.size.X, key.size.Y)
		f = &framePics{base: ebiten.NewImageFromImage(pic.Base)}
		for g, img := range pic.Glow {
			if img != nil {
				f.glow[g] = ebiten.NewImageFromImage(img)
			}
		}
		s.frames[key] = f
	}
	in := outer.Inset(frame.Width * scale)
	c := frame.Inside(style)
	k := opacity * float32(c.A) / 255
	fill := color.RGBA{uint8(float32(c.R) * k), uint8(float32(c.G) * k), uint8(float32(c.B) * k), uint8(255 * k)}
	vector.FillRect(screen, float32(in.Min.X), float32(in.Min.Y), float32(in.Dx()), float32(in.Dy()), fill, false)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	op.GeoM.Translate(float64(outer.Min.X), float64(outer.Min.Y))
	screen.DrawImage(f.base, op)
	t := float64(s.drawn) / float64(ebiten.TPS())
	for g, img := range f.glow {
		if img == nil {
			continue
		}
		op.ColorScale.Reset()
		op.ColorScale.ScaleAlpha(float32(frame.Pulse(g, t)))
		screen.DrawImage(img, op)
	}
}

func (f *framePics) deallocate() {
	f.base.Deallocate()
	for _, img := range f.glow {
		if img != nil {
			img.Deallocate()
		}
	}
}
