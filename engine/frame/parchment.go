package frame

import (
	"image"
	"image/color"
	"math"
)

// The map's parchment and the rolls it unrolls from (docs/spec/world.md, The engine's map),
// made from noise like the frames.

var (
	parchLight = hexRGB("#ead7a4")
	parchMid   = hexRGB("#d2b47a")
	parchDark  = hexRGB("#a5814e")
	parchStain = hexRGB("#8a6438")
	parchSinge = hexRGB("#4a2c14")
)

// Parchment draws a sheet of parchment w × h: tan, mottled lighter and darker, with a few faint
// stains, darkening toward its edges. Ragged, its top and bottom edges are torn and singed and
// clear beyond the tear; otherwise it fills the whole picture.
func Parchment(w, h int, ragged bool) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	const seed = 41
	for x := range w {
		fx := float64(x)
		// How far the tear comes in at this column, from the top and from the bottom.
		top, bottom := 0.0, 0.0
		if ragged {
			top = 2 + 7*fbm(fx/11, 0.5, seed+1, 4)
			bottom = 2 + 7*fbm(fx/11, 9.5, seed+2, 4)
		}
		for y := range h {
			fy := float64(y)
			in := min(fy-top, float64(h-1)-fy-bottom)
			if in < 0 {
				continue
			}
			m := fbm(fx/48, fy/48, seed, 5)
			c := mix(parchDark, parchLight, smooth(0.25, 0.75, m))
			c = mix(c, parchMid, 0.35)
			// Stains: a few large soft patches.
			if st := fbm(fx/150, fy/150, seed+3, 3); st > 0.6 {
				c = mix(c, parchStain, min(0.35, (st-0.6)*1.6))
			}
			// Fibres: a fine grain across the sheet.
			c = c.scale(0.97 + 0.06*noise(fx/1.5, fy/6, seed+4))
			// Darkening toward the edges, all four.
			edge := min(fx, float64(w-1)-fx, in)
			c = c.scale(0.72 + 0.28*smooth(0, 40, edge))
			// The singed lip of a tear.
			if ragged && in < 2.5 {
				c = mix(c, parchSinge, 1-in/2.5)
			}
			img.SetRGBA(x, y, c.rgba())
		}
	}
	return img
}

// Roll draws a rolled end of parchment w across and h long, lying upright: a cylinder of
// parchment lit from the left, with the spiral of the roll showing at both ends.
func Roll(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	capH := max(2, w/3)
	const seed = 57
	for y := range h {
		for x := range w {
			t := (float64(x)+0.5)/float64(w)*2 - 1
			fy := float64(y)
			// The ends are ellipses: inside the top one the spiral shows, outside the bottom one
			// is clear.
			ey := 0.0
			switch {
			case y < capH:
				ey = (float64(capH) - fy) / float64(capH)
			case y >= h-capH:
				ey = (fy - float64(h-capH)) / float64(capH)
			}
			r := math.Hypot(t, ey)
			if y >= h-capH && r > 1 {
				continue
			}
			if y < capH && r > 1 {
				// Above the top end's ellipse is clear.
				continue
			}
			var c rgb
			if y < capH {
				// The spiral: rings of the sheet's edge, darker between.
				a := math.Atan2(ey, t)
				ring := 0.5 + 0.5*math.Sin(r*math.Pi*7+a)
				c = mix(parchSinge, parchMid, 0.45+0.45*ring)
				c = c.scale(0.8 + 0.2*(1-r))
			} else {
				n := math.Sqrt(max(0, 1-t*t))
				light := 0.22 + 0.95*max(0, n*0.9-t*0.35)
				c = mix(parchDark, parchLight, fbm(float64(x)/3, fy/40, seed, 3)).scale(light)
				// The layers of the roll, faint lines along it.
				if math.Mod(math.Abs(t)*5+fbm(fy/30, 0, seed+1, 2), 1) < 0.12 {
					c = c.scale(0.85)
				}
			}
			img.SetRGBA(x, y, c.rgba())
		}
	}
	return img
}

// smooth is 0 at a, 1 at b and eases between them.
func smooth(a, b, v float64) float64 {
	t := max(0, min(1, (v-a)/(b-a)))
	return t * t * (3 - 2*t)
}

// Ink is the color of the map's lines and lettering on parchment.
var Ink = color.RGBA{58, 34, 16, 255}
