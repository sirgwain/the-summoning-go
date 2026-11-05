package window

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sirgwain/the-summoning-go/engine/font"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
)

// letters returns face's letters as an image, made once.
func (s *Window) letters(face *font.Face) *ebiten.Image {
	img, ok := s.fonts[face]
	if !ok {
		img = ebiten.NewImageFromImage(face.Atlas())
		s.fonts[face] = img
	}
	return img
}

// print draws text on dst in the small face with its first line's top left at (x, y), in
// white; a newline starts a line below.
// ui returns label key in words (inspect.UI).
func (s *Window) ui(key string, args ...any) string { return inspect.UI(s.v.Pack, key, args...) }

func (s *Window) print(dst *ebiten.Image, text string, x, y int) {
	s.printFace(dst, font.Small, text, x, y, 1, color.White)
}

// printIn draws text as print does, in color c.
func (s *Window) printIn(dst *ebiten.Image, text string, x, y int, c color.Color) {
	s.printFace(dst, font.Small, text, x, y, 1, c)
}

// printFace draws text in face and color c, each pixel of the lettering scale pixels square.
func (s *Window) printFace(dst *ebiten.Image, face *font.Face, text string, x, y, scale int, c color.Color) {
	atlas := s.letters(face)
	op := &ebiten.DrawImageOptions{}
	cx, cy := x, y
	for _, r := range text {
		if r == '\n' {
			cx, cy = x, cy+face.LineHeight*scale
			continue
		}
		if r != ' ' {
			op.GeoM.Reset()
			op.GeoM.Scale(float64(scale), float64(scale))
			op.GeoM.Translate(float64(cx-font.Left*scale), float64(cy))
			op.ColorScale.Reset()
			op.ColorScale.ScaleWithColor(c)
			dst.DrawImage(atlas.SubImage(face.Cell(r)).(*ebiten.Image), op)
		}
		cx += face.Advance * scale
	}
}
