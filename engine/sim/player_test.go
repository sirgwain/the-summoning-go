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
	w.Update([]input.Command{input.PickUp{}})
	w.Update([]input.Command{input.PickUp{}})
	if p.Hand == nil || p.Slots[0] == nil || len(l.Objects[cell{1, 2}]) != 0 {
		t.Fatalf("hand %v, shelf %v, floor %v", p.Hand, p.Slots[0], l.Objects[cell{1, 2}])
	}
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
	walk(w, 2, 2)
	if w.Current().ID != "level.below" || w.Player().Cell() != (cell{1, 1}) {
		t.Fatalf("the floor exit took the player to %s %v", w.Current().ID, w.Player().Cell())
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
	walk(w, 0, 4)
	if w.Current().ID != "level.below" {
		t.Error("walking into a wall exit did not take it")
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
	walk(w, 6, 4)
	if w.Current().ID != "level.below" || w.Player().HitPoints >= 30 {
		t.Errorf("falling into the pit: %s, %d hit points", w.Current().ID, w.Player().HitPoints)
	}
}

func TestTeleporterAndFloors(t *testing.T) {
	w := walkWorld(t,
		"#######",
		"#@t...#",
		"#^....#",
		"#######")
	walk(w, 2, 2)
	if c := w.Player().Cell(); c != (cell{1, 4}) {
		t.Errorf("the teleporter sent the player to %v", c)
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
