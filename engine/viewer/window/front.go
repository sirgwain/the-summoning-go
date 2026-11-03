package window

import (
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// Drawing the frontend: its menus with the debug font until the UI has its own, and scenes
// through the palette at the largest whole scale that fits.

var (
	frontColor  = color.RGBA{8, 6, 4, 255}
	chosenColor = color.RGBA{120, 80, 40, 240}
)

// sceneBannerRow is where a scene's caption and banner show.
const sceneBannerRow = 180

func (s *Window) drawFront(screen *ebiten.Image) {
	f := s.v.Front
	screen.Fill(frontColor)
	if f.Screen == viewer.ScreenScene {
		s.drawScene(screen, f.Scene)
		return
	}
	center := func(text string, y int) {
		ebitenutil.DebugPrintAt(screen, text, (f.Size.X-len(text)*viewer.CharWidth)/2, y)
	}
	center(f.Title, f.TitleAt())
	for i, l := range f.Lines {
		center(l, f.LinesAt()+i*viewer.LineHeight)
	}
	if f.Portrait != nil {
		if b := s.v.Pack.Sprites[f.Portrait.Sprite]; b != nil {
			art := s.artImage(image.Pt(b.FrameSize[0], b.FrameSize[1]))
			if err := s.sprites.DrawImage(art, *f.Portrait, 0, 0, false); err != nil && s.err == nil {
				s.err = err
			}
			s.presenter.SetColors(s.palette.At(0), s.v.FixedColors(s.palette))
			s.presenter.PresentAt(screen, art, viewer.PortraitScale, f.PortraitAt())
		}
	}
	for i, b := range f.Buttons {
		c := actionColor
		if i == f.Choice {
			c = chosenColor
		}
		r := b.Rect
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
		vector.StrokeRect(screen, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, slotEdgeColor, false)
		ebitenutil.DebugPrintAt(screen, b.Label, r.Min.X+(r.Dx()-len(b.Label)*viewer.CharWidth)/2, r.Min.Y+2)
	}
	s.drawNotice(screen)
}

// drawScene draws a scene's pictures, part, banner, caption and fade.
func (s *Window) drawScene(screen *ebiten.Image, r *viewer.SceneRun) {
	if r == nil {
		return
	}
	size := image.Pt(content.SceneWidth, content.SceneHeight)
	art := s.artImage(size)
	draw := func(img content.Image, x, y int) {
		if err := s.sprites.DrawImage(art, img, x, y, false); err != nil && s.err == nil {
			s.err = err
		}
	}
	for _, p := range r.Pictures {
		draw(p.Image, p.At.X, p.At.Y)
	}
	if st := r.Current(); st != nil && st.Part != nil {
		spread, _ := r.Spread()
		if st.Part.Background != nil {
			draw(*st.Part.Background, 0, 0)
		}
		draw(st.Part.Left, -8*spread, 0)
		draw(st.Part.Right, content.SceneWidth/2+8*spread, 0)
	}
	if r.Banner != nil {
		draw(*r.Banner, 0, sceneBannerRow)
	}
	scale := max(1, min(s.v.Screen.X/size.X, s.v.Screen.Y/size.Y))
	at := s.v.Screen.Sub(size.Mul(scale)).Div(2)
	s.presenter.SetColors(s.palette.At(0), s.v.FixedColors(s.palette))
	s.presenter.PresentAt(screen, art, scale, at)
	if r.Caption != "" {
		text := inspect.Text(s.v.Pack, r.Caption)
		ebitenutil.DebugPrintAt(screen, text, at.X+(size.X*scale-len(text)*viewer.CharWidth)/2, at.Y+sceneBannerRow*scale+(20*scale-viewer.LineHeight)/2)
	}
	if r.Fade != "" && r.Faded > 0 {
		c := parseColor(r.Fade)
		c.A = uint8(r.Faded * 255)
		vector.FillRect(screen, float32(at.X), float32(at.Y), float32(size.X*scale), float32(size.Y*scale),
			color.RGBA{uint8(int(c.R) * int(c.A) / 255), uint8(int(c.G) * int(c.A) / 255), uint8(int(c.B) * int(c.A) / 255), c.A}, false)
	}
}

// parseColor reads a color written #rrggbb.
func parseColor(s string) color.RGBA {
	if len(s) != 7 || s[0] != '#' {
		return color.RGBA{A: 255}
	}
	n, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return color.RGBA{A: 255}
	}
	return color.RGBA{uint8(n >> 16), uint8(n >> 8), uint8(n), 255}
}
