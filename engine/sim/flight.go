package sim

import (
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Flight constants (docs/spec/entities.md, Things in flight; docs/spec/items.md, Throwing).
const (
	flightSpeed = 8
	flightTicks = 1
	firedRange  = 127
	arrowBreaks = 25
	throwDelay  = 15
	// touchReach is how far ahead a thrower touches what it hits at once.
	touchReach = 4
	// bowClass is the weapon class throwing and firing practice.
	bowClass = 3
)

// flightBox is what a thing in flight collides by (docs/spec/entities.md, Collision boxes).
var flightBox = content.Box{-6, 5, 8, 8}

// Flight is a thing in flight: an object thrown or fired, or a spell.
type Flight struct {
	// Object is the object in flight, nil for a spell; Spell the spell, nil for an object.
	Object *Object
	Spell  *SpellFlight
	// U and V are its ground point; Direction its table direction; Range the steps it has left.
	U, V, Direction, Range int
	Damage                 int
	// Kind is the damage kind it does; Speed the units it moves a step, flightSpeed without one.
	// A Vanishes flight's object vanishes where it lands, as a bolt does.
	Kind     combat.Kind `json:"kind,omitempty"`
	Speed    int         `json:"speed,omitempty"`
	Vanishes bool        `json:"vanishes,omitempty"`
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
	if m := it.Missile; m != nil && m.Damage != (content.Dice{}) {
		if m.FiredOnly && !fired {
			return 1
		}
		return w.roll(m.Damage)
	}
	return o.Weight(w.pack)/10 + 1
}

// missileKind is the damage kind object o does in flight: its missile's, else the first.
func (w *World) missileKind(o *Object) combat.Kind {
	if it := w.pack.Items[o.Item]; it != nil && it.Missile != nil {
		return w.kind(it.Missile.Kind)
	}
	return combat.Physical
}

// shotOptions change a flight as it is launched: a shot's kind, speed and vanishing.
type shotOptions struct {
	kind     *combat.Kind
	speed    int
	vanishes bool
}

// launchDistance is how far ahead of its thrower, in steps of its direction, a thing in flight
// starts, by table direction (docs/spec/entities.md, Things in flight).
var launchDistance = [8]int{12, 12, 12, 14, 12, 8, 12, 12}

// launchStart returns where a flight from (u, v) along direction d starts, extra units further.
func launchStart(u, v, d, extra int) (int, int) {
	a := launchDistance[d] + extra
	return u + Dirs[d][0]*a, v + Dirs[d][1]*a
}

// launch sends object o flying from a thrower at (u, v) along direction d with range and damage,
// starting its launch distance ahead, and reports whether it flies. A thrower touching something
// hits it at once, and the object lands where its flight starts. A flight that would start in a
// blocked cell does not fly: the player's object stays with the player (too_close_to_wall),
// anyone else's is lost.
func (w *World) launch(l *Level, o *Object, u, v, d, rng, damage int, byPlayer, fired bool, thrower *Thing, opts ...shotOptions) bool {
	su, sv := launchStart(u, v, d, 0)
	f := &Flight{Object: o, U: su, V: sv, Direction: d, Range: rng, Damage: damage, ByPlayer: byPlayer, Fired: fired, Thrower: thrower,
		Kind: w.missileKind(o), next: w.step + Steps(flightTicks)}
	for _, op := range opts {
		if op.kind != nil {
			f.Kind = *op.kind
		}
		f.Speed, f.Vanishes = op.speed, op.vanishes
	}
	touched := w.flightBlocker(l, f, u+Dirs[d][0]*touchReach, v+Dirs[d][1]*touchReach, w.throwerBox(f))
	if touched != nil {
		w.flightHits(l, f, touched)
		f.Range = 1
	}
	if l.Blocked(f.Cell()) {
		if byPlayer && !f.Vanishes {
			w.message(l, cellOf(u, v), content.MsgTooCloseToWall, "")
		} else {
			w.emit(Event{Kind: EventObject, Level: l.ID, Cell: f.Cell(), Item: o.Item, Detail: "vanished"})
		}
		return false
	}
	f.glide.start(su, sv, w.step, f.next)
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: f.Cell(), Item: o.Item, Detail: "thrown"})
	l.Flights = append(l.Flights, f)
	return true
}

// flightMeets returns what flight f meets moving to (u, v): what its box overlaps where it is or
// there, tested before the terrain (docs/spec/entities.md, Things in flight).
func (w *World) flightMeets(l *Level, f *Flight, u, v int) *body {
	if b := w.flightBlocker(l, f, f.U, f.V, flightBox); b != nil {
		return b
	}
	return w.flightBlocker(l, f, u, v, flightBox)
}

// flightBlocker returns what box at (u, v) overlaps for flight f: any living
// thing or the player, but not its thrower, nor the player when the player threw it.
func (w *World) flightBlocker(l *Level, f *Flight, u, v int, box content.Box) *body {
	for _, b := range w.bodies(l) {
		switch {
		case b.thing == nil && f.ByPlayer, b.thing != nil && b.thing == f.Thrower:
			continue
		case b.thing != nil && b.thing.Template.Kind == content.ThingCreature && b.thing.HitPoints <= 0:
			continue
		}
		if overlaps(u, v, box, b.u, b.v, b.box) {
			return &b
		}
	}
	return nil
}

// flightHits hurts what flight f met, by the flight's damage kind.
func (w *World) flightHits(l *Level, f *Flight, b *body) {
	k := f.Kind
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
	if poisons && alive(t) && !w.resistsPoison(t) {
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
		if f.Speed > 0 {
			speed = f.Speed
		}
		if f.Direction == 1 || f.Direction == 5 {
			speed /= 2
		}
		u, v := f.U+Dirs[f.Direction][0]*speed, f.V+Dirs[f.Direction][1]*speed
		if b := w.flightMeets(l, f, u, v); b != nil {
			w.flightHits(l, f, b)
			w.land(l, f)
			continue
		}
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
	case !f.ByPlayer || f.Vanishes:
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
	rng := (100-min(o.Weight(w.pack), 100))/5 + 5
	if !w.launch(l, o, p.U, p.V, p.Facing, rng, w.missileDamage(o, false), true, false, nil) {
		return
	}
	p.Slots[SlotRightHand+hand] = nil
	w.takeOff(o)
	w.setDelay(hand, throwDelay)
	w.practice(SkillWeapon+bowClass, 1)
}

// roomToShoot reports whether the player has room to fire: whether its box moved a full step
// along its facing overlaps nothing but corpses (docs/spec/items.md, Engine model).
func (w *World) roomToShoot() bool {
	p, l := w.player, w.Current()
	speed := p.Template.Speed
	if speed == 0 {
		speed = 8
	}
	u, v := p.U+Dirs[p.Facing][0]*speed, p.V+Dirs[p.Facing][1]*speed
	box := p.Template.Bounds()
	for _, b := range w.bodies(l) {
		if b.thing == nil || b.thing.Template.Kind == content.ThingCreature && b.thing.HitPoints <= 0 {
			continue
		}
		if overlaps(u, v, box, b.u, b.v, b.box) {
			return false
		}
	}
	return true
}

// fire shoots with weapon o's fire action a (docs/spec/items.md, Throwing; docs/pack.md, Items):
// the first object of the ammunition's tag in what the ammunition's slot holds, or a new object
// of the shot's item paid for with the weapon's charges or the player's magic points.
func (w *World) fire(o *Object, a content.ItemAction) bool {
	p, l := w.player, w.Current()
	refuse := func(msg string) bool {
		w.message(l, p.Cell(), msg, "")
		return false
	}
	shot := content.Shot{}
	if a.Shot != nil {
		shot = *a.Shot
	}
	var arrow *Object
	var holder *Object
	at := -1
	switch {
	case a.Charges > 0:
		if a.NeedsRoom && !w.roomToShoot() {
			return refuse(content.MsgNoRoomToShoot)
		}
		if w.Quantity(o) < a.Charges {
			return refuse(content.MsgNoCharges)
		}
		arrow = w.newObject(shot.Item)
	case a.Power > 0:
		if a.NeedsRoom && !w.roomToShoot() {
			return refuse(content.MsgNoRoomToShoot)
		}
		if p.MagicPoints < a.Power {
			return refuse(content.MsgNoPower)
		}
		arrow = w.newObject(shot.Item)
	default:
		ammo := content.DefaultAmmo
		if a.Ammo != nil {
			ammo = *a.Ammo
		}
		if ammo.Slot >= 0 && ammo.Slot < len(p.Slots) {
			holder = p.Slots[ammo.Slot]
		}
		if holder == nil {
			return refuse(content.MsgNoQuiver)
		}
		if a.NeedsRoom && !w.roomToShoot() {
			return refuse(content.MsgNoRoomToShoot)
		}
		at = slices.IndexFunc(holder.Contents, func(c *Object) bool {
			it := w.pack.Items[c.Item]
			return it != nil && hasTag(it, ammo.Tag)
		})
		if at < 0 {
			return refuse(content.MsgNoArrows)
		}
		arrow = holder.Contents[at]
	}
	if arrow == nil {
		return refuse(content.MsgNoArrows)
	}
	damage := w.missileDamage(arrow, true)
	if shot.Damage != (content.Dice{}) {
		damage = w.roll(shot.Damage)
	}
	damage = max(damage+(p.Attributes[AttrAccuracy]-10)/2, 1)
	rng := content.DefaultRange
	if shot.Range != nil {
		rng = *shot.Range
	}
	opts := shotOptions{speed: shot.Speed, vanishes: holder == nil}
	if shot.Kind != "" {
		k := w.kind(shot.Kind)
		opts.kind = &k
	}
	if !w.launch(l, arrow, p.U, p.V, p.Facing, rng.Steps(p.Attributes), damage, true, true, nil, opts) {
		return false
	}
	switch {
	case holder != nil:
		holder.Contents = slices.Delete(holder.Contents, at, at+1)
	case a.Charges > 0:
		for range a.Charges {
			w.useOne(o)
		}
	default:
		p.MagicPoints -= a.Power
	}
	w.pose(2)
	p.bowRelease = w.step + 1
	w.practice(SkillWeapon+w.fireClass(o), 1)
	return true
}

// fireClass is the weapon class firing weapon o practices: its own, else the bow's.
func (w *World) fireClass(o *Object) int {
	if it := w.pack.Items[o.Item]; it != nil && it.WeaponClass != nil {
		return *it.WeaponClass
	}
	return bowClass
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

// throwerBox is the box that tests what flight f's thrower touches as it throws: the player's or
// the creature's own, or the flight's when a cell threw it.
func (w *World) throwerBox(f *Flight) content.Box {
	switch {
	case f.ByPlayer && w.player != nil:
		return w.player.Template.Bounds()
	case f.Thrower != nil:
		return f.Thrower.Template.Bounds()
	}
	return flightBox
}
