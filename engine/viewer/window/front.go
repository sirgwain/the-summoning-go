package window

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/font"
	"github.com/sirgwain/the-summoning-go/engine/frame"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// Drawing the frontend: its menus in an iron frame over the splash artwork darkened, with the
// debug font until the UI has its own, and scenes through the palette at the largest whole scale
// that fits.

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

// sceneBannerRow is where a scene's caption and banner show.
const sceneBannerRow = 180

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

// drawScene draws a scene's pictures, part, banner, caption and fade.
func (s *Window) drawScene(screen *ebiten.Image, r *viewer.SceneRun) {
	if r == nil {
		return
	}
	w, h := r.Def.Dims()
	art := s.artImage(image.Pt(w, h))
	draw := func(img content.Image, x, y int) {
		if err := s.sprites.DrawImage(art, img, x, y, false); err != nil && s.err == nil {
			s.err = err
		}
	}
	for _, p := range r.Pictures {
		if !p.Crop.Empty() {
			if err := s.sprites.DrawImagePart(art, p.Image, p.At.X, p.At.Y, p.Crop); err != nil && s.err == nil {
				s.err = err
			}
			continue
		}
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
	// The scene shows in a 320 × 200 place at the largest whole scale that fits; a taller one
	// with each row shrunk to fit.
	size := image.Pt(content.SceneWidth, content.SceneHeight)
	scale := max(1, min(s.v.Screen.X/size.X, s.v.Screen.Y/size.Y))
	at := s.v.Screen.Sub(size.Mul(scale)).Div(2)
	rowScale := float64(scale) * float64(content.SceneHeight) / float64(h)
	s.presenter.SetColors(s.palette.At(0), s.v.FixedColors(s.palette))
	s.presenter.PresentScaled(screen, art, float64(scale), rowScale, at)
	// toScreen maps a point of the scene's picture to the screen.
	toScreen := func(x, y int) image.Point {
		return image.Pt(at.X+x*scale, at.Y+int(float64(y)*rowScale))
	}
	if r.Caption != "" {
		text := inspect.Text(s.v.Pack, r.Caption)
		s.print(screen, text, at.X+(size.X*scale-len(text)*viewer.CharWidth)/2, at.Y+sceneBannerRow*scale+(20*scale-viewer.LineHeight)/2)
	}
	textScale := max(1, scale/2)
	if r.Text != "" {
		box := image.Rectangle{toScreen(r.Box.Min.X, r.Box.Min.Y), toScreen(r.Box.Max.X, r.Box.Max.Y)}
		s.centerWords(screen, r.Words(inspect.Text(s.v.Pack, r.Text)), box, textScale)
	}
	if r.Talking {
		vector.FillRect(screen, float32(at.X), float32(at.Y), float32(size.X*scale), float32(size.Y*scale),
			color.RGBA{A: sceneDarken}, false)
		for i, line := range []*viewer.SceneLine{r.Top, r.Bottom} {
			s.drawSay(screen, r, line, i == 1, at, scale, textScale)
		}
	}
	if r.Fade != "" && r.Faded > 0 {
		c := parseColor(r.Fade)
		c.A = uint8(r.Faded * 255)
		vector.FillRect(screen, float32(at.X), float32(at.Y), float32(size.X*scale), float32(size.Y*scale),
			color.RGBA{uint8(int(c.R) * int(c.A) / 255), uint8(int(c.G) * int(c.A) / 255), uint8(int(c.B) * int(c.A) / 255), c.A}, false)
	}
}

// A scene's speech panels, in the 320 × 200 place: the speaker's above, the character's below,
// each with its portrait inset at one end and its line beside it; and how dark the picture under
// them shows (alpha).
var (
	sayTop    = image.Rect(8, 8, 312, 76)
	sayBottom = image.Rect(8, 124, 312, 192)
)

const (
	sayInset    = 10
	sceneDarken = 150
)

// drawSay draws a speech panel, the character's below when bottom, with line in it if any.
func (s *Window) drawSay(screen *ebiten.Image, r *viewer.SceneRun, line *viewer.SceneLine, bottom bool, at image.Point, scale, textScale int) {
	panel := sayTop
	if bottom {
		panel = sayBottom
	}
	outer := image.Rectangle{at.Add(panel.Min.Mul(scale)), at.Add(panel.Max.Mul(scale))}
	s.drawFrame(screen, frame.Iron, outer, scale, menuOpacity)
	if line == nil {
		return
	}
	textBox := outer.Inset(sayInset * scale)
	if pt := s.v.Pack.Portraits[line.Portrait]; pt != nil {
		if b := s.v.Pack.Sprites[pt.Image.Sprite]; b != nil {
			ps := image.Pt(b.FrameSize[0], b.FrameSize[1])
			img := s.artImage(ps)
			mirrored := bottom && pt.Mirrored
			if err := s.sprites.DrawImage(img, pt.Image, 0, 0, mirrored); err != nil && s.err == nil {
				s.err = err
			}
			y := outer.Min.Y + (outer.Dy()-ps.Y*scale)/2
			x := outer.Min.X + sayInset*scale
			if bottom {
				x = outer.Max.X - sayInset*scale - ps.X*scale
				textBox.Max.X = x - sayInset*scale
			} else {
				textBox.Min.X = x + ps.X*scale + sayInset*scale
			}
			s.presenter.PresentAt(screen, img, scale, image.Pt(x, y))
		}
	}
	s.centerWords(screen, r.Words(inspect.Text(s.v.Pack, line.Text)), textBox, textScale)
}

// centerWords draws text wrapped to box, a letter's room kept at each side, each line centered
// across it and the lines centered down it, in the large face at scale.
func (s *Window) centerWords(screen *ebiten.Image, text string, box image.Rectangle, scale int) {
	face := font.Large
	lines := wrap(text, max(1, box.Dx()/(face.Advance*scale)-2))
	y := box.Min.Y + (box.Dy()-len(lines)*face.LineHeight*scale)/2
	for i, l := range lines {
		x := box.Min.X + (box.Dx()-face.Measure(l)*scale)/2
		s.printFace(screen, face, l, x, y+i*face.LineHeight*scale, scale, color.White)
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
