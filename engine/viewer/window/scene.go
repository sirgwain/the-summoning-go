package window

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/font"
	"github.com/sirgwain/the-summoning-go/engine/frame"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// Drawing a scene (docs/pack.md, Scenes): its pictures through the palette at the largest whole
// scale that fits a 320 × 200 place, then its caption, page of words, speech panels and fade.

// sceneBannerRow is where a scene's caption and banner show.
const sceneBannerRow = 180

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
		s.centerWords(screen, r.Words(inspect.Text(s.v.Pack, r.Text)), box, textScale, r.FromTop)
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
	s.centerWords(screen, r.Words(inspect.Text(s.v.Pack, line.Text)), textBox, textScale, false)
}

// sceneWordsColor is the gold of a scene's words (scenes.md, The story, in short).
var sceneWordsColor = color.RGBA{236, 200, 92, 255}

// centerWords draws text wrapped to box, a letter's room kept at each side, each line centered
// across it and the lines centered down it (or from its top when fromTop), in the large face at
// scale, in gold.
func (s *Window) centerWords(screen *ebiten.Image, text string, box image.Rectangle, scale int, fromTop bool) {
	face := font.Large
	lines := wrap(text, max(1, box.Dx()/(face.Advance*scale)-2))
	y := box.Min.Y
	if !fromTop {
		y += (box.Dy() - len(lines)*face.LineHeight*scale) / 2
	}
	for i, l := range lines {
		x := box.Min.X + (box.Dx()-face.Measure(l)*scale)/2
		s.printFace(screen, face, l, x, y+i*face.LineHeight*scale, scale, sceneWordsColor)
	}
}
