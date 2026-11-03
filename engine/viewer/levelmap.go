package viewer

import (
	"fmt"
	"image"
	"math"
	"slices"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/frame"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The level map (docs/spec/world.md, The engine's map): the cells the player has seen, by what
// they are, where the player stands, and the links it has travelled. The full map opens with M
// while playing, over the view, and takes the input while it is open; the minimap shows in a
// corner unless the level refuses the map.

// Map kinds: what a cell's ground is (floor, wall, door, water), and the symbols the features
// draw over it.
const (
	MapFloor = iota
	MapWall
	MapDoor
	MapWater
	MapExit
	MapTeleporter
	MapHole
	MapTrapdoor
	MapPlate
	MapMark
	MapSocket
	MapMouth
	MapThurisaz
	MapGebo
	MapRaido
	MapNPC
	MapLever
	MapKinds
	// MapNone is a cell without a symbol.
	MapNone = -1
)

// MapKindNames name the kinds in the map's legend and its notes.
var MapKindNames = [MapKinds]string{
	MapFloor: "Floor", MapWall: "Wall", MapDoor: "Door", MapWater: "Water", MapExit: "Stairs",
	MapTeleporter: "Teleporter", MapHole: "Pit", MapTrapdoor: "Trapdoor", MapPlate: "Plate",
	MapMark: "Rune", MapSocket: "Gem socket", MapMouth: "Mouth",
	MapThurisaz: "Thurisaz", MapGebo: "Gebo", MapRaido: "Raido", MapNPC: "NPC", MapLever: "Lever",
}

// runeSymbols are the symbols of the rune marks the game's runes teleport to, by rune: 1 the
// Thurisaz rune's, 2 the Gebo rune's, 3 the Raido rune's. Any other rune draws the plain mark.
var runeSymbols = map[string]int{"1": MapThurisaz, "2": MapGebo, "3": MapRaido}

// noteWidth is the widest a line of a mouth's words is in a note, in letters.
const noteWidth = 48

// MapCell is a seen cell of the level map.
type MapCell struct {
	Cell content.Cell
	// Kind is the ground (MapFloor, MapWall, MapDoor, MapWater), Symbol the feature's symbol or
	// MapNone.
	Kind, Symbol int
	// Open is an open door or trapdoor, Down a pressed plate; Across is a door whose wall runs
	// along the columns rather than the rows.
	Open, Down, Across bool
	// Gems and Places are a socket's gems set and places.
	Gems, Places int
	// Known is a cell with a recorded link from it, or a mouth the player has heard; Heard are
	// what the mouth has said.
	Known bool
	Heard []string
}

// MapTalker is a creature that talks, where the player last saw it.
type MapTalker struct {
	Cell content.Cell
	Name string
}

// LevelMap is a visited level's map.
type LevelMap struct {
	Level, Name string
	// Current is whether it is the current level.
	Current bool
	Size    [2]int
	Cells   []MapCell
	// Player is the player's ground point in cells (a cell's middle is its index + ½), and
	// Facing its table direction; HasPlayer is false on another level.
	Player    [2]float64
	Facing    int
	HasPlayer bool
	// Legend are the kinds the map shows, in kind order.
	Legend []int
	// Links are the recorded links from or to the level; Talkers the creatures that talk seen on
	// it.
	Links   []sim.Link
	Talkers []MapTalker
	cells   map[content.Cell]int
}

// At returns the map's cell c, if seen.
func (m *LevelMap) At(c content.Cell) (MapCell, bool) {
	i, ok := m.cells[c]
	if !ok {
		return MapCell{}, false
	}
	return m.Cells[i], true
}

// Map returns the current level's map as the player has seen it, or nil without a player.
func (v *Viewer) Map() *LevelMap {
	if v.World.Player() == nil {
		return nil
	}
	return v.LevelMapOf(v.World.Current().ID)
}

// LevelMapOf returns level id's map as the player last saw it, or nil when the world has not
// loaded it.
func (v *Viewer) LevelMapOf(id string) *LevelMap {
	l := v.World.Level(id)
	if l == nil {
		return nil
	}
	m := &LevelMap{Level: id, Name: v.levelName(id), Current: id == v.World.Current().ID, Size: l.Def.Size,
		cells: map[content.Cell]int{}}
	if p := v.World.Player(); p != nil && m.Current {
		m.Player = [2]float64{float64(p.U) / sim.CellUnits, float64(p.V) / sim.CellUnits}
		m.Facing, m.HasPlayer = p.Facing, true
	}
	for _, lk := range v.World.Links() {
		if lk.From.Level == id || lk.To.Level == id {
			m.Links = append(m.Links, lk)
		}
	}
	heard := map[content.Cell][]string{}
	for _, h := range v.World.Mouths() {
		if h.Level == id {
			for _, msg := range h.Messages {
				heard[h.Cell] = append(heard[h.Cell], inspect.Text(v.Pack, msg))
			}
		}
	}
	var shown [MapKinds]bool
	for _, t := range v.World.Talkers() {
		if t.Level == id {
			m.Talkers = append(m.Talkers, MapTalker{Cell: t.Cell, Name: inspect.ThingName(v.Pack, t.Template)})
			shown[MapNPC] = true
		}
	}
	for x := range l.Def.Size[0] {
		for y := range l.Def.Size[1] {
			c := content.Cell{x, y}
			if !l.Seen(c) {
				continue
			}
			mc, ok := mapCell(l, c)
			if !ok {
				continue
			}
			mc.Heard = heard[c]
			mc.Known = len(mc.Heard) > 0 || slices.ContainsFunc(m.Links, func(lk sim.Link) bool { return lk.From.Level == id && lk.From.Cell == c })
			m.cells[c] = len(m.Cells)
			m.Cells = append(m.Cells, mc)
			shown[mc.Kind] = true
			if mc.Symbol != MapNone {
				shown[mc.Symbol] = true
			}
		}
	}
	for k, ok := range shown {
		if ok && k != MapFloor && k != MapWall {
			m.Legend = append(m.Legend, k)
		}
	}
	return m
}

// mapCell returns what seen cell c of level l shows, false for nothing (no floor, no wall).
func mapCell(l *sim.Level, c content.Cell) (MapCell, bool) {
	mc := MapCell{Cell: c, Kind: MapFloor, Symbol: MapNone}
	wall := l.Wall[c[0]][c[1]] != content.NoFrame
	f := l.FeatureAt(c)
	switch {
	case f != nil && f.Kind == content.FeatureDoor:
		mc.Kind, mc.Open = MapDoor, f.State == content.DoorOpen
		isWall := func(dx, dy int) bool {
			n := content.Cell{c[0] + dx, c[1] + dy}
			return l.InBounds(n) && l.Wall[n[0]][n[1]] != content.NoFrame && l.FeatureAt(n) == nil
		}
		mc.Across = !(isWall(-1, 0) || isWall(1, 0)) && (isWall(0, -1) || isWall(0, 1))
	case wall:
		mc.Kind = MapWall
	default:
		ft := l.FloorType(c)
		if ft == nil {
			return mc, false
		}
		if ft.Depth > 0 {
			mc.Kind = MapWater
		}
	}
	if f == nil {
		return mc, true
	}
	switch f.Kind {
	case content.FeatureExit:
		mc.Symbol = MapExit
	case content.FeatureTeleporter:
		switch {
		case wall:
			mc.Symbol = MapExit
		case f.Armed:
			mc.Symbol = MapTeleporter
		}
	case content.FeaturePit:
		mc.Symbol, mc.Open = MapHole, true
	case content.FeatureTrapdoor:
		mc.Symbol, mc.Open = MapTrapdoor, f.Open
		if f.Open {
			mc.Symbol = MapHole
		}
	case content.FeaturePlate:
		mc.Symbol, mc.Down = MapPlate, f.Down
	case content.FeatureLever:
		mc.Symbol, mc.Open = MapLever, f.On
	case content.FeatureMark:
		mc.Symbol = MapMark
		if s, ok := runeSymbols[f.Rune]; ok {
			mc.Symbol = s
		}
	case content.FeatureSocket:
		mc.Symbol, mc.Gems, mc.Places = MapSocket, len(f.Set), len(f.Places)
	case content.FeatureMouth:
		mc.Symbol = MapMouth
	}
	return mc, true
}

// levelName returns level id's name, or "Level id" without one.
func (v *Viewer) levelName(id string) string {
	if d := v.Pack.Levels[id]; d != nil && d.Name != "" {
		return inspect.Text(v.Pack, d.Name)
	}
	return "Level " + id
}

// mapShowable reports whether the full map can show level id: visited, loaded, not refusing.
func (v *Viewer) mapShowable(id string) bool {
	l := v.World.Level(id)
	return l != nil && l.Def.Map == nil && slices.Contains(v.World.Visited(), id)
}

// MinimapShown reports whether the minimap shows: while playing, on a level that does not refuse
// its map, with no window open over the view, unless the player has hidden it.
func (v *Viewer) MinimapShown() bool {
	return v.Play && v.World.Player() != nil && !v.World.MapRefused() && !v.MapOpen && v.World.Talk() == nil &&
		!v.InventoryOpen && !v.SpellsOpen && !v.book.open && !v.LogOpen && !v.MinimapHidden
}

// Map geometry. A point of the map is in cells: (mx, my) = (v − u, (u + v) / 2) for the ground
// point (u, v), so a cell is a diamond 2 wide and 1 high, its top at (c1 − c0, (c0 + c1) / 2).
// On the screen it is Zoom times as large, Zoom being half a cell's width in screen pixels.

// mapPoint returns ground point (u, v), in cells, as a point of the map.
func mapPoint(u, v float64) [2]float64 { return [2]float64{v - u, (u + v) / 2} }

// cellMiddle returns the middle of cell c as a point of the map.
func cellMiddle(c content.Cell) [2]float64 { return mapPoint(float64(c[0])+0.5, float64(c[1])+0.5) }

// MapFrame is a map laid out on the screen: Center, a point of the map, shows at the middle of
// Area, Zoom screen pixels to a unit.
type MapFrame struct {
	Area   image.Rectangle
	Center [2]float64
	Zoom   float64
}

// Point returns map point p on the screen.
func (f MapFrame) Point(p [2]float64) (float64, float64) {
	return float64(f.Area.Min.X+f.Area.Max.X)/2 + (p[0]-f.Center[0])*f.Zoom,
		float64(f.Area.Min.Y+f.Area.Max.Y)/2 + (p[1]-f.Center[1])*f.Zoom
}

// CellPoint returns the middle of cell c on the screen.
func (f MapFrame) CellPoint(c content.Cell) (float64, float64) { return f.Point(cellMiddle(c)) }

// unproject returns the map point at screen point (x, y).
func (f MapFrame) unproject(x, y float64) [2]float64 {
	return [2]float64{(x-float64(f.Area.Min.X+f.Area.Max.X)/2)/f.Zoom + f.Center[0],
		(y-float64(f.Area.Min.Y+f.Area.Max.Y)/2)/f.Zoom + f.Center[1]}
}

// CellAt returns the cell at screen point pt.
func (f MapFrame) CellAt(pt image.Point) content.Cell {
	p := f.unproject(float64(pt.X), float64(pt.Y))
	a, b := p[0], 2*p[1]
	return content.Cell{int(math.Floor((b - a) / 2)), int(math.Floor((a + b) / 2))}
}

// The full map's layout and controls.
const (
	// mapMaxZoom is the closest the full map zooms: 32 screen pixels a cell.
	mapMaxZoom = 16
	// mapZoomStep is how much a notch of the wheel zooms.
	mapZoomStep = 1.2
	// mapDragSlop is how far the pointer moves with the button down before it drags.
	mapDragSlop = 4
	// mapMargin is the margin of cells round the fitted map, in map units.
	mapMargin = 2
)

// mapState is the full map's own state while it is open; opening the map starts it anew.
type mapState struct {
	level string
	// zoom is 0 while the map is fitted; focus is the map point at the middle.
	zoom     float64
	focus    [2]float64
	selected *content.Cell
	// arrival is the cell a follow arrived at, marked.
	arrival *content.Cell
	// follows counts the follows of each link cell, to take the next of its links.
	follows map[content.Destination]int
	// back are where the map was before each follow.
	back []mapPlace
	// press is where the left button went down, dragging whether it has moved since, last the
	// pointer's place at the last update of the drag.
	press      *image.Point
	dragging   bool
	last       image.Point
	labelFocus *content.Cell
}

// mapPlace is where the map was: its level, zoom, focus and selection.
type mapPlace struct {
	level    string
	zoom     float64
	focus    [2]float64
	selected *content.Cell
	arrival  *content.Cell
}

// FullMapView is the full map as it shows now.
type FullMapView struct {
	*LevelMap
	// Rect is the parchment on the screen; Title the level's name as it heads the map.
	Rect  image.Rectangle
	Title string
	MapFrame
	// Legend is where the legend's lines start, one LegendLine apart.
	LegendAt image.Point
	// Focus is the cell the notes and arrows are for, Selected and Arrival the cells selected and
	// arrived at by a follow.
	Focus, Selected, Arrival *content.Cell
	// Note are the focus's lines, in NoteRect.
	Note     []NoteLine
	NoteRect image.Rectangle
	// Arrows are the focus's links on its level, Labels those to other levels.
	Arrows []MapArrow
	Labels []MapLabel
}

// NoteLine is a line of the focus's note; Kind a link's kind for its color, "" for none. Said
// is a line of a mouth's words, Talker a creature's name.
type NoteLine struct {
	Text, Kind   string
	Said, Talker bool
}

// MapArrow is an arrow between two cells of the shown level: In for a link that arrives at the
// focus.
type MapArrow struct {
	From, To content.Cell
	Kind     string
	In       bool
}

// MapLabel names another level a link of the focus leads to or comes from; Button is set when
// clicking it shows that level.
type MapLabel struct {
	Rect   image.Rectangle
	Text   string
	Kind   string
	In     bool
	To     content.Destination
	Button bool
}

// LegendLine is the distance between the full map's legend's lines.
const LegendLine = LineHeight + 4

// legendWidth is the full map's legend's width for map m.
func legendWidth(m *LevelMap) int {
	w := len("You")
	for _, k := range m.Legend {
		w = max(w, len(MapKindNames[k]))
	}
	return 40 + w*CharWidth
}

// openFullMap opens the full map, as it opens each time: on the current level, fitted.
func (v *Viewer) openFullMap() {
	v.MapOpen = true
	v.mapView = mapState{level: v.World.Current().ID, follows: map[content.Destination]int{}}
}

// closeFullMap closes the full map, forgetting where it was.
func (v *Viewer) closeFullMap() {
	v.MapOpen = false
	v.mapView = mapState{}
}

// FullMap returns the full map as it shows now, or nil when it is closed.
func (v *Viewer) FullMap() *FullMapView {
	if !v.MapOpen {
		return nil
	}
	return v.fullMap(v.Cursor)
}

// fullMap lays the full map out with the pointer at cursor.
func (v *Viewer) fullMap(cursor image.Point) *FullMapView {
	st := &v.mapView
	m := v.LevelMapOf(st.level)
	if m == nil {
		return nil
	}
	fv := &FullMapView{LevelMap: m, Title: m.Name, Selected: st.selected, Arrival: st.arrival}
	if !m.Current {
		fv.Title += " - visited"
	}
	fv.Rect = image.Rect(Margin, Margin, v.Screen.X-Margin, v.Screen.Y-Margin)
	inner := fv.Rect.Inset(3 * Margin)
	fv.LegendAt = image.Pt(inner.Min.X, inner.Min.Y+3*LineHeight)
	fv.Area = image.Rect(inner.Min.X+legendWidth(m)+Margin, inner.Min.Y+2*LineHeight, inner.Max.X, inner.Max.Y)
	fit, center := mapFit(m, fv.Area)
	fv.Zoom, fv.Center = fit, center
	if st.zoom > 0 {
		fv.Zoom, fv.Center = max(fit, min(st.zoom, maxZoom(fit))), st.focus
	}
	fv.focusOn(v, cursor)
	return fv
}

// maxZoom is the closest the full map zooms when fit fits it: 32 screen pixels a cell, or twice
// fit when that is more.
func maxZoom(fit float64) float64 { return max(mapMaxZoom, 2*fit) }

// mapFit returns the zoom that fits map m's seen cells and a margin in area, and the middle of
// those cells.
func mapFit(m *LevelMap, area image.Rectangle) (float64, [2]float64) {
	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, c := range m.Cells {
		top := mapPoint(float64(c.Cell[0]), float64(c.Cell[1]))
		lo = [2]float64{min(lo[0], top[0]-1), min(lo[1], top[1])}
		hi = [2]float64{max(hi[0], top[0]+1), max(hi[1], top[1]+1)}
	}
	if len(m.Cells) == 0 {
		top := mapPoint(0, 0)
		bottom := mapPoint(float64(m.Size[0]), float64(m.Size[1]))
		lo, hi = [2]float64{-float64(m.Size[0]), top[1]}, [2]float64{float64(m.Size[1]), bottom[1]}
	}
	lo = [2]float64{lo[0] - mapMargin, lo[1] - mapMargin/2}
	hi = [2]float64{hi[0] + mapMargin, hi[1] + mapMargin/2}
	zoom := min(float64(area.Dx())/(hi[0]-lo[0]), float64(area.Dy())/(hi[1]-lo[1]))
	return max(zoom, 0.5), [2]float64{(lo[0] + hi[0]) / 2, (lo[1] + hi[1]) / 2}
}

// linkCellAt returns the cell at pt that has a feature or a link, if any.
func (fv *FullMapView) linkCellAt(pt image.Point) (content.Cell, bool) {
	if !pt.In(fv.Area) {
		return content.Cell{}, false
	}
	c := fv.CellAt(pt)
	if mc, ok := fv.At(c); ok && (mc.Symbol != MapNone || mc.Kind == MapDoor || mc.Known) {
		return c, true
	}
	if slices.ContainsFunc(fv.Links, func(lk sim.Link) bool { return lk.To.Level == fv.Level && lk.To.Cell == c }) ||
		slices.ContainsFunc(fv.Talkers, func(t MapTalker) bool { return t.Cell == c }) {
		return c, true
	}
	return content.Cell{}, false
}

// focusOn finds the focus with the pointer at cursor, and its note, arrows and labels.
func (fv *FullMapView) focusOn(v *Viewer, cursor image.Point) {
	st := &v.mapView
	if c, ok := fv.linkCellAt(cursor); ok {
		fv.Focus = &c
	} else if st.labelFocus != nil {
		// The pointer on a label of the last focus keeps it, so the label can be clicked.
		fv.Focus = st.labelFocus
		fv.links(v)
		if slices.ContainsFunc(fv.Labels, func(l MapLabel) bool { return cursor.In(l.Rect) }) {
			fv.note(v)
			return
		}
		fv.Focus, fv.Labels, fv.Arrows = nil, nil, nil
	}
	if fv.Focus == nil {
		fv.Focus = st.selected
	}
	if fv.Focus == nil {
		return
	}
	fv.links(v)
	fv.note(v)
}

// links lays out the focus's arrows and labels.
func (fv *FullMapView) links(v *Viewer) {
	c := *fv.Focus
	x, y := fv.CellPoint(c)
	at := image.Pt(int(x)+int(fv.Zoom)+8, int(y)-int(fv.Zoom)-LineHeight)
	for _, lk := range fv.Links {
		out := lk.From.Level == fv.Level && lk.From.Cell == c
		in := lk.To.Level == fv.Level && lk.To.Cell == c
		if !out && !in {
			continue
		}
		if out && in {
			continue // a link back onto its own cell shows nothing
		}
		other := lk.To
		if in {
			other = lk.From
		}
		if other.Level == fv.Level {
			if out {
				fv.Arrows = append(fv.Arrows, MapArrow{From: c, To: lk.To.Cell, Kind: lk.Kind})
			} else {
				fv.Arrows = append(fv.Arrows, MapArrow{From: lk.From.Cell, To: c, Kind: lk.Kind, In: true})
			}
			continue
		}
		text := fmt.Sprintf("%s (%d, %d)", v.levelName(other.Level), other.Cell[0], other.Cell[1])
		w := len(text)*CharWidth + 12
		fv.Labels = append(fv.Labels, MapLabel{Rect: image.Rect(at.X, at.Y, at.X+w, at.Y+LineHeight+2), Text: text,
			Kind: lk.Kind, In: in, To: other, Button: v.mapShowable(other.Level)})
		at.Y -= LineHeight + 4
	}
}

// note writes the focus's note and places it beside it.
func (fv *FullMapView) note(v *Viewer) {
	c := *fv.Focus
	mc, _ := fv.At(c)
	name := ""
	switch {
	case mc.Kind == MapDoor && mc.Open:
		name = "Door, open"
	case mc.Kind == MapDoor:
		name = "Door, closed"
	case mc.Symbol == MapSocket:
		name = fmt.Sprintf("Gem socket, %d of %d set", mc.Gems, mc.Places)
	case mc.Symbol == MapLever && mc.Open:
		name = "Lever, on"
	case mc.Symbol == MapLever:
		name = "Lever, off"
	case isRune(mc.Symbol):
		name = "Rune mark, " + MapKindNames[mc.Symbol]
	case mc.Symbol != MapNone:
		name = MapKindNames[mc.Symbol]
	}
	if name != "" {
		fv.Note = append(fv.Note, NoteLine{Text: name})
	}
	if mc.Symbol == MapMouth {
		for _, said := range mc.Heard {
			for _, line := range wrapText("\""+said+"\"", noteWidth) {
				fv.Note = append(fv.Note, NoteLine{Text: line, Said: true})
			}
		}
		if len(mc.Heard) == 0 {
			fv.Note = append(fv.Note, NoteLine{Text: "What it says is not known"})
		}
	}
	for _, t := range fv.Talkers {
		if t.Cell == c {
			fv.Note = append(fv.Note, NoteLine{Text: t.Name, Talker: true})
		}
	}
	var outs, ins int
	for _, lk := range fv.Links {
		out := lk.From.Level == fv.Level && lk.From.Cell == c
		in := lk.To.Level == fv.Level && lk.To.Cell == c
		if out == in {
			continue
		}
		place := v.levelName(lk.To.Level)
		if in {
			place = v.levelName(lk.From.Level)
		}
		if out {
			if lk.To.Level == fv.Level {
				place = fmt.Sprintf("here, %d cells", cellDistance(c, lk.To.Cell))
			} else {
				place += fmt.Sprintf(" (%d, %d)", lk.To.Cell[0], lk.To.Cell[1])
			}
			fv.Note = append(fv.Note, NoteLine{Text: "To " + place, Kind: lk.Kind})
			outs++
			continue
		}
		if lk.From.Level == fv.Level {
			place = fmt.Sprintf("here, %d cells", cellDistance(c, lk.From.Cell))
		}
		fv.Note = append(fv.Note, NoteLine{Text: "From " + place, Kind: lk.Kind})
		ins++
	}
	if outs > 0 {
		fv.Note = append(fv.Note, NoteLine{Text: "Right-click to follow"})
	} else if isLink(mc.Symbol) {
		fv.Note = append(fv.Note, NoteLine{Text: "Where it leads is not known"})
	}
	if len(fv.Note) == 0 {
		return
	}
	w := 0
	for _, l := range fv.Note {
		w = max(w, len(l.Text))
	}
	x, y := fv.CellPoint(c)
	size := image.Pt(w*CharWidth+16, len(fv.Note)*LineHeight+8)
	at := image.Pt(int(x)+int(fv.Zoom)+8, int(y)+int(fv.Zoom/2)+4)
	at.X = max(fv.Rect.Min.X, min(at.X, fv.Rect.Max.X-size.X))
	at.Y = max(fv.Rect.Min.Y, min(at.Y, fv.Rect.Max.Y-size.Y))
	fv.NoteRect = image.Rectangle{at, at.Add(size)}
}

// isRune reports whether symbol s is one of the game's rune marks.
func isRune(s int) bool { return s == MapThurisaz || s == MapGebo || s == MapRaido }

// wrapText breaks text into lines of at most n letters, at spaces.
func wrapText(text string, n int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case len([]rune(line))+1+len([]rune(word)) <= n:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// isLink reports whether symbol s is a way out: stairs, a teleporter, a pit or a trapdoor.
func isLink(s int) bool {
	return s == MapExit || s == MapTeleporter || s == MapHole || s == MapTrapdoor
}

// cellDistance is the distance from a to b in cells, the longer of the two axes.
func cellDistance(a, b content.Cell) int {
	return max(abs(a[0]-b[0]), abs(a[1]-b[1]))
}

// mapInput handles the input while the full map is open: it holds the world.
func (v *Viewer) mapInput(in Input) {
	st := &v.mapView
	fv := v.fullMap(in.Cursor)
	if in.pressed("M", KeyEscape) || fv == nil {
		v.closeFullMap()
		return
	}
	// The map's place as shown, so panning starts from it.
	place := func() {
		st.zoom, st.focus = fv.Zoom, fv.Center
	}
	switch {
	case in.pressed(KeyPageUp, KeyPageDown):
		var shown []string
		for _, id := range v.World.Visited() {
			if v.mapShowable(id) {
				shown = append(shown, id)
			}
		}
		if i := slices.Index(shown, st.level); len(shown) > 0 {
			d := 1
			if in.pressed(KeyPageUp) {
				d = -1
			}
			v.mapGo(shown[(max(i, 0)+d+len(shown))%len(shown)])
		}
		return
	case in.pressed(KeyHome):
		v.mapGo(v.World.Current().ID)
		return
	case in.pressed(KeyBackspace) && len(st.back) > 0:
		b := st.back[len(st.back)-1]
		st.back = st.back[:len(st.back)-1]
		st.level, st.zoom, st.focus, st.selected, st.arrival = b.level, b.zoom, b.focus, b.selected, b.arrival
		return
	}
	if in.Wheel != 0 && in.Cursor.In(fv.Area) {
		under := fv.unproject(float64(in.Cursor.X), float64(in.Cursor.Y))
		fit, _ := mapFit(fv.LevelMap, fv.Area)
		zoom := max(fit, min(fv.Zoom*math.Pow(mapZoomStep, in.Wheel), maxZoom(fit)))
		// The point under the pointer stays there.
		k := fv.Zoom / zoom
		st.zoom = zoom
		st.focus = [2]float64{under[0] - (under[0]-fv.Center[0])*k, under[1] - (under[1]-fv.Center[1])*k}
	}
	if in.Left.Pressed {
		p := in.Cursor
		st.press, st.dragging, st.last = &p, false, p
	}
	if st.press != nil && in.Left.Held {
		if !st.dragging && max(abs(in.Cursor.X-st.press.X), abs(in.Cursor.Y-st.press.Y)) > mapDragSlop {
			st.dragging = true
		}
		if st.dragging {
			place()
			d := in.Cursor.Sub(st.last)
			st.focus = [2]float64{st.focus[0] - float64(d.X)/fv.Zoom, st.focus[1] - float64(d.Y)/fv.Zoom}
		}
		st.last = in.Cursor
	}
	if in.Left.Released {
		if st.press != nil && !st.dragging {
			v.mapClick(fv, in.Cursor)
		}
		st.press, st.dragging = nil, false
	}
	if in.Right.Released {
		if c, ok := fv.linkCellAt(in.Cursor); ok {
			v.mapFollow(fv, c)
		}
	}
	// Remember the focus, so the pointer can go on to its labels.
	if after := v.fullMap(in.Cursor); after != nil && after.Focus != nil && len(after.Labels) > 0 {
		f := *after.Focus
		st.labelFocus = &f
	} else {
		st.labelFocus = nil
	}
}

// mapClick clicks the full map at pt: a label's button goes to its level, a cell with a feature
// or a link is selected, anything else clears the selection.
func (v *Viewer) mapClick(fv *FullMapView, pt image.Point) {
	st := &v.mapView
	for _, l := range fv.Labels {
		if pt.In(l.Rect) {
			if l.Button {
				v.mapArrive(fv, l.To)
			}
			return
		}
	}
	if c, ok := fv.linkCellAt(pt); ok {
		st.selected, st.arrival = &c, nil
		return
	}
	st.selected, st.arrival = nil, nil
}

// mapFollow follows the next recorded link from cell c of the shown level.
func (v *Viewer) mapFollow(fv *FullMapView, c content.Cell) {
	st := &v.mapView
	var outs []sim.Link
	for _, lk := range fv.Links {
		if lk.From.Level == fv.Level && lk.From.Cell == c && lk.To != lk.From {
			outs = append(outs, lk)
		}
	}
	if len(outs) == 0 {
		return
	}
	key := content.Destination{Level: fv.Level, Cell: c}
	lk := outs[st.follows[key]%len(outs)]
	if !v.mapShowable(lk.To.Level) {
		return
	}
	st.follows[key]++
	v.mapArrive(fv, lk.To)
}

// mapArrive shows destination to on the full map, centered on it at the zoom shown and selected,
// remembering where the map was for Backspace.
func (v *Viewer) mapArrive(fv *FullMapView, to content.Destination) {
	st := &v.mapView
	if !v.mapShowable(to.Level) {
		return
	}
	st.back = append(st.back, mapPlace{level: st.level, zoom: fv.Zoom, focus: fv.Center, selected: st.selected, arrival: st.arrival})
	c := to.Cell
	st.level, st.zoom, st.focus, st.selected, st.arrival = to.Level, fv.Zoom, cellMiddle(c), &c, &c
}

// mapGo shows level id fitted, with nothing selected.
func (v *Viewer) mapGo(id string) {
	st := &v.mapView
	st.level, st.zoom, st.selected, st.arrival = id, 0, nil, nil
}

// MinimapView is the minimap as it shows now: the level's map in a frame, centered on the
// player.
type MinimapView struct {
	*LevelMap
	// Frame is the frame on the screen, Scale its panel pixels' size; Plaque is under it, naming
	// the level.
	Frame  image.Rectangle
	Scale  int
	Plaque image.Rectangle
	MapFrame
}

// minimapCell is a minimap cell's width in panel pixels.
const minimapCell = 6

// Minimap returns the minimap as it shows now, or nil when it does not.
func (v *Viewer) Minimap() *MinimapView {
	if !v.MinimapShown() {
		return nil
	}
	m := v.Map()
	h := v.Hotbar()
	if m == nil || h == nil {
		return nil
	}
	s := h.Scale
	w := v.Screen.X / 4
	mv := &MinimapView{LevelMap: m, Scale: s}
	mv.Frame = image.Rect(Margin, Margin, Margin+w/s*s, Margin+w/2/s*s)
	mv.Area = mv.Frame.Inset(frame.Width * s)
	mv.Zoom = float64(minimapCell*s) / 2
	mv.Center = mapPoint(m.Player[0], m.Player[1])
	pw := (len(m.Name)*CharWidth + 2*Margin)
	mv.Plaque = image.Rect(mv.Frame.Min.X+(mv.Frame.Dx()-pw)/2, mv.Frame.Max.Y-s, mv.Frame.Min.X+(mv.Frame.Dx()+pw)/2,
		mv.Frame.Max.Y-s+LineHeight+4)
	return mv
}

// openMap opens the full map when the events show it.
func (v *Viewer) openMap(events []sim.Event) {
	for _, e := range events {
		if e.Kind == sim.EventShowMap && !v.MapOpen {
			v.openFullMap()
		}
	}
}
