package window

import (
	"image"
	"image/color"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/frame"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// FlipbookFrame is the color of a framed flipbook's border, and FlipbookBorder its width in art
// pixels.
var FlipbookFrame = color.RGBA{120, 100, 70, 255}

const FlipbookBorder = 8

// drawNotice draws the viewer's message line above the hotbar while it lasts.
func (s *Window) drawNotice(screen *ebiten.Image) {
	v := s.v
	if v.Notice == "" || v.Ticks > v.NoticeUntil {
		return
	}
	r := v.NoticeRect()
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), panelColor, false)
	s.print(screen, v.Notice, r.Min.X+4, r.Min.Y+2)
}

// drawShowing draws the view effect or sparkle playing at its place in the view, a view
// effect's box with its border.
func (s *Window) drawShowing(screen *ebiten.Image) {
	v := s.v
	sh := v.Showing
	if sh == nil {
		return
	}
	size := sh.Size(v.Pack)
	if size == (image.Point{}) {
		return
	}
	at := v.ScreenPoint(sh.At)
	if sh.Framed {
		bw := float32(FlipbookBorder * v.Scale)
		x, y := float32(at.X), float32(at.Y)
		w, h := float32(size.X*v.Scale), float32(size.Y*v.Scale)
		// The border only: the box's transparent pixels show the view.
		vector.FillRect(screen, x-bw, y-bw, w+2*bw, bw, FlipbookFrame, false)
		vector.FillRect(screen, x-bw, y+h, w+2*bw, bw, FlipbookFrame, false)
		vector.FillRect(screen, x-bw, y, bw, h, FlipbookFrame, false)
		vector.FillRect(screen, x+w, y, bw, h, FlipbookFrame, false)
	}
	art := s.artImage(size)
	if err := s.sprites.DrawImage(art, sh.Frame(), 0, 0, false); err != nil && s.err == nil {
		s.err = err
	}
	s.presenter.PresentAt(screen, art, v.Scale, at)
}

// drawReading draws the picture of a scroll being read in a small window centered above the
// hotbar, at the hotbar's scale.
func (s *Window) drawReading(screen *ebiten.Image) {
	v := s.v
	img := v.Reading
	if img == nil {
		return
	}
	b := v.Pack.Sprites[img.Sprite]
	if b == nil {
		return
	}
	scale, bottom := v.Scale, v.Screen.Y
	if h := v.Hotbar(); h != nil {
		scale, bottom = h.Scale, h.Frame.Min.Y-viewer.Margin
	}
	size := image.Pt(b.FrameSize[0], b.FrameSize[1])
	// The picture sits a frame's width in from its frame, with room as wide again round it.
	pad := 2 * frame.Width
	at := image.Pt((v.Screen.X-size.X*scale)/2, bottom-(size.Y+pad)*scale)
	r := image.Rectangle{at, at.Add(size.Mul(scale))}.Inset(-pad * scale)
	s.panelFrame(screen, content.PanelNotice, frame.Iron, r, scale, 1)
	art := s.artImage(size)
	if err := s.sprites.DrawImage(art, *img, 0, 0, false); err != nil && s.err == nil {
		s.err = err
	}
	s.presenter.PresentAt(screen, art, scale, at)
}

// drawStatusIcons draws the player's status icons in the screen's top-right corner.
func (s *Window) drawStatusIcons(screen *ebiten.Image) {
	for _, b := range s.v.StatusIcons() {
		size := b.Rect.Size().Div(b.Scale)
		art := s.artImage(size)
		if err := s.sprites.DrawImage(art, b.Image, 0, 0, false); err != nil {
			if s.err == nil {
				s.err = err
			}
			continue
		}
		s.presenter.PresentAt(screen, art, b.Scale, b.Rect.Min)
	}
}

// drawPointer names what the cursor is over on the hotbar, and draws the pointer while playing:
// the arrow, the speech bubble over something a click talks to, or the object in hand, at the
// view's scale, in place of the system's pointer.
func (s *Window) drawPointer(screen *ebiten.Image) {
	v := s.v
	cursor := v.Cursor
	if lines := v.HoverLines(cursor); len(lines) > 0 {
		w := 0
		for _, l := range lines {
			w = max(w, utf8.RuneCountInString(l))
		}
		w = w*viewer.CharWidth + 4
		h := len(lines) * viewer.LineHeight
		// Above the pointer, kept on the screen.
		at := cursor.Add(image.Pt(12, -h-4))
		at.X = min(at.X, v.Screen.X-w)
		at.Y = max(at.Y, 0)
		vector.FillRect(screen, float32(at.X), float32(at.Y), float32(w), float32(h), panelColor, false)
		for i, l := range lines {
			s.print(screen, l, at.X+2, at.Y+i*viewer.LineHeight)
		}
	}
	img, centered, ok := v.Pointer()
	ok = ok && cursor.In(image.Rectangle{Max: v.Screen})
	mode := ebiten.CursorModeVisible
	if ok && !s.scripted() {
		mode = ebiten.CursorModeHidden
	}
	if ebiten.CursorMode() != mode {
		ebiten.SetCursorMode(mode)
	}
	if !ok {
		return
	}
	scale := v.Scale
	if h := v.Hotbar(); h != nil && centered {
		scale = h.Scale
	}
	fs := v.Pack.Sprites[img.Sprite].FrameSize
	size := image.Pt(fs[0], fs[1])
	art := s.artImage(size)
	if err := s.sprites.DrawImage(art, img, 0, 0, false); err != nil && s.err == nil {
		s.err = err
	}
	at := cursor
	if centered {
		at = cursor.Sub(size.Mul(scale).Div(2))
	}
	s.presenter.PresentAt(screen, art, scale, at)
}
