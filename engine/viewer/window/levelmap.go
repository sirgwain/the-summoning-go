package window

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// mapColors are the level map's colors by cell kind.
var mapColors = map[int]color.RGBA{
	viewer.MapFloor: {120, 110, 95, 255}, viewer.MapWall: {55, 50, 48, 255}, viewer.MapDoor: {150, 100, 50, 255},
	viewer.MapExit: {80, 200, 90, 255}, viewer.MapWater: {50, 90, 190, 255}, viewer.MapHole: {15, 15, 15, 255},
	viewer.MapTeleporter: {170, 80, 200, 255}, viewer.MapMouth: {220, 220, 220, 255},
	viewer.MapPlate: {220, 190, 60, 255}, viewer.MapMark: {60, 200, 210, 255},
	viewer.MapTrapdoor: {120, 75, 40, 255}, viewer.MapSocket: {230, 110, 40, 255},
}

// mapPlayerColor is the player's cell on the level map.
var mapPlayerColor = color.RGBA{230, 40, 40, 255}

// drawLevelMap draws map m's cells as diamonds of half-width s around the middle of r, rows and
// columns turned as the view turns them, and the player's cell in red.
func drawLevelMap(screen *ebiten.Image, m *viewer.LevelMap, r image.Rectangle, s float32) {
	rows, cols := float32(m.Size[0]), float32(m.Size[1])
	// The map's corners: (0, 0) at the top, (rows, 0) at the left, (0, cols) at the right.
	ox := float32(r.Min.X+r.Max.X)/2 - (cols-rows)*s/2
	oy := float32(r.Min.Y+r.Max.Y)/2 - (cols+rows)*s/4
	paths := map[int]*vector.Path{}
	diamond := func(path *vector.Path, c content.Cell) {
		x := ox + float32(c[1]-c[0])*s
		y := oy + float32(c[0]+c[1])*s/2
		path.MoveTo(x, y)
		path.LineTo(x+s, y+s/2)
		path.LineTo(x, y+s)
		path.LineTo(x-s, y+s/2)
		path.Close()
	}
	for _, c := range m.Cells {
		if paths[c.Kind] == nil {
			paths[c.Kind] = &vector.Path{}
		}
		diamond(paths[c.Kind], c.Cell)
	}
	for kind, path := range paths {
		op := &vector.DrawPathOptions{}
		op.ColorScale.ScaleWithColor(mapColors[kind])
		vector.FillPath(screen, path, nil, op)
	}
	player := &vector.Path{}
	diamond(player, m.Player)
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(mapPlayerColor)
	vector.FillPath(screen, player, nil, op)
}

// drawMinimap draws the level map small in the top-left corner.
func (s *Window) drawMinimap(screen *ebiten.Image) {
	v := s.v
	if !v.MinimapShown() {
		return
	}
	m := v.Map()
	const size = 200
	r := image.Rect(viewer.Margin, viewer.Margin, viewer.Margin+size, viewer.Margin+size/2)
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), panelColor, false)
	cell := float32(size) / float32(m.Size[0]+m.Size[1])
	drawLevelMap(screen, m, r, cell)
}

// drawFullMap draws the level map over the whole screen, with the level's name.
func (s *Window) drawFullMap(screen *ebiten.Image) {
	v := s.v
	if !v.MapOpen {
		return
	}
	m := v.Map()
	if m == nil {
		return
	}
	screen.Fill(color.RGBA{8, 8, 12, 255})
	legendW := s.drawMapLegend(screen, m)
	r := image.Rect(legendW, 3*viewer.LineHeight, v.Screen.X, v.Screen.Y-viewer.Margin)
	cell := min(float32(r.Dx())/float32(m.Size[0]+m.Size[1]), 2*float32(r.Dy())/float32(m.Size[0]+m.Size[1]))
	drawLevelMap(screen, m, r, cell)
	name := m.Name
	if name == "" {
		name = v.World.Current().ID
	}
	ebitenutil.DebugPrintAt(screen, name, (v.Screen.X-len(name)*viewer.CharWidth)/2, viewer.LineHeight)
}

// drawMapLegend draws the full map's legend down its left side, a swatch and a name for the
// player and for each kind the map shows, and returns the width it takes.
func (s *Window) drawMapLegend(screen *ebiten.Image, m *viewer.LevelMap) int {
	const swatch = 10
	x, y := viewer.Margin, 3*viewer.LineHeight
	width := 0
	line := func(c color.Color, name string) {
		vector.FillRect(screen, float32(x), float32(y+(viewer.LineHeight-swatch)/2), swatch, swatch, c, false)
		vector.StrokeRect(screen, float32(x), float32(y+(viewer.LineHeight-swatch)/2), swatch, swatch, 1, slotEdgeColor, false)
		ebitenutil.DebugPrintAt(screen, name, x+swatch+viewer.CharWidth, y)
		width = max(width, swatch+viewer.CharWidth*(1+len(name)))
		y += viewer.LineHeight + 4
	}
	line(mapPlayerColor, "You")
	for _, k := range m.Legend {
		line(mapColors[k], viewer.MapKindNames[k])
	}
	return x + width + viewer.Margin
}
