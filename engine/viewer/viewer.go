package viewer

import (
	"fmt"
	"image"
	"sort"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

const (
	MaxScale    = 6
	panSpeed    = 6
	panSpeedFat = 24
	// clickSlop is how far, in screen pixels, the mouse may move between press and release and
	// still click rather than drag.
	clickSlop = 4

	// LogLines is how many events the log keeps.
	LogLines   = 12
	panelWidth = 456
	// CharWidth and LineHeight are the size of a character of the panel's text.
	CharWidth  = 6
	LineHeight = 16
	Margin     = 8
)

// Viewer is the viewer's state: the world it runs, the camera, the selection and the tools.
type Viewer struct {
	Pack  *content.Pack
	World *sim.World
	// Ticks counts updates, the clock palette effects run on.
	Ticks int64
	// Levels are the pack's levels in order; Shown is the one drawn.
	Levels []string
	Shown  string
	// Magnify draws floor objects with their inventory image.
	Magnify bool
	// Origin is the view pixel at the buffer's top-left; Scale the screen pixels per view pixel.
	Origin image.Point
	Scale  int
	// Screen is the size of the screen in pixels.
	Screen image.Point
	// Cmds draw the shown level, back to front, covering Bounds. Version counts their rebuilds,
	// so a shell knows when to reload them.
	Cmds    []drawlist.Cmd
	Bounds  image.Rectangle
	Version int

	// Hover is the cell under the cursor. Selected is the cell selected and Thing the thing in
	// it, if one was clicked.
	Hover, Selected *content.Cell
	Thing           string
	// Items and Things are the items to drop and templates to place; Item and Template the ones
	// chosen.
	Items, Things  []string
	Item, Template int
	Paused         bool
	// Events are the latest lines of the event log.
	Events []string

	// press is where the mouse button went down; drag where the view was last dragged from.
	press, drag *image.Point
	// wheel accumulates scrolling not yet turned into zoom steps.
	wheel float64
}

// New returns a viewer of pack p, running a world with seed, showing level (the start level if
// empty) at scale on a screen of the given size.
func New(p *content.Pack, seed uint64, level string, scale int, magnify bool, screen image.Point) (*Viewer, error) {
	w, err := sim.New(p, seed)
	if err != nil {
		return nil, err
	}
	v := &Viewer{
		Pack: p, World: w, Scale: max(1, min(scale, MaxScale)), Magnify: magnify, Screen: screen,
		Levels: sortedIDs(p.Levels), Items: sortedIDs(p.Items), Things: sortedIDs(p.Things),
	}
	if level != "" && level != w.Current().ID {
		w.Update([]input.Command{input.GoTo{Level: level}})
		if w.Current().ID != level {
			return nil, fmt.Errorf("no level %q", level)
		}
	}
	v.rebuild()
	return v, nil
}

func sortedIDs[V any](m map[string]V) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// rebuild rebuilds the draw commands from the world, recentering when the level changed.
func (v *Viewer) rebuild() {
	l := v.World.Current()
	v.Cmds = drawlist.Build(v.Pack, drawlist.SimScene(l), drawlist.Options{Magnify: v.Magnify})
	v.Bounds = drawlist.Bounds(l.Def)
	v.Version++
	if v.Shown != l.ID {
		v.Shown = l.ID
		v.Selected, v.Thing = nil, ""
		v.Recenter()
	}
}

// Recenter centers the view on the start cell on the start level, else on the level.
func (v *Viewer) Recenter() {
	if s := v.Pack.Manifest.Start; s.Level == v.Shown {
		v.CenterOn(s.Cell)
		return
	}
	center := v.Bounds.Min.Add(v.Bounds.Max).Div(2)
	v.Origin = center.Sub(v.BufferSize().Div(2))
}

// CenterOn centers the view on cell c.
func (v *Viewer) CenterOn(c content.Cell) {
	v.Origin = cellCenter(c).Sub(v.BufferSize().Div(2))
}

// cellCenter is the view pixel at the middle of cell c's ground.
func cellCenter(c content.Cell) image.Point {
	return drawlist.TileBox(c).Add(image.Pt(drawlist.TileSize/2, drawlist.TileSize/2))
}

// Select selects cell c, without a thing.
func (v *Viewer) Select(c content.Cell) { v.Selected, v.Thing = &c, "" }

// SelectThing selects a thing of the shown level and its cell.
func (v *Viewer) SelectThing(id string) error {
	t := v.World.Current().Thing(id)
	if t == nil {
		return fmt.Errorf("no thing %q on %s", id, v.Shown)
	}
	c := t.Cell()
	v.Selected, v.Thing = &c, id
	return nil
}

// BufferSize is the size in view pixels of the world buffer that covers the screen.
func (v *Viewer) BufferSize() image.Point {
	return image.Pt((v.Screen.X+v.Scale-1)/v.Scale, (v.Screen.Y+v.Scale-1)/v.Scale)
}

// ViewPoint returns the view pixel under screen point pt.
func (v *Viewer) ViewPoint(pt image.Point) image.Point { return v.Origin.Add(pt.Div(v.Scale)) }

// ScreenPoint returns the screen point of view pixel pt.
func (v *Viewer) ScreenPoint(pt image.Point) image.Point { return pt.Sub(v.Origin).Mul(v.Scale) }

// CellPoint returns the screen point at the middle of cell c.
func (v *Viewer) CellPoint(c content.Cell) image.Point {
	return v.ScreenPoint(cellCenter(c)).Add(image.Pt(v.Scale/2, v.Scale/2))
}

// Update applies one update's input: moves the camera, changes the selection and tools, and
// advances the world with the commands the input makes.
func (v *Viewer) Update(in Input) {
	v.Ticks++
	v.pan(in)
	v.scroll(in)
	cmds := v.mouse(in)

	switch {
	case in.pressed(KeyPageDown, KeyBracketRight):
		cmds = append(cmds, input.GoTo{Level: v.Levels[(v.levelIndex()+1)%len(v.Levels)]})
	case in.pressed(KeyPageUp, KeyBracketLeft):
		cmds = append(cmds, input.GoTo{Level: v.Levels[(v.levelIndex()+len(v.Levels)-1)%len(v.Levels)]})
	case in.pressed(KeyEqual, KeyNumpadAdd):
		v.zoomAt(1, v.Screen.Div(2))
	case in.pressed(KeyMinus, KeyNumpadSubtract):
		v.zoomAt(-1, v.Screen.Div(2))
	case in.pressed(KeyHome):
		v.Recenter()
	case in.pressed("M"):
		v.Magnify = !v.Magnify
		v.rebuild()
	}
	cmds = append(cmds, v.keys(in)...)

	if v.Paused && len(cmds) == 0 && !in.pressed("N") {
		return
	}
	events := v.World.Update(cmds)
	v.log(events)
	if len(events) > 0 {
		v.rebuild()
	}
	// The selection follows the selected thing, and drops it when it leaves the level.
	if v.Thing != "" {
		if t := v.World.Current().Thing(v.Thing); t != nil {
			c := t.Cell()
			v.Selected = &c
		} else {
			v.Thing = ""
		}
	}
}

func (v *Viewer) levelIndex() int {
	for i, id := range v.Levels {
		if id == v.Shown {
			return i
		}
	}
	return 0
}

// pan moves the view by the arrow and WASD keys.
func (v *Viewer) pan(in Input) {
	speed := panSpeed
	if in.held(KeyShift) {
		speed = panSpeedFat
	}
	for _, k := range []struct {
		keys   []Key
		dx, dy int
	}{
		{[]Key{KeyArrowLeft, "A"}, -1, 0},
		{[]Key{KeyArrowRight, "D"}, 1, 0},
		{[]Key{KeyArrowUp, "W"}, 0, -1},
		{[]Key{KeyArrowDown, "S"}, 0, 1},
	} {
		if in.held(k.keys...) {
			v.Origin = v.Origin.Add(image.Pt(k.dx*speed, k.dy*speed))
		}
	}
}

// mouse tracks the cell under the cursor, drags the view, and turns a left click into a
// selection, or with Shift into moving the selected thing to the clicked cell. A click is a
// release near its press; a press and release between two updates (a tap) counts too.
func (v *Viewer) mouse(in Input) []input.Command {
	cursor := in.Cursor
	if cursor.In(image.Rectangle{Max: v.Screen}) && !cursor.In(v.PanelRect()) {
		c := drawlist.CellAt(v.ViewPoint(cursor))
		v.Hover = &c
	} else {
		v.Hover = nil
	}

	if in.Left.Pressed || in.Right.Pressed {
		v.press, v.drag = &cursor, &cursor
	}
	if v.drag != nil && (in.Left.Held || in.Right.Held) {
		v.Origin = v.Origin.Sub(cursor.Sub(*v.drag).Div(v.Scale))
		moved := cursor.Sub(*v.drag).Div(v.Scale).Mul(v.Scale)
		next := v.drag.Add(moved)
		v.drag = &next
	}
	if in.Right.Released {
		v.press, v.drag = nil, nil
	}
	if !in.Left.Released {
		return nil
	}
	press := cursor
	if v.press != nil {
		press = *v.press
	}
	v.press, v.drag = nil, nil
	if d := cursor.Sub(press); d.X*d.X+d.Y*d.Y > clickSlop*clickSlop || cursor.In(v.PanelRect()) {
		return nil
	}
	pt := v.ViewPoint(cursor)
	c := drawlist.CellAt(pt)
	if in.held(KeyShift) && v.Thing != "" {
		return []input.Command{input.Move{Thing: v.Thing, Cell: c}}
	}
	v.Select(c)
	if id, _, ok := drawlist.ThingAt(v.Pack, v.Cmds, pt); ok {
		v.SelectThing(id)
	}
	return nil
}

// scroll zooms around the cursor by the scroll wheel, a step per whole wheel unit.
func (v *Viewer) scroll(in Input) {
	v.wheel += in.Wheel
	for ; v.wheel >= 1; v.wheel-- {
		v.zoomAt(1, in.Cursor)
	}
	for ; v.wheel <= -1; v.wheel++ {
		v.zoomAt(-1, in.Cursor)
	}
}

// zoomAt changes the scale, keeping the view pixel under the screen point at fixed.
func (v *Viewer) zoomAt(d int, at image.Point) {
	under := v.Origin.Add(at.Div(v.Scale))
	v.Scale = max(1, min(v.Scale+d, MaxScale))
	v.Origin = under.Sub(at.Div(v.Scale))
}

// cycle steps index i through n entries, backward with Shift.
func cycle(in Input, i, n int) int {
	if n == 0 {
		return 0
	}
	if in.held(KeyShift) {
		return (i + n - 1) % n
	}
	return (i + 1) % n
}

// keys turns the tool keys into commands on the selection.
func (v *Viewer) keys(in Input) []input.Command {
	switch {
	case in.pressed(KeySpace):
		v.Paused = !v.Paused
	case in.pressed("I"):
		v.Item = cycle(in, v.Item, len(v.Items))
	case in.pressed("C"):
		v.Template = cycle(in, v.Template, len(v.Things))
	}
	if v.Selected == nil {
		return nil
	}
	c := *v.Selected
	f := v.World.Current().FeatureAt(c)
	switch {
	case in.pressed("U"):
		return []input.Command{input.Use{Cell: c}}
	case in.pressed("T") && f != nil:
		return []input.Command{input.Switch{Feature: f.ID, Mode: input.SwitchToggle}}
	case in.pressed("E") && f != nil:
		return []input.Command{input.FollowExit{Feature: f.ID}}
	case in.pressed("O") && len(v.Items) > 0:
		return []input.Command{input.Drop{Cell: c, Item: v.Items[v.Item]}}
	case in.pressed("X"):
		return []input.Command{input.Remove{Cell: c}}
	case in.pressed("P") && len(v.Things) > 0:
		return []input.Command{input.Place{Cell: c, Thing: v.Things[v.Template]}}
	}
	if f != nil {
		hooks := inspect.Hooks(f)
		for i := range min(len(hooks), 9) {
			if in.pressed(Key(fmt.Sprintf("Digit%d", i+1))) {
				return []input.Command{input.Run{Procedure: hooks[i]}}
			}
		}
	}
	return nil
}

// log adds what happened to the event log, leaving out sounds.
func (v *Viewer) log(events []sim.Event) {
	for _, e := range events {
		var line string
		switch e.Kind {
		case sim.EventSound:
			continue
		case sim.EventMessage:
			line = fmt.Sprintf("%v says: %s", e.Cell, inspect.Message(v.Pack, e))
		default:
			line = strings.TrimPrefix(e.String(), e.Kind+" "+e.Level+" ")
			line = e.Kind + " " + line
		}
		v.Events = append(v.Events, fmt.Sprintf("%5d %s", v.World.Step(), line))
	}
	if n := len(v.Events); n > LogLines {
		v.Events = v.Events[n-LogLines:]
	}
}

// panelWidth is the inspector panel's width on the screen.
func (v *Viewer) panelWidth() int { return min(panelWidth, v.Screen.X/2) }

// PanelRect is where the inspector panel is drawn.
func (v *Viewer) PanelRect() image.Rectangle {
	w := v.panelWidth()
	return image.Rect(v.Screen.X-w, 0, v.Screen.X, len(v.PanelLines())*LineHeight+Margin)
}

// LogRect is where the event log is drawn, empty when there are no events.
func (v *Viewer) LogRect() image.Rectangle {
	if len(v.Events) == 0 {
		return image.Rectangle{}
	}
	h := len(v.Events)*LineHeight + Margin
	return image.Rect(0, v.Screen.Y-h, v.Screen.X-v.panelWidth(), v.Screen.Y)
}

// LogLines are the event log's lines, clipped to its width.
func (v *Viewer) LogLines() []string {
	n := v.LogRect().Dx()/CharWidth - 1
	lines := make([]string, len(v.Events))
	for i, s := range v.Events {
		lines[i] = clip(s, n)
	}
	return lines
}

// PanelLines are the inspector panel's lines, clipped to its width.
func (v *Viewer) PanelLines() []string {
	l := v.World.Current()
	status := fmt.Sprintf("%s (%d/%d)  step %d  scale %d", l.ID, v.levelIndex()+1, len(v.Levels), v.World.Step(), v.Scale)
	if v.Paused {
		status += "  PAUSED"
	}
	lines := []string{status}
	if v.Hover != nil {
		lines = append(lines, fmt.Sprintf("under the mouse: (%d, %d)", v.Hover[0], v.Hover[1]))
	}
	if len(v.Items) > 0 {
		lines = append(lines, "object (I): "+inspect.ItemName(v.Pack, v.Items[v.Item])+" ["+v.Items[v.Item]+"]")
	}
	if len(v.Things) > 0 {
		lines = append(lines, "thing (C):  "+v.Things[v.Template])
	}
	lines = append(lines, "")
	switch {
	case v.Selected == nil:
		lines = append(lines, "Click a cell or a thing to inspect it.")
	default:
		lines = append(lines, inspect.Cell(v.Pack, l, *v.Selected)...)
		if t := l.Thing(v.Thing); t != nil {
			lines = append(lines, "")
			lines = append(lines, inspect.Thing(v.Pack, t)...)
		}
	}
	lines = append(lines, "",
		"U use  T toggle  1-9 run hook  E follow exit",
		"O drop object  X take object  P place thing",
		"Shift+click move thing  Space pause  N step",
		"[ ] level  M magnify  Home recenter")
	n := v.panelWidth()/CharWidth - 1
	for i, s := range lines {
		lines[i] = clip(s, n)
	}
	return lines
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n && n > 3 {
		return string(r[:n-3]) + "..."
	}
	return s
}

// Dump is the viewer's state and the world's, for scripts and tests.
type Dump struct {
	Tick     int64               `json:"tick"`
	Shown    string              `json:"shown"`
	Selected *content.Cell       `json:"selected,omitempty"`
	Thing    string              `json:"thing,omitempty"`
	Item     string              `json:"item,omitempty"`
	Template string              `json:"template,omitempty"`
	Paused   bool                `json:"paused,omitempty"`
	Scale    int                 `json:"scale"`
	Origin   [2]int              `json:"origin"`
	Events   []string            `json:"events,omitempty"`
	World    *inspect.WorldState `json:"world"`
}

// Dump returns the viewer's state and the world's.
func (v *Viewer) Dump() *Dump {
	d := &Dump{
		Tick: v.Ticks, Shown: v.Shown, Selected: v.Selected, Thing: v.Thing, Paused: v.Paused,
		Scale: v.Scale, Origin: [2]int{v.Origin.X, v.Origin.Y}, Events: v.Events,
		World: inspect.State(v.World),
	}
	if len(v.Items) > 0 {
		d.Item = v.Items[v.Item]
	}
	if len(v.Things) > 0 {
		d.Template = v.Things[v.Template]
	}
	return d
}
