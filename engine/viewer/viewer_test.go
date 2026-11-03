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

func TestUseFillsASlot(t *testing.T) {
	v := newViewer(t)
	v.Select(devpack.SlotC)
	pressKey(v, "U")
	if f := feature(t, v, "door.c"); f.State != content.DoorOpening {
		t.Errorf("door %+v", f)
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
	walk("D", 40)
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
	tv := v.Talk()
	if tv == nil || len(tv.Buttons) != 2 || tv.Buttons[0].Label != "name" {
		t.Fatalf("the offered keywords: %+v", tv)
	}
	// The first keyword is highlighted; Down moves to the second, then the field, and no further.
	if tv.Selected != 0 {
		t.Errorf("highlighted %d at first", tv.Selected)
	}
	for _, want := range []int{1, 2, 2} {
		v.Update(snapshot(KeyArrowDown))
		if got := v.Talk().Selected; got != want {
			t.Errorf("after Down: highlighted %d, want %d", got, want)
		}
	}
	// Typing highlights the field; Up then Enter asks the second keyword, not the typed word.
	v.Update(snapshot(KeyArrowUp, KeyArrowUp))
	in := snapshot()
	in.Chars = []rune("zz")
	v.Update(in)
	if v.Talk().Selected != 2 || v.Typed != "zz" {
		t.Fatalf("typing: highlighted %d, typed %q", v.Talk().Selected, v.Typed)
	}
	v.Update(snapshot(KeyArrowUp))
	second := tv.Buttons[1].Label
	v.Update(snapshot(KeyEnter))
	if !strings.Contains(strings.ToLower(strings.Join(tk.Transcript, "|")), "q: "+strings.ToLower(second)) {
		t.Errorf("Enter did not ask %q: %v", second, tk.Transcript)
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
	// The game's kinds show where seen, each named in the legend: a plate, a closed trapdoor, a
	// rune mark and a gem socket.
	kinds := map[content.Cell]int{}
	for _, c := range m.Cells {
		kinds[c.Cell] = c.Kind
	}
	for c, want := range map[content.Cell]int{devpack.PlateB: MapPlate, devpack.Trapdoor: MapTrapdoor, devpack.Mark: MapMark, devpack.Socket: MapSocket} {
		got, seen := kinds[c]
		if !seen {
			t.Errorf("%s at %v not seen", MapKindNames[want], c)
			continue
		}
		if got != want || !slices.Contains(m.Legend, want) {
			t.Errorf("%v shows %s, want %s; legend %v", c, MapKindNames[got], MapKindNames[want], m.Legend)
		}
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

// TestMessageLine shows what the player takes on the message line, just above the hotbar.
func TestMessageLine(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	v.World.Update([]input.Command{input.Drop{Cell: devpack.StudyStart, Item: devpack.ItemRock}})
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{"F": true}, Cursor: image.Pt(-1, -1)})
	if v.Notice != "Rock" {
		t.Errorf("taking the rock: notice %q", v.Notice)
	}
	r, bar := v.NoticeRect(), v.Hotbar().Rect
	if r.Max.Y != bar.Min.Y-Margin || r.Min.X+r.Max.X != v.Screen.X {
		t.Errorf("message line at %v, hotbar at %v", r, bar)
	}
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{"F": true}, Cursor: image.Pt(-1, -1)})
	if v.Notice != "Nothing to take." {
		t.Errorf("taking nothing: notice %q", v.Notice)
	}
}

// TestDiagonalGrace walks a pair of keys pressed and let go a moment apart only along their
// diagonal, and a single key held on straight after a moment.
func TestDiagonalGrace(t *testing.T) {
	v := newViewer(t)
	v.SetPlay(true)
	walks := func(held ...Key) []int {
		in := Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}}
		for _, k := range held {
			in.Held[k] = true
		}
		var dirs []int
		for _, c := range v.playKeys(in) {
			if w, ok := c.(input.Walk); ok {
				dirs = append(dirs, w.Direction)
			}
		}
		return dirs
	}
	up, upRight := screenDirection(0, -1), screenDirection(1, -1)
	// W a moment before D: nothing, then the diagonal.
	if got := walks("W"); got != nil {
		t.Errorf("W alone at first walked %v", got)
	}
	if got := walks("W", "D"); !slices.Equal(got, []int{upRight}) {
		t.Errorf("W and D: %v", got)
	}
	// D let go a moment before W: no straight step.
	for range walkGrace - 1 {
		if got := walks("W"); got != nil {
			t.Fatalf("W left over from the diagonal walked %v", got)
		}
	}
	// W held on walks straight up, and keeps walking.
	for range 3 {
		if got := walks("W"); !slices.Equal(got, []int{up}) {
			t.Fatalf("W held on: %v", got)
		}
	}
	// Turning from one straight direction to another walks at once.
	if got := walks("D"); !slices.Equal(got, []int{screenDirection(1, 0)}) {
		t.Errorf("turning to D: %v", got)
	}
	walks()
}

// TestPauseMessage pauses play with P, showing the pause message until the next key, which only
// resumes.
func TestPauseMessage(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	v.Update(Input{Held: map[Key]bool{"P": true}, Pressed: map[Key]bool{"P": true}, Cursor: image.Pt(-1, -1)})
	step := v.World.Step()
	for range 300 {
		v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: image.Pt(-1, -1)})
	}
	if !v.Paused || v.World.Step() != step || v.Notice != "Paused; press a key." || v.Ticks > v.NoticeUntil {
		t.Fatalf("paused %v, steps %d, notice %q until %d", v.Paused, v.World.Step()-step, v.Notice, v.NoticeUntil)
	}
	v.Update(Input{Held: map[Key]bool{"F": true}, Pressed: map[Key]bool{"F": true}, Cursor: image.Pt(-1, -1)})
	if v.Paused || v.Ticks <= v.NoticeUntil-1 && v.Notice != "" && v.Ticks < v.NoticeUntil {
		t.Errorf("a key did not resume: paused %v, notice %q", v.Paused, v.Notice)
	}
	if v.World.Step() != step {
		t.Error("the key that resumed also acted")
	}
}

func TestSelectionLinksAndMarks(t *testing.T) {
	v := newViewer(t)
	v.Select(devpack.LeverA)
	links := v.Links()
	if len(links) != 2 || !links[0].Out || links[0].Feature != "door.a" || links[0].Kind != inspect.LinkOn ||
		links[1].Feature != "door.a" || links[1].Kind != inspect.LinkOff {
		t.Fatalf("lever.a's links: %+v", links)
	}
	kinds := map[string]string{}
	for _, m := range v.Marks(links) {
		c := v.Cmds[m.Cmd]
		kinds[c.Feature] += string(m.Kind) + " "
	}
	if kinds["lever.a"] != "selected " || kinds["door.a"] != "on off " {
		t.Errorf("marks by feature: %q", kinds)
	}

	// The door is acted on by the lever.
	v.Select(devpack.DoorA)
	links = v.Links()
	if len(links) != 2 || links[0].Out || links[0].From != "lever.a" {
		t.Errorf("door.a's links: %+v", links)
	}
	v.HideDebug = true
	if v.Links() != nil || v.Marks(links) != nil {
		t.Error("links or marks with the debug overlay hidden")
	}
}

func TestHoverMarks(t *testing.T) {
	v := newViewer(t)
	// The pointer over the lever's cell marks the lever, for a click selects it.
	v.Update(Input{Cursor: v.CellPoint(devpack.LeverA)})
	if v.Hover == nil || *v.Hover != devpack.LeverA {
		t.Fatalf("hover %v, want %v", v.Hover, devpack.LeverA)
	}
	marks := v.Marks(nil)
	if len(marks) == 0 {
		t.Fatal("no marks for the hovered lever")
	}
	for _, m := range marks {
		if m.Kind != MarkHover || v.Cmds[m.Cmd].Feature != "lever.a" {
			t.Errorf("mark %+v of %+v", m, v.Cmds[m.Cmd])
		}
	}
	// Selected too, its selection's mark comes after the hover's, so it wins.
	v.Select(devpack.LeverA)
	marks = v.Marks(v.Links())
	if last := marks[len(marks)-1]; last.Kind != MarkSelected {
		t.Errorf("last mark %+v, want the selection's", last)
	}
}

func TestEnterUses(t *testing.T) {
	v := newViewer(t)
	v.Select(devpack.LeverA)
	pressKey(v, KeyEnter)
	if f := v.World.Current().Feature("lever.a"); !f.On {
		t.Error("Enter left the selected lever off")
	}
	// Enter presses a plate as U does.
	v.Select(devpack.PlateB)
	pressKey(v, KeyEnter)
	if !v.World.Current().Feature("plate.b").Down {
		t.Error("Enter did not press the plate")
	}
}

func TestHoverThingMarks(t *testing.T) {
	v := newViewer(t)
	// Find a screen point on an opaque pixel of a thing's sprite.
	for _, c := range v.Cmds {
		if c.Thing == "" {
			continue
		}
		size := v.Pack.Sprites[c.Sprite].FrameSize
		for y := 0; y < size[1]; y++ {
			for x := 0; x < size[0]; x++ {
				pt := image.Pt(c.X+x, c.Y+y)
				if id, _, ok := drawlist.ThingAt(v.Pack, v.Cmds, pt); !ok || id != c.Thing {
					continue
				}
				v.CenterOn(drawlist.CellAt(pt))
				v.Update(Input{Cursor: v.ScreenPoint(pt)})
				if v.HoverThing != c.Thing {
					t.Fatalf("hovering %s's pixel: hover thing %q", c.Thing, v.HoverThing)
				}
				for _, m := range v.Marks(nil) {
					if m.Kind != MarkHover || v.Cmds[m.Cmd].Thing != c.Thing {
						t.Errorf("mark %+v of %+v", m, v.Cmds[m.Cmd])
					}
				}
				return
			}
		}
	}
	t.Skip("no thing drawn on the dev pack's start level")
}

func TestTravelLinksOnlyFromTheTeleporter(t *testing.T) {
	v := newViewer(t)
	tp := v.World.Current().Feature("teleporter")
	v.Select(tp.Cell)
	travel := 0
	for _, lk := range v.Links() {
		if lk.Kind == inspect.LinkTravel && lk.Out {
			travel++
		}
	}
	if travel != 2 {
		t.Errorf("the teleporter shows %d destinations, want 2", travel)
	}
	v.Select(tp.Destinations[0])
	for _, lk := range v.Links() {
		if lk.Kind == inspect.LinkTravel {
			t.Errorf("the destination shows %+v", lk)
		}
	}
}

func TestUsePressesAndLiftsAPlate(t *testing.T) {
	v := newViewer(t)
	plate := v.World.Current().Feature("plate.b")
	door := func() string { return v.World.Current().Feature("door.b").State }
	v.Select(devpack.PlateB)
	pressKey(v, "U")
	if !plate.Down || plate.Counter != plate.Threshold {
		t.Fatalf("pressed: down %v, weight %d of %d", plate.Down, plate.Counter, plate.Threshold)
	}
	if s := door(); s != content.DoorOpening && s != content.DoorOpen {
		t.Errorf("door.b %s with the plate down", s)
	}
	pressKey(v, "U")
	if plate.Down || plate.Counter != 0 {
		t.Errorf("lifted: down %v, weight %d", plate.Down, plate.Counter)
	}

	// With something on it, lifting leaves that weight, and a plate it holds down stays down.
	v.World.Update([]input.Command{input.Weigh{Cell: devpack.PlateB, Weight: 30}})
	pressKey(v, "U")
	pressKey(v, "U")
	if plate.Counter != 30 || plate.Down {
		t.Errorf("lifted with 30 on it: down %v, weight %d", plate.Down, plate.Counter)
	}
	v.World.Update([]input.Command{input.Weigh{Cell: devpack.PlateB, Weight: plate.Threshold}})
	pressKey(v, "U")
	if !plate.Down || plate.Counter != 30+plate.Threshold {
		t.Errorf("held down: down %v, weight %d", plate.Down, plate.Counter)
	}
}

func TestResetFeature(t *testing.T) {
	v := newViewer(t)
	v.Select(devpack.LeverA)
	pressKey(v, "U")
	if !feature(t, v, "lever.a").On {
		t.Fatal("the lever did not turn on")
	}
	pressKey(v, "R")
	if feature(t, v, "lever.a").On {
		t.Error("the reset lever is still on")
	}
	// The door the lever opened is reset on its own, running nothing.
	v.Select(devpack.DoorA)
	pressKey(v, "R")
	if d := feature(t, v, "door.a"); d.State != content.DoorClosed || d.Stage != 0 {
		t.Errorf("reset door %s, stage %d", d.State, d.Stage)
	}

	// A plate the tools pressed loses that weight with the reset, without running its release.
	v.Select(devpack.PlateB)
	pressKey(v, "U")
	pressKey(v, "R")
	if p := feature(t, v, "plate.b"); p.Down || p.Counter != 0 {
		t.Errorf("reset plate: down %v, weight %d", p.Down, p.Counter)
	}
	if s := feature(t, v, "door.b").State; s != content.DoorOpening && s != content.DoorOpen {
		t.Errorf("the reset ran the plate's release: door.b %s", s)
	}
	pressKey(v, "U")
	if !feature(t, v, "plate.b").Down {
		t.Error("U after the reset did not press the plate")
	}
}

func TestFollowExitCentersOnTheEntrance(t *testing.T) {
	v := newViewer(t)
	exit := v.World.Current().Feature("exit.d")
	v.Select(exit.Cell)
	pressKey(v, "E")
	if v.Shown != exit.To.Level {
		t.Fatalf("shown %s, want %s", v.Shown, exit.To.Level)
	}
	want := v.Origin
	v.CenterOn(exit.To.Cell)
	if v.Origin != want {
		t.Errorf("origin %v, centered on the entrance %v", want, v.Origin)
	}
}
