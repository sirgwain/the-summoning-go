package window

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// drawLog draws the conversation log: the search field, the people talked with or the matches
// down the left, and the chosen person's conversations on the right. It draws the window at its
// own pixels, then scales it onto the screen.
func (s *Window) drawLog(screen *ebiten.Image) {
	v := s.v
	lv := v.Log()
	if lv == nil {
		return
	}
	size := lv.Rect.Size().Div(lv.Scale)
	if s.log == nil || s.log.Bounds().Size() != size {
		if s.log != nil {
			s.log.Deallocate()
		}
		s.log = ebiten.NewImage(size.X, size.Y)
	}
	win := s.log
	win.Fill(talkColor)
	vector.StrokeRect(win, 0.5, 0.5, float32(size.X-1), float32(size.Y-1), 1, slotEdgeColor, false)
	fill := func(r image.Rectangle, c color.Color) {
		vector.FillRect(win, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
	}
	edge := func(r image.Rectangle, c color.Color) {
		vector.StrokeRect(win, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, c, false)
	}
	ebitenutil.DebugPrintAt(win, "Find:", lv.Field.Min.X-6*viewer.CharWidth, lv.Field.Min.Y+2)
	fill(lv.Field, slotColor)
	edge(lv.Field, keywordColor)
	cursor := ""
	if v.Ticks/30%2 == 0 {
		cursor = "_"
	}
	ebitenutil.DebugPrintAt(win, lv.Typed+cursor, lv.Field.Min.X+4, lv.Field.Min.Y+2)
	fill(lv.Close.Rect, actionColor)
	edge(lv.Close.Rect, slotEdgeColor)
	ebitenutil.DebugPrintAt(win, lv.Close.Label, lv.Close.Rect.Min.X+(lv.Close.Rect.Dx()-len(lv.Close.Label)*viewer.CharWidth)/2, lv.Close.Rect.Min.Y+2)

	for i, r := range lv.Rows {
		bg := slotColor
		if i == lv.Selected {
			bg = talkSelectedColor
		}
		fill(r.Rect, bg)
		x := r.Rect.Min.X + 4
		if r.HasPortrait {
			if b := v.Pack.Sprites[r.Portrait.Sprite]; b != nil {
				art := s.artImage(image.Pt(b.FrameSize[0], b.FrameSize[1]))
				if err := s.sprites.DrawImage(art, r.Portrait, 0, 0, r.Mirrored); err != nil && s.err == nil {
					s.err = err
				}
				s.presenter.PresentAt(win, art, 1, image.Pt(x, r.Rect.Min.Y+(r.Rect.Dy()-b.FrameSize[1])/2))
			}
			x += 48 + 4
		}
		ebitenutil.DebugPrintAt(win, r.Label, x, r.Rect.Min.Y+(r.Rect.Dy()-viewer.LineHeight)/2)
	}
	if lv.Empty != "" {
		ebitenutil.DebugPrintAt(win, lv.Empty, lv.Pane.Min.X, lv.Pane.Min.Y)
	}
	for _, t := range lv.Text {
		if t.Focus {
			r := t.Words[0].Rect
			for _, w := range t.Words {
				r = r.Union(w.Rect)
			}
			fill(image.Rect(lv.Pane.Min.X, r.Min.Y, lv.Pane.Max.X, r.Max.Y), talkSelectedColor)
		}
		for _, w := range t.Words {
			if t.Heading {
				vector.FillRect(win, float32(lv.Pane.Min.X), float32(w.Rect.Max.Y)-1, float32(lv.Pane.Dx()), 1, slotEdgeColor, false)
			}
			ebitenutil.DebugPrintAt(win, w.Text, w.Rect.Min.X, w.Rect.Min.Y)
			if w.Keyword {
				vector.StrokeLine(win, float32(w.Rect.Min.X), float32(w.Rect.Max.Y)-0.5, float32(w.Rect.Max.X), float32(w.Rect.Max.Y)-0.5, 1, keywordColor, false)
			}
		}
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(lv.Scale), float64(lv.Scale))
	op.GeoM.Translate(float64(lv.Rect.Min.X), float64(lv.Rect.Min.Y))
	screen.DrawImage(win, op)
}
