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

// drawBook draws the book: its tabs, the entries down the left and the chosen one's picture and
// lines on the right. It draws the window at its own pixels, then scales it onto the screen.
func (s *Window) drawBook(screen *ebiten.Image) {
	v := s.v
	bv := v.Book()
	if bv == nil {
		return
	}
	size := bv.Rect.Size().Div(bv.Scale)
	if s.book == nil || s.book.Bounds().Size() != size {
		if s.book != nil {
			s.book.Deallocate()
		}
		s.book = ebiten.NewImage(size.X, size.Y)
	}
	win := s.book
	win.Fill(talkColor)
	vector.StrokeRect(win, 0.5, 0.5, float32(size.X-1), float32(size.Y-1), 1, slotEdgeColor, false)
	fill := func(r image.Rectangle, c color.Color) {
		vector.FillRect(win, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
	}
	edge := func(r image.Rectangle, c color.Color) {
		vector.StrokeRect(win, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, c, false)
	}
	button := func(b viewer.PanelButton, bg color.Color) {
		fill(b.Rect, bg)
		edge(b.Rect, slotEdgeColor)
		ebitenutil.DebugPrintAt(win, b.Label, b.Rect.Min.X+(b.Rect.Dx()-len(b.Label)*viewer.CharWidth)/2, b.Rect.Min.Y+2)
	}
	for i, t := range bv.Tabs {
		bg := actionColor
		if i == bv.Tab {
			bg = talkSelectedColor
		}
		button(t, bg)
	}
	button(bv.Close, actionColor)
	ebitenutil.DebugPrintAt(win, "Find:", bv.Field.Min.X-6*viewer.CharWidth, bv.Field.Min.Y+2)
	fill(bv.Field, slotColor)
	edge(bv.Field, keywordColor)
	cursor := ""
	if v.Ticks/30%2 == 0 {
		cursor = "_"
	}
	ebitenutil.DebugPrintAt(win, viewer.Clip(bv.Typed+cursor, (bv.Field.Dx()-8)/viewer.CharWidth), bv.Field.Min.X+4, bv.Field.Min.Y+2)
	for i, r := range bv.Rows {
		bg := slotColor
		if i == bv.Selected {
			bg = talkSelectedColor
		}
		fill(r.Rect, bg)
	}

	// The pictures, drawn in the palette's colors.
	art := s.artImage(size)
	draw := func(img content.Image, at image.Point) {
		if err := s.sprites.DrawImage(art, img, at.X, at.Y, false); err != nil && s.err == nil {
			s.err = err
		}
	}
	for _, r := range bv.Rows {
		if !r.HasIcon {
			continue
		}
		if b := v.Pack.Sprites[r.Icon.Sprite]; b != nil {
			box := image.Rect(r.Rect.Min.X+2, r.Rect.Min.Y, r.Rect.Min.X+2+viewer.SlotSize, r.Rect.Max.Y)
			draw(r.Icon, box.Min.Add(image.Pt((box.Dx()-b.FrameSize[0])/2, (box.Dy()-b.FrameSize[1])/2)))
		}
	}
	if bv.HasPicture {
		draw(bv.Picture, bv.PictureAt)
	}
	s.presenter.PresentAt(win, art, 1, image.Point{})

	for _, r := range bv.Rows {
		x := r.Rect.Min.X + 4
		if r.HasIcon {
			x += viewer.SlotSize + 4
		}
		ebitenutil.DebugPrintAt(win, r.Label, x, r.Rect.Min.Y+(r.Rect.Dy()-viewer.LineHeight)/2)
	}
	if bv.Empty != "" {
		ebitenutil.DebugPrintAt(win, bv.Empty, bv.Pane.Min.X, bv.Pane.Min.Y)
	}
	for _, l := range bv.Lines {
		if len(l.Words) == 0 {
			continue
		}
		r := l.Words[0].Rect
		for _, w := range l.Words {
			r = r.Union(w.Rect)
		}
		switch {
		case l.Heading:
			vector.FillRect(win, float32(bv.Pane.Min.X), float32(r.Max.Y)-1, float32(bv.Pane.Dx()), 1, keywordColor, false)
		case l.Hint:
			fill(image.Rect(bv.Pane.Min.X-4, r.Min.Y, bv.Pane.Min.X-2, r.Max.Y), keywordColor)
		}
		for _, w := range l.Words {
			ebitenutil.DebugPrintAt(win, w.Text, w.Rect.Min.X, w.Rect.Min.Y)
		}
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(bv.Scale), float64(bv.Scale))
	op.GeoM.Translate(float64(bv.Rect.Min.X), float64(bv.Rect.Min.Y))
	screen.DrawImage(win, op)
}
