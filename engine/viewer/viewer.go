package viewer

import (
	"fmt"
	"image"
	"sort"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/audio"
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
	// Play drives the player with the keys and keeps the view on it, instead of the tools.
	// SeeAll draws every cell rather than only what the player sees. HideDebug hides the panel and
	// the event log and the selection's and hover's outlines: the debug overlay.
	Play, SeeAll, HideDebug bool
	// Origin is the view pixel at the buffer's top-left; Scale the screen pixels per view pixel.
	Origin image.Point
	Scale  int
	// Screen is the size of the screen in pixels; Cursor where the mouse was at the last update.
	Screen image.Point
	Cursor image.Point
	// Cmds draw the shown level, back to front, covering Bounds. Version counts their rebuilds,
	// so a shell knows when to reload them.
	Cmds    []drawlist.Cmd
	Bounds  image.Rectangle
	Version int
	// Labels are the damage shown over things, rebuilt with Cmds.
	Labels []drawlist.Label

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

	// SpellsOpen shows the spell panel, and Entered is the gesture entered on it. Typed is the
	// word typed in a conversation, and Reading the
	// picture of a scroll being read, shown above the hotbar until the next key or click.
	SpellsOpen bool
	Entered    string
	// InventoryOpen shows the inventory panel; MapOpen the full level map.
	InventoryOpen, MapOpen bool
	// SaveDir is where saves go, empty for none; Notice is a line shown at the top of the screen
	// until the update NoticeUntil.
	SaveDir     string
	Notice      string
	NoticeUntil int64
	// Napping is whether N runs the world fast (nap.go); napHeldOff whether N, not released since
	// the last nap or refusal, may not start another yet.
	Napping    bool
	napHeldOff bool
	// Audio plays the world's sounds; nil plays none.
	Audio *audio.Mixer
	// Said is a mouth's message showing beside it, nil when none is.
	Said    *Said
	Typed   string
	Reading *content.Image
	// Front is the frontend while it shows, taking the input; Quit is set when it asks to quit.
	// Seed is the seed the world was made with, from which new games take theirs.
	Front *Frontend
	Quit  bool
	Seed  uint64

	// press is where the mouse button went down; drag where the view was last dragged from.
	press, drag *image.Point
	// wheel accumulates scrolling not yet turned into zoom steps.
	wheel float64
	// seen is where the view last saw the player, in view pixels, and on which level.
	seen      image.Point
	seenLevel string
	// deadFor counts the updates the player has been dead while playing.
	deadFor int
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
		SeeAll: true, Seed: seed,
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
	scene := drawlist.SimScene(v.World, l)
	if v.SeeAll {
		scene.Visible = nil
	}
	v.Cmds = drawlist.Build(v.Pack, scene, drawlist.Options{Magnify: v.Magnify})
	v.Labels = drawlist.Labels(v.Pack, scene)
	v.Bounds = drawlist.Bounds(l.Def)
	v.Version++
	if v.Shown != l.ID {
		v.Shown = l.ID
		v.Selected, v.Thing = nil, ""
		v.Recenter()
	}
}

// SetPlay switches between playing, with the keys driving the player and the view showing what
// it sees, and the tools. It does nothing to a pack without a player.
func (v *Viewer) SetPlay(play bool) {
	if play && v.World.Player() == nil {
		return
	}
	v.Play, v.SeeAll = play, !play
	v.rebuild()
	v.Recenter()
}

// follow scrolls the view after the player as the game does (docs/spec/player.md, View): it
// stays put while the player is in the middle half of the view, scrolls along an axis by as far
// as the player moved when a move leaves it outside, and centers on a player outside the view.
func (v *Viewer) follow() {
	p := v.World.Player()
	if p == nil {
		return
	}
	at := drawlist.RefPoint(p.U, p.V)
	moved := at.Sub(v.seen)
	v.seen = at
	buf := v.BufferSize()
	rel := at.Sub(v.Origin)
	if v.seenLevel != v.Shown || !rel.In(image.Rectangle{Max: buf}) {
		v.Recenter()
		return
	}
	lo, hi := buf.Div(4), buf.Mul(3).Div(4)
	if rel.X < lo.X && moved.X < 0 || rel.X > hi.X && moved.X > 0 {
		v.Origin.X += moved.X
	}
	if rel.Y < lo.Y && moved.Y < 0 || rel.Y > hi.Y && moved.Y > 0 {
		v.Origin.Y += moved.Y
	}
}

// Recenter centers the view on the player while playing, else on the start cell on the start
// level, else on the level.
func (v *Viewer) Recenter() {
	if p := v.World.Player(); v.Play && p != nil {
		v.seen, v.seenLevel = drawlist.RefPoint(p.U, p.V), v.Shown
		v.Origin = v.seen.Sub(v.BufferSize().Div(2))
		return
	}
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
	v.Cursor = in.Cursor
	if v.Front != nil {
		v.frontUpdate(in)
		return
	}
	acted := in.anyPressed() || in.Left.Released || in.Right.Released
	if v.Said != nil && (v.Ticks > v.Said.Until || acted) {
		v.Said = nil
	}
	if acted {
		v.Reading = nil
	}
	if cmds, ok := v.modal(in); ok {
		v.updateWorld(cmds)
		return
	}
	v.pan(in)
	v.scroll(in)
	cmds := v.mouse(in)

	switch {
	case in.pressed(KeyEqual, KeyNumpadAdd):
		v.zoomAt(1, v.Screen.Div(2))
	case in.pressed(KeyMinus, KeyNumpadSubtract):
		v.zoomAt(-1, v.Screen.Div(2))
	case in.pressed("M") && !v.Play:
		v.Magnify = !v.Magnify
		v.rebuild()
	case in.pressed(KeyF12):
		v.SetPlay(!v.Play)
	case in.pressed(KeyBackquote):
		v.HideDebug = !v.HideDebug
	case in.pressed("V"):
		v.SeeAll = !v.SeeAll
		v.rebuild()
	}
	if v.Play {
		cmds = append(cmds, v.playKeys(in)...)
	} else {
		cmds = append(cmds, v.levelKeys(in)...)
		cmds = append(cmds, v.keys(in)...)
	}

	if v.nap(in, cmds) {
		return
	}
	// Paused, the tools still send their commands; play stops altogether. N steps either.
	if v.Paused && (v.Play || len(cmds) == 0) && !in.pressed("N") {
		return
	}
	v.updateWorld(cmds)
}

// modal handles input while a conversation, a scroll's picture or the spell panel takes it,
// returning the commands and true when one does.
func (v *Viewer) modal(in Input) ([]input.Command, bool) {
	switch {
	case v.World.Talk() != nil:
		if l := v.World.Talk().Line; l != nil && l.Kind == sim.LineMenu {
			v.openMenu(l.Menu)
			return nil, true
		}
		return v.talkInput(in), true
	case v.MapOpen:
		if in.anyPressed() || in.Left.Released || in.Right.Released {
			v.MapOpen = false
		}
		return nil, true
	case v.SpellsOpen:
		cmds := v.spellKeys(in)
		if sp := v.SpellPanel(); sp != nil && in.Cursor.In(sp.Rect) && (in.Left.Released || in.Right.Released) {
			v.press, v.drag = nil, nil
			return append(cmds, v.spellClick(sp, in.Cursor, in.Right.Released)...), true
		}
		return append(cmds, v.mouse(in)...), true
	}
	return nil, false
}

// updateWorld runs the world's update with cmds and brings the view up to date.
func (v *Viewer) updateWorld(cmds []input.Command) []sim.Event {
	events := v.World.Update(cmds)
	v.log(events)
	v.hear(events)
	v.sound(events)
	v.autosave(events)
	v.transcribe(events)
	v.openMap(events)
	for _, e := range events {
		if e.Kind == sim.EventMagic && e.Detail == "read" {
			if it := v.Pack.Items[e.Item]; it != nil && it.Picture != nil {
				v.Reading = it.Picture
			}
		}
	}
	// Shown damage runs out without an event, so a view showing some rebuilds each update.
	if len(events) > 0 || len(v.Labels) > 0 {
		v.rebuild()
	}
	if v.Play {
		v.follow()
	}
	v.afterPlay()
	// The selection follows the selected thing, and drops it when it leaves the level.
	if v.Thing != "" {
		if t := v.World.Current().Thing(v.Thing); t != nil {
			c := t.Cell()
			v.Selected = &c
		} else {
			v.Thing = ""
		}
	}
	return events
}

func (v *Viewer) levelIndex() int {
	for i, id := range v.Levels {
		if id == v.Shown {
			return i
		}
	}
	return 0
}

// pan moves the view by the arrow and WASD keys, unless they walk the player.
func (v *Viewer) pan(in Input) {
	if v.Play {
		return
	}
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

// mouse tracks the cell under the cursor, drags the view, and turns clicks into commands. A
// left click selects a cell or a thing, or with Shift moves the selected thing to the clicked
// cell. While playing, clicks on the hotbar and on the player work objects with the hand instead
// (slots.go), and a right click puts the hand's object on a shelf. A click is a release near its
// press; a press and release between two updates (a tap) counts too.
func (v *Viewer) mouse(in Input) []input.Command {
	cursor := in.Cursor
	bar := v.Hotbar()
	inv := v.Inventory()
	onInv := inv != nil && cursor.In(inv.Rect)
	onBar := bar != nil && cursor.In(bar.Rect) || onInv
	if cursor.In(image.Rectangle{Max: v.Screen}) && !cursor.In(v.PanelRect()) && !onBar {
		c := drawlist.CellAt(v.ViewPoint(cursor))
		v.Hover = &c
	} else {
		v.Hover = nil
	}

	if in.Left.Pressed || in.Right.Pressed {
		v.press, v.drag = &cursor, &cursor
		if onBar {
			v.drag = nil
		}
	}
	if v.drag != nil && (in.Left.Held || in.Right.Held) {
		v.Origin = v.Origin.Sub(cursor.Sub(*v.drag).Div(v.Scale))
		moved := cursor.Sub(*v.drag).Div(v.Scale).Mul(v.Scale)
		next := v.drag.Add(moved)
		v.drag = &next
	}
	clicked := func() bool {
		press := cursor
		if v.press != nil {
			press = *v.press
		}
		v.press, v.drag = nil, nil
		d := cursor.Sub(press)
		return d.X*d.X+d.Y*d.Y <= clickSlop*clickSlop && !cursor.In(v.PanelRect())
	}
	if in.Right.Released {
		switch {
		case !clicked() || !v.Play || bar == nil:
			return nil
		case onInv:
			return v.inventoryClick(inv, cursor, true, false)
		case onBar:
			return bar.click(v, cursor, true)
		}
		return v.rightClick(v.onPlayer(v.ViewPoint(cursor)))
	}
	if !in.Left.Released || !clicked() {
		return nil
	}
	if onInv {
		return v.inventoryClick(inv, cursor, false, in.held(KeyShift))
	}
	if onBar {
		if cursor.In(bar.Screen(bar.PortraitRect)) {
			v.ToggleInventory()
			return nil
		}
		if b, ok := slotAt(bar.Slots, bar.at(cursor)); ok && in.held(KeyShift) {
			return []input.Command{input.Read{Slot: b.Slot}}
		}
		return bar.click(v, cursor, false)
	}
	if id, ok := v.TalkTarget(cursor); ok && v.World.Player().Hand == nil {
		return []input.Command{input.Talk{Thing: id}}
	}
	pt := v.ViewPoint(cursor)
	if v.Play && v.onPlayer(pt) {
		if in.held(KeyShift) && v.World.Player().Hand != nil {
			return []input.Command{input.Read{Slot: input.SlotHand}}
		}
		return v.clickPlayer()
	}
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

// functionKeys are the function keys the viewer reads, F1 on.
const functionKeys = 8

// playKeys turns the keys into the player's commands, laid out as the game's are:
//
//   - WASD or the arrows walk, two held for a diagonal; Home, PgUp, End and PgDn walk the
//     diagonals.
//   - Space acts with the right hand (Shift+Space its second action) and X throws from it; Q acts
//     with the left hand (Shift+Q) and Z throws from it.
//   - F takes from the floor, G puts down, Enter pulls the lever underfoot or uses a key on the
//     keyhole underfoot, E talks, H recenters the view, P pauses.
//   - I or Tab opens and closes the inventory panel, as clicking the portrait does.
//   - F5 saves to the quick save and F8 loads it; F9 cycles the sound; M shows the level map.
func (v *Viewer) playKeys(in Input) []input.Command {
	if in.held(KeyControl) {
		return nil
	}
	var cmds []input.Command
	var x, y int
	if in.held(KeyArrowLeft, "A", KeyHome, KeyEnd) {
		x--
	}
	if in.held(KeyArrowRight, "D", KeyPageUp, KeyPageDown) {
		x++
	}
	if in.held(KeyArrowUp, "W", KeyHome, KeyPageUp) {
		y--
	}
	if in.held(KeyArrowDown, "S", KeyEnd, KeyPageDown) {
		y++
	}
	if x != 0 || y != 0 {
		cmds = append(cmds, input.Walk{Direction: screenDirection(x, y)})
	}
	action := 0
	if in.held(KeyShift) {
		action = 1
	}
	for _, k := range []struct {
		key Key
		cmd input.Command
	}{
		{KeySpace, input.Strike{Hand: 0, Action: action}},
		{"Q", input.Strike{Hand: 1, Action: action}},
		{"X", input.Throw{Hand: 0}},
		{"Z", input.Throw{Hand: 1}},
		{"F", input.PickUp{}},
		{"G", input.PutDown{}},
		{KeyEnter, input.Activate{}},
		{"E", input.Talk{}},
		{"C", input.Talk{}},
	} {
		if in.pressed(k.key) {
			cmds = append(cmds, k.cmd)
		}
	}
	cmds = append(cmds, v.castKeys(in)...)
	switch {
	case in.pressed("I", KeyTab):
		v.ToggleInventory()
	case in.pressed("M"):
		cmds = append(cmds, input.ShowMap{})
	case in.pressed("O"):
		cmds = append(cmds, input.Menu{})
	case in.pressed(KeyF5):
		v.SaveGame(QuickSave)
	case in.pressed(KeyF8):
		v.LoadGame(QuickSave)
	case in.pressed(KeyF9) && v.Audio != nil:
		v.cycleMusic()
	case in.pressed(KeyEscape) && v.InventoryOpen:
		v.InventoryOpen = false
	case in.pressed("H"):
		v.Recenter()
	case in.pressed("P"):
		v.Paused = !v.Paused
	}
	return cmds
}

// levelKeys are the tools' keys that change level and recenter: Page Up and Page Down or [ and ],
// and Home.
func (v *Viewer) levelKeys(in Input) []input.Command {
	switch {
	case in.pressed(KeyPageDown, KeyBracketRight):
		return []input.Command{input.GoTo{Level: v.Levels[(v.levelIndex()+1)%len(v.Levels)]}}
	case in.pressed(KeyPageUp, KeyBracketLeft):
		return []input.Command{input.GoTo{Level: v.Levels[(v.levelIndex()+len(v.Levels)-1)%len(v.Levels)]}}
	case in.pressed(KeyHome):
		v.Recenter()
	}
	return nil
}

// screenDirection returns the table direction that walks toward screen direction (x, y), each
// −1, 0 or 1 and not both 0: the player's facing p, clockwise from screen up, walks table
// direction (p + 7) mod 8 (docs/spec/player.md, Facing and walking).
func screenDirection(x, y int) int {
	facings := map[[2]int]int{
		{0, -1}: 0, {1, -1}: 1, {1, 0}: 2, {1, 1}: 3, {0, 1}: 4, {-1, 1}: 5, {-1, 0}: 6, {-1, -1}: 7,
	}
	return (facings[[2]int{x, y}] + 7) % 8
}

// log adds what happened to the event log, leaving out sounds and music.
func (v *Viewer) log(events []sim.Event) {
	for _, e := range events {
		var line string
		switch e.Kind {
		case sim.EventSound, sim.EventMusic:
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
	if v.HideDebug {
		return image.Rectangle{}
	}
	w := v.panelWidth()
	return image.Rect(v.Screen.X-w, 0, v.Screen.X, len(v.PanelLines())*LineHeight+Margin)
}

// LogRect is where the event log is drawn, empty when there are no events.
func (v *Viewer) LogRect() image.Rectangle {
	if len(v.Events) == 0 || v.HideDebug {
		return image.Rectangle{}
	}
	h := len(v.Events)*LineHeight + Margin
	bottom := v.Screen.Y
	if bar := v.Hotbar(); bar != nil {
		bottom = bar.Rect.Min.Y - Margin
	}
	return image.Rect(0, bottom-h, v.Screen.X-v.panelWidth(), bottom)
}

// LogLines are the event log's lines, clipped to its width.
func (v *Viewer) LogLines() []string {
	n := v.LogRect().Dx()/CharWidth - 1
	lines := make([]string, len(v.Events))
	for i, s := range v.Events {
		lines[i] = Clip(s, n)
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
	if v.Play {
		status += "  PLAYING"
	}
	lines := []string{status}
	if p := v.World.Player(); v.Play && p != nil {
		lines = append(lines, inspect.Player(v.Pack, p)...)
		lines = append(lines, "", "WASD/arrows walk (two: diagonal), Home PgUp End PgDn",
			"Space right hand (Shift: 2nd)  X throw right",
			"Q left hand (Shift: 2nd)  Z throw left",
			"F take  G put down  Enter lever/key  E/C talk",
			"F1-F4 cast (Shift: next spell)  F10 spells",
			"Shift+click read",
			"Click the player: take / put down; click a slot:",
			"swap with the hand; right click: hand to a shelf",
			"I/Tab or click the portrait: inventory",
			"H recenter  P pause  N step  M magnify",
			"V see all  F12 tools  ~ debug")
		if v.Selected != nil {
			lines = append(lines, "")
			lines = append(lines, inspect.Cell(v.Pack, l, *v.Selected)...)
		}
		n := v.panelWidth()/CharWidth - 1
		for i, s := range lines {
			lines[i] = Clip(s, n)
		}
		return lines
	}
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
		"[ ] level  M magnify  Home recenter  V see all",
		"F12 play  ~ hide the debug overlay")
	n := v.panelWidth()/CharWidth - 1
	for i, s := range lines {
		lines[i] = Clip(s, n)
	}
	return lines
}

// Clip shortens s to n characters, ending it with "..." when it is cut.
func Clip(s string, n int) string {
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
	Play     bool                `json:"play,omitempty"`
	Scale    int                 `json:"scale"`
	Origin   [2]int              `json:"origin"`
	Events   []string            `json:"events,omitempty"`
	Labels   []drawlist.Label    `json:"labels,omitempty"`
	World    *inspect.WorldState `json:"world"`
}

// Dump returns the viewer's state and the world's.
func (v *Viewer) Dump() *Dump {
	d := &Dump{
		Tick: v.Ticks, Shown: v.Shown, Selected: v.Selected, Thing: v.Thing, Paused: v.Paused, Play: v.Play,
		Scale: v.Scale, Origin: [2]int{v.Origin.X, v.Origin.Y}, Events: v.Events, Labels: v.Labels,
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
