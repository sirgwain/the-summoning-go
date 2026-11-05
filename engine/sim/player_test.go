package sim

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// Wall shapes of the walk pack's wall set.
const (
	wallBlock = iota
	wallFace
	wallBottom // a diagonal wall filling the bottom corner
)

// walkPack builds a pack whose start level is drawn by a map, a string per row x and a character
// per column y:
//
//	# wall   e wall with an exit   \ diagonal wall (bottom corner)   . floor   ~ deep water
//	^ fire   @ the start   L lever opening door D   P plate (100) holding door D open
//	D door   O pit   X floor exit   T trap   B boulder   G guard   M mouth   r a rock
//	t teleporter to the cell two columns right
//
// Exits and the pit lead to a 3×3 level "level.below" at (1, 1).
func walkPack(rows ...string) *content.Pack {
	p := content.NewPack(content.Manifest{ID: "walk", Player: "thing.player",
		Start: content.Start{Level: "level.walk"}})
	p.Messages = &content.Messages{Messages: map[string]string{content.MsgNoRoom: "msg.full"}}
	p.Walls["walls"] = &content.WallSet{ID: "walls", Shapes: []*content.WallShape{
		wallBlock: {}, wallFace: {Frame: 1, Face: true}, wallBottom: {Frame: 2, Diagonal: content.CornerBottom}}}
	p.Things["thing.player"] = &content.Thing{ID: "thing.player", Kind: content.ThingPlayer, Speed: 8,
		Weight: 100, HitPoints: content.Dice{Bonus: 30}, MagicPoints: content.Dice{Bonus: 20}}
	p.Things["thing.trap"] = &content.Thing{ID: "thing.trap", Kind: content.ThingTrap, Weight: 100, Footprint: 20}
	p.Things["thing.boulder"] = &content.Thing{ID: "thing.boulder", Kind: content.ThingBoulder, Speed: 8, Weight: 100}
	p.Things["thing.guard"] = &content.Thing{ID: "thing.guard", Kind: content.ThingCreature, Weight: 100,
		HitPoints: content.Dice{Bonus: 20}, Behaviors: []string{content.BehaviorStationary}}
	p.Items["item.rock"] = &content.Item{ID: "item.rock", Weight: 60, Size: 3}
	p.Items["item.key"] = &content.Item{ID: "item.key", Weight: 1, Size: 1}
	p.Items["item.bag"] = &content.Item{ID: "item.bag", Weight: 1, Size: 2,
		Container: &content.Container{Capacity: 2, Largest: 2}}

	below := &content.Destination{Level: "level.below", Cell: cell{1, 1}}
	l := &content.Level{ID: "level.walk", Walls: "walls", Size: [2]int{len(rows), len(rows[0])},
		FloorTypes: []content.FloorType{{}, {Depth: 4, Sinks: true},
			{Enter: &content.Effect{Damage: content.Dice{Bonus: 2}, Type: "fire"}, Tick: &content.Effect{Damage: content.Dice{Bonus: 2}}}}}
	l.Floor, l.Wall = content.NewGrid(l.Size[0], l.Size[1]), content.NewGrid(l.Size[0], l.Size[1])
	l.Procedures = map[string][]content.Action{"open": {{On: ptr("door")}}, "close": {{Off: ptr("door")}}}
	for x, row := range rows {
		for y, ch := range row {
			c := cell{x, y}
			id := func(kind string) string { return kind + "." + string(rune('0'+x)) + "." + string(rune('0'+y)) }
			l.Floor[x][y] = 0
			switch ch {
			case '#':
				l.Wall[x][y] = wallBlock
			case 'e':
				l.Wall[x][y] = wallFace
				l.Features = append(l.Features, &content.Feature{ID: id("exit"), Kind: content.FeatureExit, Cell: c, To: below})
			case '\\':
				l.Wall[x][y] = wallBottom
			case '~':
				l.Floor[x][y] = 1
			case '^':
				l.Floor[x][y] = 2
			case '@':
				p.Manifest.Start.Cell = c
			case 'L':
				l.Features = append(l.Features, &content.Feature{ID: "lever", Kind: content.FeatureLever, Cell: c, Armed: true, OnTurnOn: "open", OnTurnOff: "close"})
			case 'P':
				l.Features = append(l.Features, &content.Feature{ID: "plate", Kind: content.FeaturePlate, Cell: c, Armed: true, Threshold: 100, OnPress: "open", OnRelease: "close"})
			case 'D':
				l.Features = append(l.Features, &content.Feature{ID: "door", Kind: content.FeatureDoor, Cell: c, State: content.DoorClosed})
			case 'O':
				l.Features = append(l.Features, &content.Feature{ID: "pit", Kind: content.FeaturePit, Cell: c, To: below})
			case 'X':
				l.Features = append(l.Features, &content.Feature{ID: "exit", Kind: content.FeatureExit, Cell: c, To: below})
			case 'M':
				l.Features = append(l.Features, &content.Feature{ID: "mouth", Kind: content.FeatureMouth, Cell: c, Message: "msg.hello"})
			case 't':
				l.Features = append(l.Features, &content.Feature{ID: "tele", Kind: content.FeatureTeleporter, Cell: c, Armed: true,
					Destinations: []content.Cell{{x, y + 2}, {x, y + 2}}})
			case 'T', 'B', 'G':
				tmpl := map[rune]string{'T': "thing.trap", 'B': "thing.boulder", 'G': "thing.guard"}[ch]
				l.Things = append(l.Things, &content.PlacedThing{ID: tmpl[len("thing."):], Thing: tmpl, Cell: c})
			case 'r':
				l.Objects = append(l.Objects, &content.FloorObject{Cell: c, Object: content.Object{Item: "item.rock"}})
			case '.':
			default:
				panic("walkPack: unknown map character " + string(ch))
			}
		}
	}
	p.Levels[l.ID] = l
	b := &content.Level{ID: "level.below", Walls: "walls", Size: [2]int{3, 3}, FloorTypes: []content.FloorType{{}}}
	b.Floor, b.Wall = content.NewGrid(3, 3), content.NewGrid(3, 3)
	for x := range 3 {
		for y := range 3 {
			b.Floor[x][y] = 0
		}
	}
	p.Levels[b.ID] = b
	return p
}

func walkWorld(t *testing.T, rows ...string) *World {
	t.Helper()
	w, err := New(walkPack(rows...), 1)
	if err != nil {
		t.Fatal(err)
	}
	if w.Player() == nil {
		t.Fatal("no player")
	}
	return w
}

// walk asks for n steps along d, one update each, and returns the events.
func walk(w *World, d, n int) []Event {
	var all []Event
	for range n {
		all = append(all, w.Update([]input.Command{input.Walk{Direction: d}})...)
	}
	return all
}

func TestPlayerStartsAtStart(t *testing.T) {
	w := walkWorld(t,
		"#####",
		"#.@.#",
		"#####")
	p := w.Player()
	if p.Cell() != (cell{1, 2}) || p.U != 48 || p.V != 80 {
		t.Errorf("player at %v (%d, %d)", p.Cell(), p.U, p.V)
	}
	if p.HitPoints != 30 || p.MaxHitPoints != 30 || p.MagicPoints != 20 {
		t.Errorf("hit points %d/%d, magic %d", p.HitPoints, p.MaxHitPoints, p.MagicPoints)
	}
}

func TestWalkStopsAtWalls(t *testing.T) {
	w := walkWorld(t,
		"#######",
		"#@....#",
		"#######")
	p := w.Player()
	walk(w, 2, 2)
	if p.V != 48+16 || p.U != 48 {
		t.Fatalf("two steps along v: (%d, %d)", p.U, p.V)
	}
	walk(w, 2, 40)
	if p.V != 6*32-1 {
		t.Errorf("walked up to the wall at v %d, want %d", p.V, 6*32-1)
	}
	walk(w, 0, 10)
	if p.U != 32 {
		t.Errorf("walked up to the wall at u %d, want 32", p.U)
	}
	// Screen right (direction 1) moves half the speed on each axis.
	w2 := walkWorld(t,
		"#####",
		"#...#",
		"#.@.#",
		"#...#",
		"#####")
	p2 := w2.Player()
	walk(w2, 1, 1)
	if p2.U != 80-4 || p2.V != 80+4 || p2.Facing != 1 {
		t.Errorf("a step screen right went to (%d, %d) facing %d", p2.U, p2.V, p2.Facing)
	}
}

func TestPlayerWaitsItsInterval(t *testing.T) {
	pk := walkPack(
		"#######",
		"#@....#",
		"#######")
	pk.Things["thing.player"].IntervalMS = 100 // 6 steps
	w, err := New(pk, 1)
	if err != nil {
		t.Fatal(err)
	}
	walk(w, 2, 12)
	if v := w.Player().V; v != 48+16 {
		t.Errorf("after 12 steps at an interval of 6 the player is at v %d", v)
	}
}

func TestSlideAlongDiagonalWall(t *testing.T) {
	w := walkWorld(t,
		"#####",
		"#...#",
		"#.@\\#",
		"#...#",
		"#####")
	p := w.Player()
	walk(w, 2, 2) // up against the diagonal wall at (2, 3)
	u := p.U
	walk(w, 2, 1)
	// Walking +v into a wall filling the bottom (+u, +v) corner slides toward −u.
	if p.U != u-slide {
		t.Errorf("slid to u %d from %d", p.U, u)
	}
}

// TestWallOutOfSight takes a diagonal wall whose neighbor cannot be seen as the shape its pairs
// say while it cannot be seen: it no longer slides the player, and the level keeps its shape.
func TestWallOutOfSight(t *testing.T) {
	w := walkWorld(t,
		"#####",
		"#..##",
		"#.@\\#",
		"#..##",
		"#####")
	l := w.Current()
	w.pack.Walls["walls"].Shapes[wallBottom].Unseen = [][2]int{{2, wallBlock}}
	walk(w, 2, 2)
	if l.Shape(cell{2, 3}) != wallBlock || l.Wall[2][3] != wallBottom {
		t.Fatalf("shape %d, own %d", l.Shape(cell{2, 3}), l.Wall[2][3])
	}
	if l.Shape(cell{1, 3}) != wallBlock || len(l.Taken()) != 1 {
		t.Errorf("taken %v", l.Taken())
	}
	st, err := w.State()
	if err != nil {
		t.Fatal(err)
	}
	if st.Levels[0].Grids[1][2][3] != wallBottom {
		t.Error("the save holds the shape taken")
	}
	r, err := Restore(w.pack, st)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Current().Shape(cell{2, 3}); got != wallBlock {
		t.Errorf("restored: shape %d", got)
	}
	p := w.Player()
	u := p.U
	walk(w, 2, 1)
	if p.U != u {
		t.Errorf("slid along a wall taken as a block: u %d from %d", p.U, u)
	}
}

func TestPushTrap(t *testing.T) {
	w := walkWorld(t,
		"########",
		"#@.T...#",
		"########")
	trap, p := w.Current().Thing("trap"), w.Player()
	before := trap.V
	walk(w, 2, 10)
	if trap.V <= before {
		t.Fatalf("the trap did not move: v %d", trap.V)
	}
	if p.V >= trap.V || trap.V-p.V < (16+20)/2 {
		t.Errorf("player at v %d overlaps the trap at %d", p.V, trap.V)
	}
	walk(w, 2, 40)
	if limit := 7*32 - 1 - (20/2 - 1); trap.V > limit {
		t.Errorf("the trap went into the wall: v %d > %d", trap.V, limit)
	}
}

// Two pushable things lying on each other, like corpses fallen in one place, can be pushed apart.
func TestPushStackedApart(t *testing.T) {
	pk := walkPack(
		"########",
		"#@.T...#",
		"########")
	l := pk.Levels["level.walk"]
	second := *l.Things[0]
	second.ID = "trap2"
	l.Things = append(l.Things, &second)
	w, err := New(pk, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, b := w.Current().Thing("trap"), w.Current().Thing("trap2")
	before := [2]int{a.V, b.V}
	walk(w, 2, 10)
	if a.V == before[0] && b.V == before[1] {
		t.Fatal("neither stacked trap moved")
	}
	if a.V == b.V {
		t.Errorf("the traps did not come apart: both at v %d", a.V)
	}
}

func TestWalkingIntoBoulderRollsIt(t *testing.T) {
	w := walkWorld(t,
		"##########",
		"#@.B....G#",
		"##########")
	b, guard := w.Current().Thing("boulder"), w.Current().Thing("guard")
	walk(w, 2, 8)
	if b.Rolling == nil || b.Rolling.Direction != 2 || b.Rolling.Shuttle {
		t.Fatalf("the boulder is not rolling along 2: %+v", b.Rolling)
	}
	for range 200 {
		w.Update(nil)
	}
	if b.Rolling != nil {
		t.Error("the boulder did not stop at the guard")
	}
	if guard.HitPoints >= 20 {
		t.Errorf("the guard was not hurt: %d", guard.HitPoints)
	}
	if d := guard.V - b.V; d != 16 {
		t.Errorf("the boulder stopped %d short of the guard, want 16", d)
	}
}

func TestShuttlingBoulderReverses(t *testing.T) {
	pk := walkPack(
		"#######",
		"#..B..#",
		"#@....#",
		"#######")
	pk.Levels["level.walk"].Procedures["shuttle"] = []content.Action{{Roll: &content.Roll{Thing: "boulder", Direction: 2, Shuttle: true}}}
	w, err := New(pk, 1)
	if err != nil {
		t.Fatal(err)
	}
	b := w.Current().Thing("boulder")
	w.Update([]input.Command{input.Run{Procedure: "shuttle"}})
	reversed := false
	for range 200 {
		w.Update(nil)
		if b.Rolling.Direction == 6 {
			reversed = true
		}
	}
	if !reversed || b.Rolling == nil {
		t.Errorf("the boulder did not shuttle: %+v", b.Rolling)
	}
}

func TestPlayerPressesPlate(t *testing.T) {
	w := walkWorld(t,
		"#######",
		"#@.P.D#",
		"#######")
	walk(w, 2, 6)
	if f := w.Current().Feature("plate"); !f.Down || f.Counter != 100 {
		t.Fatalf("the player did not press the plate: %+v", f)
	}
	settle(w)
	if w.Current().Blocked(cell{1, 5}) {
		t.Error("the door did not open")
	}
	walk(w, 6, 6)
	if w.Current().Feature("plate").Down {
		t.Error("the plate stayed down after the player left")
	}
}

func TestPickUpSwapAndPutDown(t *testing.T) {
	w := walkWorld(t,
		"#####",
		"#@r.#",
		"#####")
	p, l := w.Player(), w.Current()
	l.Objects[cell{1, 2}] = append(l.Objects[cell{1, 2}], &Object{Item: "item.rock"})
	walk(w, 2, 4)
	// The take key puts the first rock on a shelf; taking into the hand puts the second there.
	l.Objects[cell{1, 2}] = append(l.Objects[cell{1, 2}], &Object{Item: "item.rock"})
	if ev := w.Update([]input.Command{input.PickUp{}}); !has(ev, EventMessage, content.MsgTaken) {
		t.Errorf("taking the rock named nothing: %v", ev)
	}
	if p.Hand != nil || p.Slots[0] == nil {
		t.Fatalf("the take key: hand %v, shelf %v", p.Hand, p.Slots[0])
	}
	w.Update([]input.Command{input.PickUp{Hand: true}})
	if p.Hand == nil || p.Slots[0] == nil || len(l.Objects[cell{1, 2}]) != 1 {
		t.Fatalf("hand %v, shelf %v, floor %v", p.Hand, p.Slots[0], l.Objects[cell{1, 2}])
	}
	l.Objects[cell{1, 2}] = nil
	w.Update([]input.Command{input.PutDown{}})
	if p.Hand != nil || len(l.Objects[cell{1, 2}]) != 1 {
		t.Fatal("the rock was not put down")
	}
	w.Update([]input.Command{input.Swap{Slot: 0}})
	if p.Hand == nil || p.Slots[0] != nil {
		t.Error("swapping shelf 0 did not take the rock into the hand")
	}
	if ev := w.Update([]input.Command{input.Swap{Slot: SlotHead}}); !has(ev, EventRefused, "") {
		t.Error("a worn slot took the rock")
	}
	// A full inventory leaves the object on the floor.
	for i := range Shelves {
		p.Slots[i] = &Object{Item: "item.key"}
	}
	if ev := w.Update([]input.Command{input.PickUp{}}); !has(ev, EventMessage, content.MsgNoRoom) || len(l.Objects[cell{1, 2}]) != 1 {
		t.Errorf("picking up with no room: %v", ev)
	}
	l.Objects[cell{1, 2}] = nil
	if ev := w.Update([]input.Command{input.PickUp{}}); !has(ev, EventMessage, content.MsgNothingToTake) {
		t.Errorf("picking up nothing: %v", ev)
	}
}

// TestTakeUnderCreature walks the player up to a creature from below on screen: the thin boxes at
// their feet let the player stand 12 units off on each axis, inside the creature's cell, and take
// what lies there (docs/spec/entities.md, Collision boxes).
func TestTakeUnderCreature(t *testing.T) {
	p := walkPack(
		"#####",
		"#G..#",
		"#.@.#",
		"#####")
	p.Things["thing.player"].Box = &content.Box{-10, 9, 16, 6}
	p.Things["thing.guard"].Box = &content.Box{-10, 9, 16, 12}
	w, err := New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	l, pl := w.Current(), w.Player()
	l.Objects[cell{1, 1}] = append(l.Objects[cell{1, 1}], w.newObject("item.key"))
	walk(w, 7, 6)
	if pl.U != 60 || pl.V != 60 || pl.Cell() != (cell{1, 1}) {
		t.Fatalf("the player stopped at (%d, %d) in %v, want (60, 60) in the guard's cell", pl.U, pl.V, pl.Cell())
	}
	if ev := w.Update([]input.Command{input.PickUp{}}); has(ev, EventMessage, content.MsgNothingToTake) || len(l.Objects[cell{1, 1}]) != 0 {
		t.Errorf("the key under the guard was not taken: %v", ev)
	}
}

func TestPack(t *testing.T) {
	w := walkWorld(t,
		"####",
		"#@.#",
		"####")
	p := w.Player()
	p.Hand = &Object{Item: "item.bag"}
	w.Update([]input.Command{input.Swap{Slot: SlotLeftHand}})
	if p.Pack(w.Pack()) == nil {
		t.Fatal("the bag in the left hand is not a pack")
	}
	p.Hand = &Object{Item: "item.rock"}
	if ev := w.Update([]input.Command{input.Swap{Slot: SlotPack}}); !has(ev, EventRefused, "") {
		t.Error("the bag took a rock bigger than it holds")
	}
	p.Hand = &Object{Item: "item.key"}
	w.Update([]input.Command{input.Swap{Slot: SlotPack + 1}})
	if p.Hand != nil || p.Slot(w.Pack(), SlotPack) == nil {
		t.Fatal("the key did not go into the pack")
	}
	w.Update([]input.Command{input.Swap{Slot: SlotPack}})
	if p.Hand == nil || p.Hand.Item != "item.key" || len(p.Pack(w.Pack()).Contents) != 0 {
		t.Error("the key did not come out of the pack")
	}
}

func TestActivateLeverAndMouth(t *testing.T) {
	w := walkWorld(t,
		"######",
		"#@L.D#",
		"#..M.#",
		"######")
	if ev := w.Update([]input.Command{input.Activate{}}); !has(ev, EventRefused, "nothing to use") {
		t.Errorf("activating with nothing near: %v", ev)
	}
	walk(w, 2, 4)
	w.Update([]input.Command{input.Activate{}})
	settle(w)
	if w.Current().Blocked(cell{1, 4}) {
		t.Error("the lever did not open the door")
	}
	ev := w.Update([]input.Command{input.Activate{}})
	settle(w)
	if !w.Current().Blocked(cell{1, 4}) {
		t.Errorf("using the lever again did not close the door: %v", ev)
	}
	walk(w, 3, 1)
	walk(w, 2, 2)
	walk(w, 4, 2)
	if ev := w.Update([]input.Command{input.Activate{}}); len(ev) == 0 || ev[0].Text != "msg.hello" {
		t.Errorf("activating beside the mouth: %v (player at %v)", ev, w.Player().Cell())
	}
}

func TestExitsAndPits(t *testing.T) {
	w := walkWorld(t,
		"#####",
		"#@X.#",
		"#####")
	ev := walk(w, 2, 2)
	if w.Current().ID != "level.below" || w.Player().Cell() != (cell{1, 1}) {
		t.Fatalf("the floor exit took the player to %s %v", w.Current().ID, w.Player().Cell())
	}
	if !has(ev, EventTransition, content.TransitionExitFloor) {
		t.Errorf("no floor exit transition: %v", ev)
	}

	w = walkWorld(t,
		"##e##",
		"#.@.#",
		"#####")
	walk(w, 1, 6) // screen right is not along an axis
	if w.Current().ID != "level.walk" {
		t.Fatal("walking diagonally into a wall exit took it")
	}
	w = walkWorld(t,
		"##e##",
		"#.@.#",
		"#####")
	ev = walk(w, 0, 4)
	if w.Current().ID != "level.below" {
		t.Error("walking into a wall exit did not take it")
	}
	if !has(ev, EventTransition, content.TransitionExitWall) {
		t.Errorf("no wall exit transition: %v", ev)
	}

	w = walkWorld(t,
		"#####",
		"#@O.#",
		"#####")
	walk(w, 2, 4) // stepping down carefully
	if w.Current().ID != "level.below" || w.Player().HitPoints != 30 {
		t.Errorf("stepping into the pit: %s, %d hit points", w.Current().ID, w.Player().HitPoints)
	}
	w = walkWorld(t,
		"#####",
		"#.O@#",
		"#####")
	ev = walk(w, 6, 4)
	if w.Current().ID != "level.below" || w.Player().HitPoints >= 30 {
		t.Errorf("falling into the pit: %s, %d hit points", w.Current().ID, w.Player().HitPoints)
	}
	if !has(ev, EventTransition, content.TransitionFall) {
		t.Errorf("no fall transition: %v", ev)
	}
}

func TestTeleporterAndFloors(t *testing.T) {
	w := walkWorld(t,
		"#######",
		"#@t...#",
		"#^....#",
		"#######")
	ev := walk(w, 2, 2)
	if c := w.Player().Cell(); c != (cell{1, 4}) {
		t.Errorf("the teleporter sent the player to %v", c)
	}
	if !has(ev, EventTransition, content.TransitionTeleporter) {
		t.Errorf("no teleporter transition: %v", ev)
	}
	w = walkWorld(t,
		"#######",
		"#@....#",
		"#^....#",
		"#######")
	walk(w, 4, 2)
	if hp := w.Player().HitPoints; hp != 28 {
		t.Errorf("entering fire left %d hit points", hp)
	}
	for range Steps(statusTicks) + 1 {
		w.Update(nil)
	}
	if hp := w.Player().HitPoints; hp != 26 {
		t.Errorf("a status tick on fire left %d hit points", hp)
	}
}

func TestDoorClosingOnPlayerReopens(t *testing.T) {
	w := walkWorld(t,
		"#####",
		"#@D.#",
		"#####")
	w.Update([]input.Command{input.Switch{Feature: "door", Mode: input.SwitchOn}})
	settle(w)
	walk(w, 2, 4)
	if w.Player().Cell() != (cell{1, 2}) {
		t.Fatalf("player at %v", w.Player().Cell())
	}
	w.Update([]input.Command{input.Switch{Feature: "door", Mode: input.SwitchOff}})
	settle(w)
	settle(w)
	if f := w.Current().Feature("door"); f.State != content.DoorOpen {
		t.Errorf("the door closed on the player: %s", f.State)
	}
}

func TestVisibility(t *testing.T) {
	w := walkWorld(t,
		"#######",
		"#@.#..#",
		"#..D..#",
		"#######")
	l := w.Current()
	for _, c := range []cell{{1, 1}, {0, 0}, {3, 3}, {2, 3}, {0, 3}} {
		if !l.Visible(c) {
			t.Errorf("%v is not visible from the start", c)
		}
	}
	for _, c := range []cell{{1, 4}, {2, 5}} {
		if l.Visible(c) {
			t.Errorf("%v behind the closed door is visible", c)
		}
	}
	w.Update([]input.Command{input.Switch{Feature: "door", Mode: input.SwitchOn}})
	settle(w)
	if !l.Visible(cell{1, 5}) || !l.Seen(cell{1, 5}) {
		t.Error("the open door does not show the room behind it")
	}
	w.Update([]input.Command{input.Switch{Feature: "door", Mode: input.SwitchOff}})
	settle(w)
	if l.Visible(cell{1, 5}) || !l.Seen(cell{1, 5}) {
		t.Error("the closed door shows the room behind it, or it was forgotten as seen")
	}
}

func TestPlayerSpeed(t *testing.T) {
	pk := walkPack(
		"####",
		"#@.#",
		"####")
	tm := pk.Things["thing.player"]
	tm.IntervalMS = 275 // 5 ticks
	tm.Stats = map[string]int{AttrStrength: 18, AttrAgility: 20, AttrEndurance: 18}
	w, err := New(pk, 1)
	if err != nil {
		t.Fatal(err)
	}
	p := w.Player()
	steps := func(ticks int) int64 { return int64(float64(ticks)*1000/ticksPerSecond*StepsPerSecond/1000 + 0.5) }
	// Agility 20, carrying nothing (+1 for a light load): one tick faster than the base.
	if got := w.PlayerInterval(); got != steps(4) {
		t.Errorf("agile and unladen: %d steps, want %d", got, steps(4))
	}
	// 180 carried against a limit of 180: agility −1 to 19, and 2 ticks more.
	p.Hand = &Object{Item: "item.rock"}
	p.Slots[0], p.Slots[1] = &Object{Item: "item.rock"}, &Object{Item: "item.rock"}
	if got := w.PlayerInterval(); got != steps(6) {
		t.Errorf("at the load limit: %d steps, want %d", got, steps(6))
	}
	// Agility 8 and endurance 6, carrying a little: 2 + 3 ticks more.
	p.Hand, p.Slots[0], p.Slots[1] = &Object{Item: "item.rock"}, nil, nil
	p.Attributes[AttrAgility], p.Attributes[AttrEndurance] = 8, 6
	if got := w.PlayerInterval(); got != steps(10) {
		t.Errorf("clumsy and tired: %d steps, want %d", got, steps(10))
	}
	// A bonus of −2 from powers, back to agility 20: 2 ticks.
	p.Hand, p.IntervalBonus = nil, -2
	p.Attributes[AttrAgility], p.Attributes[AttrEndurance] = 20, 18
	if got := w.PlayerInterval(); got != steps(2) {
		t.Errorf("with a bonus of −2: %d steps, want %d", got, steps(2))
	}
}

// TestFigureLooks draws each figure layer as its own look until an item worn in a worn slot (not
// a hand) gives it another, with the female sprite for a female portrait.
func TestFigureLooks(t *testing.T) {
	pk := walkPack("@.")
	pk.Things["thing.player"].Figure = []content.FigureLayer{
		{Name: "body", FigureLook: content.FigureLook{Sprite: "bare", Female: "bare.f"}},
		{Name: "head", FigureLook: content.FigureLook{Sprite: "hair"}},
	}
	pk.Items["item.mail"] = &content.Item{ID: "item.mail", Wear: content.WearBody,
		Looks: map[string]content.FigureLook{"body": {Sprite: "mail"}}}
	pk.Items["item.helm"] = &content.Item{ID: "item.helm", Wear: content.WearHead,
		Looks: map[string]content.FigureLook{"head": {Sprite: "helm", Female: "helm.f"}}}
	pk.Portraits["portrait.f"] = &content.Portrait{ID: "portrait.f", Female: true}
	w, err := New(pk, 1)
	if err != nil {
		t.Fatal(err)
	}
	p := w.Player()
	check := func(want ...string) {
		t.Helper()
		got := p.FigureLooks(pk)
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("looks %v, want %v", got, want)
		}
	}
	check("bare", "hair")
	p.Slots[SlotBody] = &Object{Item: "item.mail"}
	p.Slots[SlotRightHand] = &Object{Item: "item.helm"} // a hand shows no worn look
	check("mail", "hair")
	p.Slots[SlotHead] = p.Slots[SlotRightHand]
	p.Portrait = "portrait.f"
	check("mail", "helm.f")
	p.Slots[SlotBody] = nil
	check("bare.f", "helm.f")
}

// TestOpenContainer opens a bag on a shelf as the pack: its contents are the pack slots, and a
// swap takes from it; opened again it closes, and the bag in the left hand is the pack again, as
// it is once the opened bag leaves its slot.
func TestOpenContainer(t *testing.T) {
	w := walkWorld(t, "@.")
	pk := w.Pack()
	p := w.Player()
	shelfBag := &Object{Item: "item.bag", Contents: []*Object{{Item: "item.key"}}}
	handBag := &Object{Item: "item.bag"}
	p.Slots[3], p.Slots[SlotLeftHand] = shelfBag, handBag
	if p.Pack(pk) != handBag {
		t.Fatal("the pack is not the bag in the left hand at first")
	}
	w.Update([]input.Command{input.Open{Slot: 3}})
	if p.Pack(pk) != shelfBag || p.Slot(pk, SlotPack) == nil {
		t.Fatalf("opened: pack %+v", p.Pack(pk))
	}
	w.Update([]input.Command{input.Swap{Slot: SlotPack}})
	if p.Hand == nil || p.Hand.Item != "item.key" || len(shelfBag.Contents) != 0 {
		t.Fatalf("took %+v; bag holds %d", p.Hand, len(shelfBag.Contents))
	}
	w.Update([]input.Command{input.Swap{Slot: SlotPack}})
	if p.Hand != nil || len(shelfBag.Contents) != 1 {
		t.Fatalf("put back: hand %+v, bag holds %d", p.Hand, len(shelfBag.Contents))
	}
	w.Update([]input.Command{input.Open{Slot: 3}})
	if p.Pack(pk) != handBag {
		t.Error("opened again, the shelf bag did not close")
	}
	w.Update([]input.Command{input.Open{Slot: 3}, input.Swap{Slot: 3}})
	if p.Hand != shelfBag || p.Pack(pk) != handBag {
		t.Errorf("taken from its slot, the opened bag is still the pack: %+v", p.Pack(pk))
	}
	w.Update([]input.Command{input.Open{Slot: 4}})
	if p.Pack(pk) != handBag {
		t.Error("opening an empty shelf changed the pack")
	}
}

func TestMoveSlot(t *testing.T) {
	w := walkWorld(t, "@.")
	pk := w.Pack()
	pk.Items["item.sword"] = &content.Item{ID: "item.sword", Size: 2, Wear: content.WearHand, WeaponClass: ptr(0)}
	pk.Items["item.axe"] = &content.Item{ID: "item.axe", Size: 2, Wear: content.WearHand, WeaponClass: ptr(1)}
	pk.Items["item.shield"] = &content.Item{ID: "item.shield", Size: 2, Wear: content.WearHand, Armor: 2}
	pk.Items["item.helm"] = &content.Item{ID: "item.helm", Size: 2, Wear: content.WearHead, Armor: 1}
	p := w.Player()
	chest := &Object{Item: "item.bag", Contents: []*Object{{Item: "item.key"}}}
	sword, axe, shield, helm := &Object{Item: "item.sword"}, &Object{Item: "item.axe"}, &Object{Item: "item.shield"}, &Object{Item: "item.helm"}
	p.Slots[0], p.Slots[1], p.Slots[2], p.Slots[3], p.Slots[SlotRightHand] = chest, sword, shield, helm, axe

	do(w, input.MoveSlot{Slot: 1, To: input.ToWorn})
	if p.Slots[SlotRightHand] != sword || p.Slots[1] != axe {
		t.Fatalf("sword to the right hand: hand %+v, shelf %+v", p.Slots[SlotRightHand], p.Slots[1])
	}
	do(w, input.MoveSlot{Slot: 2, To: input.ToWorn}, input.MoveSlot{Slot: 3, To: input.ToWorn})
	if p.Slots[SlotLeftHand] != shield || p.Slots[2] != nil || p.Slots[SlotHead] != helm {
		t.Fatalf("shield to the left hand, helm on: left %+v, head %+v", p.Slots[SlotLeftHand], p.Slots[SlotHead])
	}
	if ev := do(w, input.MoveSlot{Slot: SlotHead, To: input.ToWorn}); !has(ev, EventRefused, "already worn") {
		t.Error("moving a worn helm to its own slot was not refused")
	}

	do(w, input.Open{Slot: 0})
	do(w, input.MoveSlot{Slot: SlotRightHand, To: input.ToPack})
	if p.Slots[SlotRightHand] != nil || len(chest.Contents) != 2 || chest.Contents[1] != sword {
		t.Fatalf("sword into the chest: hand %+v, chest %+v", p.Slots[SlotRightHand], chest.Contents)
	}
	if ev := do(w, input.MoveSlot{Slot: 1, To: input.ToPack}); !has(ev, EventRefused, "the bag has no such place") {
		t.Error("a full chest took the axe")
	}
	if p.Slots[1] != axe || p.Hand != nil {
		t.Fatalf("a refused move moved: shelf %+v, hand %+v", p.Slots[1], p.Hand)
	}
	if ev := do(w, input.MoveSlot{Slot: 0, To: input.ToPack}); !has(ev, EventRefused, "that is the pack") {
		t.Error("the chest went into itself")
	}

	do(w, input.MoveSlot{Slot: SlotPack, To: input.ToShelf})
	if len(chest.Contents) != 1 || p.Slots[2] == nil || p.Slots[2].Item != "item.key" {
		t.Fatalf("key to the first empty shelf: chest %+v, shelf %+v", chest.Contents, p.Slots[2])
	}
	for i := range Shelves {
		if p.Slots[i] == nil {
			p.Slots[i] = &Object{Item: "item.rock"}
		}
	}
	if ev := do(w, input.MoveSlot{Slot: SlotPack, To: input.ToShelf}); !has(ev, EventMessage, content.MsgNoRoom) || len(chest.Contents) != 1 {
		t.Error("with no empty shelf the chest's object moved, or no message")
	}
}

// TestPlayerGlides draws the walking player gliding between its places, finishes the glide at
// once when no walk is asked for, and draws a long move such as a placement as a jump.
func TestPlayerGlides(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) { p.Things["thing.player"].IntervalMS = 275 },
		"#######",
		"#@....#",
		"#######")
	p := w.Player()
	walk := input.Walk{Direction: 2}
	// Holding the key asks for the walk every update, as the viewer does.
	hold := func() {
		last := p.last
		for p.last == last {
			w.Update([]input.Command{walk})
		}
	}
	hold()
	from := p.V
	hold()
	if p.V != from+8 || w.Step() != p.last+1 {
		t.Fatalf("a step went from v %d to %d, want +8", from, p.V)
	}
	g := p.Glide(w.Step())
	if g.FromV != from || g.Left != g.Span-1 || g.Span < 4 {
		t.Fatalf("just stepped, glide %+v, want from v %d with a step of its span gone", g, from)
	}
	w.Update([]input.Command{walk})
	if left := p.Glide(w.Step()).Left; left != g.Left-1 {
		t.Errorf("walking on, %d steps of the glide left, want %d", left, g.Left-1)
	}
	// Letting go: the glide finishes within two steps.
	w.Update(nil)
	if g := p.Glide(w.Step()); g.Span == 0 || g.Left > hurrySteps {
		t.Errorf("let go, glide %+v, want at most %d steps left", g, hurrySteps)
	}
	w.Update(nil)
	w.Update(nil)
	if g := p.Glide(w.Step()); g.Span != 0 || w.Gliding() {
		t.Errorf("two steps after letting go, still gliding %+v", g)
	}
	// Let go before its next update, the held walk takes no further step.
	stopped := p.V
	for end := p.next; w.Step() <= end; {
		w.Update(nil)
	}
	if p.V != stopped {
		t.Errorf("let go, it stepped on from v %d to %d", stopped, p.V)
	}
	// A tap after the update counts, though let go before the next.
	w.Update([]input.Command{walk})
	for end := p.next; w.Step() <= end; {
		w.Update(nil)
	}
	if p.V != stopped+8 {
		t.Errorf("a tap moved it from v %d to %d, want one step", stopped, p.V)
	}
	w.Update([]input.Command{input.PlacePlayer{Cell: content.Cell{1, 5}}})
	if g := p.Glide(w.Step()); g.Span != 0 {
		t.Errorf("placed, glides %+v, want drawn where it is", g)
	}
}

// TestWalkFromStandStartsSooner has a walk asked for from a stand come once half the
// interval has passed since the player's last update, not at the end of it.
func TestWalkFromStandStartsSooner(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) { p.Things["thing.player"].IntervalMS = 275 },
		"#######",
		"#@....#",
		"#######")
	p := w.Player()
	playerUpdate(w)
	last, next := p.last, p.next
	w.Update([]input.Command{input.Walk{Direction: 2}})
	half := last + (next-last+1)/2
	for w.Step() < half {
		if p.last != last {
			t.Fatalf("walked at step %d, before half the interval (%d)", p.last, half)
		}
		w.Update([]input.Command{input.Walk{Direction: 2}})
	}
	w.Update([]input.Command{input.Walk{Direction: 2}})
	if p.last != half || p.next != half+(next-last) {
		t.Errorf("walked at step %d, next %d; want at %d, next %d", p.last, p.next, half, half+next-last)
	}
	// Walking on, the next step comes when due.
	at := p.next
	for p.last == half {
		w.Update([]input.Command{input.Walk{Direction: 2}})
	}
	if p.last != at {
		t.Errorf("the second step came at %d, want %d", p.last, at)
	}
}

// TestPushedGlides draws a pushed corpse gliding along with the player that pushes it, over the
// player's longer wait.
func TestPushedGlides(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) { p.Things["thing.player"].IntervalMS = 275 },
		"########",
		"#@G....#",
		"########")
	p, g := w.Player(), w.Current().Thing("guard")
	g.HitPoints = 0
	v0 := g.V
	for g.V == v0 && w.Step() < 500 {
		w.Update([]input.Command{input.Walk{Direction: 2}})
	}
	if g.V == v0 {
		t.Fatal("the corpse was not pushed")
	}
	gg, pg := g.Glide(w.Step()), p.Glide(w.Step())
	if gg.Span == 0 || gg.Left != pg.Left || gg.Span != pg.Span || gg.FromV != v0 {
		t.Errorf("the corpse glides %+v, the player %+v; want them together, from v %d", gg, pg, v0)
	}
	if int64(gg.Span) != p.next-p.last {
		t.Errorf("glide span %d, want the push's wait %d", gg.Span, p.next-p.last)
	}
}

// TestRunning runs with a walk asked to run: 2 ticks off the interval, exertion as it goes, no
// attacks, throws or casts, and an end as soon as the run is let go (docs/spec/player.md, Running).
func TestRunning(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) { p.Things["thing.player"].IntervalMS = 275 },
		"####################",
		"#@.................#",
		"####################")
	p := w.Player()
	p.Slots[SlotRightHand] = &Object{Item: "item.sword"}
	steps := func(ticks int) int64 { return int64(float64(ticks)*1000/ticksPerSecond*StepsPerSecond/1000 + 0.5) }
	// Agility 60 takes a tick off the base 5.
	if got := w.PlayerInterval(); got != steps(4) {
		t.Fatalf("walking: %d steps, want %d", got, steps(4))
	}
	run := input.Walk{Direction: 2, Run: true}
	hold := func(cmd input.Command) {
		last := p.last
		for p.last == last {
			w.Update([]input.Command{cmd})
		}
	}
	hold(run)
	if !p.Running || w.PlayerInterval() != steps(2) {
		t.Fatalf("running %v at %d steps, want running at %d", p.Running, w.PlayerInterval(), steps(2))
	}
	for _, cmd := range []input.Command{input.Strike{Hand: 0}, input.Strike{Hand: 1}, input.Throw{Hand: 0}, input.Cast{School: 0}} {
		if ev := w.Update([]input.Command{run, cmd}); !has(ev, EventRefused, "the player is running") {
			t.Errorf("%T%+v while running was not refused: %v", cmd, cmd, ev)
		}
	}
	from := p.Exertion
	for range 24 {
		hold(run)
	}
	if !p.Running || p.Exertion <= from && p.Fatigue == 0 {
		t.Errorf("24 running steps: running %v, exertion %d from %d", p.Running, p.Exertion, from)
	}
	// Walking on without Shift ends the run at once: the strike is taken.
	if ev := w.Update([]input.Command{input.Walk{Direction: 2}, input.Strike{Hand: 0}}); p.Running || has(ev, EventRefused, "") {
		t.Errorf("walking on, running %v, events %v", p.Running, ev)
	}
}

// TestRunEndsAtAWall ends a run at a step that does not move.
func TestRunEndsAtAWall(t *testing.T) {
	w := fightWorld(t, nil, "###", "#@#", "###")
	p := w.Player()
	for range 4 {
		w.Update([]input.Command{input.Walk{Direction: 2, Run: true}})
	}
	if p.Running {
		t.Error("running into a wall")
	}
}
