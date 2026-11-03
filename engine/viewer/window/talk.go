package window

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// drawTalk draws the open conversation: the line beside its portrait, its keywords underlined,
// or the offered keywords and the field. It draws the window at its own pixels, then scales it
// onto the screen.
func (s *Window) drawTalk(screen *ebiten.Image) {
	v := s.v
	tv := v.Talk()
	if tv == nil {
		return
	}
	size := tv.Rect.Size().Div(tv.Scale)
	if s.talk == nil || s.talk.Bounds().Size() != size {
		if s.talk != nil {
			s.talk.Deallocate()
		}
		s.talk = ebiten.NewImage(size.X, size.Y)
	}
	win := s.talk
	win.Fill(talkColor)
	vector.StrokeRect(win, 0.5, 0.5, float32(size.X-1), float32(size.Y-1), 1, slotEdgeColor, false)
	fill := func(r image.Rectangle, c color.Color) {
		vector.FillRect(win, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
	}
	draw := func(img content.Image, at image.Point, scale int, mirrored bool) {
		b := v.Pack.Sprites[img.Sprite]
		if b == nil {
			return
		}
		art := s.artImage(image.Pt(b.FrameSize[0], b.FrameSize[1]))
		if err := s.sprites.DrawImage(art, img, 0, 0, mirrored); err != nil && s.err == nil {
			s.err = err
		}
		s.presenter.PresentAt(win, art, scale, at)
	}
	if tv.HasPortrait {
		draw(tv.Portrait, tv.PortraitAt, tv.PortraitScale, tv.Mirrored)
	}
	for i, img := range tv.Icons {
		draw(img, tv.IconsAt[i], 2, false)
	}
	for _, w := range tv.Words {
		ebitenutil.DebugPrintAt(win, w.Text, w.Rect.Min.X, w.Rect.Min.Y)
		if w.Keyword {
			vector.StrokeLine(win, float32(w.Rect.Min.X), float32(w.Rect.Max.Y)-0.5, float32(w.Rect.Max.X), float32(w.Rect.Max.Y)-0.5, 1, keywordColor, false)
		}
	}
	// The highlighted button or field, what Enter takes, is lit and edged in the keyword color.
	edge := func(r image.Rectangle, selected bool) {
		c := slotEdgeColor
		if selected {
			c = keywordColor
		}
		vector.StrokeRect(win, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, c, false)
	}
	for i, b := range tv.Buttons {
		selected := !tv.Waiting && i == tv.Selected
		bg := actionColor
		if selected {
			bg = talkSelectedColor
		}
		fill(b.Rect, bg)
		edge(b.Rect, selected)
		ebitenutil.DebugPrintAt(win, b.Label, b.Rect.Min.X+(b.Rect.Dx()-len(b.Label)*viewer.CharWidth)/2, b.Rect.Min.Y+2)
	}
	if !tv.Waiting {
		fill(tv.Field, slotColor)
		edge(tv.Field, tv.Selected == len(tv.Buttons))
		cursor := ""
		if v.Ticks/30%2 == 0 {
			cursor = "_"
		}
		ebitenutil.DebugPrintAt(win, tv.Typed+cursor, tv.Field.Min.X+4, tv.Field.Min.Y+2)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(tv.Scale), float64(tv.Scale))
	op.GeoM.Translate(float64(tv.Rect.Min.X), float64(tv.Rect.Min.Y))
	screen.DrawImage(win, op)
}

// drawSaid draws a mouth's message beside the mouth: wrapped lines on a dark backing, hanging
// above the mouth and kept on the screen, at twice the debug font's size.
func (s *Window) drawSaid(screen *ebiten.Image) {
	v := s.v
	said := v.Said
	if said == nil {
		return
	}
	const scale, width = 2, 36
	lines := wrap(said.Text, width)
	size := image.Pt(width*viewer.CharWidth+8, len(lines)*viewer.LineHeight+8)
	if s.said == nil || s.said.Bounds().Size() != size {
		if s.said != nil {
			s.said.Deallocate()
		}
		s.said = ebiten.NewImage(size.X, size.Y)
	}
	s.said.Fill(talkColor)
	vector.StrokeRect(s.said, 0.5, 0.5, float32(size.X-1), float32(size.Y-1), 1, slotEdgeColor, false)
	for i, l := range lines {
		ebitenutil.DebugPrintAt(s.said, l, 4, 4+i*viewer.LineHeight)
	}
	at := v.ScreenPoint(said.At)
	shown := size.Mul(scale)
	x := min(max(at.X-shown.X/2, viewer.Margin), v.Screen.X-shown.X-viewer.Margin)
	y := max(at.Y-shown.Y-viewer.Margin, viewer.Margin)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(s.said, op)
}

// wrap breaks text into lines of at most n characters, at spaces.
func wrap(text string, n int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case len([]rune(line))+1+len([]rune(word)) <= n:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
