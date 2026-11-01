package viewer

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
)

func newViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := New(devpack.Build(), 1, "", 2, false, image.Pt(1280, 800))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// run runs a script on a new viewer of the dev pack and returns the viewer.
func run(t *testing.T, script string) *Viewer {
	t.Helper()
	s, err := ParseScript([]byte(script))
	if err != nil {
		t.Fatal(err)
	}
	v := newViewer(t)
	if err := NewRunner(s, t.TempDir()).Run(v); err != nil {
		t.Fatal(err)
	}
	return v
}

func feature(t *testing.T, v *Viewer, id string) inspect.FeatureState {
	t.Helper()
	for _, f := range v.Dump().World.Levels[v.Shown].Features {
		if f.ID == id {
			return f
		}
	}
	t.Fatalf("no feature %s", id)
	return inspect.FeatureState{}
}

func logged(v *Viewer, s string) bool {
	return slices.ContainsFunc(v.Events, func(e string) bool { return strings.Contains(e, s) })
}

func TestClickAndUseOpensDoor(t *testing.T) {
	v := run(t, `[
		{"at": 0, "click": [7, 3]},
		{"at": 1, "keys": ["U"]},
		{"at": 60}]`)
	if v.Selected == nil || *v.Selected != devpack.LeverA {
		t.Errorf("selected %v", v.Selected)
	}
	if f := feature(t, v, "lever.a"); !f.On {
		t.Errorf("lever %+v", f)
	}
	if f := feature(t, v, "door.a"); f.State != content.DoorOpen || f.Shows != "open" {
		t.Errorf("door %+v", f)
	}
	if !logged(v, "feature=lever.a on") {
		t.Errorf("log %q", v.Events)
	}
}

func TestClickSelectsThingAndShiftClickMovesIt(t *testing.T) {
	v := run(t, `[
		{"at": 0, "click": [9, 20]},
		{"at": 1, "click": [7, 9], "hold": ["shift"]}]`)
	if v.Thing != "guard" || *v.Selected != devpack.PlateB {
		t.Fatalf("thing %q in %v", v.Thing, v.Selected)
	}
	if f := feature(t, v, "plate.b"); !f.Down {
		t.Errorf("the guard did not press the plate: %+v", f)
	}
	if f := feature(t, v, "door.b"); f.State != content.DoorOpening {
		t.Errorf("door %+v", f)
	}
}

func TestDropChosenItemInSlot(t *testing.T) {
	// Items are in id order; the key is fourth.
	v := run(t, `[
		{"at": 0, "select": [7, 15], "keys": ["I"]},
		{"at": 1, "keys": ["I"]},
		{"at": 2, "keys": ["I"]},
		{"at": 3, "keys": ["O"]}]`)
	if d := v.Dump(); d.Item != devpack.ItemKey {
		t.Fatalf("item %q", d.Item)
	}
	if f := feature(t, v, "door.c"); f.State != content.DoorOpening {
		t.Errorf("door %+v", f)
	}
	// Shift steps back.
	v = run(t, `[{"at": 0, "keys": ["I"], "hold": ["Shift"]}]`)
	if v.Item != len(v.Items)-1 {
		t.Errorf("item %d of %d", v.Item, len(v.Items))
	}
}

func TestPauseAndStep(t *testing.T) {
	v := run(t, `[{"at": 0, "keys": ["Space"]}, {"at": 10}]`)
	if !v.Paused || v.World.Step() != 0 {
		t.Errorf("paused %v at step %d", v.Paused, v.World.Step())
	}
	v = run(t, `[{"at": 0, "keys": ["Space"]}, {"at": 5, "keys": ["N"]}, {"at": 6, "select": [7, 3], "keys": ["U"]}]`)
	if v.World.Step() != 2 {
		t.Errorf("step %d, want one for N and one for the command", v.World.Step())
	}
}

func TestChangeLevel(t *testing.T) {
	// Levels go in id order, wrapping: the study follows the hall.
	v := run(t, `[{"at": 0, "select": [7, 3], "keys": ["PageDown"]}]`)
	if v.Shown != devpack.Study || v.Selected != nil {
		t.Errorf("shown %s, selected %v", v.Shown, v.Selected)
	}
	v = run(t, `[{"at": 0, "keys": ["BracketRight"]}, {"at": 1, "keys": ["BracketLeft"]}]`)
	if v.Shown != devpack.Hall {
		t.Errorf("shown %s", v.Shown)
	}
}

func TestPanAndZoom(t *testing.T) {
	v := newViewer(t)
	o := v.Origin
	v.Update(Input{Held: map[Key]bool{KeyArrowRight: true}})
	if v.Origin.X != o.X+panSpeed {
		t.Errorf("origin %v from %v", v.Origin, o)
	}
	at := image.Pt(400, 300)
	under := v.ViewPoint(at)
	v.Update(Input{Cursor: at, Wheel: 1})
	if v.Scale != 3 || v.ViewPoint(at) != under {
		t.Errorf("scale %d, %v under the cursor, was %v", v.Scale, v.ViewPoint(at), under)
	}
	v.Update(Input{Pressed: map[Key]bool{"M": true}})
	if !v.Magnify {
		t.Error("not magnified")
	}
}

func TestDragDoesNotClick(t *testing.T) {
	v := newViewer(t)
	o := v.Origin
	v.Update(Input{Cursor: image.Pt(100, 100), Left: Button{Held: true, Pressed: true}})
	v.Update(Input{Cursor: image.Pt(140, 100), Left: Button{Held: true}})
	v.Update(Input{Cursor: image.Pt(140, 100), Left: Button{Released: true}})
	if v.Selected != nil {
		t.Errorf("a drag selected %v", v.Selected)
	}
	if v.Origin.X != o.X-20 {
		t.Errorf("origin %v from %v", v.Origin, o)
	}
}

func TestDumpFilesAndDeterminism(t *testing.T) {
	script := `[
		{"at": 0, "select": [7, 3], "keys": ["U"]},
		{"at": 30, "dump": "a/lever.json", "shot": "lever.png"}]`
	read := func() []byte {
		s, err := ParseScript([]byte(script))
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		r := NewRunner(s, dir)
		v := newViewer(t)
		var shots []string
		for !r.Done() {
			in, err := r.Before(v)
			if err != nil {
				t.Fatal(err)
			}
			v.Update(in)
			got, err := r.After(v)
			if err != nil {
				t.Fatal(err)
			}
			shots = append(shots, got...)
		}
		if len(shots) != 1 || shots[0] != filepath.Join(dir, "lever.png") {
			t.Errorf("shots %v", shots)
		}
		data, err := os.ReadFile(filepath.Join(dir, "a", "lever.json"))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	a, b := read(), read()
	if string(a) != string(b) {
		t.Error("the same script dumped different states")
	}
	var d Dump
	if err := json.Unmarshal(a, &d); err != nil {
		t.Fatal(err)
	}
	if d.Tick != 31 || d.World.Current != devpack.Hall || d.Selected == nil {
		t.Errorf("dump %+v", d)
	}
}

func TestParseScript(t *testing.T) {
	s, err := ParseScript([]byte(`[{"at": 5, "keys": ["u"]}, {"at": 1, "keys": ["digit3", "pageup"]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if s[0].At != 1 || s[0].Keys[0] != "Digit3" || s[0].Keys[1] != "PageUp" || s[1].Keys[0] != "U" {
		t.Errorf("script %+v", s)
	}
	for _, bad := range []string{
		`[{"at": 0, "keys": ["F11"]}]`,
		`[{"at": 0, "clik": [1, 1]}]`,
		`[{"at": -1}]`,
	} {
		if _, err := ParseScript([]byte(bad)); err == nil {
			t.Errorf("%s: no error", bad)
		}
	}
	v := newViewer(t)
	s, _ = ParseScript([]byte(`[{"at": 0, "select_thing": "nobody"}]`))
	if err := NewRunner(s, t.TempDir()).Run(v); err == nil {
		t.Error("selecting a missing thing did not fail")
	}
}

// TestPlayScript plays the dev pack by script, walking with held keys to lever A and using it;
// then by key presses picks a rock up (F), shelves it, takes it back with a click on its shelf and
// puts it down (G).
func TestPlayScript(t *testing.T) {
	// The player steps every n updates: 12 steps up and right (−u) from the start, then 4 up and
	// left (−v).
	n := newViewer(t).World.PlayerInterval()
	v := run(t, fmt.Sprintf(`[
		{"at": 0, "play": true},
		{"at": 1, "hold": ["W", "D"], "for": %d},
		{"at": %d, "hold": ["W", "A"], "for": %d},
		{"at": %d, "keys": ["Enter"]}
	]`, 12*n, 12*n+1, 4*n, 16*n+3))
	if !v.Play || v.SeeAll {
		t.Fatal("the script did not switch to playing")
	}
	p := v.World.Player()
	if c := p.Cell(); c != devpack.LeverA {
		t.Fatalf("walked to %v (%d, %d)", c, p.U, p.V)
	}
	if f := feature(t, v, "lever.a"); f.Shows != "on" {
		t.Errorf("Enter did not pull lever A: %s", f.Shows)
	}
	if got := v.Dump().World.Player; got == nil || got.Cell != p.Cell() {
		t.Errorf("the dump's player %+v", got)
	}

	v = newViewer(t)
	v.SetPlay(true)
	v.World.Update([]input.Command{input.PlacePlayer{Cell: devpack.RocksB[0]}})
	key := func(k Key, held ...Key) {
		in := Input{Held: map[Key]bool{k: true}, Pressed: map[Key]bool{k: true}}
		for _, h := range held {
			in.Held[h] = true
		}
		v.Update(in)
	}
	key("F")
	if p := v.World.Player(); p.Hand != nil || p.Slots[0] == nil || p.Slots[0].Item != devpack.ItemRock {
		t.Fatalf("F did not put the rock on shelf 0: hand %v, shelf %v", p.Hand, p.Slots[0])
	}
	r, _ := v.Hotbar().SlotRect(0)
	tapAt(v, r.Min.Add(r.Max).Div(2))
	if p := v.World.Player(); p.Hand == nil || p.Slots[0] != nil {
		t.Fatal("clicking shelf 0 did not take the rock off it")
	}
	key("G")
	if p := v.World.Player(); p.Hand != nil || len(v.World.Current().Objects[devpack.RocksB[0]]) != 1 {
		t.Error("G did not put the rock down")
	}
	if !slices.ContainsFunc(v.PanelLines(), func(s string) bool { return strings.HasPrefix(s, "Player at") }) {
		t.Error("the panel does not show the player")
	}
}

func TestScreenDirections(t *testing.T) {
	for _, c := range []struct{ x, y, d int }{{0, -1, 7}, {1, -1, 0}, {1, 0, 1}, {1, 1, 2}, {0, 1, 3}, {-1, 1, 4}, {-1, 0, 5}, {-1, -1, 6}} {
		if d := screenDirection(c.x, c.y); d != c.d {
			t.Errorf("screen (%d, %d) walks %d, want %d", c.x, c.y, d, c.d)
		}
	}
}

// TestViewFollowsPlayer checks the view stays put while the player walks in its middle, and
// scrolls with the player once it walks out of it.
func TestViewFollowsPlayer(t *testing.T) {
	v := newViewer(t)
	v.SetPlay(true)
	start := v.Origin
	walk := func(k Key, steps int) {
		for range steps * int(v.World.PlayerInterval()) {
			v.Update(Input{Held: map[Key]bool{k: true}, Pressed: map[Key]bool{}})
		}
	}
	walk("D", 2)
	if v.Origin != start {
		t.Fatalf("the view moved from %v to %v while the player walked in its middle", start, v.Origin)
	}
	walk("D", 30)
	p := v.World.Player()
	rel := drawlist.RefPoint(p.U, p.V).Sub(v.Origin)
	if v.Origin == start {
		t.Fatal("the view did not scroll after the player")
	}
	if buf := v.BufferSize(); rel.X > buf.X*3/4+8 {
		t.Errorf("the player at %v went past the view's middle (%v)", rel, buf)
	}
}

// TestToggleDebug hides the debug overlay with ~ and shows it again.
func TestToggleDebug(t *testing.T) {
	v := run(t, `[{"at": 0, "play": true, "keys": ["Backquote"]}]`)
	if !v.HideDebug || !v.PanelRect().Empty() || !v.LogRect().Empty() {
		t.Fatalf("~ did not hide the text: panel %v, log %v", v.PanelRect(), v.LogRect())
	}
	v.Update(Input{Held: map[Key]bool{KeyBackquote: true}, Pressed: map[Key]bool{KeyBackquote: true}})
	if v.HideDebug || v.PanelRect().Empty() {
		t.Error("~ did not show the text again")
	}
}

// TestGameKeys checks the play keys follow the game's layout.
func TestGameKeys(t *testing.T) {
	v := newViewer(t)
	v.SetPlay(true)
	press := func(k Key, held ...Key) []input.Command {
		in := Input{Held: map[Key]bool{k: true}, Pressed: map[Key]bool{k: true}}
		for _, h := range held {
			in.Held[h] = true
		}
		return v.playKeys(in)
	}
	for _, c := range []struct {
		key  Key
		held []Key
		want input.Command
	}{
		{KeySpace, nil, input.Strike{Hand: 0, Action: 0}},
		{KeySpace, []Key{KeyShift}, input.Strike{Hand: 0, Action: 1}},
		{"Q", nil, input.Strike{Hand: 1, Action: 0}},
		{"Q", []Key{KeyShift}, input.Strike{Hand: 1, Action: 1}},
		{"X", nil, input.Throw{Hand: 0}},
		{"Z", nil, input.Throw{Hand: 1}},
		{"F", nil, input.PickUp{}},
		{"G", nil, input.PutDown{}},
		{KeyEnter, nil, input.Activate{}},
		{KeyHome, nil, input.Walk{Direction: 6}},     // up and left: −v
		{KeyPageUp, nil, input.Walk{Direction: 0}},   // up and right: −u
		{KeyEnd, nil, input.Walk{Direction: 4}},      // down and left: +u
		{KeyPageDown, nil, input.Walk{Direction: 2}}, // down and right: +v
	} {
		got := press(c.key, c.held...)
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("%s with %v: %v, want %v", c.key, c.held, got, c.want)
		}
	}
}

// TestTalkWaitsForAKey talks to the study's sage with input as a shell reads it, every key in
// the snapshot and only some down: a line stays until a key goes down.
func TestTalkWaitsForAKey(t *testing.T) {
	v, err := New(devpack.Build(), 1, devpack.Study, 2, false, image.Pt(1280, 800))
	if err != nil {
		t.Fatal(err)
	}
	v.SetPlay(true)
	v.World.Update([]input.Command{input.PlacePlayer{Cell: content.Cell{1, 3}}})
	snapshot := func(down ...Key) Input {
		in := Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: image.Pt(-1, -1)}
		for _, k := range Keys {
			in.Held[k], in.Pressed[k] = false, false
		}
		for _, k := range down {
			in.Held[k], in.Pressed[k] = true, true
		}
		return in
	}
	v.Update(snapshot("E"))
	tk := v.World.Talk()
	if tk == nil || tk.Line == nil {
		t.Fatal("E did not start the sage's conversation")
	}
	for range 5 {
		v.Update(snapshot())
	}
	if tk.Line == nil || tk.Line.Text != "talk.sage.hello" {
		t.Fatalf("the first line went by without a key: %+v", tk.Line)
	}
	v.Update(snapshot(KeySpace))
	if tk.Line != nil {
		t.Errorf("Space did not go on: %+v", tk.Line)
	}
	if tv := v.Talk(); tv == nil || len(tv.Buttons) != 2 || tv.Buttons[0].Label != "name" {
		t.Errorf("the offered keywords: %+v", tv)
	}
}

// playIn returns a playing viewer of the dev pack on level with the player on cell c.
func playIn(t *testing.T, level string, c content.Cell) *Viewer {
	t.Helper()
	v, err := New(devpack.Build(), 1, level, 2, false, image.Pt(1280, 800))
	if err != nil {
		t.Fatal(err)
	}
	v.SetPlay(true)
	v.World.Update([]input.Command{input.PlacePlayer{Cell: c}})
	v.rebuild()
	v.Recenter()
	return v
}

// tapAt taps the left button at screen point pt.
func tapAt(v *Viewer, pt image.Point) {
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: pt, Left: Button{Pressed: true, Released: true}})
}

// TestSelectingScrollReadsIt takes the scroll from a shelf onto the pointer: that reads it,
// teaching its icons and showing its picture until the next key.
func TestSelectingScrollReadsIt(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	p := v.World.Player()
	v.World.Update([]input.Command{input.Drop{Cell: devpack.StudyStart, Item: devpack.ItemScroll}, input.PickUp{}})
	if p.Slots[0] == nil || p.Magic.Knows('I') {
		t.Fatalf("the scroll is not on shelf 0 unread: %+v, icons %q", p.Slots[0], p.Magic.Icons)
	}
	r, _ := v.Hotbar().SlotRect(0)
	tapAt(v, r.Min.Add(r.Max).Div(2))
	if p.Hand == nil || p.Hand.Item != devpack.ItemScroll || p.Magic.Icons != "BDFIJ" {
		t.Fatalf("selecting the scroll: hand %+v, icons %q", p.Hand, p.Magic.Icons)
	}
	if v.Reading == nil || *v.Reading != *v.Pack.Items[devpack.ItemScroll].Picture {
		t.Errorf("the scroll's picture is not shown: %+v", v.Reading)
	}
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{KeySpace: true}, Cursor: image.Pt(-1, -1)})
	if v.Reading != nil {
		t.Error("a key did not put the picture away")
	}
}

// TestTalkPointer moves the pointer over the sage: it becomes the speech bubble, and a click
// there opens the conversation, twice the size and centered in the view above the hotbar.
func TestTalkPointer(t *testing.T) {
	v := playIn(t, devpack.Study, content.Cell{2, 4})
	var at image.Point
	for _, c := range v.Cmds {
		if c.Thing == "sage" {
			at = v.ScreenPoint(image.Pt(c.X, c.Y)).Add(image.Pt(v.Scale*4, v.Scale*4))
		}
	}
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: at})
	if img, _, ok := v.Pointer(); !ok || img != *v.Pack.Pointers.Talk {
		t.Fatalf("the pointer over the sage is %+v (%v)", img, ok)
	}
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: image.Pt(5, 5)})
	if img, _, _ := v.Pointer(); img != v.Pack.Pointers.Arrow {
		t.Errorf("the pointer away from the sage is %+v", img)
	}
	tapAt(v, at)
	tv := v.Talk()
	if tv == nil {
		t.Fatal("a click on the sage did not talk")
	}
	bar := v.Hotbar().Rect
	mid := tv.Rect.Min.Add(tv.Rect.Max).Div(2)
	if tv.Scale != 2 || tv.Rect.Max.Y > bar.Min.Y || abs(mid.X-v.Screen.X/2) > 1 || abs(mid.Y-(bar.Min.Y-Margin)/2) > 1 {
		t.Errorf("the window is at %v, scale %d; the hotbar at %v", tv.Rect, tv.Scale, bar)
	}
}

// TestMouthSaysBesideIt clicks the hall's mouth: no conversation opens, and its message shows
// beside it until the player acts.
func TestMouthSaysBesideIt(t *testing.T) {
	v := playIn(t, devpack.Hall, content.Cell{devpack.Mouth[0], devpack.Mouth[1] - 1})
	f := v.World.Current().Feature("mouth")
	tl, ok := v.mouthTopLeft(f)
	if !ok {
		t.Fatal("the mouth has no look")
	}
	at := v.ScreenPoint(tl).Add(image.Pt(v.Scale*4, v.Scale*4))
	if id, ok := v.TalkTarget(at); !ok || id != "mouth" {
		t.Fatalf("the pointer over the mouth targets %q (%v)", id, ok)
	}
	tapAt(v, at)
	if v.World.Talk() != nil {
		t.Error("the mouth opened a conversation")
	}
	if v.Said == nil || !strings.Contains(v.Said.Text, "mechanisms") {
		t.Fatalf("the mouth said %+v", v.Said)
	}
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{"W": true}, Cursor: image.Pt(-1, -1)})
	if v.Said != nil {
		t.Error("acting did not put the message away")
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// TestQuickSaveAndLoad saves with F5, walks on, and loads with F8: the player is back where it
// was saved.
func TestQuickSaveAndLoad(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	v.SaveDir = t.TempDir()
	at := v.World.Player().Cell()
	press := func(k Key) {
		v.Update(Input{Held: map[Key]bool{k: true}, Pressed: map[Key]bool{k: true}, Cursor: image.Pt(-1, -1)})
	}
	press(KeyF5)
	if v.Notice != "Saved" {
		t.Fatalf("notice %q", v.Notice)
	}
	v.World.Update([]input.Command{input.PlacePlayer{Cell: devpack.StudyScroll}})
	press(KeyF8)
	if got := v.World.Player().Cell(); got != at || v.Notice != "Loaded" {
		t.Errorf("loaded at %v, want %v; notice %q", got, at, v.Notice)
	}
}

// TestLevelMap opens the map with M in the hall: it shows the seen cells and the player, and
// closes on the next key. The cellar refuses it with its message and hides the minimap.
func TestLevelMap(t *testing.T) {
	v := playIn(t, devpack.Hall, devpack.HallStart)
	press := func(k Key) {
		v.Update(Input{Held: map[Key]bool{k: true}, Pressed: map[Key]bool{k: true}, Cursor: image.Pt(-1, -1)})
	}
	if !v.MinimapShown() {
		t.Error("no minimap in the hall")
	}
	press("M")
	m := v.Map()
	if !v.MapOpen || m == nil || len(m.Cells) == 0 || m.Name != "The Hall of Mechanisms" || m.Player != devpack.HallStart {
		t.Fatalf("map open %v: %+v", v.MapOpen, m)
	}
	press(KeySpace)
	if v.MapOpen {
		t.Error("a key did not close the map")
	}
	v.World.Update([]input.Command{input.GoTo{Level: devpack.Cellar}})
	v.rebuild()
	if v.MinimapShown() {
		t.Error("the minimap shows in the cellar, which refuses the map")
	}
	press("M")
	if v.MapOpen || !slices.ContainsFunc(v.Events, func(s string) bool { return strings.Contains(s, "confuses") }) {
		t.Errorf("the cellar's refusal: open %v, events %v", v.MapOpen, v.Events)
	}
}
