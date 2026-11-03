package viewer

import (
	"fmt"
	"image"
	"math"
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

	// Hover is the cell under the cursor and HoverThing the thing drawn there, if any: what a
	// click selects. Selected is the cell selected and Thing the thing in it, if one was clicked.
	Hover, Selected *content.Cell
	HoverThing      string
	Thing           string
	Paused          bool
	// Events are the latest lines of the event log.
	Events []string

	// SpellsOpen shows the spell panel, and Entered is the gesture entered on it. Typed is the
	// word typed in a conversation, and Reading the
	// picture of a scroll being read, shown above the hotbar until the next key or click.
	SpellsOpen bool
	Entered    string
	// bookScroll is the first spellbook row the spell panel shows, and bookWheel the wheel's
	// scrolling not yet turned into rows.
	bookScroll int
	bookWheel  float64
	// InventoryOpen shows the inventory panel; MapOpen the full level map.
	InventoryOpen, MapOpen bool
	// LogOpen shows the conversation log (convlog.go) and LogTyped is the search typed in it;
	// logChosen is the template whose conversations it shows, logSelected the match highlighted,
	// logScroll how far its pane is scrolled and logFocus the line a match chose.
	LogOpen     bool
	LogTyped    string
	logChosen   string
	logSelected int
	logScroll   int
	logFocus    *[2]int
	// SaveDir is where saves go, empty for none; Notice is the message line, shown above the
	// hotbar until the update NoticeUntil: the game's messages and the viewer's own.
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
	Said  *Said
	Typed string
	// TalkSelected is the conversation's highlighted keyword button, or the field after them
	// (TalkView.Selected); talkCursor is where the pointer was when it last chose one.
	TalkSelected int
	talkCursor   image.Point
	Reading      *content.Image
	// Front is the frontend while it shows, taking the input; Quit is set when it asks to quit.
	// Seed is the seed the world was made with, from which new games take theirs.
	Front *Frontend
	Quit  bool
	Seed  uint64
	// Showing is the view effect or sparkle playing, nil when none is; shows are the effect and
	// sparkle events waiting their turn (showing.go). Flipbooks are the pack's flipbooks and
	// Flipbook the one the tools play.
	Showing   *Showing
	shows     []sim.Event
	Flipbooks []string
	Flipbook  int

	// press is where the mouse button went down; drag where the view was last dragged from.
	press, drag *image.Point
	// wheel accumulates scrolling not yet turned into zoom steps.
	wheel float64
	// seen is where the view last saw the player, in view pixels, and on which level.
	seen      image.Point
	seenLevel string
	// deadFor counts the updates the player has been dead while playing.
	deadFor int
	// magnified is whether the world magnified floor objects at the last rebuild.
	magnified bool
	// weights is the weight the tools put on plates, by level and plate.
	weights map[[2]string]int
	// arrival is where an exit followed by the tools leads, centered on once the world has
	// followed it.
	arrival *content.Destination
	// walked is the screen direction the keys last walked, zero for none; walkWaiting is a
	// straight direction held without walking yet, for walkWait updates (walkNow).
	walked, walkWaiting [2]int
	walkWait            int
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
		Levels:    sortedIDs(p.Levels),
		Flipbooks: sortedIDs(p.Flipbooks), SeeAll: true, Seed: seed,
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
		scene.Visible, scene.Taken = nil, nil
	}
	v.magnified = v.World.Magnified()
	v.Cmds = drawlist.Build(v.Pack, scene, drawlist.Options{Magnify: v.Magnify || v.magnified})
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
	if v.Holding() {
		v.updateShowing(in)
		return
	}
	acted := in.anyPressed() || in.Left.Released || in.Right.Released
	// Paused while playing, the game waits for a key or a click, showing the pause message.
	if v.Play && v.Paused {
		if acted {
			v.Paused, v.NoticeUntil = false, v.Ticks
			v.press, v.drag = nil, nil
		}
		return
	}
	if v.Said != nil && (v.Ticks > v.Said.Until || acted) {
		v.Said = nil
	}
	if acted {
		// Esc that puts a scroll's picture away is used up, so it does not open the menu too.
		if v.Reading != nil && in.pressed(KeyEscape) {
			in = in.without(KeyEscape)
		}
		v.Reading = nil
	}
	// The conversation log holds the world while it is open.
	if v.LogOpen {
		v.logInput(in)
		return
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
	case in.pressed("B"):
		v.Flipbook = cycle(in, v.Flipbook, len(v.Flipbooks))
	case in.pressed("K") && len(v.Flipbooks) > 0:
		v.PlayFlipbook(v.Flipbooks[v.Flipbook])
		return
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
		sp := v.SpellPanel()
		if sp != nil {
			v.spellWheel(sp, in)
		}
		if sp != nil && in.Cursor.In(sp.Rect) && (in.Left.Released || in.Right.Released) {
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
	v.queueShows(events)
	v.log(events)
	v.hear(events)
	v.announce(events)
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
	if len(events) > 0 || len(v.Labels) > 0 || v.World.Magnified() != v.magnified {
		v.rebuild()
	}
	if a := v.arrival; a != nil {
		if v.Shown == a.Level {
			v.CenterOn(a.Cell)
		}
		v.arrival = nil
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
	onPanel := cursor.In(v.PanelRect())
	if cursor.In(image.Rectangle{Max: v.Screen}) && !onPanel && !onBar {
		pt := v.ViewPoint(cursor)
		c := drawlist.CellAt(pt)
		v.Hover = &c
		v.HoverThing, _, _ = drawlist.ThingAt(v.Pack, v.Cmds, pt)
	} else {
		v.Hover, v.HoverThing = nil, ""
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
		return d.X*d.X+d.Y*d.Y <= clickSlop*clickSlop && !onPanel
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
		return v.rightClick(true)
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

// keys turns the tool keys into commands on the selection (ToolBindings, which the help shows:
// change both together).
func (v *Viewer) keys(in Input) []input.Command {
	if in.pressed(KeySpace) {
		v.Paused = !v.Paused
	}
	if v.Selected == nil {
		return nil
	}
	c := *v.Selected
	f := v.World.Current().FeatureAt(c)
	use := in.pressed("U") || in.pressed(KeyEnter)
	switch {
	case use && f != nil && f.Kind == content.FeaturePlate:
		return v.pressPlate(f)
	case use && f != nil && f.Kind == content.FeatureSlot && f.Accepts != "":
		// A slot takes the item it accepts, as though it were put in.
		return []input.Command{input.Drop{Cell: c, Item: f.Accepts}}
	case use:
		return []input.Command{input.Use{Cell: c}}
	case in.pressed("R") && f != nil:
		// The weight the tools put on a plate goes with the reset.
		key := [2]string{v.Shown, f.ID}
		w := v.weights[key]
		delete(v.weights, key)
		return []input.Command{input.Reset{Feature: f.ID, Unweigh: w}}
	case in.pressed("T") && f != nil:
		return []input.Command{input.Switch{Feature: f.ID, Mode: input.SwitchToggle}}
	case in.pressed("E") && f != nil:
		if f.To != nil {
			to := *f.To
			v.arrival = &to
		}
		return []input.Command{input.FollowExit{Feature: f.ID}}
	case in.pressed("X"):
		return []input.Command{input.Remove{Cell: c}}
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

// pressPlate presses plate f with the weight it lacks to go down, or, when the tools pressed it,
// takes that weight off again, leaving what lies or stands on it. A plate that what is on it
// already holds down stays down.
func (v *Viewer) pressPlate(f *sim.Feature) []input.Command {
	key := [2]string{v.Shown, f.ID}
	if w := v.weights[key]; w > 0 {
		delete(v.weights, key)
		return []input.Command{input.Weigh{Cell: f.Cell, Weight: -w}}
	}
	w := f.Threshold - f.Counter
	if w <= 0 {
		v.notice(fmt.Sprintf("%s already bears %d of %d", f.ID, f.Counter, f.Threshold))
		return nil
	}
	if v.weights == nil {
		v.weights = map[[2]string]int{}
	}
	v.weights[key] = w
	return []input.Command{input.Weigh{Cell: f.Cell, Weight: w}}
}

// functionKeys are the function keys the viewer reads, F1 on.
const functionKeys = 8

// playKeys turns the keys into the player's commands, laid out as the game's (PlayBindings, which
// the help shows: change both together). A single walk key waits a moment before walking
// straight, in case it is half of a diagonal (walkNow).
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
	if v.walkNow(x, y) {
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
	case in.pressed("L"):
		v.OpenLog()
	case in.pressed(KeyEscape) && v.InventoryOpen:
		v.InventoryOpen = false
	case in.pressed("O", KeyEscape):
		cmds = append(cmds, input.Menu{})
	case in.pressed(KeyF5):
		v.SaveGame(QuickSave)
	case in.pressed(KeyF8):
		v.LoadGame(QuickSave)
	case in.pressed(KeyF9) && v.Audio != nil:
		v.cycleMusic()
	case in.pressed("H"):
		v.Recenter()
	case in.pressed("P"):
		v.pause()
	}
	return cmds
}

// pause pauses play until the next key or click, showing the pack's pause message meanwhile.
func (v *Viewer) pause() {
	v.Paused = true
	text := "Paused"
	if m := v.Pack.Messages; m != nil {
		if key, ok := m.Messages[content.MsgPaused]; ok {
			text = inspect.Text(v.Pack, key)
		}
	}
	v.Notice, v.NoticeUntil = text, math.MaxInt64
}

// walkGrace is how many updates a straight walk waits when it may be half of a diagonal: the
// second key of a pair pressed a moment after the first, or one let go a moment before the other.
const walkGrace = 5

// walkNow reports whether to walk toward screen direction (x, y) this update, given the keys
// held, and remembers the walk. A diagonal walks at once. A straight direction walks at once when
// it goes on from walking straight that way or turns from another; starting from a stand, or
// left over from a diagonal it is half of, it walks only once held walkGrace updates, so a pair
// of keys not pressed or released together walks only along the diagonal.
func (v *Viewer) walkNow(x, y int) bool {
	dir := [2]int{x, y}
	switch {
	case x == 0 && y == 0:
		v.walked, v.walkWait = [2]int{}, 0
		return false
	case x != 0 && y != 0, v.walked == dir:
	case v.walked == [2]int{} || v.walked[0] != 0 && v.walked[1] != 0 && (x != 0 && v.walked[0] == x || y != 0 && v.walked[1] == y):
		if v.walkWaiting != dir {
			v.walkWaiting, v.walkWait = dir, 0
		}
		if v.walkWait++; v.walkWait < walkGrace {
			return false
		}
	}
	v.walked, v.walkWait = dir, 0
	return true
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
	return v.panelRect(len(v.Panel()))
}

// panelRect is where an inspector panel of n lines is drawn.
func (v *Viewer) panelRect(n int) image.Rectangle {
	return image.Rect(v.Screen.X-v.panelWidth(), 0, v.Screen.X, n*LineHeight+Margin)
}

// Overlay is the debug overlay as a draw shows it, worked out once: the selection's links, the
// draw commands to outline, the inspector panel and the event log, each with where it goes. All
// are empty while the overlay is hidden.
type Overlay struct {
	Links     []SelectionLink
	Marks     []Mark
	Panel     []inspect.Line
	PanelRect image.Rectangle
	Log       []string
	LogRect   image.Rectangle
}

// Overlay returns the debug overlay.
func (v *Viewer) Overlay() Overlay {
	if v.HideDebug {
		return Overlay{}
	}
	links := v.Links()
	panel := v.panel(links)
	return Overlay{
		Links: links, Marks: v.Marks(links), Panel: panel, PanelRect: v.panelRect(len(panel)),
		Log: v.LogLines(), LogRect: v.LogRect(),
	}
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
func (v *Viewer) PanelLines() []string { return inspect.Texts(v.Panel()) }

// Panel is the inspector panel's lines, clipped to its width, with the kind of link each
// describes, if any: the selected feature's actions and destinations, and what acts on the
// selection.
func (v *Viewer) Panel() []inspect.Line { return v.panel(v.Links()) }

// panel is the inspector panel's lines, given the selection's links.
func (v *Viewer) panel(links []SelectionLink) []inspect.Line {
	l := v.World.Current()
	n := v.panelWidth()/CharWidth - 1
	status := fmt.Sprintf("%s (%d/%d)  step %d  scale %d", l.ID, v.levelIndex()+1, len(v.Levels), v.World.Step(), v.Scale)
	if v.Paused {
		status += "  PAUSED"
	}
	if v.Play {
		status += "  PLAYING"
	}
	var lines []inspect.Line
	text := func(ss ...string) {
		for _, s := range ss {
			lines = append(lines, inspect.Line{Text: s})
		}
	}
	text(status)
	if p := v.World.Player(); v.Play && p != nil {
		text(inspect.Player(v.Pack, p)...)
		text("")
		text(helpLines(PlayBindings, n)...)
		text(helpLines(CommonBindings, n)...)
		text(v.flipbookLine())
		if v.Selected != nil {
			text("")
			lines = append(lines, inspect.CellLines(v.Pack, l, *v.Selected)...)
		}
	} else {
		if v.Hover != nil {
			text(fmt.Sprintf("under the mouse: (%d, %d)", v.Hover[0], v.Hover[1]))
		}
		text("")
		if v.Selected == nil {
			text("Click a cell or a thing to inspect it.")
		} else {
			lines = append(lines, inspect.CellLines(v.Pack, l, *v.Selected)...)
			if t := l.Thing(v.Thing); t != nil {
				text("")
				lines = append(lines, inspect.ThingLines(v.Pack, t)...)
			}
			first := true
			for _, lk := range links {
				if lk.Out {
					continue
				}
				if first {
					text("", "Acted on by:")
					first = false
				}
				lines = append(lines, inspect.Line{Text: fmt.Sprintf("  %s %s", lk.From, lk.Text()), Kind: lk.Kind, Item: lk.Item})
			}
		}
		text("")
		text(helpLines(ToolBindings, n)...)
		text(helpLines(CommonBindings, n)...)
		text(v.flipbookLine())
	}
	for i := range lines {
		lines[i].Text = Clip(lines[i].Text, n)
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
	return &Dump{
		Tick: v.Ticks, Shown: v.Shown, Selected: v.Selected, Thing: v.Thing, Paused: v.Paused, Play: v.Play,
		Scale: v.Scale, Origin: [2]int{v.Origin.X, v.Origin.Y}, Events: v.Events, Labels: v.Labels,
		World: inspect.State(v.World),
	}
}
