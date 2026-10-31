package devpack

import (
	"math"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Floor variants.
const (
	floorPlain = iota
	floorFlags
	floorWater
	floorLava
)

// floorTile draws a floor variant as a 64×32 diamond with transparent corners: the floor set is a
// diamond set (docs/pack.md, Floor sets), so each cell shows its own floor whatever its
// neighbors are.
func floorTile(variant int) *canvas {
	c := newCanvas(64, 32, clear)
	var paint func(x, y int) byte
	switch variant {
	case floorPlain:
		paint = func(x, y int) byte { return byte(stone + 3 + noise(x, y, 2)) }
	case floorFlags:
		paint = func(x, y int) byte {
			if (x+2*y)%16 == 0 || (2*y-x+64)%16 == 0 {
				return stone + 1
			}
			return byte(stone + 2 + noise(x, y, 2))
		}
	case floorWater:
		paint = func(x, y int) byte { return byte(water + (x+2*y)/6%4) }
	case floorLava:
		paint = func(x, y int) byte { return byte(lava + (x/3+y/2+noise(x/4, y/4, 2))%4) }
	}
	in := diamond(32, 16, 64)
	c.fill(in, paint)
	c.outline(in, stone+1)
	return c
}

// Blocks are 64×56: a cell's diamond at the bottom, (32, 40) its center, and a box rising from it.
// They are drawn at blockOffset from the tile box.
const blockW, blockH = 64, 56

var blockOffset = content.Offset{-16, -24}

// blockFace says which side of a block a face is on.
const (
	faceNone = iota
	faceLeft
	faceRight
)

// lowerEdge is the y of the lower edge, at x, of a cell diamond centered at height cy.
func lowerEdge(cy, x float64) float64 { return cy + 16*(1-math.Abs(x-32)/32) }

// block draws a box h pixels tall on a cell's diamond: top from ramp top, sides from ramp side
// (left darker), with courses of brick unless planks, a recessed panel on face, a doorway on
// door, and a hatched top if hatched.
type blockArt struct {
	h             float64
	top, side     byte
	planks        bool
	panel, door   int
	hatched, rune bool
}

func (b blockArt) draw() *canvas {
	c := newCanvas(blockW, blockH, clear)
	topY := 40 - b.h
	top := diamond(32, topY, 64)
	sides := func(x, y float64) bool { return y >= lowerEdge(topY, x) && y < lowerEdge(40, x) && x >= 0 && x < 64 }
	all := func(x, y float64) bool { return top(x, y) || sides(x, y) }
	c.fill(sides, func(x, y int) byte {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		rel := int(lowerEdge(40, fx) - fy)
		shade := b.side + 3
		if x < 32 {
			shade = b.side + 1
		}
		if b.planks {
			if x%8 == 0 {
				return shade - 1
			}
			return shade + byte(noise(x, 0, 2))
		}
		row := rel / 6
		if rel%6 == 0 || (x+row*8)%16 == 0 {
			return shade - 1
		}
		return shade
	})
	c.fill(top, func(x, y int) byte {
		if b.hatched && (x+2*y)%6 == 0 {
			return b.top + 2
		}
		return byte(int(b.top) + 4 + noise(x, y, 2))
	})
	for _, f := range []struct {
		side int
		cx   float64
	}{{faceLeft, 16}, {faceRight, 48}} {
		region := func(x, y float64, half, lo, hi float64) bool {
			rel := lowerEdge(40, x) - y
			return math.Abs(x-f.cx) < half && rel >= lo && rel < hi
		}
		if b.panel == f.side {
			c.fill(func(x, y float64) bool { return region(x, y, 9, 4, b.h-4) }, solid(b.side+5))
		}
		if b.door == f.side {
			c.fill(func(x, y float64) bool { return region(x, y, 7, 0, b.h-6) }, solid(gold+5))
			c.fill(func(x, y float64) bool { return region(x, y, 5, 0, b.h-8) }, solid(black))
		}
	}
	if b.rune {
		c.outline(ellipse(32, topY, 10, 5), gold+6)
		c.line(24, int(topY), 40, int(topY), gold+6)
	}
	c.outline(all, gray)
	c.outline(top, gray+2)
	return c
}

// Wall set shapes, and the frames of the walls bank.
const (
	shapeBlock     = 0 // a plain wall
	shapeFaceLeft  = 1 // a wall whose left face can carry an exit
	shapeDiagonal  = 2 // a wall marked diagonal
	shapeFaceRight = 3 // a wall whose right face can carry an exit
	frameExitLeft  = 4
	frameExitRight = 5
)

func walls() []*canvas {
	wall := blockArt{h: 24, top: gray + 6, side: gray + 4}
	face := func(side int) blockArt { w := wall; w.panel = side; return w }
	diag := wall
	diag.hatched = true
	exit := func(side int) blockArt { w := wall; w.door = side; return w }
	return []*canvas{
		wall.draw(), face(faceLeft).draw(), diag.draw(), face(faceRight).draw(),
		exit(faceLeft).draw(), exit(faceRight).draw(),
	}
}

// doors are a door's stages, closed (0) to nearly open (3), then sealed.
func doors() []*canvas {
	var out []*canvas
	for s := range 4 {
		out = append(out, blockArt{h: float64(24 - 6*s), top: wood + 1, side: wood + 1, planks: true}.draw())
	}
	return append(out, blockArt{h: 24, top: purple, side: purple, rune: true}.draw())
}

// Feature frames are 64×48, drawn at featureOffset: the cell's diamond centered at (32, 32).
const featureW, featureH = 64, 48

var featureOffset = content.Offset{-16, -16}

// Frames of the features bank.
const (
	framePlateUp = iota
	framePlateDown
	frameSlot
	frameTeleArmed
	frameTeleDisarmed
	framePit
	frameTrapOpen
	frameTrapClosed
	frameSocketEmpty
	frameSocketFull
	frameSocketSet // .. frameSocketSet+5
	frameMouth     = frameSocketSet + socketPlaces
	frameMark      = frameMouth + 1
	frameExitFloor = frameMark + 1
	frameLeverOff  = frameExitFloor + 1
	frameLeverOn   = frameLeverOff + 1
)

const socketPlaces = 6

func socketDot(i int) (float64, float64) {
	a := float64(i) * math.Pi / 3
	return 32 + 13*math.Cos(a), 32 + 6.5*math.Sin(a)
}

func features() []*canvas {
	frame := func(draw func(c *canvas)) *canvas {
		c := newCanvas(featureW, featureH, clear)
		draw(c)
		return c
	}
	var out []*canvas
	out = append(out,
		frame(func(c *canvas) { // plate up
			c.fill(diamond(32, 34, 40), solid(stone))
			c.fill(diamond(32, 31, 40), func(x, y int) byte { return byte(stone + 5 + noise(x, y, 2)) })
			c.outline(diamond(32, 31, 40), stone+7)
		}),
		frame(func(c *canvas) { // plate down
			c.fill(diamond(32, 32, 40), solid(stone+2))
			c.outline(diamond(32, 32, 40), stone)
		}),
		frame(func(c *canvas) { // slot
			c.fill(diamond(32, 32, 28), solid(gray+9))
			c.outline(diamond(32, 32, 28), gray+4)
			c.fill(ellipse(32, 30, 2.5, 2.5), solid(black))
			c.rect(31, 30, 33, 35, black)
		}),
		frame(func(c *canvas) { // teleporter armed
			c.fill(ellipse(32, 32, 16, 8), solid(blue+1))
			c.fill(and(ellipse(32, 32, 22, 11), not(ellipse(32, 32, 16, 8))), func(x, y int) byte { return byte(glow + (x+y)/3%4) })
		}),
		frame(func(c *canvas) { // teleporter disarmed
			c.fill(ellipse(32, 32, 16, 8), solid(gray+2))
			c.fill(and(ellipse(32, 32, 22, 11), not(ellipse(32, 32, 16, 8))), solid(gray+6))
		}),
		frame(func(c *canvas) { // pit
			c.fill(diamond(32, 32, 56), solid(black))
			c.outline(diamond(32, 32, 56), gray+3)
		}),
		frame(func(c *canvas) { // trapdoor open
			c.fill(diamond(32, 32, 52), solid(black))
			c.outline(diamond(32, 32, 52), wood+3)
		}),
		frame(func(c *canvas) { // trapdoor closed
			c.fill(diamond(32, 32, 52), func(x, y int) byte {
				if (x-2*y+64)%8 == 0 {
					return wood + 1
				}
				return wood + 4
			})
			c.outline(diamond(32, 32, 52), wood)
		}),
	)
	disc := func(c *canvas) {
		c.fill(ellipse(32, 32, 20, 10), solid(stone+6))
		c.outline(ellipse(32, 32, 20, 10), stone+1)
	}
	gem := func(c *canvas, i int, v byte) {
		x, y := socketDot(i)
		c.fill(ellipse(x, y, 2.5, 2.5), solid(v))
	}
	gemColor := func(i int) byte { return []byte{red + 6, blue + 6, green + 6, gold + 6, purple + 6, gray + 15}[i] }
	out = append(out,
		frame(func(c *canvas) { // socket empty
			disc(c)
			for i := range socketPlaces {
				gem(c, i, black)
			}
		}),
		frame(func(c *canvas) { // socket full
			disc(c)
			c.outline(ellipse(32, 32, 22, 11), glow+2)
			for i := range socketPlaces {
				gem(c, i, gemColor(i))
			}
		}),
	)
	for i := range socketPlaces {
		out = append(out, frame(func(c *canvas) { gem(c, i, gemColor(i)) }))
	}
	out = append(out,
		frame(func(c *canvas) { // mouth
			c.fill(ellipse(32, 32, 11, 5), solid(red+3))
			c.outline(ellipse(32, 32, 11, 5), red)
			c.line(22, 32, 42, 32, black)
		}),
		frame(func(c *canvas) { // mark
			c.fill(and(ellipse(32, 32, 15, 7.5), not(ellipse(32, 32, 12, 6))), solid(purple+5))
			c.line(24, 28, 40, 36, purple+5)
			c.line(24, 36, 40, 28, purple+5)
		}),
		frame(func(c *canvas) { // exit in the floor: stairs down
			c.fill(diamond(32, 32, 52), solid(black))
			for k := range 4 {
				c.outline(diamond(32, float64(29+2*k), float64(44-9*k)), byte(stone+7-k))
			}
			c.outline(diamond(32, 32, 52), gold+5)
		}),
	)
	lever := func(on bool) *canvas {
		return frame(func(c *canvas) {
			c.fill(diamond(32, 32, 16), solid(gray+3))
			tip, knob := 26, byte(red+5)
			if on {
				tip, knob = 38, green+5
			}
			c.line(32, 31, tip, 14, wood+5)
			c.line(33, 31, tip+1, 14, wood+3)
			c.fill(ellipse(float64(tip)+0.5, 14, 2.5, 2.5), solid(knob))
		})
	}
	return append(out, lever(false), lever(true))
}

// Items, in the order of the items banks.
var itemArt = []struct {
	name string
	draw func(c *canvas, s float64)
}{
	{"key", func(c *canvas, s float64) {
		c.fill(and(ellipse(.3*s, .3*s, .22*s, .22*s), not(ellipse(.3*s, .3*s, .1*s, .1*s))), solid(gold+5))
		c.line(int(.42*s), int(.42*s), int(.9*s), int(.9*s), gold+5)
		c.line(int(.7*s), int(.7*s), int(.6*s), int(.85*s), gold+4)
	}},
	{"rock", func(c *canvas, s float64) {
		c.fill(ellipse(.5*s, .58*s, .42*s, .32*s), func(x, y int) byte { return byte(gray + 6 + noise(x, y, 3)) })
		c.outline(ellipse(.5*s, .58*s, .42*s, .32*s), gray+3)
	}},
	{"red_gem", func(c *canvas, s float64) {
		c.fill(diamond(.5*s, .5*s, .8*s), solid(red+5))
		c.outline(diamond(.5*s, .5*s, .8*s), red+2)
	}},
	{"blue_gem", func(c *canvas, s float64) {
		c.fill(diamond(.5*s, .5*s, .8*s), solid(blue+5))
		c.outline(diamond(.5*s, .5*s, .8*s), blue+2)
	}},
	{"sphere", func(c *canvas, s float64) {
		c.fill(ellipse(.5*s, .5*s, .38*s, .38*s), solid(purple+4))
		c.fill(ellipse(.38*s, .38*s, .1*s, .1*s), solid(purple+7))
	}},
	{"sword", func(c *canvas, s float64) {
		c.line(int(.15*s), int(.85*s), int(.85*s), int(.15*s), gray+13)
		c.line(int(.2*s), int(.6*s), int(.4*s), int(.8*s), wood+4)
	}},
	{"sack", func(c *canvas, s float64) {
		c.fill(ellipse(.5*s, .62*s, .36*s, .32*s), solid(tan+3))
		c.rect(int(.4*s), int(.18*s), int(.6*s), int(.34*s), tan+2)
	}},
	{"bag", func(c *canvas, s float64) {
		c.fill(ellipse(.5*s, .62*s, .36*s, .32*s), solid(purple+3))
		c.rect(int(.4*s), int(.18*s), int(.6*s), int(.34*s), gold+5)
	}},
	{"boots", func(c *canvas, s float64) {
		c.rect(int(.25*s), int(.2*s), int(.45*s), int(.75*s), wood+2)
		c.rect(int(.25*s), int(.65*s), int(.6*s), int(.8*s), wood+1)
		c.rect(int(.55*s), int(.2*s), int(.75*s), int(.75*s), wood+3)
		c.rect(int(.55*s), int(.65*s), int(.9*s), int(.8*s), wood+2)
	}},
	{"mail", func(c *canvas, s float64) {
		c.fill(ellipse(.5*s, .55*s, .32*s, .36*s), func(x, y int) byte { return byte(gray + 8 + (x+y)%3) })
		c.rect(int(.4*s), int(.15*s), int(.6*s), int(.25*s), black)
	}},
	{"potion", func(c *canvas, s float64) {
		c.fill(ellipse(.5*s, .62*s, .3*s, .3*s), solid(red+4))
		c.rect(int(.42*s), int(.12*s), int(.58*s), int(.36*s), gray+10)
	}},
}

func items(size int) []*canvas {
	out := make([]*canvas, len(itemArt))
	for i, it := range itemArt {
		out[i] = newCanvas(size, size, clear)
		it.draw(out[i], float64(size))
	}
	return out
}

// directions are the screen directions of the eight facings (docs/pack.md, Conventions).
var directions = func() [8][2]float64 {
	du := []float64{-1, -1, 0, 1, 1, 1, 0, -1}
	dv := []float64{0, 1, 1, 1, 0, -1, -1, -1}
	var out [8][2]float64
	for d := range 8 {
		x, y := dv[d]-du[d], (du[d]+dv[d])/2
		n := math.Hypot(x, y)
		out[d] = [2]float64{x / n, y / n}
	}
	return out
}()

// Figures are 24×40, standing on (12, 36), with an arrow on the ground showing their facing.
const figureW, figureH = 24, 40

var figureAnchor = content.Offset{12, 36}

// figureCorpse is the frame of a figure lying dead.
const figureCorpse = 3 * 8

// figure draws a figure facing each direction, standing (frames 0–7) and in two strides
// (frames 8–15 and 16–23), one leg forward in each; and lying dead (frame 24).
func figure(body, skin byte) []*canvas {
	out := make([]*canvas, 3*8, 3*8+1)
	for i := range out {
		d, pose := i%8, i/8
		c := newCanvas(figureW, figureH, clear)
		c.fill(ellipse(12, 36, 9, 3.5), solid(gray+2))
		dir := directions[d]
		c.line(12, 36, 12+int(math.Round(dir[0]*9)), 36+int(math.Round(dir[1]*4)), gold+6)
		left, right := 0, 0
		switch pose {
		case 1:
			left, right = -2, 1
		case 2:
			left, right = 1, -2
		}
		c.rect(8+left, 26, 11+left, 36, body)
		c.rect(13+right, 26, 16+right, 36, body)
		c.fill(ellipse(12, 21, 6, 8), solid(body+3))
		c.outline(ellipse(12, 21, 6, 8), body)
		c.fill(ellipse(12, 9, 4.5, 4.5), solid(skin))
		c.outline(ellipse(12, 9, 4.5, 4.5), skin-3)
		// Eyes look the way the figure faces, when it faces the viewer.
		if dir[1] > 0 {
			ex := 12 + int(math.Round(dir[0]*2))
			c.set(ex-2, 9, black)
			c.set(ex+1, 9, black)
		}
		out[i] = c
	}
	dead := newCanvas(figureW, figureH, clear)
	dead.fill(ellipse(12, 34, 11, 4), solid(body))
	dead.fill(ellipse(4, 33, 3.5, 3), solid(skin))
	return append(out, dead)
}

// boulderFrames is how many frames a boulder's roll turns through.
const boulderFrames = 4

// boulders draws a boulder turning: a spot on it moves round a quarter turn each frame.
func boulders() []*canvas {
	var out []*canvas
	for f := range boulderFrames {
		c := newCanvas(32, 32, clear)
		ball := ellipse(16, 17, 13, 12)
		c.fill(ball, func(x, y int) byte {
			shade := 7 - (x+y)/10
			return byte(gray + max(2, shade+noise(x/3, y/3, 2)))
		})
		c.fill(ellipse(11, 11, 3, 2.5), solid(gray+10))
		a := float64(f) * math.Pi / 2
		c.fill(ellipse(16+7*math.Cos(a), 17+6*math.Sin(a), 2.5, 2), solid(gray+1))
		c.outline(ball, gray+1)
		out = append(out, c)
	}
	return out
}

// crateW and crateH are the size of a crate's frame; its ground point is at (20, 30), the middle
// of its base, a diamond as wide as its 20-unit footprint.
const crateW, crateH = 40, 42

// crate draws a wooden crate, a block a player can push.
func crate() *canvas {
	c := newCanvas(crateW, crateH, clear)
	const cx, base, h = 20, 30, 14
	edge := func(cy, x float64) float64 { return cy + 10*(1-math.Abs(x-cx)/20) }
	top := diamond(cx, base-h, 40)
	sides := func(x, y float64) bool { return x >= 0 && x < 40 && y >= edge(base-h, x) && y < edge(base, x) }
	c.fill(sides, func(x, y int) byte {
		shade := byte(wood + 3)
		if x < cx {
			shade = wood + 1
		}
		if y%5 == 0 {
			shade--
		}
		return shade
	})
	c.fill(top, solid(wood+5))
	c.outline(top, wood)
	c.line(cx, base-h+10, cx, base+10, wood)
	return c
}
