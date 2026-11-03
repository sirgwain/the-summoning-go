package summoning

// Tests on the game's own files (AGENTS.md, rule 3). They run only when given them:
//
//   - SUMMONING_DATA is the directory of the game's data files; the pack is imported from it.
//   - SUMMONING_SAVES lists directories (separated as PATH is) holding the game's saves, one per
//     case: <case>/SAVES/dNNN.sav or <case>.sav. A case with no save is skipped.
//
// Each case loads a save made just before something happens in the game, does what makes it
// happen, and checks what the engine does.

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

var gameData struct {
	once sync.Once
	pack *content.Pack
	si   *SaveImporter
	err  error
}

// gamePack imports the pack from SUMMONING_DATA once, skipping the test without it.
func gamePack(t *testing.T) (*content.Pack, *SaveImporter) {
	t.Helper()
	dir := os.Getenv("SUMMONING_DATA")
	if dir == "" {
		t.Skip("SUMMONING_DATA is not set")
	}
	g := &gameData
	g.once.Do(func() {
		src, err := ReadSource(dir)
		if err != nil {
			g.err = err
			return
		}
		if g.pack, _, g.err = Import(src, Options{NoAudio: true}); g.err != nil {
			return
		}
		g.si, g.err = NewSaveImporter(src)
	})
	if g.err != nil {
		t.Fatal(g.err)
	}
	return g.pack, g.si
}

// gameSave finds case name's save in SUMMONING_SAVES, skipping the test without one.
func gameSave(t *testing.T, name string) string {
	t.Helper()
	for _, dir := range filepath.SplitList(os.Getenv("SUMMONING_SAVES")) {
		if dir == "" {
			continue
		}
		if m, _ := filepath.Glob(filepath.Join(dir, name, "SAVES", "d*.sav")); len(m) > 0 {
			return m[0]
		}
		if p := filepath.Join(dir, name+".sav"); fileExists(p) {
			return p
		}
	}
	t.Skipf("no save for %s in SUMMONING_SAVES", name)
	return ""
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// gameWorld loads case name's save into a world of the imported pack.
func gameWorld(t *testing.T, name string) *sim.World {
	t.Helper()
	p, si := gamePack(t)
	data, err := os.ReadFile(gameSave(t, name))
	if err != nil {
		t.Fatal(err)
	}
	sf, err := ReadSave(data)
	if err != nil {
		t.Fatal(err)
	}
	st, _, err := si.Import(p, sf, 1)
	if err != nil {
		t.Fatal(err)
	}
	w, err := sim.Restore(p, st)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// run updates w up to n times with the commands next gives each update, until stop holds for
// the events so far, and returns them; it fails when stop never holds.
func run(t *testing.T, w *sim.World, n int, next func() []input.Command, stop func([]sim.Event) bool) []sim.Event {
	t.Helper()
	var all []sim.Event
	for range n {
		var cmds []input.Command
		if next != nil {
			cmds = next()
		}
		all = append(all, w.Update(cmds)...)
		if stop(all) {
			return all
		}
	}
	t.Fatalf("not done after %d updates; on %s at %v", n, w.Current().ID, w.Player().Cell())
	return nil
}

func hasKind(kind string) func([]sim.Event) bool {
	return func(ev []sim.Event) bool {
		return slices.ContainsFunc(ev, func(e sim.Event) bool { return e.Kind == kind })
	}
}

func effectIn(ev []sim.Event) (sim.Event, bool) {
	i := slices.IndexFunc(ev, func(e sim.Event) bool { return e.Kind == sim.EventEffect })
	if i < 0 {
		return sim.Event{}, false
	}
	return ev[i], true
}

// toward is the direction that steps from cell a to its neighbor b.
func toward(a, b content.Cell) int {
	return slices.Index(sim.Dirs[:], [2]int{b[0] - a[0], b[1] - a[1]})
}

// walk walks along d each update.
func walk(d int) func() []input.Command {
	return func() []input.Command { return []input.Command{input.Walk{Direction: d}} }
}

// thingAt is the id of the living thing standing on cell c of the current level.
func thingAt(t *testing.T, w *sim.World, c content.Cell) string {
	t.Helper()
	for _, th := range w.Current().Things {
		if th.Cell() == c {
			return th.ID
		}
	}
	t.Fatalf("no thing at %v on %s", c, w.Current().ID)
	return ""
}

// talkThrough presses Enter each update, as the game takes it: past a text, or, waiting for a
// word, asking the first keyword offered (the button selected at first).
func talkThrough(w *sim.World) func() []input.Command {
	return func() []input.Command {
		t := w.Talk()
		switch {
		case t == nil:
			return nil
		case t.Line != nil:
			return []input.Command{input.Continue{}}
		}
		for _, k := range t.Offered {
			if k != "" {
				return []input.Command{input.Ask{Word: keywordWord(w.Pack(), k)}}
			}
		}
		return []input.Command{input.Leave{}}
	}
}

// keywordWord is the word that asks keyword id.
func keywordWord(p *content.Pack, id string) string {
	for _, kw := range p.Keywords.Keywords {
		if kw.ID == id {
			return inspect.Text(p, kw.Word)
		}
	}
	return id
}

// Walking into a pit, a fall or a trapdoor plays its view effect, and the level changes after it
// (rules.md, Exits; scenes.md, What plays them); teleporters play none and keep the level.
func TestGameTransitions(t *testing.T) {
	for _, c := range []struct {
		name     string
		into     content.Cell
		level    string
		at       content.Cell
		flipbook string
	}{
		{"pit_3", content.Cell{9, 29}, "level.04", content.Cell{9, 29}, effectID(viewFall)},
		{"fall_3", content.Cell{4, 30}, "level.04", content.Cell{4, 31}, effectID(viewFall)},
		{"fall_6", content.Cell{8, 23}, "level.10", content.Cell{46, 54}, effectID(viewClimbDown)},
		{"trapdoor_3", content.Cell{14, 29}, "level.04", content.Cell{14, 29}, effectID(viewFall)},
		{"teleport_3_a", content.Cell{2, 31}, "level.03", content.Cell{6, 31}, ""},
		{"teleport_3_b", content.Cell{13, 43}, "level.03", content.Cell{15, 46}, ""},
		{"teleport_6", content.Cell{1, 5}, "level.06", content.Cell{35, 22}, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := gameWorld(t, c.name)
			from := w.Player().Cell()
			if c.name == "trapdoor_3" {
				// Step off the plate the player stands on, away from the trapdoor, and back:
				// that opens it; then go on onto it.
				away := toward(c.into, from)
				run(t, w, 600, walk(away), func([]sim.Event) bool { return w.Player().Cell() != from })
				ev := run(t, w, 1200, walk((away+4)%8), func(ev []sim.Event) bool {
					_, ok := effectIn(ev)
					return ok || hasKind(sim.EventLevelChanged)(ev)
				})
				checkTransition(t, w, ev, c.flipbook, c.level, c.at)
				return
			}
			d := toward(from, c.into)
			if d < 0 {
				t.Fatalf("%v is not beside %v", c.into, from)
			}
			ev := run(t, w, 600, walk(d), func(ev []sim.Event) bool {
				_, ok := effectIn(ev)
				return ok || hasKind(sim.EventLevelChanged)(ev) || w.Player().Cell() == c.at
			})
			checkTransition(t, w, ev, c.flipbook, c.level, c.at)
		})
	}
}

// checkTransition checks a transition's events: the effect, and the level change after it.
func checkTransition(t *testing.T, w *sim.World, ev []sim.Event, flipbook, level string, at content.Cell) {
	t.Helper()
	e, ok := effectIn(ev)
	switch {
	case flipbook == "" && ok:
		t.Errorf("played %s", e.Detail)
	case flipbook != "" && (!ok || e.Detail != flipbook || e.Thing != sim.PlayerID):
		t.Errorf("effect %+v, want %s by the player", e, flipbook)
	case flipbook != "" && hasKind(sim.EventLevelChanged)(ev):
		t.Error("the level changed with the effect, not after it")
	}
	for i := 0; w.Waiting() && i < 2; i++ {
		w.Update(nil)
	}
	if w.Current().ID != level || w.Player().Cell() != at {
		t.Errorf("arrived on %s at %v, want %s at %v", w.Current().ID, w.Player().Cell(), level, at)
	}
}

// Conversations, the socket, Mending and a death blow play their view effects (scenes.md, What
// plays them).
func TestGameViewEffects(t *testing.T) {
	// first holds the events of the update start runs, which can already hold the effect.
	var first []sim.Event
	update := func(w *sim.World, cmds ...input.Command) { first = append(first, w.Update(cmds)...) }
	talkTo := func(c content.Cell) func(t *testing.T, w *sim.World) func() []input.Command {
		return func(t *testing.T, w *sim.World) func() []input.Command {
			update(w, input.Talk{Thing: thingAt(t, w, c)})
			return talkThrough(w)
		}
	}
	for _, c := range []struct {
		name    string
		effect  int
		updates int
		// start does what plays the effect, in an update of its own, and returns the commands
		// of the updates after.
		start func(t *testing.T, w *sim.World) func() []input.Command
		// by is the creature the effect is placed by: nil for the player, else its cell; byNone
		// is an effect placed by no creature (an object's conversation has no speaker).
		by     *content.Cell
		byNone bool
	}{
		{"ghost", 3, 3000, talkTo(content.Cell{31, 31}), &content.Cell{31, 31}, false},
		{"dunstan", 4, 3000, talkTo(content.Cell{14, 36}), &content.Cell{14, 36}, false},
		{"guards", 6, 3000, talkTo(content.Cell{3, 51}), &content.Cell{3, 51}, false},
		{"steelsmiter", 7, 3000, talkTo(content.Cell{59, 7}), &content.Cell{59, 7}, false},
		{"honorah", 11, 3000, talkTo(content.Cell{45, 5}), &content.Cell{45, 5}, false},
		{"kern", 13, 3000, talkTo(content.Cell{50, 28}), &content.Cell{50, 28}, false},
		{"murchad", 14, 3000, talkTo(content.Cell{30, 59}), &content.Cell{30, 59}, false},
		{"padrake", 15, 3000, talkTo(content.Cell{44, 31}), &content.Cell{44, 31}, false},
		{"rukn", 10, 3000, func(t *testing.T, w *sim.World) func() []input.Command {
			// Use the object in hand whose power starts Rukn's conversation.
			p := w.Player()
			hand := slices.IndexFunc(p.Slots[sim.SlotRightHand:sim.SlotRightHand+2], func(o *sim.Object) bool {
				return o != nil && w.Pack().Items[o.Item].Conversation != ""
			})
			if hand < 0 {
				t.Fatal("no object in hand starts a conversation")
			}
			update(w, input.Strike{Hand: hand})
			return talkThrough(w)
		}, nil, true},
		{"socket", 8, 600, func(t *testing.T, w *sim.World) func() []input.Command {
			// Step onto the socket, take the gem from the first shelf and put it down.
			from := w.Player().Cell()
			socket := content.Cell{from[0], from[1] - 1}
			run(t, w, 600, walk(toward(from, socket)), func([]sim.Event) bool { return w.Player().Cell() == socket })
			update(w, input.Swap{Slot: 0})
			sent := false
			return func() []input.Command {
				if sent {
					return nil
				}
				sent = true
				return []input.Command{input.PutDown{}}
			}
		}, nil, false},
		{"join_halves", 9, 1200, func(t *testing.T, w *sim.World) func() []input.Command {
			// Cast the school's selected spell, Mending, holding the staff's halves.
			p := w.Player()
			school := slices.IndexFunc(p.Magic.Selected[:], func(s string) bool { return s != "" })
			if school < 0 {
				t.Fatal("no spell selected")
			}
			update(w, input.Cast{School: school})
			return talkThrough(w)
		}, nil, false},
		{"special_attack", 12, 60 * 90, func(t *testing.T, w *sim.World) func() []input.Command {
			// Wait for the creature to come and strike.
			return nil
		}, nil, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := gameWorld(t, c.name)
			first = nil
			next := c.start(t, w)
			ev := first
			if _, ok := effectIn(ev); !ok {
				ev = append(ev, run(t, w, c.updates, next, func(ev []sim.Event) bool { _, ok := effectIn(ev); return ok })...)
			}
			e, _ := effectIn(ev)
			if want := effectID(c.effect); e.Detail != want {
				t.Errorf("played %s, want %s", e.Detail, want)
			}
			switch {
			case c.byNone && (e.Thing != "" || e.Feature != ""):
				t.Errorf("placed by %q, want no creature", e.Thing)
			case c.byNone:
			case c.by == nil && e.Thing != sim.PlayerID:
				t.Errorf("placed by %q, want the player", e.Thing)
			case c.by != nil && e.Cell != *c.by:
				t.Errorf("placed by %q at %v, want the creature at %v", e.Thing, e.Cell, *c.by)
			}
		})
	}
}

// TestGameDoorSight checks that the wall beside the vertical door at (44, 14) of level 1 follows
// the door: with it closed its upper floor half has no check direction, and with it open the half
// facing the doorway has one (world.md, Floor halves).
func TestGameDoorSight(t *testing.T) {
	p, _ := gamePack(t)
	l := p.Levels["level.01"]
	for _, ds := range l.DoorSight {
		if ds.Cell != (content.Cell{44, 15}) {
			continue
		}
		if len(ds.Doors) != 1 || ds.Doors[0] != (content.Cell{44, 14}) {
			t.Fatalf("doors %v", ds.Doors)
		}
		set := p.Walls[l.Walls]
		shape := l.Wall[44][15]
		open, closed := set.Shape(ds.As[0][shape]).Sight, set.Shape(ds.As[1][shape]).Sight
		if closed[0] != content.NoFrame || open[0] == content.NoFrame {
			t.Errorf("open %v, closed %v", open, closed)
		}
		return
	}
	t.Error("no door sight for (44, 15)")
}

// TestGameWallsOutOfSight stands the player in the room south of the vertical door at (20, 31) of
// level 1. The junctions on either side of the door join walls behind it that cannot be seen, so
// they are taken as shapes that end there (world.md, Walls out of sight); the level keeps its own.
func TestGameWallsOutOfSight(t *testing.T) {
	p, _ := gamePack(t)
	w, err := sim.New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	w.Update([]input.Command{input.GoTo{Level: "level.01"}})
	w.Update([]input.Command{input.PlacePlayer{Cell: content.Cell{26, 33}}})
	w.Update(nil)
	l := w.Current()
	for _, c := range []struct {
		cell      content.Cell
		own, took int
	}{{content.Cell{20, 30}, 5, 0xB}, {content.Cell{20, 33}, 9, 1}} {
		if !l.Visible(c.cell) || l.Wall[c.cell[0]][c.cell[1]] != c.own || l.Shape(c.cell) != c.took {
			t.Errorf("%v: visible %v, own %d, taken as %d; want %d as %d", c.cell, l.Visible(c.cell),
				l.Wall[c.cell[0]][c.cell[1]], l.Shape(c.cell), c.own, c.took)
		}
	}
}

// TestGameStory: the game's import plays the story in short on a tall screen, with
// Rowena's portrait in her panel, and at the end Rowena alone under the last page.
func TestGameStory(t *testing.T) {
	p, _ := gamePack(t)
	c := p.Creation
	s := p.Scenes[c.Story]
	if s == nil {
		t.Fatalf("no story %q", c.Story)
	}
	if w, h := s.Dims(); w != 320 || h != 400 {
		t.Errorf("story size %dx%d", w, h)
	}
	says, rowena, kept := 0, 0, 0
	for _, st := range s.Steps {
		if st.Say != nil {
			says++
			if st.Say.Portrait == rowenaID {
				rowena++
			}
		}
		if st.KeepText {
			kept++
		}
		if len(st.ByWeapon) != 0 && len(st.ByWeapon) != len(c.WeaponClasses) {
			t.Errorf("a by-weapon step of %d for %d classes", len(st.ByWeapon), len(c.WeaponClasses))
		}
	}
	if says != len(storyLines) || rowena != 4 || kept != 1 {
		t.Errorf("%d lines, %d Rowena's, %d clears keeping the page", says, rowena, kept)
	}
	if last := s.Steps[len(s.Steps)-2]; last.Picture == nil || last.MS != storyGoneMS {
		t.Errorf("Rowena alone: %+v", last)
	}
	if b := p.Sprites[p.Portraits[rowenaID].Image.Sprite]; b == nil || b.FrameSize != [2]int{48, 48} {
		t.Errorf("Rowena's portrait: %+v", b)
	}
}
