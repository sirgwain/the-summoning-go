package window

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sirgwain/the-summoning-go/engine/font"
)

// letters returns the engine's lettering as an image, made once.
func (s *Window) letters() *ebiten.Image {
	if s.font == nil {
		s.font = ebiten.NewImageFromImage(font.Atlas())
	}
	return s.font
}

// print draws text on dst with its first line's top left at (x, y), in white; a newline starts
// a line below.
func (s *Window) print(dst *ebiten.Image, text string, x, y int) {
	s.printScaled(dst, text, x, y, 1, color.White)
}

// printIn draws text as print does, in color c.
func (s *Window) printIn(dst *ebiten.Image, text string, x, y int, c color.Color) {
	s.printScaled(dst, text, x, y, 1, c)
}

// printScaled draws text as printIn does, each pixel of the lettering scale pixels square.
func (s *Window) printScaled(dst *ebiten.Image, text string, x, y, scale int, c color.Color) {
	atlas := s.letters()
	op := &ebiten.DrawImageOptions{}
	cx, cy := x, y
	for _, r := range text {
		if r == '\n' {
			cx, cy = x, cy+font.LineHeight*scale
			continue
		}
		if r != ' ' {
			op.GeoM.Reset()
			op.GeoM.Scale(float64(scale), float64(scale))
			op.GeoM.Translate(float64(cx-font.Left*scale), float64(cy))
			op.ColorScale.Reset()
			op.ColorScale.ScaleWithColor(c)
			dst.DrawImage(atlas.SubImage(font.Cell(r)).(*ebiten.Image), op)
		}
		cx += font.Advance * scale
	}
}
