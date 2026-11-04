package sim

import (
	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Flight constants (docs/spec/entities.md, Things in flight; docs/spec/items.md, Throwing).
const (
	flightSpeed     = 8
	flightFootprint = 8
	flightTicks     = 1
	firedRange      = 127
	arrowBreaks     = 25
	throwDelay      = 15
	// touchReach is how far ahead of its footprint a thrower touches what it hits at once.
	touchReach = 4
	// bowClass is the weapon class throwing and firing practice.
	bowClass = 3
)

// Flight is a thing in flight: an object thrown or fired, or a spell.
type Flight struct {
	// Object is the object in flight, nil for a spell; Spell the spell, nil for an object.
	Object *Object
	Spell  *SpellFlight
	// U and V are its ground point; Direction its table direction; Range the steps it has left.
	U, V, Direction, Range int
	Damage                 int
	// ByPlayer flights land and turn creatures hostile; others vanish. Thrower is the thing that
	// threw it, if a creature did. Fired is an object shot rather than thrown.
	ByPlayer bool
	Fired    bool
	Thrower  *Thing `json:"-"`
	waited   bool
	next     int64
	// glide is where it was at its last update, to draw it gliding.
	glide glide
}

// still has flight f drawn where it is until it moves again.
func (f *Flight) still() { f.glide = glide{} }

// Cell returns the cell the flight is over.
func (f *Flight) Cell() content.Cell { return cellOf(f.U, f.V) }

// missileDamage is the damage object o does in flight: its missile's dice (1 thrown when only
// firing rolls them), or weight / 10 + 1.
func (w *World) missileDamage(o *Object, fired bool) int {
	it := w.pack.Items[o.Item]
	if it.Missile != nil {
		if it.Missile.FiredOnly && !fired {
			return 1
		}
		return w.roll(it.Missile.Damage)
	}
	return o.Weight(w.pack)/10 + 1
}

// launch sends object o flying from (u, v) along direction d with range and damage. A thrower
// touching something already hits it at once, and the object lands at its feet.
func (w *World) launch(l *Level, o *Object, u, v, d, rng, damage int, byPlayer, fired bool, thrower *Thing) {
	f := &Flight{Object: o, U: u, V: v, Direction: d, Range: rng, Damage: damage, ByPlayer: byPlayer, Fired: fired, Thrower: thrower,
		next: w.step + Steps(flightTicks)}
	f.glide.start(u, v, w.step, f.next)
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: f.Cell(), Item: o.Item, Detail: "thrown"})
	if b := w.flightBlocker(l, f, u+Dirs[d][0]*touchReach, v+Dirs[d][1]*touchReach, flightFootprint+touchReach); b != nil {
		w.flightHits(l, f, b)
		w.land(l, f)
		return
	}
	l.Flights = append(l.Flights, f)
}

// flightBlocker returns what a flight at (u, v) with a footprint n wide overlaps: any living
// thing or the player, but not its thrower, nor the player when the player threw it.
func (w *World) flightBlocker(l *Level, f *Flight, u, v, n int) *body {
	for _, b := range w.bodies(l) {
		switch {
		case b.thing == nil && f.ByPlayer, b.thing != nil && b.thing == f.Thrower:
			continue
		case b.thing != nil && b.thing.Template.Kind == content.ThingCreature && b.thing.HitPoints <= 0:
			continue
		}
		if overlaps(u, v, n, b.u, b.v, b.n) {
			return &b
		}
	}
	return nil
}

// flightHits hurts what flight f met: arrows as arrow damage, anything else physical.
func (w *World) flightHits(l *Level, f *Flight, b *body) {
	k := combat.Physical
	if it := w.pack.Items[f.Object.Item]; it != nil && hasTag(it, "arrows") {
		k = combat.Arrow
	}
	it := w.pack.Items[f.Object.Item]
	poisons := it != nil && it.PoisonsThrown
	if b.thing == nil {
		w.learnItem(f.Object.Item, StageUsed)
		w.hurtPlayer(f.Damage, k, f.Object.Item)
		if poisons {
			w.poisonPlayer()
		}
		return
	}
	t := b.thing
	if f.ByPlayer {
		w.turnHostile(t)
	}
	if t.Template.Kind == content.ThingCreature && l == w.Current() {
		w.learnItem(f.Object.Item, StageUsed)
		if poisons {
			w.meetDefense(l, t, content.BehaviorResistsPoison)
		}
	}
	w.hurtThing(l, t, f.Damage, k)
	if poisons && alive(t) && !t.Template.Has(content.BehaviorResistsPoison) {
		t.Poisoned = true
	}
}

// updateFlights moves level l's things in flight whose tick has come.
func (w *World) updateFlights(l *Level) {
	for _, f := range append([]*Flight(nil), l.Flights...) {
		if w.step < f.next {
			continue
		}
		f.next = w.step + Steps(flightTicks)
		f.glide.start(f.U, f.V, w.step, f.next)
		if f.Spell != nil {
			w.updateSpell(l, f)
			continue
		}
		if !f.waited {
			f.waited = true
			continue
		}
		if f.Range--; f.Range <= 0 {
			w.land(l, f)
			continue
		}
		speed := flightSpeed
		if f.Direction == 1 || f.Direction == 5 {
			speed /= 2
		}
		u, v := f.U+Dirs[f.Direction][0]*speed, f.V+Dirs[f.Direction][1]*speed
		c := cellOf(u, v)
		if l.Blocked(c) {
			if door := l.byCell[c]; door != nil && door.Kind == content.FeatureDoor && door.SealedBy != "" && door.SealedBy == f.Object.Item {
				w.unseal(l, door)
				w.removeFlight(l, f)
				w.emit(Event{Kind: EventObject, Level: l.ID, Cell: c, Item: f.Object.Item, Detail: "used"})
				continue
			}
			w.land(l, f)
			continue
		}
		if b := w.flightBlocker(l, f, u, v, flightFootprint); b != nil {
			w.flightHits(l, f, b)
			w.land(l, f)
			continue
		}
		from := f.Cell()
		f.U, f.V = u, v
		if to := f.Cell(); to != from {
			if t := l.byCell[to]; t != nil && t.Kind == content.FeatureTeleporter && t.Armed && !t.FlyOver {
				dest := w.jump(l, t)
				f.U, f.V = CellUnits*dest[0]+f.U-CellUnits*to[0], CellUnits*dest[1]+f.V-CellUnits*to[1]
				f.Direction = (f.Direction + t.Turn + 8) % 8
				f.still()
			}
		}
	}
}

// unseal opens sealed door f at once.
func (w *World) unseal(l *Level, f *Feature) {
	f.SealedBy, f.Stage, f.State = "", stageOpen, content.DoorOpen
	w.playSparkle(l, f.Sparkle, f.Cell, "", f.ID)
	w.emit(Event{Kind: EventSound, Level: l.ID, Cell: f.Cell, Detail: soundUnseal})
	w.featureChanged(l, f)
}

func (w *World) removeFlight(l *Level, f *Flight) {
	for i, x := range l.Flights {
		if x == f {
			l.Flights = append(l.Flights[:i], l.Flights[i+1:]...)
			return
		}
	}
}

// land ends flight f: the player's object comes down on the cell below it, an arrow breaking 1
// time in 25 and a bottle into its broken item; anyone else's vanishes.
func (w *World) land(l *Level, f *Flight) {
	w.removeFlight(l, f)
	c, o := f.Cell(), f.Object
	it := w.pack.Items[o.Item]
	switch {
	case !f.ByPlayer:
		w.emit(Event{Kind: EventObject, Level: l.ID, Cell: c, Item: o.Item, Detail: "vanished"})
		return
	case hasTag(it, "arrows") && w.chance(1, arrowBreaks):
		w.emit(Event{Kind: EventObject, Level: l.ID, Cell: c, Item: o.Item, Detail: "broke"})
		return
	case it.BreaksInto != "":
		w.emit(Event{Kind: EventObject, Level: l.ID, Cell: c, Item: o.Item, Detail: "broke"})
		if l.ID == w.current && l.Visible(c) {
			w.cue(c, content.CueShatter)
		}
		o = w.newObject(it.BreaksInto)
		w.broke(l, it)
	}
	w.drop(l, c, o, byWorld, 0)
}

// throwFrom throws the object in hand (0 right, 1 left) along the player's facing
// (docs/spec/items.md, Throwing).
func (w *World) throwFrom(hand int) {
	p, l := w.player, w.Current()
	o := p.Slots[SlotRightHand+hand]
	switch {
	case p.Dead:
		w.refuse("the player is dead")
		return
	case hand < 0 || hand > 1 || o == nil:
		w.refuse("nothing to throw")
		return
	case p.Delays[hand] > 0:
		w.refuse("the hand is not ready")
		return
	case w.refuseRunning():
		return
	case w.HandBlocked(hand):
		w.refuse("the hand is blocked")
		return
	case w.deepWater():
		w.message(l, p.Cell(), content.MsgCannotThrow, "")
		return
	}
	p.Slots[SlotRightHand+hand] = nil
	w.takeOff(o)
	w.setDelay(hand, throwDelay)
	rng := (100-min(o.Weight(w.pack), 100))/5 + 5
	w.launch(l, o, p.U, p.V, p.Facing, rng, w.missileDamage(o, false), true, false, nil)
	w.practice(SkillWeapon+bowClass, 1)
}

// fire shoots the quiver's first arrow with a bow (docs/spec/items.md, Throwing).
func (w *World) fire() bool {
	p, l := w.player, w.Current()
	quiver := p.Slots[SlotQuiver]
	speed := p.Template.Speed
	switch {
	case quiver == nil:
		w.message(l, p.Cell(), content.MsgNoQuiver, "")
		return false
	case len(quiver.Contents) == 0:
		w.message(l, p.Cell(), content.MsgNoArrows, "")
		return false
	case w.blocker(l, nil, true, p.U+Dirs[p.Facing][0]*speed, p.V+Dirs[p.Facing][1]*speed, p.Template.Size(), nil) != nil:
		w.message(l, p.Cell(), content.MsgNoRoomToShoot, "")
		return false
	}
	arrow := quiver.Contents[0]
	quiver.Contents = quiver.Contents[1:]
	damage := max(w.missileDamage(arrow, true)+(p.Attributes[AttrAccuracy]-10)/2, 1)
	w.launch(l, arrow, p.U, p.V, p.Facing, min(5*p.Attributes[AttrStrength], firedRange), damage, true, true, nil)
	w.pose(2)
	p.bowRelease = w.step + 1
	w.practice(SkillWeapon+bowClass, 1)
	return true
}

// shootItem fires a new object of item from cell c along direction d (arrow traps); it vanishes
// when it lands.
func (w *World) shootItem(l *Level, c content.Cell, item string, d int) {
	if w.pack.Items[item] == nil {
		return
	}
	o := w.newObject(item)
	u, v := cellCenter(c)
	w.launch(l, o, u, v, d, firedRange, w.missileDamage(o, true), false, true, nil)
}

// DeepWater reports whether the player stands in water too deep to act in, depth 3 or more,
// unless levitating; held objects are not drawn there.
func (w *World) DeepWater() bool { return w.player != nil && w.deepWater() }

// deepWater reports whether the player stands in water too deep to act in, depth 3 or more,
// unless levitating.
func (w *World) deepWater() bool {
	p := w.player
	ft := w.Current().FloorType(p.Cell())
	return ft != nil && ft.Depth >= 3 && !p.Status.Levitating
}

// refuse reports a refused command of the player's.
func (w *World) refuse(why string) {
	cell := content.Cell{}
	if w.player != nil {
		cell = w.player.Cell()
	}
	w.emit(Event{Kind: EventRefused, Level: w.current, Cell: cell, Detail: why})
}

// intoQuiver puts arrows into the worn quiver when it has room, and reports whether it did.
func (w *World) intoQuiver(o *Object) bool {
	q := w.player.Slots[SlotQuiver]
	it := w.pack.Items[o.Item]
	if q == nil || it == nil || !hasTag(it, "arrows") {
		return false
	}
	box := w.pack.Items[q.Item].Container
	if box == nil || len(q.Contents) >= box.Capacity || it.Size > box.Largest {
		return false
	}
	q.Contents = append(q.Contents, o)
	return true
}
