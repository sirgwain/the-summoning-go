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

// source is how a kind of glowing frame lights: from its glowing pixels (the animated ones, and
// the bright ones too when bright is set), of which it needs at least fewest.
type source struct {
	bright                 bool
	fewest                 int
	strength, radius, drop float64
	flicker, speed         float64
}

var sources = map[drawlist.Glow]source{
	drawlist.GlowDecoration: {fewest: 4, strength: 0.30, radius: 64, drop: 8, flicker: 0.08, speed: 1},
	drawlist.GlowSpell:      {bright: true, fewest: 4, strength: 0.45, radius: 48, flicker: 0.15, speed: 3},
	drawlist.GlowThing:      {fewest: 4, strength: 0.35, radius: 56, flicker: 0.10, speed: 2},
	drawlist.GlowObject:     {fewest: 4, strength: 0.25, radius: 48, flicker: 0.05, speed: 1},
	drawlist.GlowFloor:      {fewest: 32, strength: 0.25, radius: 48, flicker: 0.05, speed: 1},
}

// shield is the light a shield gives, from the middle of the player's frame.
type shield struct {
	color                            [3]float64
	strength, radius, flicker, speed float64
}

var (
	fireShield      = shield{color: [3]float64{1, 0.55, 0.2}, strength: 0.50, radius: 72, flicker: 0.15, speed: 3}
	lightningShield = shield{color: [3]float64{0.65, 0.8, 1}, strength: 0.50, radius: 72, flicker: 0.35, speed: 7}
)

// shields are the shields each shield glow gives light from.
var shields = map[drawlist.Glow][]shield{
	drawlist.GlowFireShield:      {fireShield},
	drawlist.GlowLightningShield: {lightningShield},
	drawlist.GlowShields:         {fireShield, lightningShield},
}

const (
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
	part   drawlist.Part
	bright bool
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
		if c.Sprite == "" {
			continue
		}
		if sh, ok := shields[c.Glow]; ok {
			out = append(out, f.shields(c, sh)...)
			continue
		}
		src, ok := sources[c.Glow]
		if !ok {
			continue
		}
		g := f.frame(frameKey{c.Sprite, c.Frame, c.Part, src.bright})
		if g.n < src.fewest {
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

// shields returns the lights of shields sh around the player drawn by c, centered on the middle
// of its frame.
func (f *Finder) shields(c drawlist.Cmd, sh []shield) []Light {
	b := f.pack.Sprites[c.Sprite]
	if b == nil {
		return nil
	}
	x, y := float64(c.X)+float64(b.FrameSize[0])/2, float64(c.Y)+float64(b.FrameSize[1])/2
	out := make([]Light, len(sh))
	for i, s := range sh {
		out[i] = Light{X: x, Y: y, Radius: s.radius, Color: s.color, Strength: s.strength, Flicker: s.flicker, Speed: s.speed}
	}
	return out
}

// frame returns the glow of a frame, or of the part of it drawn, worked out once. Positions are
// from the top-left of what is drawn.
func (f *Finder) frame(k frameKey) glow {
	if g, ok := f.frames[k]; ok {
		return g
	}
	var g glow
	b := f.pack.Sprites[k.sprite]
	if b != nil && b.Sheet != nil && k.frame >= 0 && k.frame < b.Frames {
		r := b.FrameRect(k.frame)
		switch k.part {
		case drawlist.UpperHalf:
			r.Max.Y = min(r.Max.Y, r.Min.Y+drawlist.HalfHeight)
		case drawlist.LowerHalf:
			r.Min.Y = min(r.Max.Y, r.Min.Y+drawlist.HalfHeight)
		}
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				i := int(b.Sheet.ColorIndexAt(x, y))
				if b.Transparent != nil && i == *b.Transparent {
					continue
				}
				c := f.steady[i]
				if !f.animated[i] && !(k.bright && (c[0]+c[1]+c[2])/3 >= brightLevel) {
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
