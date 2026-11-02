package render

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
)

// Mark is a draw command the glow outlines, in a color.
type Mark struct {
	Cmd   int
	Color color.RGBA
}

// GlowRadius is how far the glow reaches past a marked sprite's edge, in screen pixels.
const GlowRadius = 5

// glowShaderSrc lights the screen pixels outside the mask within GlowRadius of it, in the color of
// the nearest masked pixel, fading with the distance and pulsing with Time. The mask is in view
// pixels, drawn Scale times as large.
const glowShaderSrc = `//kage:unit pixels
package main

var Scale float
var Time float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	if imageSrc0At(srcPos).a > 0 {
		return vec4(0)
	}
	const r = 5
	best := float(r + 1)
	rgb := vec3(0)
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			d := length(vec2(float(dx), float(dy)))
			if d < best {
				m := imageSrc0At(srcPos + vec2(float(dx), float(dy))/Scale)
				if m.a > 0 {
					best, rgb = d, m.rgb/m.a
				}
			}
		}
	}
	if best > float(r) {
		return vec4(0)
	}
	a := clamp(1-(best-1)/float(r), 0, 1)
	a = sqrt(a) * (0.75 + 0.25*sin(Time*6))
	return vec4(rgb*a, a)
}
`

// Glow draws a glowing outline around marked draw commands of a level view.
type Glow struct {
	shader *ebiten.Shader
	mask   *ebiten.Image
}

// NewGlow compiles the glow shader.
func NewGlow() (*Glow, error) {
	sh, err := ebiten.NewShader([]byte(glowShaderSrc))
	if err != nil {
		return nil, fmt.Errorf("glow shader: %w", err)
	}
	return &Glow{shader: sh}, nil
}

// Draw outlines the marked commands of view v onto dst, which shows a buffer of size view pixels
// with view pixel origin at its top-left, scale screen pixels to a view pixel; t is the time in
// seconds, which the glow pulses with. A marked sprite's whole silhouette glows, even where what
// is drawn in front of it covers it.
func (g *Glow) Draw(dst *ebiten.Image, v *LevelView, marks []Mark, origin, size image.Point, scale int, t float64) {
	if len(marks) == 0 {
		return
	}
	if g.mask == nil || g.mask.Bounds().Size() != size {
		if g.mask != nil {
			g.mask.Deallocate()
		}
		g.mask = ebiten.NewImage(size.X, size.Y)
	}
	g.mask.Clear()
	marks = slices.Clone(marks)
	slices.SortStableFunc(marks, func(a, b Mark) int { return a.Cmd - b.Cmd })
	pad := (GlowRadius + scale - 1) / scale
	var used image.Rectangle
	var op colorm.DrawImageOptions
	for _, m := range marks {
		if m.Cmd < 0 || m.Cmd >= len(v.cmds) || v.cmds[m.Cmd].Sprite == "" {
			continue
		}
		c, img := v.cmds[m.Cmd], v.images[m.Cmd]
		at := image.Pt(c.X, c.Y).Sub(origin)
		r := img.Bounds().Sub(img.Bounds().Min).Add(at)
		if !r.Inset(-pad).Overlaps(g.mask.Bounds()) {
			continue
		}
		used = used.Union(r.Inset(-pad))
		// The frame's pixels, all in the mark's color.
		var cm colorm.ColorM
		cm.Scale(0, 0, 0, 1)
		cm.Translate(float64(m.Color.R)/255, float64(m.Color.G)/255, float64(m.Color.B)/255, 0)
		op.GeoM.Reset()
		op.GeoM.Translate(float64(at.X), float64(at.Y))
		colorm.DrawImage(g.mask, img, cm, &op)
	}
	used = used.Intersect(g.mask.Bounds())
	if used.Empty() {
		return
	}
	var sop ebiten.DrawRectShaderOptions
	sop.Images[0] = g.mask.SubImage(used).(*ebiten.Image)
	sop.Uniforms = map[string]any{"Scale": float32(scale), "Time": float32(t)}
	sop.GeoM.Scale(float64(scale), float64(scale))
	sop.GeoM.Translate(float64(used.Min.X*scale), float64(used.Min.Y*scale))
	dst.DrawRectShader(used.Dx(), used.Dy(), g.shader, &sop)
}
