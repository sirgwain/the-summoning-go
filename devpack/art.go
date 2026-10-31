package devpack

import (
	"image/color"
	"math"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Palette layout: ramps of 8 shades, dark to light, and animated runs at the top.
const (
	black = 0
	// gray is a ramp of 16, the rest ramps of 8.
	gray   = 1
	stone  = 17
	wood   = 25
	red    = 33
	gold   = 41
	green  = 49
	blue   = 57
	purple = 65
	tan    = 73
	// water, lava and glow are runs of 4 that the palette rotates.
	water = 224
	lava  = 228
	glow  = 232
	// clear is the transparent index of every sprite.
	clear = 255
)

func ramp(c []color.RGBA, first, n int, dark, light color.RGBA) {
	for i := range n {
		t := float64(i) / float64(n-1)
		c[first+i] = color.RGBA{lerp(dark.R, light.R, t), lerp(dark.G, light.G, t), lerp(dark.B, light.B, t), 255}
	}
}

func lerp(a, b uint8, t float64) uint8 {
	return uint8(math.Round(float64(a) + (float64(b)-float64(a))*t))
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

// palette returns the pack's palette: ramps for the art, and rotating runs for water, lava and
// teleporter glow, so the pack shows palette effects working.
func palette() *content.Palette {
	p := &content.Palette{ID: "palette.dev"}
	c := p.Colors[:]
	for i := range c {
		c[i] = rgb(0, 0, 0)
	}
	ramp(c, gray, 16, rgb(16, 16, 20), rgb(236, 236, 240))
	ramp(c, stone, 8, rgb(52, 44, 36), rgb(176, 160, 132))
	ramp(c, wood, 8, rgb(48, 26, 10), rgb(196, 140, 80))
	ramp(c, red, 8, rgb(64, 8, 8), rgb(255, 120, 110))
	ramp(c, gold, 8, rgb(80, 56, 0), rgb(255, 236, 120))
	ramp(c, green, 8, rgb(8, 48, 12), rgb(140, 240, 120))
	ramp(c, blue, 8, rgb(10, 20, 72), rgb(130, 170, 255))
	ramp(c, purple, 8, rgb(40, 10, 60), rgb(220, 150, 255))
	ramp(c, tan, 8, rgb(90, 56, 36), rgb(250, 210, 170))
	copy(c[water:], []color.RGBA{rgb(20, 50, 130), rgb(30, 70, 160), rgb(50, 100, 190), rgb(30, 70, 160)})
	copy(c[lava:], []color.RGBA{rgb(150, 20, 0), rgb(210, 70, 0), rgb(255, 150, 20), rgb(210, 70, 0)})
	copy(c[glow:], []color.RGBA{rgb(20, 90, 110), rgb(40, 170, 200), rgb(140, 240, 255), rgb(40, 170, 200)})
	c[clear] = rgb(255, 0, 255)
	p.Effects = []content.PaletteEffect{
		{Kind: content.EffectRotate, PeriodMS: 250, First: water, Size: 4, Groups: 2},
		{Kind: content.EffectRotate, PeriodMS: 120, First: glow, Size: 4, Groups: 1},
	}
	return p
}

// canvas is one frame's palette indexes.
type canvas struct {
	w, h int
	pix  []byte
}

func newCanvas(w, h int, fill byte) *canvas {
	c := &canvas{w: w, h: h, pix: make([]byte, w*h)}
	for i := range c.pix {
		c.pix[i] = fill
	}
	return c
}

func (c *canvas) set(x, y int, v byte) {
	if x >= 0 && y >= 0 && x < c.w && y < c.h {
		c.pix[y*c.w+x] = v
	}
}

func (c *canvas) at(x, y int) byte {
	if x >= 0 && y >= 0 && x < c.w && y < c.h {
		return c.pix[y*c.w+x]
	}
	return clear
}

// fill sets every pixel whose center is in shape, to the color paint returns for it.
func (c *canvas) fill(in func(x, y float64) bool, paint func(x, y int) byte) {
	for y := range c.h {
		for x := range c.w {
			if in(float64(x)+0.5, float64(y)+0.5) {
				c.set(x, y, paint(x, y))
			}
		}
	}
}

// outline paints the pixels of shape that border pixels outside it.
func (c *canvas) outline(in func(x, y float64) bool, v byte) {
	inside := func(x, y int) bool { return in(float64(x)+0.5, float64(y)+0.5) }
	for y := range c.h {
		for x := range c.w {
			if inside(x, y) && (!inside(x-1, y) || !inside(x+1, y) || !inside(x, y-1) || !inside(x, y+1)) {
				c.set(x, y, v)
			}
		}
	}
}

func (c *canvas) line(x0, y0, x1, y1 int, v byte) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	err := dx + dy
	for {
		c.set(x0, y0, v)
		if x0 == x1 && y0 == y1 {
			return
		}
		if e2 := 2 * err; e2 >= dy {
			err += dy
			x0 += sx
		} else {
			err += dx
			y0 += sy
		}
	}
}

func (c *canvas) rect(x0, y0, x1, y1 int, v byte) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c.set(x, y, v)
		}
	}
}

func solid(v byte) func(x, y int) byte { return func(int, int) byte { return v } }

// diamond is the isometric diamond centered at (cx, cy), w wide and w/2 tall.
func diamond(cx, cy, w float64) func(x, y float64) bool {
	return func(x, y float64) bool { return math.Abs(x-cx)/(w/2)+math.Abs(y-cy)/(w/4) <= 1 }
}

// ellipse is the ellipse centered at (cx, cy) with radii rx, ry.
func ellipse(cx, cy, rx, ry float64) func(x, y float64) bool {
	return func(x, y float64) bool {
		dx, dy := (x-cx)/rx, (y-cy)/ry
		return dx*dx+dy*dy <= 1
	}
}

func and(a, b func(x, y float64) bool) func(x, y float64) bool {
	return func(x, y float64) bool { return a(x, y) && b(x, y) }
}

func not(a func(x, y float64) bool) func(x, y float64) bool {
	return func(x, y float64) bool { return !a(x, y) }
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func sign(a int) int {
	switch {
	case a < 0:
		return -1
	case a > 0:
		return 1
	}
	return 0
}

// noise is a fixed pseudo-random value 0..n-1 for a pixel, for texture.
func noise(x, y, n int) int {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ h>>13) * 1274126177
	return int((h ^ h>>16) % uint32(n))
}

// bank makes a sprite bank of frames, each a canvas of the same size, transparent where clear.
func bank(p *content.Pack, id string, frames []*canvas, anchor *content.Offset) *content.SpriteBank {
	w, h := frames[0].w, frames[0].h
	data := make([][]byte, len(frames))
	for i, f := range frames {
		data[i] = f.pix
	}
	t := clear
	transparent := &t
	sheet, columns := content.NewSheet(data, w, h, 8, p.Palettes[paletteID], transparent, clear)
	b := &content.SpriteBank{
		ID: id, FrameSize: content.Offset{w, h}, Columns: columns, Frames: len(frames),
		Transparent: transparent, Anchor: anchor, Sheet: sheet,
	}
	p.Sprites[id] = b
	return b
}
