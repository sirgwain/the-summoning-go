// Package frame draws the engine's panel frames: borders generated from noise and rules, at any
// size, so the UI needs no original art (docs/design.md, UI). It uses only the standard library.
package frame

import (
	"image"
	"image/color"
	"math"
)

const (
	// Width is the frame's thickness on each side, in the panel's pixels.
	Width = 8
	// Corner is the side of a corner piece.
	Corner = 12
	// GlowGroups is how many groups of lights breathe out of step.
	GlowGroups = 3
	// glowPeriod is a breath's length, in seconds.
	glowPeriod = 4.0
)

// Style is a frame's look.
type Style int

const (
	// Iron is blue-black brushed iron with rivets and a sapphire at each corner.
	Iron Style = iota
	// Runic is dark slate cut with runes lit blue.
	Runic
	// Hull is riveted hull plate with amber indicator lights, for science fiction.
	Hull
)

// styles are the built-in styles' painters, by style; Names their names in a pack's skin.
var styles = map[Style]painter{Iron: iron{}, Runic: runic{}, Hull: hull{}}

// Names are the built-in styles by name.
var Names = map[string]Style{"iron": Iron, "runic": Runic, "hull": Hull}

// lit are the styles with lights.
var lit = map[Style]bool{Runic: true, Hull: true}

// Picture is a frame drawn at one size: Base is the frame with its inside clear, and Glow, for
// a style with lights, the lights of each group, to be drawn over Base at Pulse's strength.
type Picture struct {
	Base *image.RGBA
	Glow [GlowGroups]*image.RGBA
}

// Pulse is how bright group g's lights are at time t in seconds, 0..1.
func Pulse(g int, t float64) float64 {
	return 0.5 + 0.5*math.Sin(2*math.Pi*t/glowPeriod+float64(g)*2*math.Pi/GlowGroups)
}

// Draw draws a frame of style with its outer size w × h, the frame Width inside that.
func Draw(style Style, w, h int) *Picture {
	p := &Picture{Base: image.NewRGBA(image.Rect(0, 0, w, h))}
	st := painterOf(style)
	if lit[style] {
		for g := range p.Glow {
			p.Glow[g] = image.NewRGBA(p.Base.Rect)
		}
	}
	for y := range h {
		for x := range w {
			d := min(x, y, w-1-x, h-1-y)
			if d >= Width {
				continue
			}
			var px pixel
			cx, cy := cornerAt(x, w), cornerAt(y, h)
			if cx >= 0 && cy >= 0 {
				px = st.corner(cx, cy)
			} else {
				var e edge
				switch d {
				case y:
					e = edge{side: top, t: x, u: y, span: w}
				case x:
					e = edge{side: left, t: y, u: x, span: h}
				case h - 1 - y:
					e = edge{side: bottom, t: x, u: h - 1 - y, span: w}
				default:
					e = edge{side: right, t: y, u: w - 1 - x, span: h}
				}
				px = st.edge(e, x, y)
			}
			p.Base.SetRGBA(x, y, px.c.rgba())
			if px.glow > 0 {
				a := uint8(math.Round(255 * px.glow))
				g := px.light
				p.Glow[px.group].SetRGBA(x, y, color.RGBA{premul(g[0], a), premul(g[1], a), premul(g[2], a), a})
			}
		}
	}
	return p
}

// Inside is the color a panel framed in style fills its inside with, before any see-through.
func Inside(style Style) color.RGBA { return painterOf(style).inside() }

func painterOf(style Style) painter {
	if p, ok := styles[style]; ok {
		return p
	}
	return iron{}
}

// cornerAt is where coordinate v lies in a corner piece along a side of length n, or -1 when it
// lies between the corners.
func cornerAt(v, n int) int {
	switch {
	case v < Corner:
		return v
	case v >= n-Corner:
		return v - (n - Corner)
	}
	return -1
}

func premul(c float64, a uint8) uint8 {
	return uint8(math.Round(clamp(c) * float64(a) / 255))
}

// side is which side of the frame an edge pixel is on.
type side int

const (
	top side = iota
	left
	bottom
	right
)

// lit reports whether a side faces the light, from the top left.
func (s side) lit() bool { return s == top || s == left }

// edge is an edge pixel: t along its side, u across it from the outside in, span the side's
// length.
type edge struct {
	side side
	t, u int
	span int
}

// pixel is a frame pixel: its color, and for a light its strength, color and group.
type pixel struct {
	c     rgb
	glow  float64
	light rgb
	group int
}

type painter interface {
	inside() color.RGBA
	edge(e edge, x, y int) pixel
	// corner paints a corner piece's pixel at (x, y) within it, from its top left.
	corner(x, y int) pixel
}

// bevel shades c across the frame: a dark line outside and in, a lip toward the light.
func bevel(c rgb, e edge, outline rgb) rgb {
	switch e.u {
	case 0, Width - 1:
		return outline
	case 1:
		if e.side.lit() {
			return c.scale(1.35)
		}
		return c.scale(0.7)
	case Width - 2:
		if e.side.lit() {
			return c.scale(0.7)
		}
		return c.scale(1.3)
	}
	return c
}

// plate shades a corner piece as a raised square.
func plate(c rgb, x, y int, outline rgb) rgb {
	switch {
	case x == 0 || y == 0 || x == Corner-1 || y == Corner-1:
		return outline
	case x == 1 || y == 1:
		return c.scale(1.35)
	case x == Corner-2 || y == Corner-2:
		return c.scale(0.65)
	}
	return c
}

// iron is the hotbar's and the inventory panel's frame.
type iron struct{}

var (
	ironRamp    = ramp("#14171e", "#1c222b", "#252c37", "#2f3846", "#3a4556", "#4a576a")
	ironOutline = hexRGB("#05070a")
)

func (iron) inside() color.RGBA { return color.RGBA{11, 14, 20, 240} }

func (iron) edge(e edge, x, y int) pixel {
	c := pick(ironRamp, fbm(float64(e.t)*0.05, float64(e.u)*0.9+float64(e.side)*9, 9, 3)*1.15-0.05)
	switch e.u {
	case 2:
		c = c.scale(0.8)
	case 5:
		c = c.scale(1.12)
	case 3, 4:
		// Rivets, two pixels square, spaced evenly between the corners.
		if n := (e.span - 2*Corner) / 12; n > 0 {
			step := float64(e.span-2*Corner) / float64(n)
			k := math.Floor((float64(e.t-Corner) + 0.5) / step)
			mid := int(float64(Corner) + step*(k+0.5))
			if dt := e.t - mid; dt == -1 || dt == 0 {
				switch {
				case dt == -1 && e.u == 3:
					c = hexRGB("#a7b3c3")
				case dt == 0 && e.u == 4:
					c = hexRGB("#0b0d11")
				default:
					c = hexRGB("#5f6c7d")
				}
				return pixel{c: c}
			}
		}
	}
	return pixel{c: bevel(c, e, ironOutline)}
}

func (iron) corner(x, y int) pixel {
	const mid = (Corner - 1) / 2.0
	m := math.Abs(float64(x)-mid) + math.Abs(float64(y)-mid)
	if m <= 5 {
		var c string
		switch left, up := float64(x) < mid, float64(y) < mid; {
		case m > 4:
			c = "#06102a"
		case x == 4 && y == 3:
			c = "#ffffff"
		case m < 1.5:
			c = "#4f8ff0"
		case up && left:
			c = "#8fc2ff"
		case up:
			c = "#3f7fe2"
		case left:
			c = "#2a60c6"
		default:
			c = "#123a8c"
		}
		return pixel{c: hexRGB(c)}
	}
	return pixel{c: plate(hexRGB("#2b3340"), x, y, ironOutline)}
}

// runic is the spell panel's frame.
type runic struct{}

var (
	slateRamp    = ramp("#0f131a", "#161b24", "#1d242f", "#252e3b", "#2e3948")
	runeDim      = hexRGB("#2a64aa")
	runeBright   = hexRGB("#9adcff")
	runeHalo     = hexRGB("#1f4a78")
	slateOutline = hexRGB("#05070a")
)

// runes are the glyphs cut in the slate, three pixels wide and four high.
var runes = [][4]string{
	{"#.#", ".#.", ".#.", ".#."},
	{"#..", "##.", "#.#", "#.."},
	{".#.", "###", ".#.", ".#."},
	{"#.#", "###", "#.#", "#.#"},
	{".#.", "#.#", "#.#", ".#."},
	{"##.", "#.#", "##.", "#.#"},
	{"#..", "#..", "###", "..#"},
	{"#.#", "#.#", ".#.", "#.#"},
	{"###", ".#.", ".#.", ".#."},
}

// runeStep is the distance from one rune to the next along an edge.
const runeStep = 10

func (runic) inside() color.RGBA { return color.RGBA{10, 13, 19, 240} }

func (runic) edge(e edge, x, y int) pixel {
	fx, fy := float64(x), float64(y)
	c := pick(slateRamp, fbm(fx*0.06+fy*0.02, fy*0.35-fx*0.05, 4, 2)*1.2-0.1)
	inner := e.span - 2*Corner
	if n := inner / runeStep; n > 0 && e.u >= 1 && e.u <= 6 {
		// The runes are centered between the corners.
		t := e.t - Corner - (inner-n*runeStep)/2
		k := floorDiv(t, runeStep)
		if k >= 0 && k < n {
			g := runes[int(hash(k, int(e.side), 31)*float64(len(runes)))]
			gx, gy := t-k*runeStep-2, e.u-2
			on := func(a, b int) bool { return a >= 0 && a < 3 && b >= 0 && b < 4 && g[b][a] == '#' }
			group := (k + int(e.side)) % GlowGroups
			switch {
			case on(gx, gy):
				return pixel{c: runeDim, glow: 1, light: runeBright, group: group}
			case on(gx-1, gy) || on(gx+1, gy) || on(gx, gy-1) || on(gx, gy+1):
				c = mix(c, runeHalo, 0.35)
				return pixel{c: bevel(c, e, slateOutline), glow: 0.35, light: runeHalo, group: group}
			}
		}
	}
	return pixel{c: bevel(c, e, slateOutline)}
}

func (runic) corner(x, y int) pixel {
	const mid = (Corner - 1) / 2.0
	r := math.Hypot(float64(x)-mid, float64(y)-mid)
	switch {
	case r < 1:
		return pixel{c: runeDim, glow: 1, light: hexRGB("#c8ecff"), group: 0}
	case r >= 3.4 && r < 4.4:
		return pixel{c: runeDim, glow: 1, light: runeBright, group: 0}
	case r >= 2.7 && r < 3.4 || r >= 4.4 && r < 5.1:
		return pixel{c: mix(hexRGB("#141a24"), runeHalo, 0.4), glow: 0.4, light: runeHalo, group: 0}
	}
	return pixel{c: plate(hexRGB("#1b212b"), x, y, slateOutline)}
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

// rgb is a color with channels 0..255, unclamped while it is worked.
type rgb [3]float64

func hexRGB(s string) rgb {
	var c rgb
	for i := range c {
		var v int
		for _, ch := range s[1+2*i : 3+2*i] {
			v *= 16
			switch {
			case ch >= '0' && ch <= '9':
				v += int(ch - '0')
			case ch >= 'a' && ch <= 'f':
				v += int(ch-'a') + 10
			}
		}
		c[i] = float64(v)
	}
	return c
}

func ramp(hexes ...string) []rgb {
	r := make([]rgb, len(hexes))
	for i, h := range hexes {
		r[i] = hexRGB(h)
	}
	return r
}

// pick is the step of ramp r at t, 0..1, clamped: the ramp is banded as an 8-bit palette is.
func pick(r []rgb, t float64) rgb {
	return r[max(0, min(len(r)-1, int(math.Floor(t*float64(len(r))))))]
}

func (c rgb) scale(k float64) rgb { return rgb{c[0] * k, c[1] * k, c[2] * k} }

func mix(a, b rgb, t float64) rgb {
	return rgb{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, a[2] + (b[2]-a[2])*t}
}

func clamp(v float64) float64 { return max(0, min(255, v)) }

func (c rgb) rgba() color.RGBA {
	return color.RGBA{uint8(math.Round(clamp(c[0]))), uint8(math.Round(clamp(c[1]))), uint8(math.Round(clamp(c[2]))), 255}
}

// hash is a number 0..1 from x, y and seed s, the same each time.
func hash(x, y, s int) float64 {
	h := int32(x)*374761393 + int32(y)*668265263 + int32(s)*982451653
	u := uint32(h)
	u = (u ^ u>>13) * 1274126177
	u ^= u >> 16
	return float64(u) / 4294967296
}

// noise is smooth value noise.
func noise(x, y float64, s int) float64 {
	xi, yi := math.Floor(x), math.Floor(y)
	xf, yf := x-xi, y-yi
	u, v := xf*xf*(3-2*xf), yf*yf*(3-2*yf)
	ix, iy := int(xi), int(yi)
	a, b := hash(ix, iy, s), hash(ix+1, iy, s)
	c, d := hash(ix, iy+1, s), hash(ix+1, iy+1, s)
	return a + (b-a)*u + (c-a)*v + (a-b-c+d)*u*v
}

// fbm is octaves of noise, each twice as fine and half as strong, 0..1.
func fbm(x, y float64, s, octaves int) float64 {
	t, a, f, n := 0.0, 0.5, 1.0, 0.0
	for i := range octaves {
		t += a * noise(x*f, y*f, s+i*17)
		n += a
		a *= 0.5
		f *= 2
	}
	return t / n
}
