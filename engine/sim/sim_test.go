package sim

import (
	"slices"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

type cell = content.Cell

func ptr[T any](v T) *T { return &v }

// testPack builds a 5×5 level holding one of each mechanism, and a 3×3 level below it.
//
//	   0     1      2       3        4
//	0  .     door   .       wall     socket
//	1  lever .      tele    .        .
//	2  guard .      plate   .        .
//	3  slot  .      .       mouth    deep
//	4  .     .      .       trapdoor pit
func testPack() *content.Pack {
	p := content.NewPack(content.Manifest{ID: "test", Start: content.Start{Level: "level.00"}})
	p.Messages = &content.Messages{Messages: map[string]string{
		content.MsgLeverStuck: "msg.stuck", content.MsgSlotWrongItem: "msg.wrong",
		content.MsgSlotStuck: "msg.nothing", content.MsgNoDropHere: "msg.lose",
	}}
	for id, w := range map[string]int{"key": 1, "rock": 60, "gem1": 1, "gem2": 1, "sphere": 6, "sword": 10} {
		p.Items["item."+id] = &content.Item{ID: "item." + id, Weight: w, Random: id == "sword"}
	}
	p.Items["item.sack"] = &content.Item{ID: "item.sack", Weight: 2, Container: &content.Container{Capacity: 4, Largest: 9}}
	p.Items["item.bag"] = &content.Item{ID: "item.bag", Weight: 1, Container: &content.Container{Capacity: 4, Largest: 9, Weightless: true}}
	p.Things["thing.guard"] = &content.Thing{ID: "thing.guard", Kind: content.ThingCreature, Weight: 100, HitPoints: content.Dice{Count: 2, Sides: 4}}
	p.Things["thing.ghost"] = &content.Thing{ID: "thing.ghost", Kind: content.ThingCreature, Weight: 100,
		HitPoints: content.Dice{Bonus: 5}, Behaviors: []string{content.BehaviorResistsTeleport}}
	p.Things["thing.boulder"] = &content.Thing{ID: "thing.boulder", Kind: content.ThingBoulder, Weight: 100}

	n := content.NoFrame
	grid := func(rows, cols, v int) content.Grid {
		g := content.NewGrid(rows, cols)
		for x := range g {
			for y := range g[x] {
				g[x][y] = v
			}
		}
		return g
	}
	l := &content.Level{
		ID: "level.00", Size: [2]int{5, 5},
		FloorTypes: []content.FloorType{{}, {Sinks: true, Depth: 4}},
		Floor:      grid(5, 5, 0), Wall: grid(5, 5, n),
	}
	l.Floor[3][4] = 1
	l.Wall[0][3] = 3
	below := &content.Destination{Level: "level.01", Cell: cell{2, 2}}
	l.Features = []*content.Feature{
		{ID: "door", Kind: content.FeatureDoor, Cell: cell{0, 1}, State: content.DoorClosed},
		{ID: "sealed", Kind: content.FeatureDoor, Cell: cell{1, 1}, State: content.DoorClosed, SealedBy: "item.sphere"},
		{ID: "lever", Kind: content.FeatureLever, Cell: cell{1, 0}, Armed: true, OnTurnOn: "open", OnTurnOff: "close"},
		{ID: "plate", Kind: content.FeaturePlate, Cell: cell{2, 2}, Armed: true, Threshold: 100, OnPress: "open", OnRelease: "close"},
		{ID: "slot", Kind: content.FeatureSlot, Cell: cell{3, 0}, Accepts: "item.key", Armed: true, OnInsert: "open"},
		{ID: "tele", Kind: content.FeatureTeleporter, Cell: cell{1, 2}, Armed: true, Destinations: []content.Cell{{1, 4}, {3, 2}}},
		{ID: "pit", Kind: content.FeaturePit, Cell: cell{4, 4}, To: below},
		{ID: "trapdoor", Kind: content.FeatureTrapdoor, Cell: cell{4, 3}, To: below},
		{ID: "socket", Kind: content.FeatureSocket, Cell: cell{0, 4}, Places: []string{"item.gem1", "item.gem2"}, OnComplete: "socket"},
		{ID: "mouth", Kind: content.FeatureMouth, Cell: cell{3, 3}, Message: "msg.hi"},
	}
	l.Procedures = map[string][]content.Action{
		"open":   {{On: ptr("door")}},
		"close":  {{Off: ptr("door")}},
		"socket": {{SetFloor: &content.SetFloor{Cell: cell{0, 3}, Type: 0}}},
		"say":    {{SetMessage: &content.SetMessage{Feature: "mouth", Message: "msg.bye"}}},
		"rewire": {{SetHook: &content.SetHook{Feature: "lever", Hook: content.HookTurnOn, Procedure: ptr("say")}}},
		"make":   {{Create: &content.Create{Cell: cell{2, 2}, Item: "item.rock"}}, {Create: &content.Create{Cell: cell{3, 4}, Item: "item.rock"}}},
		"roll":   {{Roll: &content.Roll{Thing: "boulder", Direction: 2}}},
	}
	l.Things = []*content.PlacedThing{
		{ID: "guard", Thing: "thing.guard", Cell: cell{2, 0}},
		{ID: "boulder", Thing: "thing.boulder", Cell: cell{4, 0}},
	}
	p.Levels[l.ID] = l
	p.Levels["level.01"] = &content.Level{ID: "level.01", Size: [2]int{3, 3},
		FloorTypes: []content.FloorType{{}}, Floor: grid(3, 3, 0), Wall: grid(3, 3, n)}
	return p
}

func newWorld(t *testing.T, edit ...func(p *content.Pack)) *World {
	t.Helper()
	p := testPack()
	for _, e := range edit {
		e(p)
	}
	w, err := New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// do runs one update with cmds and returns its events.
func do(w *World, cmds ...input.Command) []Event { return w.Update(cmds) }

// settle runs updates until moving doors have had time to finish, returning all events.
func settle(w *World) []Event {
	var all []Event
	for range 4*Steps(doorStepTicks) + 2 {
		all = append(all, w.Update(nil)...)
	}
	return all
}

// spinOut runs updates until a creature that came onto a teleporter has spun and jumped.
func spinOut(w *World) []Event {
	var all []Event
	for range Steps(1)*spinTicks + 2 {
		all = append(all, w.Update(nil)...)
	}
	return all
}

func has(events []Event, kind, detail string) bool {
	return slices.ContainsFunc(events, func(e Event) bool { return e.Kind == kind && (detail == "" || e.Detail == detail) })
}

func door(w *World) *Feature { return w.Current().Feature("door") }

func objects(w *World, c cell) []string {
	var items []string
	for _, o := range w.Current().Objects[c] {
		items = append(items, o.Item)
	}
	return items
}

func TestLeverOpensAndClosesDoor(t *testing.T) {
	w := newWorld(t)
	ev := do(w, input.Use{Cell: cell{1, 0}})
	lever := w.Current().Feature("lever")
	if !lever.On || !has(ev, EventProcedure, "open") || door(w).State != content.DoorOpening {
		t.Fatalf("after pulling: lever on %v, door %s, events %v", lever.On, door(w).State, ev)
	}
	if w.Current().Blocked(cell{0, 1}) == false {
		t.Error("a door opening lets things through")
	}
	ev = settle(w)
	if d := door(w); d.State != content.DoorOpen || d.Stage != stageOpen || w.Current().Blocked(cell{0, 1}) {
		t.Fatalf("door %s stage %d", d.State, d.Stage)
	}
	var stages []string
	for _, e := range ev {
		if e.Kind == EventFeature && e.Feature == "door" {
			stages = append(stages, e.Detail)
		}
	}
	if want := []string{"1", "2", "3", "open"}; !slices.Equal(stages, want) {
		t.Errorf("door stages %v, want %v", stages, want)
	}

	do(w, input.Use{Cell: cell{1, 0}})
	settle(w)
	if lever.On || door(w).State != content.DoorClosed {
		t.Errorf("after pulling back: lever on %v, door %s", lever.On, door(w).State)
	}
}

func TestDoorReversesAndToggles(t *testing.T) {
	w := newWorld(t)
	do(w, input.Switch{Feature: "door", Mode: input.SwitchOn})
	for range Steps(doorStepTicks) * 2 {
		do(w)
	}
	if door(w).Stage != 2 {
		t.Fatalf("stage %d after two ticks", door(w).Stage)
	}
	do(w, input.Switch{Feature: "door", Mode: input.SwitchToggle})
	if door(w).State != content.DoorClosing {
		t.Fatal("toggling a moving door does not reverse it")
	}
	settle(w)
	if door(w).State != content.DoorClosed {
		t.Errorf("door %s", door(w).State)
	}
	do(w, input.Switch{Feature: "door", Mode: input.SwitchToggle})
	settle(w)
	if door(w).State != content.DoorOpen {
		t.Errorf("toggling a closed door: %s", door(w).State)
	}
}

func TestDisarmedLever(t *testing.T) {
	w := newWorld(t)
	do(w, input.Switch{Feature: "lever", Mode: input.SwitchOff})
	ev := do(w, input.Use{Cell: cell{1, 0}})
	if !has(ev, EventMessage, content.MsgLeverStuck) || ev[0].Text != "msg.stuck" || w.Current().Feature("lever").On {
		t.Errorf("events %v", ev)
	}
}

func TestPlate(t *testing.T) {
	w := newWorld(t)
	plate := w.Current().Feature("plate")
	ev := do(w, input.Drop{Cell: cell{2, 2}, Item: "item.rock"})
	if plate.Counter != 60 || plate.Down || has(ev, EventProcedure, "") {
		t.Fatalf("one rock: counter %d, down %v", plate.Counter, plate.Down)
	}
	ev = do(w, input.Drop{Cell: cell{2, 2}, Item: "item.rock"})
	if plate.Counter != 120 || !plate.Down || !has(ev, EventProcedure, "open") {
		t.Fatalf("two rocks: counter %d, down %v, events %v", plate.Counter, plate.Down, ev)
	}
	ev = do(w, input.Drop{Cell: cell{2, 2}, Item: "item.key"})
	if has(ev, EventProcedure, "") {
		t.Error("a plate already down runs again")
	}
	do(w, input.Remove{Cell: cell{2, 2}})
	ev = do(w, input.Remove{Cell: cell{2, 2}})
	if plate.Counter != 60 || plate.Down || !has(ev, EventProcedure, "close") {
		t.Errorf("after taking: counter %d, down %v, events %v", plate.Counter, plate.Down, ev)
	}
}

func TestPlateArmingReevaluates(t *testing.T) {
	w := newWorld(t)
	plate := w.Current().Feature("plate")
	do(w, input.Switch{Feature: "plate", Mode: input.SwitchOff})
	ev := do(w, input.Drop{Cell: cell{2, 2}, Item: "item.rock"}, input.Drop{Cell: cell{2, 2}, Item: "item.rock"})
	if plate.Down || has(ev, EventProcedure, "") {
		t.Fatal("a disarmed plate went down")
	}
	ev = do(w, input.Switch{Feature: "plate", Mode: input.SwitchOn})
	if !plate.Down || !has(ev, EventProcedure, "open") {
		t.Errorf("arming a loaded plate: down %v, events %v", plate.Down, ev)
	}
}

func TestThingsPressPlates(t *testing.T) {
	w := newWorld(t)
	plate := w.Current().Feature("plate")
	ev := do(w, input.Move{Thing: "guard", Cell: cell{2, 2}})
	if plate.Counter != 100 || !has(ev, EventProcedure, "open") {
		t.Fatalf("guard on the plate: counter %d", plate.Counter)
	}
	ev = do(w, input.Move{Thing: "guard", Cell: cell{2, 3}})
	if plate.Counter != 0 || !has(ev, EventProcedure, "close") {
		t.Errorf("guard off the plate: counter %d", plate.Counter)
	}
	do(w, input.Place{Thing: "thing.ghost", Cell: cell{2, 2}})
	if plate.Counter != 0 {
		t.Errorf("a thing that resists teleporting pressed the plate: %d", plate.Counter)
	}
}

func TestPlateStartsWithPlacedWeight(t *testing.T) {
	w := newWorld(t, func(p *content.Pack) {
		p.Levels["level.00"].Objects = []*content.FloorObject{{Cell: cell{2, 2}, Object: content.Object{
			Item: "item.sack", Contents: []content.Object{{Item: "item.rock"}}}}}
	})
	plate := w.Current().Feature("plate")
	if plate.Counter != 62 || plate.Down {
		t.Errorf("counter %d, down %v", plate.Counter, plate.Down)
	}
}

func TestContainerWeight(t *testing.T) {
	p := testPack()
	rock := &Object{Item: "item.rock"}
	if w := (&Object{Item: "item.sack", Contents: []*Object{rock}}).Weight(p); w != 62 {
		t.Errorf("sack with a rock weighs %d", w)
	}
	if w := (&Object{Item: "item.bag", Contents: []*Object{rock}}).Weight(p); w != 1 {
		t.Errorf("bag of lightness with a rock weighs %d", w)
	}
}

func TestSlot(t *testing.T) {
	w := newWorld(t)
	ev := do(w, input.Drop{Cell: cell{3, 0}, Item: "item.rock"})
	if !has(ev, EventMessage, content.MsgSlotWrongItem) || ev[0].Item != "item.rock" || len(objects(w, cell{3, 0})) != 0 {
		t.Fatalf("wrong item: %v", ev)
	}
	ev = do(w, input.Drop{Cell: cell{3, 0}, Item: "item.key"})
	if !has(ev, EventObject, "used") || !has(ev, EventProcedure, "open") || len(objects(w, cell{3, 0})) != 0 {
		t.Fatalf("key: %v", ev)
	}
	do(w, input.Switch{Feature: "slot", Mode: input.SwitchOff})
	ev = do(w, input.Drop{Cell: cell{3, 0}, Item: "item.key"})
	if !has(ev, EventMessage, content.MsgSlotStuck) || has(ev, EventProcedure, "") {
		t.Errorf("disarmed slot: %v", ev)
	}
}

func TestSlotFills(t *testing.T) {
	w := newWorld(t)
	slot := w.Current().Feature("slot")
	do(w, input.Drop{Cell: cell{3, 0}, Item: "item.key"})
	if !slot.Full {
		t.Fatal("a slot its hook left armed is not full")
	}
	ev := do(w, input.Drop{Cell: cell{3, 0}, Item: "item.key"})
	if !has(ev, EventMessage, content.MsgSlotStuck) || has(ev, EventProcedure, "open") {
		t.Fatalf("full slot: %v", ev)
	}
	do(w, input.Switch{Feature: "slot", Mode: input.SwitchOff})
	do(w, input.Switch{Feature: "slot", Mode: input.SwitchOn})
	if ev := do(w, input.Drop{Cell: cell{3, 0}, Item: "item.key"}); !has(ev, EventMessage, content.MsgSlotStuck) {
		t.Errorf("re-armed full slot: %v", ev)
	}
}

func TestSlotStaysOpen(t *testing.T) {
	for name, open := range map[string][]content.Action{
		"disarms itself": {{Off: ptr("slot")}},
		"moves on":       {{SetHook: &content.SetHook{Feature: "slot", Hook: content.HookInsert, Procedure: ptr("close")}}},
		"takes another":  {{SetAccepts: &content.SetAccepts{Feature: "slot", Item: "item.rock"}}},
		"hands it back":  {{Create: &content.Create{Cell: cell{0, 0}, Item: "item.key"}}},
	} {
		w := newWorld(t, func(p *content.Pack) { p.Levels["level.00"].Procedures["open"] = open })
		do(w, input.Drop{Cell: cell{3, 0}, Item: "item.key"})
		if w.Current().Feature("slot").Full {
			t.Errorf("%s: the slot is full", name)
		}
	}
}

func TestTeleporterSendsObjectsAndThings(t *testing.T) {
	w := newWorld(t)
	tele := w.Current().Feature("tele")
	do(w, input.Drop{Cell: cell{1, 2}, Item: "item.key"})
	if got := objects(w, cell{1, 4}); !slices.Equal(got, []string{"item.key"}) || tele.Next != 1 {
		t.Fatalf("first key at (1, 4): %v, next %d", got, tele.Next)
	}
	do(w, input.Move{Thing: "guard", Cell: cell{1, 2}})
	if g := w.Current().Thing("guard"); g.Cell() != (cell{1, 2}) || g.spin == 0 || tele.Next != 1 {
		t.Errorf("the guard did not spin on the teleporter: at %v, spin %d, next %d", g.Cell(), g.spin, tele.Next)
	}
	ev := spinOut(w)
	if g := w.Current().Thing("guard"); g.Cell() != (cell{3, 2}) || tele.Next != 0 {
		t.Errorf("guard at %v, next %d", g.Cell(), tele.Next)
	}
	if has(ev, EventSound, soundTeleport) {
		t.Error("a creature through a teleporter made the teleport sound")
	}
	do(w, input.Place{Thing: "thing.ghost", Cell: cell{1, 2}})
	if g := w.Current().ThingsAt(cell{1, 2}); len(g) != 1 {
		t.Error("a thing that resists teleporting was moved")
	}
}

func TestArmingTeleporterSendsWhatIsOnIt(t *testing.T) {
	w := newWorld(t)
	do(w, input.Switch{Feature: "tele", Mode: input.SwitchOff})
	do(w, input.Drop{Cell: cell{1, 2}, Item: "item.key"})
	if len(objects(w, cell{1, 2})) != 1 {
		t.Fatal("a disarmed teleporter moved an object")
	}
	do(w, input.Switch{Feature: "tele", Mode: input.SwitchOn})
	if len(objects(w, cell{1, 2})) != 0 || len(objects(w, cell{1, 4})) != 1 {
		t.Errorf("after arming: %v, %v", objects(w, cell{1, 2}), objects(w, cell{1, 4}))
	}
}

func TestTeleporterLoopEnds(t *testing.T) {
	w := newWorld(t, func(p *content.Pack) {
		p.Levels["level.00"].Features[5].Destinations = []content.Cell{{1, 2}, {1, 2}}
	})
	do(w, input.Drop{Cell: cell{1, 2}, Item: "item.key"})
	if len(objects(w, cell{1, 2})) != 1 {
		t.Error("an object teleported onto its own teleporter did not come to rest")
	}
}

func TestPitAndTrapdoor(t *testing.T) {
	w := newWorld(t)
	ev := do(w, input.Drop{Cell: cell{4, 4}, Item: "item.key"})
	below := w.Level("level.01")
	if below == nil || len(below.Objects[cell{2, 2}]) != 1 || !has(ev, EventObject, "fell") {
		t.Fatalf("key through the pit: %v", ev)
	}
	ev = do(w, input.Move{Thing: "guard", Cell: cell{4, 4}})
	if w.Current().Thing("guard") != nil || !has(ev, EventThing, "fell") {
		t.Error("the guard did not fall")
	}

	do(w, input.Drop{Cell: cell{4, 3}, Item: "item.rock"})
	if len(objects(w, cell{4, 3})) != 1 {
		t.Fatal("an object fell through a closed trapdoor")
	}
	do(w, input.Switch{Feature: "trapdoor", Mode: input.SwitchOn})
	if len(objects(w, cell{4, 3})) != 0 || len(below.Objects[cell{2, 2}]) != 2 {
		t.Error("opening the trapdoor did not drop the rock")
	}
	if w.Current().ID != "level.00" {
		t.Error("objects falling changed the current level")
	}
}

func TestSocket(t *testing.T) {
	w := newWorld(t)
	socket := w.Current().Feature("socket")
	do(w, input.Drop{Cell: cell{0, 4}, Item: "item.gem2"})
	ev := do(w, input.Drop{Cell: cell{0, 4}, Item: "item.gem2"})
	if !slices.Equal(socket.Set, []int{1}) || socket.LookState() != "empty" || has(ev, EventFeature, "") {
		t.Fatalf("one gem: set %v", socket.Set)
	}
	do(w, input.Drop{Cell: cell{0, 4}, Item: "item.rock"})
	if len(objects(w, cell{0, 4})) != 0 {
		t.Error("an object stayed on the socket")
	}
	ev = do(w, input.Drop{Cell: cell{0, 4}, Item: "item.gem1"})
	if socket.LookState() != "full" || !has(ev, EventFeature, "full") || !has(ev, EventMap, "") ||
		w.Current().Wall[0][3] != content.NoFrame || w.Current().Blocked(cell{0, 3}) {
		t.Errorf("both gems: %v", ev)
	}
}

func TestDeepWater(t *testing.T) {
	w := newWorld(t)
	ev := do(w, input.Drop{Cell: cell{3, 4}, Item: "item.key"})
	if !has(ev, EventMessage, content.MsgNoDropHere) || len(objects(w, cell{3, 4})) != 0 {
		t.Fatalf("player drop: %v", ev)
	}
	ev = do(w, input.Run{Procedure: "make"})
	if !has(ev, EventObject, "sank") || len(objects(w, cell{3, 4})) != 0 || len(objects(w, cell{2, 2})) != 1 {
		t.Errorf("created objects: %v", ev)
	}
}

func TestSealedDoor(t *testing.T) {
	w := newWorld(t)
	sealed := w.Current().Feature("sealed")
	do(w, input.Switch{Feature: "sealed", Mode: input.SwitchOn})
	settle(w)
	if sealed.State != content.DoorClosed || sealed.LookState() != "sealed" {
		t.Fatal("switching opened a sealed door")
	}
	do(w, input.Drop{Cell: cell{1, 1}, Item: "item.key"})
	if sealed.State != content.DoorClosed {
		t.Fatal("the wrong object opened a sealed door")
	}
	ev := do(w, input.Drop{Cell: cell{1, 1}, Item: "item.sphere"})
	if sealed.State != content.DoorOpen || sealed.SealedBy != "" || !has(ev, EventSound, soundUnseal) {
		t.Errorf("sphere: %v", ev)
	}
}

func TestDoorClosingOnThings(t *testing.T) {
	w := newWorld(t)
	do(w, input.Switch{Feature: "door", Mode: input.SwitchOn})
	settle(w)
	do(w, input.Move{Thing: "guard", Cell: cell{0, 1}}, input.Move{Thing: "boulder", Cell: cell{0, 1}})
	do(w, input.Switch{Feature: "door", Mode: input.SwitchOff})
	ev := settle(w)
	if w.Current().Thing("boulder") != nil || !has(ev, EventThing, "crushed") {
		t.Error("the boulder was not crushed")
	}
	settle(w) // closing, then reopening
	if w.Current().Thing("guard") == nil || door(w).State != content.DoorOpen {
		t.Errorf("the door did not reopen on the guard: %s", door(w).State)
	}
}

func TestMouthAndProcedureActions(t *testing.T) {
	w := newWorld(t)
	ev := do(w, input.Use{Cell: cell{3, 3}})
	if !has(ev, EventMessage, "") || ev[0].Text != "msg.hi" {
		t.Fatalf("mouth: %v", ev)
	}
	do(w, input.Run{Procedure: "rewire"})
	if w.Current().Feature("lever").OnTurnOn != "say" {
		t.Fatal("set_hook did not rewire the lever")
	}
	do(w, input.Use{Cell: cell{1, 0}})
	if ev := do(w, input.Use{Cell: cell{3, 3}}); ev[0].Text != "msg.bye" {
		t.Errorf("mouth after set_message: %v", ev)
	}
	if w.pack.Levels["level.00"].Features[2].OnTurnOn != "open" {
		t.Error("the simulation changed the pack")
	}
	if ev := do(w, input.Run{Procedure: "roll"}); !has(ev, EventThing, "rolling") || w.Current().Thing("boulder").Rolling == nil {
		t.Errorf("roll: %v", ev)
	}
}

func TestLevels(t *testing.T) {
	w := newWorld(t)
	ev := do(w, input.FollowExit{Feature: "pit"})
	if w.Current().ID != "level.01" || !has(ev, EventLevelChanged, "") {
		t.Fatalf("follow exit: %v", ev)
	}
	do(w, input.GoTo{Level: "level.00"})
	do(w, input.Drop{Cell: cell{0, 0}, Item: "item.key"})
	do(w, input.GoTo{Level: "level.01"}, input.GoTo{Level: "level.00"})
	if len(objects(w, cell{0, 0})) != 1 {
		t.Error("a level did not keep its state")
	}
	if ev := do(w, input.GoTo{Level: "level.99"}); !has(ev, EventRefused, "") {
		t.Error("going to a missing level was not refused")
	}
}

func TestRandomObjectsAndHitPoints(t *testing.T) {
	edit := func(p *content.Pack) {
		p.Levels["level.00"].Objects = []*content.FloorObject{{Cell: cell{0, 0}, Object: content.Object{Item: content.RandomItem}}}
	}
	w := newWorld(t, edit)
	if got := objects(w, cell{0, 0}); !slices.Equal(got, []string{"item.sword"}) {
		t.Errorf("random object %v", got)
	}
	hp := w.Current().Thing("guard").HitPoints
	if hp < 2 || hp > 8 {
		t.Errorf("2d4 rolled %d", hp)
	}
	if again := newWorld(t, edit).Current().Thing("guard").HitPoints; again != hp {
		t.Errorf("same seed rolled %d then %d", hp, again)
	}
}

// A random object chosen inside a container that does not take it is lost (formats.md, Entity
// stream).
func TestRandomObjectInContainer(t *testing.T) {
	w := newWorld(t, func(p *content.Pack) {
		p.Items["item.sword"].Size = 5
		p.Items["item.pouch"] = &content.Item{ID: "item.pouch", Weight: 1, Container: &content.Container{Capacity: 4, Largest: 1}}
		random := []content.Object{{Item: content.RandomItem}, {Item: "item.key"}}
		p.Levels["level.00"].Objects = []*content.FloorObject{
			{Cell: cell{0, 0}, Object: content.Object{Item: "item.sack", Contents: random}},
			{Cell: cell{0, 0}, Object: content.Object{Item: "item.pouch", Contents: random}},
		}
	})
	var got [][]string
	for _, o := range w.Current().Objects[cell{0, 0}] {
		var in []string
		for _, c := range o.Contents {
			in = append(in, c.Item)
		}
		got = append(got, in)
	}
	if want := [][]string{{"item.sword", "item.key"}, {"item.key"}}; !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("contents %v, want %v", got, want)
	}
}

func TestRefusals(t *testing.T) {
	w := newWorld(t)
	for _, c := range []input.Command{
		input.Use{Cell: cell{0, 0}},
		input.Drop{Cell: cell{9, 9}, Item: "item.key"},
		input.Drop{Cell: cell{0, 0}, Item: "item.nope"},
		input.Remove{Cell: cell{0, 0}},
		input.Place{Cell: cell{0, 0}, Thing: "thing.nope"},
		input.Move{Thing: "nobody", Cell: cell{0, 0}},
		input.Switch{Feature: "mouth", Mode: input.SwitchOn},
		input.Switch{Feature: "nope", Mode: input.SwitchOn},
		input.Run{Procedure: "nope"},
		input.FollowExit{Feature: "lever"},
	} {
		if ev := do(w, c); !has(ev, EventRefused, "") {
			t.Errorf("%T %+v: %v", c, c, ev)
		}
	}
}

func TestEventString(t *testing.T) {
	e := Event{Kind: EventMessage, Level: "level.00", Cell: cell{1, 2}, Text: "msg.hi", Detail: "lever_stuck"}
	if s := e.String(); !strings.Contains(s, "text=msg.hi") || !strings.Contains(s, "lever_stuck") {
		t.Errorf("%q", s)
	}
}
