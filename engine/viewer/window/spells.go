package window

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// drawSpellPanel draws the spell panel: the icons known, the gesture entered, the buttons and the
// spellbook.
func (s *Window) drawSpellPanel(screen *ebiten.Image) {
	v := s.v
	sp := v.SpellPanel()
	if sp == nil {
		return
	}
	fill := func(r image.Rectangle, c color.Color) {
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
	}
	stroke := func(r image.Rectangle, c color.Color) {
		vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, c, false)
	}
	fill(sp.Rect.Inset(-viewer.Margin/2), hotbarColor)
	for _, b := range sp.Icons {
		fill(sp.Screen(b.Rect), slotColor)
		stroke(sp.Screen(b.Rect), slotEdgeColor)
	}
	for _, b := range sp.Buttons {
		r := sp.Screen(b.Rect)
		fill(r, actionColor)
		stroke(r, slotEdgeColor)
		ebitenutil.DebugPrintAt(screen, b.Label, r.Min.X+(r.Dx()-len(b.Label)*viewer.CharWidth)/2, r.Min.Y+(r.Dy()-viewer.LineHeight)/2)
	}
	for _, h := range sp.Headers {
		fill(sp.Screen(h.Rect), actionColor)
	}
	for _, e := range sp.Book {
		fill(sp.Screen(e.Rect), slotColor)
		r := sp.Screen(e.Button)
		fill(r, actionColor)
		stroke(r, slotEdgeColor)
		ebitenutil.DebugPrintAt(screen, "Memorize", r.Min.X+(r.Dx()-8*viewer.CharWidth)/2, r.Min.Y+(r.Dy()-viewer.LineHeight)/2)
	}
	if !sp.Bar.Empty() {
		fill(sp.Screen(sp.Bar), slotColor)
		stroke(sp.Screen(sp.Bar), slotEdgeColor)
		fill(sp.Screen(sp.Thumb), slotEdgeColor)
	}
	art := s.artImage(sp.Rect.Size().Div(sp.Scale))
	draw := func(img content.Image, r image.Rectangle) {
		b := v.Pack.Sprites[img.Sprite]
		if b == nil {
			return
		}
		at := r.Min.Add(image.Pt((r.Dx()-b.FrameSize[0])/2, (r.Dy()-b.FrameSize[1])/2))
		if err := s.sprites.DrawImage(art, img, at.X, at.Y, false); err != nil && s.err == nil {
			s.err = err
		}
	}
	for _, b := range append(sp.Icons, sp.Entered...) {
		if b.Known {
			draw(b.Icon, b.Rect)
		}
	}
	for _, e := range sp.Book {
		if e.Icon != nil {
			draw(*e.Icon, e.IconRect)
		}
	}
	s.presenter.PresentAt(screen, art, sp.Scale, sp.Rect.Min)
	for _, b := range sp.Icons {
		if b.Known {
			r := sp.Screen(b.LetterRect)
			ebitenutil.DebugPrintAt(screen, string(rune(b.Letter)), r.Min.X+(r.Dx()-viewer.CharWidth)/2, r.Min.Y+(r.Dy()-viewer.LineHeight)/2)
		}
	}
	for _, h := range sp.Headers {
		r := sp.Screen(h.Rect)
		ebitenutil.DebugPrintAt(screen, viewer.Clip(h.Label, (r.Dx()-4)/viewer.CharWidth), r.Min.X+4, r.Min.Y+(r.Dy()-viewer.LineHeight)/2)
	}
	for _, e := range sp.Book {
		r, end := sp.Screen(e.Rect), sp.Screen(e.Button).Min.X-2
		x := sp.Screen(e.IconRect).Max.X + 4
		ebitenutil.DebugPrintAt(screen, viewer.Clip(e.Label, (end-x)/viewer.CharWidth), x, r.Min.Y+(r.Dy()-viewer.LineHeight)/2)
	}
}
