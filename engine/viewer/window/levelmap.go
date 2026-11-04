package window

import (
	"image"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/font"
	"github.com/sirgwain/the-summoning-go/engine/frame"
	"github.com/sirgwain/the-summoning-go/engine/sim"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// The level map drawn as ink (docs/spec/world.md, The engine's map): the minimap straight over
// the view, and the full map on parchment unrolling over the view.

// Ink colors of the ground.
var (
	floorWash   = color.NRGBA{246, 234, 200, 90}
	floorEdge   = color.NRGBA{110, 76, 40, 70}
	waterWash   = color.NRGBA{40, 110, 200, 140}
	waterEdge   = color.NRGBA{200, 230, 250, 200}
	wallTop     = color.RGBA{98, 64, 36, 255}
	wallLeft    = color.RGBA{72, 46, 26, 255}
	wallRight   = color.RGBA{52, 33, 18, 255}
	wallLit     = color.RGBA{160, 120, 78, 255}
	wallShade   = color.RGBA{30, 18, 9, 255}
	doorColor   = color.RGBA{120, 58, 22, 255}
	symbolEdge  = color.RGBA{30, 18, 9, 230}
	knownColor  = color.RGBA{255, 250, 210, 255}
	noteColor   = color.RGBA{38, 24, 12, 225}
	noteEdge    = color.RGBA{170, 128, 70, 255}
	noteText    = color.RGBA{242, 228, 192, 255}
	focusColor  = color.RGBA{255, 236, 150, 255}
	shadowColor = color.RGBA{0, 0, 0, 150}
)

// symbolColors are the colors of the symbols, the legend's and their glow's.
var symbolColors = map[int]color.RGBA{
	viewer.MapDoor: doorColor, viewer.MapWater: {60, 120, 200, 255},
	viewer.MapExit: {250, 200, 70, 255}, viewer.MapTeleporter: {70, 130, 255, 255},
	viewer.MapHole: {225, 80, 40, 255}, viewer.MapTrapdoor: {190, 120, 50, 255},
	viewer.MapPlate: {245, 242, 230, 255}, viewer.MapMark: {255, 110, 50, 255},
	viewer.MapSocket: {220, 90, 230, 255}, viewer.MapMouth: {250, 250, 250, 255},
	viewer.MapThurisaz: {80, 225, 80, 255}, viewer.MapGebo: {255, 170, 30, 255}, viewer.MapRaido: {240, 45, 40, 255},
	viewer.MapNPC: {40, 215, 195, 255}, viewer.MapLever: {245, 242, 230, 255},
}

// mapPlayerColor is the player's marker.
var mapPlayerColor = color.RGBA{225, 35, 30, 255}

// linkColors are the colors of the links' arrows, by the feature's kind.
var linkColors = map[string]color.RGBA{
	content.FeatureTeleporter: {80, 140, 255, 255},
	content.FeatureExit:       {245, 185, 50, 255},
	content.FeaturePit:        {235, 80, 50, 255},
	content.FeatureTrapdoor:   {235, 80, 50, 255},
}

// Timing of the full map's unrolling, and its rolls' size.
const (
	unrollTime = 400 * time.Millisecond
	rollUpTime = 200 * time.Millisecond
	// parchmentScale is how many screen pixels a pixel of the parchment takes.
	parchmentScale = 2
)

// mapPics are the map's pictures and its state on the screen, kept between draws.
type mapPics struct {
	// glow is a soft round spot, white, for the symbols' halos.
	glow *ebiten.Image
	// sheet is the full map's parchment and roll its rolled end, at their sizes; miniMask the
	// minimap's edge fade.
	sheet, roll, miniMask *ebiten.Image
	// page is the full map as last drawn, and its place on the screen, kept so it can roll up
	// once it has closed.
	page     *ebiten.Image
	pageRect image.Rectangle
	// open is whether the full map was open at the last draw; since when it opened or closed.
	open    bool
	closing bool
	since   time.Time
	// cells is the minimap's own picture, clipped to its place.
	cells *ebiten.Image
}

// glowImage returns the round spot the halos are drawn with.
func (s *Window) glowImage() *ebiten.Image {
	if s.maps.glow == nil {
		const n = 64
		img := image.NewRGBA(image.Rect(0, 0, n, n))
		for y := range n {
			for x := range n {
				d := math.Hypot(float64(x)+0.5-n/2, float64(y)+0.5-n/2) / (n / 2)
				a := math.Max(0, 1-d)
				a = a * a
				k := uint8(255 * a)
				img.SetRGBA(x, y, color.RGBA{k, k, k, k})
			}
		}
		s.maps.glow = ebiten.NewImageFromImage(img)
	}
	return s.maps.glow
}

// sized returns *img if it is size, else a new image of that size made by draw.
func sized(img **ebiten.Image, size image.Point, draw func() image.Image) *ebiten.Image {
	if *img != nil && (*img).Bounds().Size() == size {
		return *img
	}
	if *img != nil {
		(*img).Deallocate()
	}
	*img = ebiten.NewImageFromImage(draw())
	return *img
}

// blank returns *img cleared if it is size, else a new empty image of that size.
func blank(img **ebiten.Image, size image.Point) *ebiten.Image {
	if *img != nil && (*img).Bounds().Size() == size {
		(*img).Clear()
		return *img
	}
	if *img != nil {
		(*img).Deallocate()
	}
	*img = ebiten.NewImage(size.X, size.Y)
	return *img
}

// mapDrawing draws a level map's cells onto dst through frame f, dst's top-left being off on
// the screen; t is the time in seconds, which the glow breathes and the player pulses by.
type mapDrawing struct {
	s   *Window
	dst *ebiten.Image
	m   *viewer.LevelMap
	f   viewer.MapFrame
	off image.Point
	t   float64
}

// at returns the point of the map p on dst.
func (d *mapDrawing) at(p [2]float64) (float32, float32) {
	x, y := d.f.Point(p)
	return float32(x) - float32(d.off.X), float32(y) - float32(d.off.Y)
}

// middle returns cell c's middle on dst, raised by lift units.
func (d *mapDrawing) middle(c content.Cell, lift float64) (float32, float32) {
	x, y := d.f.CellPoint(c)
	return float32(x) - float32(d.off.X), float32(y-lift*d.f.Zoom) - float32(d.off.Y)
}

// diamond adds cell c's diamond, raised by lift units, to path.
func (d *mapDrawing) diamond(path *vector.Path, c content.Cell, lift float64) {
	cx, cy := d.middle(c, lift)
	z := float32(d.f.Zoom)
	path.MoveTo(cx, cy-z/2)
	path.LineTo(cx+z, cy)
	path.LineTo(cx, cy+z/2)
	path.LineTo(cx-z, cy)
	path.Close()
}

// visible reports whether cell c shows on dst.
func (d *mapDrawing) visible(c content.Cell) bool {
	cx, cy := d.middle(c, 0)
	z := float32(d.f.Zoom) * 2
	b := d.dst.Bounds()
	return cx > float32(b.Min.X)-z && cy > float32(b.Min.Y)-z && cx < float32(b.Max.X)+z && cy < float32(b.Max.Y)+z
}

func fill(dst *ebiten.Image, path *vector.Path, c color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.FillPath(dst, path, nil, op)
}

func stroke(dst *ebiten.Image, path *vector.Path, width float32, c color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.StrokePath(dst, path, &vector.StrokeOptions{Width: width, LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound}, op)
}

// wallLift is how far a wall stands above the floor: a quarter of a cell's height, in units.
const wallLift = 0.25

// draw draws the map: the ground, the walls, the symbols with their glow, the known links' dots
// and the player.
func (d *mapDrawing) draw() {
	var floor, water, waterLine, sideL, sideR, tops, lit, shade, doors, doorFloor vector.Path
	z := float32(d.f.Zoom)
	var symbols []viewer.MapCell
	for _, mc := range d.m.Cells {
		if !d.visible(mc.Cell) {
			continue
		}
		c := mc.Cell
		switch mc.Kind {
		case viewer.MapWall:
			// The block's sides, from its foot to its raised top, then the top.
			// A run of walls is one block: its sides and edges show only where it meets
			// something that is not a wall, so no seams cross it.
			bx, by := d.middle(c, 0)
			tx, ty := d.middle(c, wallLift)
			if !d.wall(content.Cell{c[0] + 1, c[1]}) {
				sideL.MoveTo(bx-z, by)
				sideL.LineTo(bx, by+z/2)
				sideL.LineTo(tx, ty+z/2)
				sideL.LineTo(tx-z, ty)
				sideL.Close()
			}
			if !d.wall(content.Cell{c[0], c[1] + 1}) {
				sideR.MoveTo(bx, by+z/2)
				sideR.LineTo(bx+z, by)
				sideR.LineTo(tx+z, ty)
				sideR.LineTo(tx, ty+z/2)
				sideR.Close()
				shade.MoveTo(tx+z, ty)
				shade.LineTo(tx, ty+z/2)
			}
			d.diamond(&tops, c, wallLift)
			if !d.wall(content.Cell{c[0], c[1] - 1}) {
				lit.MoveTo(tx-z, ty)
				lit.LineTo(tx, ty-z/2)
			}
		case viewer.MapWater:
			d.diamond(&water, c, 0)
			// A lighter line where the water meets a floor that is not water.
			cx, cy := d.middle(c, 0)
			edges := [4]struct {
				n      content.Cell
				x0, y0 float32
				x1, y1 float32
			}{
				{content.Cell{c[0] - 1, c[1]}, cx - z, cy, cx, cy - z/2},
				{content.Cell{c[0], c[1] + 1}, cx, cy - z/2, cx + z, cy},
				{content.Cell{c[0] + 1, c[1]}, cx + z, cy, cx, cy + z/2},
				{content.Cell{c[0], c[1] - 1}, cx, cy + z/2, cx - z, cy},
			}
			for _, e := range edges {
				if n, ok := d.m.At(e.n); ok && n.Kind != viewer.MapWater && n.Kind != viewer.MapWall {
					waterLine.MoveTo(e.x0, e.y0)
					waterLine.LineTo(e.x1, e.y1)
				}
			}
		case viewer.MapDoor:
			d.diamond(&doorFloor, c, 0)
			d.doorBar(&doors, mc)
		default:
			d.diamond(&floor, c, 0)
		}
		if mc.Symbol != viewer.MapNone || mc.Known {
			symbols = append(symbols, mc)
		}
	}
	fill(d.dst, &floor, floorWash)
	fill(d.dst, &doorFloor, floorWash)
	stroke(d.dst, &floor, 1, floorEdge)
	stroke(d.dst, &doorFloor, 1, floorEdge)
	fill(d.dst, &water, waterWash)
	stroke(d.dst, &water, 1, color.NRGBA{40, 80, 130, 90})
	stroke(d.dst, &waterLine, 1.5, waterEdge)
	fill(d.dst, &sideL, wallLeft)
	fill(d.dst, &sideR, wallRight)
	fill(d.dst, &tops, wallTop)
	stroke(d.dst, &lit, max(1, z/8), wallLit)
	stroke(d.dst, &shade, max(1, z/8), wallShade)
	stroke(d.dst, &doors, max(2, z/3.5), color.RGBA{20, 10, 4, 255})
	stroke(d.dst, &doors, max(1.2, z/5), doorColor)
	// The halos first, under every symbol, breathing from two thirds to full.
	breath := float32(2.0/3 + (1.0/3)*(0.5+0.5*math.Sin(d.t*math.Pi)))
	glow := d.s.glowImage()
	for _, mc := range symbols {
		if mc.Symbol == viewer.MapNone {
			continue
		}
		d.halo(glow, mc, symbolColors[mc.Symbol], 3.4, breath)
	}
	for _, mc := range symbols {
		d.symbol(mc)
	}
	for _, t := range d.m.Talkers {
		if d.visible(t.Cell) {
			d.talker(glow, t.Cell, breath)
		}
	}
	for _, mc := range symbols {
		if !mc.Known {
			continue
		}
		cx, cy := d.middle(mc.Cell, d.lift(mc))
		x, y := cx+z*0.45, cy-z*0.35
		d.spot(glow, x, y, z*0.9, knownColor, 0.9*breath)
		vector.FillCircle(d.dst, x, y, max(1.5, z/7)+1, symbolEdge, true)
		vector.FillCircle(d.dst, x, y, max(1.5, z/7), knownColor, true)
	}
	if d.m.HasPlayer {
		d.player()
	}
}

// wall reports whether cell c is a seen wall.
func (d *mapDrawing) wall(c content.Cell) bool {
	mc, ok := d.m.At(c)
	return ok && mc.Kind == viewer.MapWall
}

// lift is how far cell mc's symbol is raised: onto its wall's top, for a wall.
func (d *mapDrawing) lift(mc viewer.MapCell) float64 {
	if mc.Kind == viewer.MapWall {
		return wallLift
	}
	return 0
}

// halo draws a soft spot of color c round cell mc's symbol, size times the cell's half-width
// across, at opacity a.
func (d *mapDrawing) halo(glow *ebiten.Image, mc viewer.MapCell, c color.RGBA, size, a float32) {
	cx, cy := d.middle(mc.Cell, d.lift(mc))
	d.spot(glow, cx, cy, float32(d.f.Zoom)*size, c, a)
}

// spot draws the glow image centered at (x, y), w across, tinted c at opacity a.
func (d *mapDrawing) spot(glow *ebiten.Image, x, y, w float32, c color.RGBA, a float32) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	k := float64(w) / float64(glow.Bounds().Dx())
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(float64(x-w/2), float64(y-w/2))
	op.ColorScale.ScaleWithColor(c)
	op.ColorScale.ScaleAlpha(a)
	d.dst.DrawImage(glow, op)
	// A brighter core, added, so the light seems to come from the symbol.
	op.Blend = ebiten.BlendLighter
	op.GeoM.Reset()
	op.GeoM.Scale(k/2, k/2)
	op.GeoM.Translate(float64(x-w/4), float64(y-w/4))
	op.ColorScale.ScaleAlpha(0.5)
	d.dst.DrawImage(glow, op)
}

// doorBar adds door mc's bar, along the line of its wall, to path: with a gap while open.
func (d *mapDrawing) doorBar(path *vector.Path, mc viewer.MapCell) {
	// Along the rows a step is (−1, ½) of the map; along the columns (1, ½).
	dir := [2]float64{-1, 0.5}
	if mc.Across {
		dir = [2]float64{1, 0.5}
	}
	m := [2]float64{float64(mc.Cell[1] - mc.Cell[0]), float64(mc.Cell[0]+mc.Cell[1])/2 + 0.5}
	seg := func(a, b float64) {
		x0, y0 := d.at([2]float64{m[0] + dir[0]*a, m[1] + dir[1]*a})
		x1, y1 := d.at([2]float64{m[0] + dir[0]*b, m[1] + dir[1]*b})
		path.MoveTo(x0, y0)
		path.LineTo(x1, y1)
	}
	if mc.Open {
		seg(-0.42, -0.22)
		seg(0.22, 0.42)
		return
	}
	seg(-0.42, 0.42)
}

// symbol draws cell mc's symbol, about half the cell's width, in ink outlined dark.
func (d *mapDrawing) symbol(mc viewer.MapCell) {
	if mc.Symbol == viewer.MapNone {
		return
	}
	cx, cy := d.middle(mc.Cell, d.lift(mc))
	r := float32(d.f.Zoom) / 2 // half the symbol's width
	c := symbolColors[mc.Symbol]
	edge := max(1, r/3)
	switch mc.Symbol {
	case viewer.MapExit:
		// Stairs: three stepped bars, rising to the right.
		for i := range 3 {
			fi := float32(i)
			x := cx - r + fi*r*2/3
			y := cy + r/2 - fi*r/2
			w, h := r*2-fi*r*2/3, r/2.6
			vector.FillRect(d.dst, x-edge/2, y-edge/2, w+edge, h+edge, symbolEdge, true)
			vector.FillRect(d.dst, x, y, w, h, c, true)
		}
	case viewer.MapTeleporter:
		vector.StrokeCircle(d.dst, cx, cy, r*0.85, max(2, r/2.2)+edge, symbolEdge, true)
		vector.StrokeCircle(d.dst, cx, cy, r*0.85, max(1.5, r/2.2), c, true)
	case viewer.MapHole:
		vector.FillCircle(d.dst, cx, cy, r+edge/2, color.RGBA{190, 150, 100, 255}, true)
		vector.FillCircle(d.dst, cx, cy, r*0.8, color.RGBA{14, 8, 4, 255}, true)
		vector.StrokeCircle(d.dst, cx, cy, r+edge/2, max(1, edge/1.5), c, true)
	case viewer.MapTrapdoor:
		vector.StrokeRect(d.dst, cx-r*0.8, cy-r*0.8, r*1.6, r*1.6, max(2, r/3)+edge, symbolEdge, true)
		vector.StrokeRect(d.dst, cx-r*0.8, cy-r*0.8, r*1.6, r*1.6, max(1.5, r/3), c, true)
	case viewer.MapPlate:
		k := float32(0.7)
		if mc.Down {
			k = 0.55
			c = color.RGBA{185, 180, 165, 255}
		}
		vector.FillRect(d.dst, cx-r*k-edge/2, cy-r*k-edge/2, 2*r*k+edge, 2*r*k+edge, symbolEdge, true)
		vector.FillRect(d.dst, cx-r*k, cy-r*k, 2*r*k, 2*r*k, c, true)
	case viewer.MapMark:
		vector.StrokeCircle(d.dst, cx, cy, r*0.85, max(1, r/4)+edge, symbolEdge, true)
		vector.StrokeCircle(d.dst, cx, cy, r*0.85, max(1, r/4), c, true)
		vector.FillCircle(d.dst, cx, cy, r*0.35+edge/2, symbolEdge, true)
		vector.FillCircle(d.dst, cx, cy, r*0.35, c, true)
	case viewer.MapSocket:
		var p vector.Path
		p.MoveTo(cx, cy-r)
		p.LineTo(cx+r*0.7, cy)
		p.LineTo(cx, cy+r)
		p.LineTo(cx-r*0.7, cy)
		p.Close()
		if mc.Places > 0 && mc.Gems > 0 {
			fill(d.dst, &p, color.NRGBA{c.R, c.G, c.B, uint8(80 + 175*mc.Gems/mc.Places)})
		}
		stroke(d.dst, &p, max(2, r/3)+edge, symbolEdge)
		stroke(d.dst, &p, max(1.5, r/3), c)
	case viewer.MapLever:
		// A base bar, and a handle leaning left while off and right while on, with a knob.
		lean := float32(-0.6)
		if mc.Open {
			lean = 0.6
		}
		var p vector.Path
		p.MoveTo(cx-r*0.8, cy+r*0.7)
		p.LineTo(cx+r*0.8, cy+r*0.7)
		p.MoveTo(cx, cy+r*0.7)
		p.LineTo(cx+lean*r, cy-r*0.6)
		w := max(1.5, r/3)
		stroke(d.dst, &p, w+edge*1.5, symbolEdge)
		stroke(d.dst, &p, w, c)
		vector.FillCircle(d.dst, cx+lean*r, cy-r*0.6, r*0.32+edge/2, symbolEdge, true)
		vector.FillCircle(d.dst, cx+lean*r, cy-r*0.6, r*0.32, c, true)
	case viewer.MapThurisaz, viewer.MapGebo, viewer.MapRaido:
		d.rune(mc.Symbol, cx, cy, r, c, edge)
	case viewer.MapMouth:
		var p vector.Path
		p.MoveTo(cx-r*0.2, cy+r*0.5)
		p.LineTo(cx-r*0.7, cy+r)
		p.LineTo(cx+r*0.1, cy+r*0.6)
		p.Close()
		fill(d.dst, &p, c)
		vector.FillCircle(d.dst, cx, cy, r*0.75+edge/2, symbolEdge, true)
		vector.FillCircle(d.dst, cx, cy, r*0.75, c, true)
	}
}

// rune draws rune mark symbol sym as its letter, about 2r tall, in color c outlined dark.
func (d *mapDrawing) rune(sym int, cx, cy, r float32, c color.RGBA, edge float32) {
	var p vector.Path
	stem := cx - r*0.35
	switch sym {
	case viewer.MapThurisaz:
		// A stem with a thorn to its right.
		p.MoveTo(stem, cy-r)
		p.LineTo(stem, cy+r)
		p.MoveTo(stem, cy-r*0.5)
		p.LineTo(cx+r*0.5, cy)
		p.LineTo(stem, cy+r*0.5)
	case viewer.MapGebo:
		// An X.
		p.MoveTo(cx-r*0.75, cy-r*0.8)
		p.LineTo(cx+r*0.75, cy+r*0.8)
		p.MoveTo(cx+r*0.75, cy-r*0.8)
		p.LineTo(cx-r*0.75, cy+r*0.8)
	case viewer.MapRaido:
		// An R: a stem, a bowl off its top, a leg from the bowl.
		p.MoveTo(stem, cy+r)
		p.LineTo(stem, cy-r)
		p.LineTo(cx+r*0.5, cy-r*0.5)
		p.LineTo(stem, cy)
		p.LineTo(cx+r*0.55, cy+r)
	}
	w := max(1.5, r/2.6)
	stroke(d.dst, &p, w+edge*1.5, symbolEdge)
	stroke(d.dst, &p, w, c)
}

// talker draws a creature that talks on cell c: a small figure, a head over a body, in its
// color, glowing.
func (d *mapDrawing) talker(glow *ebiten.Image, c content.Cell, breath float32) {
	cx, cy := d.middle(c, 0)
	z := float32(d.f.Zoom)
	col := symbolColors[viewer.MapNPC]
	d.spot(glow, cx, cy-z*0.3, z*3, col, breath)
	head, body := max(2, z*0.22), max(3, z*0.42)
	hy := cy - z*0.55
	var b vector.Path
	b.MoveTo(cx-body*0.8, cy+z*0.25)
	b.LineTo(cx-body*0.5, hy+head)
	b.LineTo(cx+body*0.5, hy+head)
	b.LineTo(cx+body*0.8, cy+z*0.25)
	b.Close()
	stroke(d.dst, &b, 2.5, symbolEdge)
	fill(d.dst, &b, col)
	vector.FillCircle(d.dst, cx, hy, head+1.2, symbolEdge, true)
	vector.FillCircle(d.dst, cx, hy, head, col, true)
}

// player draws the player's marker: a circle with a pointer along its facing, outlined dark,
// pulsing once a second.
func (d *mapDrawing) player() {
	m := d.m
	x, y := d.at(mapPointOf(m.Player))
	z := float32(d.f.Zoom)
	r := max(3, z*0.55)
	dir := sim.Dirs[m.Facing%len(sim.Dirs)]
	// A step along the facing, as a direction on the map.
	ax, ay := float32(dir[1]-dir[0]), float32(dir[0]+dir[1])/2
	n := float32(math.Hypot(float64(ax), float64(ay)))
	if n > 0 {
		ax, ay = ax/n, ay/n
	}
	pulse := float32(0.75 + 0.25*math.Sin(d.t*2*math.Pi))
	d.spot(d.s.glowImage(), x, y, r*5, mapPlayerColor, 0.6*pulse)
	var p vector.Path
	p.MoveTo(x+ax*r*1.9, y+ay*r*1.9)
	p.LineTo(x-ay*r*0.8, y+ax*r*0.8)
	p.LineTo(x+ay*r*0.8, y-ax*r*0.8)
	p.Close()
	stroke(d.dst, &p, 3, symbolEdge)
	vector.FillCircle(d.dst, x, y, r+1.5, symbolEdge, true)
	c := scaleColor(mapPlayerColor, pulse)
	fill(d.dst, &p, c)
	vector.FillCircle(d.dst, x, y, r, c, true)
	vector.FillCircle(d.dst, x-r/3, y-r/3, r/3, color.RGBA{255, 190, 170, 200}, true)
}

// mapPointOf is the map point of ground point p, in cells.
func mapPointOf(p [2]float64) [2]float64 { return [2]float64{p[1] - p[0], (p[0] + p[1]) / 2} }

func scaleColor(c color.RGBA, k float32) color.RGBA {
	return color.RGBA{uint8(float32(c.R) * k), uint8(float32(c.G) * k), uint8(float32(c.B) * k), c.A}
}

// mapTime is the time the map's lights go by.
func (s *Window) mapTime() float64 { return float64(s.drawn) / float64(ebiten.TPS()) }

// minimapOpacity is how opaque the minimap's ink is over the view.
const minimapOpacity = 0.85

// drawMinimap draws the minimap in the top-left corner: the level about the player, inked
// straight over the view with no frame or parchment and fading out toward its edges, and the
// level's name under it.
func (s *Window) drawMinimap(screen *ebiten.Image) {
	mv := s.v.Minimap()
	if mv == nil {
		return
	}
	size := mv.Area.Size()
	page := blank(&s.maps.cells, size)
	d := &mapDrawing{s: s, dst: page, m: mv.LevelMap, f: mv.MapFrame, off: mv.Area.Min, t: s.mapTime()}
	d.draw()
	// Fade the edges out over a cell's width.
	fade := float64(mv.Zoom * 2)
	mask := sized(&s.maps.miniMask, size, func() image.Image { return edgeFade(size, fade) })
	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn}
	page.DrawImage(mask, op)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(mv.Area.Min.X), float64(mv.Area.Min.Y))
	op.ColorScale.ScaleAlpha(minimapOpacity)
	screen.DrawImage(page, op)
	// The level's name, shadowed so it reads over the view.
	p := mv.Plaque
	x, y := p.Min.X+viewer.Margin, p.Min.Y+2
	s.printIn(screen, mv.Name, x+1, y+1, color.RGBA{0, 0, 0, 200})
	s.printIn(screen, mv.Name, x, y, color.RGBA{235, 215, 160, 255})
}

// edgeFade is a mask of size, opaque in the middle and fading to clear over fade pixels toward
// each edge.
func edgeFade(size image.Point, fade float64) image.Image {
	img := image.NewAlpha(image.Rectangle{Max: size})
	for y := range size.Y {
		for x := range size.X {
			d := math.Min(math.Min(float64(x)+0.5, float64(size.X-x)-0.5), math.Min(float64(y)+0.5, float64(size.Y-y)-0.5))
			a := math.Min(1, d/fade)
			a = a * a * (3 - 2*a)
			img.SetAlpha(x, y, color.Alpha{uint8(255 * a)})
		}
	}
	return img
}

// drawFullMap draws the full map: the view darkened, and the parchment unrolling from the
// middle between its two rolls, or rolling up again once the map has closed.
func (s *Window) drawFullMap(screen *ebiten.Image) {
	fv := s.v.FullMap()
	st := &s.maps
	now := time.Now()
	switch {
	case fv != nil && !st.open:
		st.open, st.closing, st.since = true, false, now
	case fv == nil && st.open:
		st.open, st.closing, st.since = false, true, now
	}
	var u float64
	switch {
	case fv != nil:
		u = easeOut(float64(now.Sub(st.since)) / float64(unrollTime))
		s.drawMapPage(fv)
	case st.closing && st.page != nil:
		u = 1 - easeOut(float64(now.Sub(st.since))/float64(rollUpTime))
		if u <= 0 {
			st.closing = false
			return
		}
	default:
		return
	}
	if s.scripted() {
		// A script's screenshots show the map as it stands, not part way.
		u = 1
		if fv == nil {
			return
		}
	}
	r := st.pageRect
	vector.FillRect(screen, 0, 0, float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy()), color.RGBA{0, 0, 0, uint8(170 * u)}, false)
	half := int(float64(r.Dx()) / 2 * u)
	mid := r.Min.X + r.Dx()/2
	if half > 0 {
		src := image.Rect(r.Dx()/2-half, 0, r.Dx()/2+half, r.Dy())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(mid-half), float64(r.Min.Y))
		screen.DrawImage(st.page.SubImage(src).(*ebiten.Image), op)
	}
	// The rolls, each half over the sheet's edge.
	rw := viewer.MapRollWidth(screen.Bounds().Size())
	roll := sized(&st.roll, image.Pt(rw, r.Dy()+rw).Div(parchmentScale), func() image.Image {
		return frame.Roll(rw/parchmentScale, (r.Dy()+rw)/parchmentScale)
	})
	for _, x := range []int{mid - half - rw/2, mid + half - rw/2} {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(parchmentScale, parchmentScale)
		op.GeoM.Translate(float64(x+3), float64(r.Min.Y-rw/2+4))
		op.ColorScale.ScaleWithColor(shadowColor)
		screen.DrawImage(roll, op)
		op.ColorScale.Reset()
		op.GeoM.Translate(-3, -4)
		screen.DrawImage(roll, op)
	}
}

// easeOut eases t, 0..1, so that it slows toward the end.
func easeOut(t float64) float64 {
	t = max(0, min(1, t))
	return 1 - (1-t)*(1-t)*(1-t)
}

// drawMapPage draws the full map onto its page: the parchment, the title, the legend, the cells
// and the focus's notes and links.
func (s *Window) drawMapPage(fv *viewer.FullMapView) {
	st := &s.maps
	size := fv.Rect.Size()
	sheet := sized(&st.sheet, size.Div(parchmentScale), func() image.Image {
		return frame.Parchment(size.X/parchmentScale, size.Y/parchmentScale, true)
	})
	page := blank(&st.page, size)
	st.pageRect = fv.Rect
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(size.X)/float64(sheet.Bounds().Dx()), float64(size.Y)/float64(sheet.Bounds().Dy()))
	page.DrawImage(sheet, op)
	off := fv.Rect.Min
	t := s.mapTime()

	// The cells, clipped to the sheet inside its torn edges and rolls.
	clip := page.SubImage(image.Rectangle{Max: size}.Inset(2 * viewer.Margin)).(*ebiten.Image)
	d := &mapDrawing{s: s, dst: clip, m: fv.LevelMap, f: fv.MapFrame, off: off, t: t}
	d.draw()
	s.drawMapLinks(d, fv)

	// The title, inked at the top.
	scale := 2
	if size.X < 700 {
		scale = 1
	}
	tw := len(fv.Title) * font.LargeAdvance * scale
	s.printFace(page, font.Large, fv.Title, (size.X-tw)/2, 3*viewer.Margin, scale, color.RGBA{96, 54, 22, 255})

	s.drawMapLegend(page, fv, off, t)
	s.drawMapNote(page, fv, off)
	// The keys along the bottom, wrapped onto more lines when the sheet is narrow.
	keys := []string{"Wheel zoom", "Drag pan", "Right-click follow", "Backspace back", "PgUp/PgDn levels", "Home here", "M close"}
	n := fv.Inner.Dx() / viewer.CharWidth
	var lines []string
	for _, k := range keys {
		if last := len(lines) - 1; last >= 0 && len(lines[last])+2+len(k) <= n {
			lines[last] += "  " + k
		} else {
			lines = append(lines, k)
		}
	}
	for i, l := range lines {
		y := size.Y - 3*viewer.Margin - (len(lines)-i)*viewer.LineHeight
		s.printIn(page, l, (size.X-len(l)*viewer.CharWidth)/2, y, color.RGBA{225, 205, 160, 255})
	}
}

// drawMapLegend draws the legend down the map's left side, in a box headed Key: each kind's
// symbol and its name, the player's first when it is on the level shown.
func (s *Window) drawMapLegend(page *ebiten.Image, fv *viewer.FullMapView, off image.Point, t float64) {
	at := fv.LegendAt.Sub(off)
	first := 0
	if fv.HasPlayer {
		first = 1
	}
	n := first + len(fv.Legend)
	w := 0
	for _, k := range fv.Legend {
		w = max(w, len(viewer.MapKindNames[k]))
	}
	w = max(w, 3)*viewer.CharWidth + 40
	box := image.Rect(at.X-6, at.Y-6, at.X+w, at.Y+n*viewer.LegendLine+2)
	vector.FillRect(page, float32(box.Min.X), float32(box.Min.Y-font.LargeLineHeight/2), float32(box.Dx()),
		float32(box.Dy()+font.LargeLineHeight/2), color.RGBA{196, 165, 112, 235}, false)
	vector.StrokeRect(page, float32(box.Min.X), float32(box.Min.Y), float32(box.Dx()), float32(box.Dy()), 2, frame.Ink, true)
	s.printFace(page, font.Large, "Key", box.Min.X+8, box.Min.Y-font.LargeLineHeight+4, 1, color.RGBA{96, 54, 22, 255})
	// Each line's symbol is drawn by a one-cell map of its own.
	line := func(i int, name string, mc viewer.MapCell, talker bool) {
		y := at.Y + i*viewer.LegendLine
		lm := &viewer.LevelMap{Cells: []viewer.MapCell{mc}}
		switch {
		case talker:
			lm.Cells, lm.Talkers = nil, []viewer.MapTalker{{}}
		case i == 0 && fv.HasPlayer:
			lm.Cells, lm.HasPlayer, lm.Player, lm.Facing = nil, true, [2]float64{0.5, 0.5}, 2
		}
		f := viewer.MapFrame{Area: image.Rect(at.X, y, at.X+20, y+viewer.LineHeight).Add(off), Center: [2]float64{0, 0.5}, Zoom: 9}
		dd := &mapDrawing{s: s, dst: page, m: lm, f: f, off: off, t: t}
		dd.draw()
		s.printIn(page, name, at.X+26, y, color.RGBA{240, 226, 190, 255})
	}
	if fv.HasPlayer {
		line(0, "You", viewer.MapCell{}, false)
	}
	for i, k := range fv.Legend {
		mc := viewer.MapCell{Kind: viewer.MapFloor, Symbol: k}
		switch k {
		case viewer.MapDoor, viewer.MapWater:
			mc.Kind, mc.Symbol = k, viewer.MapNone
		case viewer.MapNPC:
			mc.Symbol = viewer.MapNone
		}
		line(first+i, viewer.MapKindNames[k], mc, k == viewer.MapNPC)
	}
}

// drawMapLinks draws the focus's outline, its links' arrows on the level, and its labels for
// other levels; the selection and a follow's arrival are ringed.
func (s *Window) drawMapLinks(d *mapDrawing, fv *viewer.FullMapView) {
	z := float32(fv.Zoom)
	pulse := float32(0.7 + 0.3*math.Sin(d.t*4))
	ring := func(c content.Cell, col color.RGBA, w float32) {
		var p vector.Path
		mc, _ := fv.At(c)
		d.diamond(&p, c, d.lift(mc))
		stroke(d.dst, &p, w+2, symbolEdge)
		stroke(d.dst, &p, w, col)
	}
	if a := fv.Arrival; a != nil {
		x, y := d.middle(*a, 0)
		vector.StrokeCircle(d.dst, x, y, z*1.2, 3, symbolEdge, true)
		vector.StrokeCircle(d.dst, x, y, z*1.2, 2, scaleColor(focusColor, pulse), true)
	}
	if sel := fv.Selected; sel != nil {
		ring(*sel, focusColor, 2)
	}
	if f := fv.Focus; f != nil && (fv.Selected == nil || *f != *fv.Selected) {
		ring(*f, scaleColor(focusColor, pulse), 1.5)
	}
	for i, a := range fv.Arrows {
		c := linkColors[a.Kind]
		x0, y0 := d.middle(a.From, 0)
		x1, y1 := d.middle(a.To, 0)
		curve(d.dst, x0, y0, x1, y1, float32(i%3+1)*0.18, c, a.In)
		vector.StrokeCircle(d.dst, x1, y1, max(3, z*0.6), 3, symbolEdge, true)
		vector.StrokeCircle(d.dst, x1, y1, max(3, z*0.6), 2, c, true)
	}
	for _, l := range fv.Labels {
		c := linkColors[l.Kind]
		r := l.Rect.Sub(d.off)
		fx, fy := d.middle(*fv.Focus, 0)
		curve(d.dst, fx, fy, float32(r.Min.X), float32(r.Min.Y+r.Dy()/2), 0.15, c, l.In)
		bg := noteColor
		if l.Button && d.s.v.Cursor.In(l.Rect) {
			bg = color.RGBA{90, 60, 25, 235}
		}
		vector.FillRect(d.dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), bg, false)
		vector.StrokeRect(d.dst, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, c, false)
		d.s.printIn(d.dst, l.Text, r.Min.X+6, r.Min.Y+1, noteText)
	}
}

// curve draws a curved arrow from (x0, y0) to (x1, y1), bowed by bend of its length to its left,
// dark under its color, dashed if asked.
func curve(dst *ebiten.Image, x0, y0, x1, y1, bend float32, c color.RGBA, dashed bool) {
	dx, dy := x1-x0, y1-y0
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length < 2 {
		return
	}
	// The control point, off the middle to the left.
	cx, cy := (x0+x1)/2+dy*bend, (y0+y1)/2-dx*bend
	at := func(t float32) (float32, float32) {
		u := 1 - t
		return u*u*x0 + 2*u*t*cx + t*t*x1, u*u*y0 + 2*u*t*cy + t*t*y1
	}
	steps := max(8, int(length/6))
	var p vector.Path
	for i := range steps {
		if dashed && i%2 == 1 {
			continue
		}
		xa, ya := at(float32(i) / float32(steps))
		xb, yb := at(float32(i+1) / float32(steps))
		if dashed || i == 0 {
			p.MoveTo(xa, ya)
		}
		p.LineTo(xb, yb)
	}
	stroke(dst, &p, 4.5, symbolEdge)
	stroke(dst, &p, 2.5, c)
	// The head, along the curve's last stretch.
	hx, hy := at(0.9)
	ux, uy := x1-hx, y1-hy
	n := float32(math.Hypot(float64(ux), float64(uy)))
	if n == 0 {
		return
	}
	ux, uy = ux/n, uy/n
	const head = 11
	var h vector.Path
	h.MoveTo(x1, y1)
	h.LineTo(x1-ux*head-uy*head/2, y1-uy*head+ux*head/2)
	h.LineTo(x1-ux*head+uy*head/2, y1-uy*head-ux*head/2)
	h.Close()
	stroke(dst, &h, 2, symbolEdge)
	fill(dst, &h, c)
}

// drawMapNote draws the focus's note in its box: its name, then where it leads and where it was
// come to from, each link's line marked in its color.
func (s *Window) drawMapNote(page *ebiten.Image, fv *viewer.FullMapView, off image.Point) {
	if len(fv.Note) == 0 {
		return
	}
	r := fv.NoteRect.Sub(off)
	vector.FillRect(page, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), noteColor, false)
	vector.StrokeRect(page, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, noteEdge, false)
	for i, l := range fv.Note {
		y := r.Min.Y + 4 + i*viewer.LineHeight
		c := noteText
		switch {
		case i == 0:
			c = focusColor
		case l.Said:
			c = color.RGBA{255, 244, 214, 255}
		case l.Talker:
			c = symbolColors[viewer.MapNPC]
		}
		if l.Kind != "" {
			vector.FillRect(page, float32(r.Min.X+4), float32(y+3), 3, float32(viewer.LineHeight-6), linkColors[l.Kind], false)
		}
		s.printIn(page, l.Text, r.Min.X+10, y, c)
	}
}
