package sim

import (
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/combat"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// maxDepth bounds procedures running procedures, and teleporter chains, so a loop in a level
// cannot hang the world.
const maxDepth = 32

// Door sounds and effects, named for the presentation.
const (
	soundDoor     = content.CueDoor
	soundUnseal   = content.CueUnseal
	soundLever    = content.CueLever
	soundTeleport = content.CueTeleport
)

// doorStepTicks is how often a moving door steps (docs/spec/rules.md, Unknowns).
const doorStepTicks = 1

func (w *World) featureChanged(l *Level, f *Feature) {
	w.emit(Event{Kind: EventFeature, Level: l.ID, Cell: f.Cell, Feature: f.ID, Detail: f.LookState()})
}

// advance moves the level's doors one step on.
func (w *World) advance(l *Level) {
	for _, f := range l.Features {
		if f.Kind != content.FeatureDoor || !f.Moving() || w.step < f.next {
			continue
		}
		f.next = w.step + Steps(doorStepTicks)
		if f.State == content.DoorOpening {
			f.Stage++
		} else {
			f.Stage--
		}
		switch {
		case f.Stage >= stageOpen:
			f.Stage, f.State = stageOpen, content.DoorOpen
		case f.Stage <= stageClosed:
			f.Stage, f.State = stageClosed, content.DoorClosed
			w.doorClosed(l, f)
		default:
			w.emit(Event{Kind: EventSound, Level: l.ID, Cell: f.Cell, Detail: soundDoor})
		}
		w.featureChanged(l, f)
	}
}

// doorClosed handles a door closing on its cell: boulders, traps and corpses there are removed;
// anything else is hurt 1 point with a 3 in 10 chance, and the door opens again
// (docs/spec/rules.md, Doors). Pushing what is in the way aside comes with movement.
func (w *World) doorClosed(l *Level, f *Feature) {
	reopen := false
	if p := w.player; p != nil && w.current == l.ID && p.Cell() == f.Cell {
		reopen = true
		if w.chance(3, 10) {
			w.hurtPlayerAloud(1, combat.Physical, "a door")
		}
	}
	for _, t := range l.ThingsAt(f.Cell) {
		if t.Template.Kind != content.ThingCreature || t.HitPoints <= 0 {
			w.removeThing(l, t, "crushed")
			continue
		}
		reopen = true
		if w.chance(3, 10) {
			t.HitPoints--
			w.emit(Event{Kind: EventThing, Level: l.ID, Cell: f.Cell, Thing: t.ID, Detail: "hurt"})
		}
	}
	if reopen {
		f.State = content.DoorOpening
		f.next = w.step + Steps(doorStepTicks)
	}
}

// moveDoor starts door f toward open or closed, or reverses it.
func (w *World) moveDoor(l *Level, f *Feature, open bool) {
	if f.SealedBy != "" {
		return
	}
	switch {
	case open && f.State == content.DoorOpen, !open && f.State == content.DoorClosed:
		return
	case open:
		f.State = content.DoorOpening
	default:
		f.State = content.DoorClosing
	}
	f.next = w.step + Steps(doorStepTicks)
	w.emit(Event{Kind: EventSound, Level: l.ID, Cell: f.Cell, Detail: soundDoor})
	w.featureChanged(l, f)
}

// switchFeature switches feature f on (open, arm), off, or over (docs/spec/rules.md, Level
// procedures).
func (w *World) switchFeature(l *Level, f *Feature, mode string, depth int) {
	on := func(current bool) bool {
		switch mode {
		case input.SwitchOn:
			return true
		case input.SwitchOff:
			return false
		}
		return !current
	}
	switch f.Kind {
	case content.FeatureDoor:
		if mode == input.SwitchToggle && f.Moving() {
			w.moveDoor(l, f, f.State == content.DoorClosing)
		} else {
			w.moveDoor(l, f, on(f.State == content.DoorOpen))
		}
	case content.FeatureTrapdoor:
		open := on(f.Open)
		if open == f.Open {
			return
		}
		f.Open = open
		w.featureChanged(l, f)
		if open {
			w.openedBelow(l, f, depth)
		}
	case content.FeatureTeleporter:
		armed := on(f.Armed)
		if armed == f.Armed {
			return
		}
		f.Armed = armed
		w.featureChanged(l, f)
		if armed {
			w.teleportCell(l, f, depth)
		}
	case content.FeatureLever, content.FeaturePlate, content.FeatureSlot:
		armed := on(f.Armed)
		changed := armed != f.Armed
		f.Armed = armed
		if changed {
			w.featureChanged(l, f)
		}
		if armed && f.Kind == content.FeaturePlate {
			w.evaluatePlate(l, f, depth)
		}
	default:
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: f.Cell, Feature: f.ID, Detail: "a " + f.Kind + " does not switch"})
	}
}

// evaluatePlate brings a plate's position in line with its counter: a down plate below its
// threshold rises, an up plate at it goes down, each running its hook.
func (w *World) evaluatePlate(l *Level, f *Feature, depth int) {
	switch {
	case f.Down && f.Counter < f.Threshold:
		f.Down = false
		w.featureChanged(l, f)
		w.run(l, f.OnRelease, depth+1)
	case !f.Down && f.Counter >= f.Threshold:
		f.Down = true
		w.featureChanged(l, f)
		w.run(l, f.OnPress, depth+1)
	}
}

// weigh adds weight (negative to remove) to the plate on cell c, running its hook when the
// counter crosses the threshold and the plate is armed (docs/spec/rules.md, Weight plates).
func (w *World) weigh(l *Level, c content.Cell, weight, depth int) {
	f := l.byCell[c]
	if f == nil || f.Kind != content.FeaturePlate || weight == 0 {
		return
	}
	before := f.Counter
	f.Counter = max(0, f.Counter+weight)
	switch {
	case before < f.Threshold && f.Counter >= f.Threshold && f.Armed:
		f.Down = true
		w.featureChanged(l, f)
		w.run(l, f.OnPress, depth+1)
	case before >= f.Threshold && f.Counter < f.Threshold && f.Armed:
		f.Down = false
		w.featureChanged(l, f)
		w.run(l, f.OnRelease, depth+1)
	}
}

// resetFeature puts feature id of level l back as the level's definition starts it: its state,
// hooks and settings, running nothing. A plate keeps the weight on it, less unweigh, and is down
// when that reaches its threshold. False when the level has no such feature.
func (w *World) resetFeature(l *Level, id string, unweigh int) bool {
	f := l.byID[id]
	i := slices.IndexFunc(l.Def.Features, func(fd *content.Feature) bool { return fd.ID == id })
	if f == nil || i < 0 {
		return false
	}
	fresh := newFeature(l.Def.Features[i])
	if f.Kind == content.FeaturePlate {
		fresh.Counter = max(0, f.Counter-unweigh)
		fresh.Down = fresh.Counter >= fresh.Threshold
	}
	*f = *fresh
	w.featureChanged(l, f)
	return true
}

// use uses what is on cell c: pulls a lever or hears a mouth.
func (w *World) use(l *Level, c content.Cell) {
	f := l.byCell[c]
	switch {
	case f == nil:
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: c, Detail: "nothing to use"})
	case f.Kind == content.FeatureLever:
		if !f.Armed {
			w.message(l, c, content.MsgLeverStuck, "")
			return
		}
		hook := f.OnTurnOn
		if f.On {
			hook = f.OnTurnOff
		}
		f.On = !f.On
		w.emit(Event{Kind: EventSound, Level: l.ID, Cell: c, Detail: soundLever})
		w.featureChanged(l, f)
		w.run(l, hook, 0)
	case f.Kind == content.FeatureMouth:
		w.emit(Event{Kind: EventMessage, Level: l.ID, Cell: c, Feature: f.ID, Text: f.Message})
		w.heardMouth(l, f, f.Message)
	default:
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: c, Feature: f.ID, Detail: "a " + f.Kind + " is not used"})
	}
}

// message shows the engine message key, naming item when it takes one.
func (w *World) message(l *Level, c content.Cell, key, item string) {
	if w.quiet(key) {
		return
	}
	text := ""
	if m := w.pack.Messages; m != nil {
		text = m.Messages[key]
	}
	w.emit(Event{Kind: EventMessage, Level: l.ID, Cell: c, Text: text, Item: item, Detail: key})
}

// Who puts an object down decides what happens when it cannot stay (docs/spec/rules.md,
// Dropping objects).
const (
	byPlayer = iota
	byWorld
)

// drop puts object o down on cell c in the order the spec gives. A player's drop that is refused
// keeps the object in hand: drop returns false.
func (w *World) drop(l *Level, c content.Cell, o *Object, by, depth int) bool {
	f := l.byCell[c]
	upPlate := f != nil && f.Kind == content.FeaturePlate && !f.Down
	if ft := l.FloorType(c); ft != nil && ft.Sinks && !upPlate {
		if by == byPlayer {
			w.message(l, c, content.MsgNoDropHere, "")
			return false
		}
		w.emit(Event{Kind: EventObject, Level: l.ID, Cell: c, Item: o.Item, Detail: "sank"})
		return true
	}
	if f != nil {
		switch {
		case f.Kind == content.FeatureSlot:
			return w.insert(l, f, o, depth)
		case f.Kind == content.FeatureSocket:
			w.setGem(l, f, o, depth)
			return true
		case f.Kind == content.FeaturePit || f.Kind == content.FeatureTrapdoor && f.Open:
			w.fall(l, f, o, depth)
			return true
		case f.Kind == content.FeatureTeleporter && f.Armed && depth < maxDepth:
			return w.drop(l, w.jump(l, f), o, byWorld, depth+1)
		case f.Kind == content.FeatureDoor && f.SealedBy != "" && o.Item == f.SealedBy:
			w.unseal(l, f)
		}
	}
	l.Objects[c] = append(l.Objects[c], o)
	w.emit(Event{Kind: EventObject, Level: l.ID, Cell: c, Item: o.Item, Detail: "landed"})
	w.weigh(l, c, o.Weight(w.pack), depth)
	return true
}

// insert puts an object in a slot: the wrong item, or a disarmed slot, refuse it; the right one
// is used up and runs the slot's hook.
func (w *World) insert(l *Level, f *Feature, o *Object, depth int) bool {
	switch {
	case o.Item != f.Accepts:
		w.message(l, f.Cell, content.MsgSlotWrongItem, o.Item)
		return false
	case !f.Armed:
		w.message(l, f.Cell, content.MsgSlotStuck, o.Item)
		return false
	}
	w.emit(Event{Kind: EventObject, Level: l.ID, Cell: f.Cell, Item: o.Item, Detail: "used"})
	w.run(l, f.OnInsert, depth+1)
	return true
}

// setGem fills a socket's place for the object; every object is used up. When all places are
// filled the socket runs its hook (docs/spec/rules.md, Gem socket).
func (w *World) setGem(l *Level, f *Feature, o *Object, depth int) {
	w.emit(Event{Kind: EventObject, Level: l.ID, Cell: f.Cell, Item: o.Item, Detail: "used"})
	i := slices.Index(f.Places, o.Item)
	if i < 0 || slices.Contains(f.Set, i) {
		return
	}
	f.Set = append(f.Set, i)
	slices.Sort(f.Set)
	w.featureChanged(l, f)
	if len(f.Set) == len(f.Places) {
		w.playEffect(l, f.Effect, f.Cell, w.playerRef())
		w.afterEffect(func() { w.run(l, f.OnComplete, depth+1) })
	}
}

// fall sends an object through a pit or open trapdoor to the level and cell it leads to.
func (w *World) fall(l *Level, f *Feature, o *Object, depth int) {
	w.emit(Event{Kind: EventObject, Level: l.ID, Cell: f.Cell, Item: o.Item, Detail: "fell"})
	dest, err := w.load(f.To.Level)
	if err != nil {
		return
	}
	w.drop(dest, f.To.Cell, o, byWorld, depth+1)
}

// jump returns where teleporter f sends things now, and flips it to its other destination.
func (w *World) jump(l *Level, f *Feature) content.Cell {
	to := f.Destinations[f.Next]
	f.Next = 1 - f.Next
	w.featureChanged(l, f)
	return to
}

// teleportCell sends what is on an armed teleporter's cell on: objects along the chain, things
// to the destination (docs/spec/rules.md, In-level teleporters).
func (w *World) teleportCell(l *Level, f *Feature, depth int) {
	objs := l.Objects[f.Cell]
	if len(objs) > 0 {
		delete(l.Objects, f.Cell)
		for _, o := range objs {
			w.weigh(l, f.Cell, -o.Weight(w.pack), depth)
			w.drop(l, f.Cell, o, byWorld, depth+1)
		}
	}
	for _, t := range l.ThingsAt(f.Cell) {
		w.entered(l, t, depth+1)
	}
	if p := w.player; p != nil && w.current == l.ID && p.Cell() == f.Cell {
		w.teleportPlayer(l, f, depth+1)
	}
}

// openedBelow drops what is on a trapdoor that has opened: objects fall to its destination and
// things on it fall.
func (w *World) openedBelow(l *Level, f *Feature, depth int) {
	objs := l.Objects[f.Cell]
	delete(l.Objects, f.Cell)
	for _, o := range objs {
		w.fall(l, f, o, depth)
	}
	for _, t := range l.ThingsAt(f.Cell) {
		w.entered(l, t, depth+1)
	}
	if p := w.player; p != nil && w.current == l.ID && p.Cell() == f.Cell {
		w.fallPlayer(f, -1)
	}
}

// moveThing moves thing t to cell to, keeping its place within the cell, and runs the effects
// of leaving its cell and entering the new one.
func (w *World) moveThing(l *Level, t *Thing, to content.Cell, depth int) {
	from := t.Cell()
	du, dv := t.U-CellUnits*from[0], t.V-CellUnits*from[1]
	t.U, t.V = CellUnits*to[0]+du, CellUnits*to[1]+dv
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: to, Thing: t.ID, Detail: "moved"})
	if from != to {
		w.weigh(l, from, -t.weight(), depth)
		w.weigh(l, to, t.weight(), depth)
	}
	w.entered(l, t, depth)
}

// entered runs the effects of the cell thing t stands in: teleporters move it and pits take it,
// unless it resists (docs/spec/rules.md, Teleporters and exits).
func (w *World) entered(l *Level, t *Thing, depth int) {
	if depth >= maxDepth {
		return
	}
	f := l.byCell[t.Cell()]
	if t.Template.Has(content.BehaviorResistsTeleport) {
		if f != nil && (f.Kind == content.FeatureTeleporter && f.Armed || f.Kind == content.FeaturePit || f.Kind == content.FeatureTrapdoor && f.Open) {
			w.meetDefense(l, t, content.BehaviorResistsTeleport)
		}
		return
	}
	switch {
	case f == nil:
	case f.Kind == content.FeatureTeleporter && f.Armed:
		if t.Template.Kind == content.ThingCreature {
			// A creature or corpse is first set down in the middle of the teleporter's cell, and is
			// crushed if something stands there; a living one then spins before it jumps.
			u, v := cellCenter(f.Cell)
			if w.blocker(l, t, false, u, v, t.Template.Bounds(), nil) != nil {
				t.U, t.V = u, v
				w.removeThing(l, t, "crushed")
				return
			}
			t.U, t.V = u, v
			if alive(t) && !t.Template.Has(content.BehaviorAnimated) {
				if t.spin == 0 {
					t.spin, t.spinNext = spinTicks, w.step+Steps(1)
				}
				return
			}
		}
		w.teleportThing(l, t, w.jump(l, f), depth+1)
	case (f.Kind == content.FeaturePit || f.Kind == content.FeatureTrapdoor && f.Open) && t.Template.Kind == content.ThingCreature:
		w.removeThing(l, t, "fell")
	}
}

// teleportThing places thing t, sent by a teleporter, at the center of cell to and runs the
// effects of the cells it leaves and enters. Something already standing there crushes t instead:
// it is removed where it arrived, a creature dropping what it carries (docs/spec/entities.md,
// Placement). It reports whether t is still in the level.
func (w *World) teleportThing(l *Level, t *Thing, to content.Cell, depth int) bool {
	from := t.Cell()
	u, v := cellCenter(to)
	if w.blocker(l, t, false, u, v, t.Template.Bounds(), nil) != nil {
		w.weigh(l, from, -t.weight(), depth)
		t.U, t.V = u, v
		l.removeThing(t)
		w.emit(Event{Kind: EventThing, Level: l.ID, Cell: to, Thing: t.ID, Detail: "crushed"})
		for _, o := range t.Items {
			w.drop(l, to, o, byWorld, 0)
		}
		return false
	}
	t.U, t.V = u, v
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: to, Thing: t.ID, Detail: "moved"})
	if from != to {
		w.weigh(l, from, -t.weight(), depth)
		w.weigh(l, to, t.weight(), depth)
	}
	w.entered(l, t, depth)
	return l.Thing(t.ID) == t
}

// removeThing takes t out of the level, saying why, and lifts its weight off any plate; a
// creature that does not fall drops what it carries.
func (w *World) removeThing(l *Level, t *Thing, why string) {
	l.removeThing(t)
	w.weigh(l, t.Cell(), -t.weight(), 0)
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: why})
	if why != "fell" {
		for _, o := range t.Items {
			w.drop(l, t.Cell(), o, byWorld, 0)
		}
	}
}
