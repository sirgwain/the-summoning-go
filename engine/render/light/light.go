// Package light finds the light the view's fires and spells give, and how it flickers
// (docs/spec/world.md, Light and shadow). It does not depend on Ebitengine.
package light

import (
	"image/color"
	"math"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
)

// Ambient is how bright the view is away from any light.
const Ambient = 0.86

// MaxFactor is the most a pixel's color is multiplied by.
const MaxFactor = 1.25

// ShadowDepth is how much a shadow darkens at its center.
const ShadowDepth = 0.35

// Light is a light centered at view pixel (X, Y), reaching Radius pixels across and half as far
// up and down, in Color (its largest component 1) at Strength, flickering by Flicker at Speed.
type Light struct {
	X, Y     float64
	Radius   float64
	Color    [3]float64
	Strength float64
	Flicker  float64
	Speed    float64
}

// At returns the light's strength at time t, in seconds.
func (l Light) At(t float64) float64 {
	a := 0.37*l.X + 0.61*l.Y
	w := l.Speed * t
	return l.Strength * (1 + l.Flicker*(math.Sin(2.3*w+a)+math.Sin(3.7*w+1.7*a))/2)
}

// Reach returns how much of a light's strength reaches view pixel (x, y).
func (l Light) Reach(x, y float64) float64 {
	dx, dy := (x-l.X)/l.Radius, 2*(y-l.Y)/l.Radius
	d2 := dx*dx + dy*dy
	if d2 >= 1 {
		return 0
	}
	return (1 - d2) * (1 - d2)
}

// source is how a kind of glowing frame lights.
type source struct {
	bright                 bool
	strength, radius, drop float64
	flicker, speed         float64
}

var sources = map[drawlist.Glow]source{
	drawlist.GlowDecoration: {strength: 0.30, radius: 64, drop: 8, flicker: 0.08, speed: 1},
	drawlist.GlowSpell:      {bright: true, strength: 0.45, radius: 48, flicker: 0.15, speed: 3},
}

const (
	// minGlowing is the fewest glowing pixels a frame lights with.
	minGlowing = 4
	// fullGlowing is the count of glowing pixels a frame lights fully with.
	fullGlowing = 16
	// brightLevel is the mean component, of full, at which a spell's pixel glows.
	brightLevel = 0.6
)

// glow is what a frame's glowing pixels give: their count, their mean position from the
// frame's top-left, and the mean of their steady colors.
type glow struct {
	n     int
	x, y  float64
	color [3]float64
}

type frameKey struct {
	sprite string
	frame  int
	glow   drawlist.Glow
}

// Finder finds the lights of draw commands, from a pack's sprites in a palette.
type Finder struct {
	pack     *content.Pack
	steady   [256][3]float64
	animated [256]bool
	frames   map[frameKey]glow
}

// NewFinder returns a finder for pack p's sprites in palette pal.
func NewFinder(p *content.Pack, pal *content.Palette) *Finder {
	f := &Finder{pack: p, frames: map[frameKey]glow{}}
	colors, animated := Steady(pal)
	f.animated = animated
	for i, c := range colors {
		f.steady[i] = [3]float64{float64(c.R) / 255, float64(c.G) / 255, float64(c.B) / 255}
	}
	return f
}

// Steady returns each color's steady color in palette p, and whether an effect animates it.
func Steady(p *content.Palette) (colors [256]color.RGBA, animated [256]bool) {
	colors = p.Colors
	mean := func(cs []color.RGBA) color.RGBA {
		var r, g, b int
		for _, c := range cs {
			r, g, b = r+int(c.R), g+int(c.G), b+int(c.B)
		}
		n := len(cs)
		return color.RGBA{uint8((r + n/2) / n), uint8((g + n/2) / n), uint8((b + n/2) / n), 0xFF}
	}
	for _, e := range p.Effects {
		switch e.Kind {
		case content.EffectRotate:
			for g := range e.Groups {
				base := e.First + g*e.Size
				m := mean(p.Colors[base : base+e.Size])
				for i := base; i < base+e.Size; i++ {
					colors[i], animated[i] = m, true
				}
			}
		case content.EffectCycle:
			if len(e.Colors) > 0 {
				colors[e.Index], animated[e.Index] = mean(e.Colors), true
			}
		}
	}
	return colors, animated
}

// Find returns the lights the commands give.
func (f *Finder) Find(cmds []drawlist.Cmd) []Light {
	var out []Light
	for _, c := range cmds {
		src, ok := sources[c.Glow]
		if !ok || c.Sprite == "" {
			continue
		}
		g := f.frame(frameKey{c.Sprite, c.Frame, c.Glow}, src.bright)
		if g.n < minGlowing {
			continue
		}
		hue := g.color
		top := max(hue[0], hue[1], hue[2])
		if top <= 0 {
			continue
		}
		for i := range hue {
			hue[i] /= top
		}
		out = append(out, Light{
			X: float64(c.X) + g.x, Y: float64(c.Y) + g.y + src.drop,
			Radius: src.radius, Color: hue,
			Strength: src.strength * min(1, float64(g.n)/fullGlowing),
			Flicker:  src.flicker, Speed: src.speed,
		})
	}
	return out
}

// frame returns the glow of a frame, worked out once.
func (f *Finder) frame(k frameKey, bright bool) glow {
	if g, ok := f.frames[k]; ok {
		return g
	}
	var g glow
	b := f.pack.Sprites[k.sprite]
	if b != nil && b.Sheet != nil && k.frame >= 0 && k.frame < b.Frames {
		r := b.FrameRect(k.frame)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				i := int(b.Sheet.ColorIndexAt(x, y))
				if b.Transparent != nil && i == *b.Transparent {
					continue
				}
				c := f.steady[i]
				if !f.animated[i] && !(bright && (c[0]+c[1]+c[2])/3 >= brightLevel) {
					continue
				}
				g.n++
				g.x += float64(x-r.Min.X) + 0.5
				g.y += float64(y-r.Min.Y) + 0.5
				for j := range c {
					g.color[j] += c[j]
				}
			}
		}
	}
	if g.n > 0 {
		n := float64(g.n)
		g.x, g.y = g.x/n, g.y/n
		for j := range g.color {
			g.color[j] /= n
		}
	}
	f.frames[k] = g
	return g
}
