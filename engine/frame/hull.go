package frame

import (
	"image/color"
	"math"
)

// hull is riveted hull plate, warm grey with hazard-worn seams, and amber indicator lights that
// blink in groups: a frame for science fiction without art of its own.
type hull struct{}

var (
	hullRamp    = ramp("#1a1c1f", "#24272b", "#2e3237", "#393e44", "#454b52", "#545b63")
	hullOutline = hexRGB("#060708")
	amberDim    = hexRGB("#5a3a0c")
	amberBright = hexRGB("#ffb347")
	amberHalo   = hexRGB("#7a4a10")
)

// hullStep is the distance from one plate seam to the next along an edge; a light sits
// mid-plate on every other plate.
const hullStep = 28

func (hull) inside() color.RGBA { return color.RGBA{12, 13, 15, 240} }

func (hull) edge(e edge, x, y int) pixel {
	// Brushed along the edge, a little mottled across it.
	c := pick(hullRamp, fbm(float64(e.t)*0.03, float64(e.u)*0.7+float64(e.side)*7, 5, 3)*1.1)
	inner := e.span - 2*Corner
	t := e.t - Corner
	if inner > 0 && t >= 0 && t < inner && e.u >= 1 && e.u <= Width-2 {
		k := t / hullStep
		at := t - k*hullStep
		switch {
		case at == 0 && e.u >= 2 && e.u <= Width-3:
			// A seam between plates.
			return pixel{c: hullOutline.scale(2)}
		case (at == 3 || at == hullStep-3) && (e.u == 3 || e.u == Width-4):
			// Rivets at each plate's ends.
			if e.u == 3 {
				return pixel{c: hexRGB("#8a9099")}
			}
			return pixel{c: hexRGB("#0c0d0f")}
		case k%2 == 1 && abs(at-hullStep/2) <= 1 && (e.u == 3 || e.u == 4):
			// An indicator light, two pixels square with a dim rim.
			group := (k/2 + int(e.side)) % GlowGroups
			if at-hullStep/2 >= 0 {
				return pixel{c: amberDim, glow: 1, light: amberBright, group: group}
			}
			return pixel{c: mix(c, amberHalo, 0.5), glow: 0.4, light: amberHalo, group: group}
		}
	}
	return pixel{c: bevel(c, e, hullOutline)}
}

func (hull) corner(x, y int) pixel {
	const mid = (Corner - 1) / 2.0
	// A bolted square with a hazard stripe across it.
	if math.Abs(float64(x)-mid) <= 3 && math.Abs(float64(y)-mid) <= 3 {
		if (x+y)%4 < 2 {
			return pixel{c: hexRGB("#c08a1e")}
		}
		return pixel{c: hexRGB("#17181a")}
	}
	return pixel{c: plate(hexRGB("#3a3f45"), x, y, hullOutline)}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
