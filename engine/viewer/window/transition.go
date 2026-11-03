package window

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// Drawing transitions and lost objects (engine/viewer/transition.go decides what they show).

// transitionShaderSrc draws a frame of a transition from the old view (image 0) and the new
// (image 1), both the screen's size: as two panels, or as one view rippled, tinted and cut to a
// circle. Every image is read at positions on image 0's texture, from its origin.
const transitionShaderSrc = `//kage:unit pixels
package main

var Panels float
var OldDst vec4
var OldSrc vec4
var NewDst vec4
var NewSrc vec4
var New float
var Shift float
var Rows float
var Amount float
var Center vec2
var Radius float
var Tint vec4

func inside(p vec2, r vec4) bool {
	return p.x >= r.x && p.y >= r.y && p.x < r.z && p.y < r.w
}

func stretch(p vec2, dst vec4, src vec4) vec2 {
	return src.xy + (p-dst.xy)*(src.zw-src.xy)/(dst.zw-dst.xy)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	p := dstPos.xy
	if Panels > 0.5 {
		if inside(p, NewDst) {
			return imageSrc1At(imageSrc0Origin() + stretch(p, NewDst, NewSrc))
		}
		if inside(p, OldDst) {
			return imageSrc0At(imageSrc0Origin() + stretch(p, OldDst, OldSrc))
		}
		return Tint
	}
	q := p
	if Shift != 0 {
		t := fract(p.y / (2 * Rows))
		q.x = clamp(q.x+Shift*(abs(t*4-2)-1), 0, imageSrc0Size().x-1)
	}
	c := imageSrc0At(imageSrc0Origin() + q)
	if New > 0.5 {
		c = imageSrc1At(imageSrc0Origin() + q)
	}
	c = mix(c, Tint, Amount)
	if Radius >= 0 {
		c = mix(Tint, c, clamp(Radius-distance(p, Center)+0.5, 0, 1))
	}
	return c
}
`

// transitions holds what drawing transitions needs: the shader, and two world images of the
// screen's size, one drawn into each frame and the other kept as the old view while a transition
// plays.
type transitions struct {
	shader *ebiten.Shader
	worlds [2]*ebiten.Image
	// cur is the world image drawn into; seq the transition whose old view the other holds.
	cur, seq int
	// icon holds a lost object's image in colors, by size.
	icons map[image.Point]*ebiten.Image
}

func newTransitions() (*transitions, error) {
	sh, err := ebiten.NewShader([]byte(transitionShaderSrc))
	if err != nil {
		return nil, fmt.Errorf("transition shader: %w", err)
	}
	return &transitions{shader: sh, icons: map[image.Point]*ebiten.Image{}}, nil
}

// world returns the world image to draw this frame into, keeping the last frame's as the old
// view when transition t has started since.
func (tr *transitions) world(size image.Point, t *viewer.Transition) *ebiten.Image {
	for i, w := range tr.worlds {
		if w == nil || w.Bounds().Size() != size {
			if w != nil {
				w.Deallocate()
			}
			tr.worlds[i] = ebiten.NewImage(size.X, size.Y)
		}
	}
	if t != nil && t.Seq != tr.seq {
		tr.seq = t.Seq
		tr.cur = 1 - tr.cur
	}
	w := tr.worlds[tr.cur]
	w.Clear()
	return w
}

// draw draws the world image drawn this frame onto screen, through transition t when one plays.
func (tr *transitions) draw(screen *ebiten.Image, t *viewer.Transition, scale int) {
	now := tr.worlds[tr.cur]
	if t == nil {
		screen.DrawImage(now, nil)
		return
	}
	size := now.Bounds().Size()
	f := t.Frame(size, scale)
	rect := func(r image.Rectangle) []float32 {
		return []float32{float32(r.Min.X), float32(r.Min.Y), float32(r.Max.X), float32(r.Max.Y)}
	}
	flag := func(b bool) float32 {
		if b {
			return 1
		}
		return 0
	}
	c := f.Color
	var op ebiten.DrawRectShaderOptions
	op.Images[0], op.Images[1] = tr.worlds[1-tr.cur], now
	op.Uniforms = map[string]any{
		"Panels": flag(f.Panels), "OldDst": rect(f.OldDst), "OldSrc": rect(f.OldSrc),
		"NewDst": rect(f.NewDst), "NewSrc": rect(f.NewSrc), "New": flag(f.New),
		"Shift": float32(f.Shift), "Rows": float32(max(1, f.Rows)), "Amount": float32(f.Amount),
		"Center": []float32{float32(f.Center.X), float32(f.Center.Y)}, "Radius": float32(f.Radius),
		"Tint": []float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, 1},
	}
	screen.DrawRectShader(size.X, size.Y, tr.shader, &op)
}

// shatterPieces is how many pieces a side of a shattering image breaks into, and lostFlash how
// much of its time its box flashes.
const (
	shatterPieces = 3
	lostFlash     = 0.35
)

// drawLost draws the objects lost from the player's slots, each going by its style.
func (s *Window) drawLost(screen *ebiten.Image) {
	for _, l := range s.v.LostItems() {
		b := s.v.Pack.Sprites[l.Image.Sprite]
		if b == nil {
			continue
		}
		size := image.Pt(b.FrameSize[0], b.FrameSize[1])
		icon, ok := s.tr.icons[size]
		if !ok {
			icon = ebiten.NewImage(size.X, size.Y)
			s.tr.icons[size] = icon
		}
		icon.Clear()
		art := s.artImage(size)
		if err := s.sprites.DrawImage(art, l.Image, 0, 0, false); err != nil {
			if s.err == nil {
				s.err = err
			}
			continue
		}
		s.presenter.PresentAt(icon, art, 1, image.Point{})
		at := l.Rect.Min.Add(l.Rect.Size().Sub(size.Mul(l.Scale)).Div(2))
		switch l.Style {
		case content.LostFade:
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(l.Scale), float64(l.Scale))
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			op.ColorScale.ScaleAlpha(float32(1 - l.Age))
			screen.DrawImage(icon, op)
		case content.LostShatter:
			// The box flashes as the image breaks.
			if f := 1 - l.Age/lostFlash; f > 0 {
				r := l.Rect.Inset(l.Scale)
				vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()),
					color.RGBA{255, 240, 200, uint8(220 * f)}, false)
			}
			drawShatter(screen, icon, size, at, l.Scale, l.Age)
		}
	}
}

// drawShatter draws icon, of size art pixels, broken into pieces flying out from its middle,
// falling, turning and fading as age goes from 0 to 1.
func drawShatter(screen, icon *ebiten.Image, size, at image.Point, scale int, age float64) {
	sc := float64(scale)
	mid := [2]float64{float64(size.X) / 2, float64(size.Y) / 2}
	for i := range shatterPieces {
		for j := range shatterPieces {
			r := image.Rect(size.X*i/shatterPieces, size.Y*j/shatterPieces,
				size.X*(i+1)/shatterPieces, size.Y*(j+1)/shatterPieces)
			if r.Empty() {
				continue
			}
			// Each piece flies away from the middle, a little up, then falls.
			c := [2]float64{float64(r.Min.X+r.Max.X) / 2, float64(r.Min.Y+r.Max.Y) / 2}
			dx, dy := c[0]-mid[0], c[1]-mid[1]
			n := float64(i*shatterPieces + j)
			spin := (math.Mod(n*2.3, 2) - 1) * 4
			x := c[0] + dx*4*age
			y := c[1] + (dy*3-10)*age + 40*age*age
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-float64(r.Dx())/2, -float64(r.Dy())/2)
			op.GeoM.Rotate(spin * age)
			op.GeoM.Translate(x, y)
			op.GeoM.Scale(sc, sc)
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			op.ColorScale.ScaleAlpha(float32(1 - age*age))
			screen.DrawImage(icon.SubImage(r).(*ebiten.Image), op)
		}
	}
}
