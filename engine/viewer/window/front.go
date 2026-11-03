package window

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/font"
	"github.com/sirgwain/the-summoning-go/engine/frame"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// Drawing the frontend: its menus in an iron frame over the splash artwork darkened, lettered in
// the engine's own font. Scenes are drawn by scene.go.

var (
	frontColor = color.RGBA{8, 6, 4, 255}
	// A menu button's iron plaque: plain, under the pointer and chosen, with its edges.
	plaqueColor        = color.RGBA{28, 34, 44, 255}
	plaqueHoverColor   = color.RGBA{40, 48, 63, 255}
	plaqueChosenColor  = color.RGBA{50, 66, 94, 255}
	plaqueLightColor   = color.RGBA{84, 98, 120, 255}
	plaqueDarkColor    = color.RGBA{6, 8, 12, 255}
	sapphireLightColor = color.RGBA{143, 194, 255, 255}
	sapphireDarkColor  = color.RGBA{42, 96, 198, 255}
)

const (
	// menuBrightness and creationBrightness are how bright the splash artwork shows behind
	// the main menu and behind the other screens; backdropStep is how far it moves each draw
	// toward that, half a second from full.
	menuBrightness     = 0.45
	creationBrightness = 0.3
	backdropStep       = (1 - menuBrightness) / 30
	// menuPad is the room inside a menu's frame round what it shows, in screen pixels.
	menuPad = 12
	// menuOpacity is how solid a menu panel's inside is.
	menuOpacity = 0.85
)

func (s *Window) drawFront(screen *ebiten.Image) {
	f := s.v.Front
	screen.Fill(frontColor)
	switch f.Screen {
	case viewer.ScreenSplash:
		s.backdrop = 1
		s.drawSplash(screen, f.SplashShown())
		return
	case viewer.ScreenScene:
		s.drawScene(screen, f.Scene)
		return
	}
	lettered := false
	if !f.OverPlay() {
		target := float32(creationBrightness)
		if f.Screen == viewer.ScreenMenu {
			target = menuBrightness
		}
		if s.backdrop < target {
			s.backdrop = min(target, s.backdrop+backdropStep)
		} else {
			s.backdrop = max(target, s.backdrop-backdropStep)
		}
		art := s.drawSplashArt(screen, s.backdrop)
		if f.Screen == viewer.ScreenMenu {
			s.drawSplashTitle(screen, art, 1)
			lettered = true
		}
	}
	panel := s.drawMenuPanel(screen, f, lettered)
	center := func(text string, y int) {
		s.printFace(screen, font.Large, text, (f.Size.X-font.Large.Measure(text))/2, y, 1, color.White)
	}
	if !lettered {
		center(f.Title, f.TitleAt())
	}
	for i, l := range f.Lines {
		center(l, f.LinesAt()+i*viewer.MenuLineHeight)
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
	cursor := image.Pt(ebiten.CursorPosition())
	for i, b := range f.Buttons {
		r, w := b.Rect, font.Large.Measure(b.Label)
		// The sapphires go only where there is room beside the label.
		gems := (r.Dx()-w)/2 >= r.Dy()
		drawPlaque(screen, r, i == f.Choice, cursor.In(r), gems, menuScale(f))
		s.printFace(screen, font.Large, b.Label, r.Min.X+(r.Dx()-w)/2, r.Min.Y+2, 1, color.White)
	}
	if f.RollsShown > 0 && !panel.Empty() {
		text := fmt.Sprintf("Rolls: %d", f.Rolls)
		a := f.RollsShown
		c := color.RGBA{uint8(200 * a), uint8(220 * a), uint8(255 * a), uint8(255 * a)}
		s.printFace(screen, font.Large, text, (f.Size.X-font.Large.Measure(text))/2, panel.Max.Y+viewer.MenuLineHeight/2, 1, c)
	}
	s.drawNotice(screen)
}

// menuScale is the screen pixels per frame pixel of a menu's frame.
func menuScale(f *viewer.Frontend) int { return max(1, min(4, f.Size.X/600)) }

// drawMenuPanel draws the iron frame round what a menu shows: its title (unless the splash
// letters it), lines, portrait and buttons. It returns the frame's rectangle, empty with none.
func (s *Window) drawMenuPanel(screen *ebiten.Image, f *viewer.Frontend, lettered bool) image.Rectangle {
	var r image.Rectangle
	add := func(q image.Rectangle) {
		if r.Empty() {
			r = q
		} else {
			r = r.Union(q)
		}
	}
	line := func(text string, y int) {
		if w := font.Large.Measure(text); w > 0 {
			x := (f.Size.X - w) / 2
			add(image.Rect(x, y, x+w, y+viewer.MenuLineHeight))
		}
	}
	if !lettered {
		line(f.Title, f.TitleAt())
	}
	for i, l := range f.Lines {
		line(l, f.LinesAt()+i*viewer.MenuLineHeight)
	}
	if f.Portrait != nil {
		if b := s.v.Pack.Sprites[f.Portrait.Sprite]; b != nil {
			at := f.PortraitAt()
			add(image.Rectangle{at, at.Add(image.Pt(b.FrameSize[0], b.FrameSize[1]).Mul(viewer.PortraitScale))})
		}
	}
	for _, b := range f.Buttons {
		add(b.Rect)
	}
	if r.Empty() {
		return r
	}
	scale := menuScale(f)
	outer := r.Inset(-(menuPad + frame.Width*scale))
	// The frame is whole frame pixels.
	outer.Max.X += (scale - outer.Dx()%scale) % scale
	outer.Max.Y += (scale - outer.Dy()%scale) % scale
	s.drawFrame(screen, frame.Iron, outer, scale, menuOpacity)
	return outer
}

// drawPlaque draws a menu button's iron plaque in r, its edges k pixels: lit steel when chosen,
// with a sapphire at each end if gems, lighter under the pointer.
func drawPlaque(screen *ebiten.Image, r image.Rectangle, chosen, hovered, gems bool, k int) {
	fill := func(q image.Rectangle, c color.Color) {
		vector.FillRect(screen, float32(q.Min.X), float32(q.Min.Y), float32(q.Dx()), float32(q.Dy()), c, false)
	}
	face, light := plaqueColor, plaqueLightColor
	switch {
	case chosen:
		face, light = plaqueChosenColor, sapphireLightColor
	case hovered:
		face = plaqueHoverColor
	}
	k = min(k, r.Dy()/4)
	fill(r, plaqueDarkColor)
	fill(image.Rect(r.Min.X, r.Min.Y, r.Max.X-k, r.Max.Y-k), light)
	fill(image.Rect(r.Min.X+k, r.Min.Y+k, r.Max.X-k, r.Max.Y-k), face)
	if !chosen || !gems {
		return
	}
	// A cut sapphire at each end: light above, dark below.
	rad := float32(r.Dy()-2*k) / 2 * 0.45
	cy := float32(r.Min.Y+r.Max.Y) / 2
	for _, cx := range []float32{float32(r.Min.X+k+3) + rad, float32(r.Max.X-k-3) - rad} {
		for i, c := range []color.RGBA{sapphireLightColor, sapphireDarkColor} {
			dy := -rad
			if i == 1 {
				dy = rad
			}
			var p vector.Path
			p.MoveTo(cx-rad, cy)
			p.LineTo(cx, cy+dy)
			p.LineTo(cx+rad, cy)
			p.Close()
			op := &vector.DrawPathOptions{AntiAlias: true}
			op.ColorScale.ScaleWithColor(c)
			vector.FillPath(screen, &p, nil, op)
		}
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
