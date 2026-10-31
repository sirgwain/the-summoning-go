package sim

import (
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// PlayerID is the id events and draw lists give the player.
const PlayerID = "player"

// Inventory slots (docs/spec/items.md, Inventory).
const (
	// Shelves are slots 0 .. Shelves−1.
	Shelves       = 20
	SlotHead      = 20
	SlotArms      = 21
	SlotLeftHand  = 22
	SlotRightHand = 23
	SlotFeet      = 24
	SlotQuiver    = 25
	SlotBody      = 26
	SlotNeck      = 27
	// SlotCount is the number of fixed slots; SlotPack is the first pack slot after them.
	SlotCount = 28
	SlotPack  = SlotCount
)

// Player timing (docs/spec/player.md, Facing and walking; Status tick).
const (
	// idleUpdates is how many updates without walking show the idle frame.
	idleUpdates = 8
	// statusTicks is how often the status tick comes.
	statusTicks = 109
	// playerWeight is what the player adds to a plate when its template gives none.
	playerWeight = 100
)

// Attribute names, as a player template's stats give them (docs/spec/player.md, Attributes).
const (
	AttrStrength  = "strength"
	AttrAgility   = "agility"
	AttrEndurance = "endurance"
	AttrAccuracy  = "accuracy"
	AttrTalent    = "talent"
	AttrPower     = "power"
	// StatIntervalBonus is the player template's stat giving the starting interval bonus.
	StatIntervalBonus = "interval_bonus"
)

// Player is the player: it belongs to no level and goes with the world from level to level.
type Player struct {
	Template *content.Thing
	// U and V are its ground point on the current level, in world units.
	U, V int
	// Facing is a table direction.
	Facing                      int
	HitPoints, MaxHitPoints     int
	MagicPoints, MaxMagicPoints int
	// Attributes are the player's attributes by name.
	Attributes map[string]int
	// IntervalBonus is the ticks worn objects and powers add to the interval.
	IntervalBonus int
	// Hand is the object held on the pointer; Slots the inventory's fixed slots.
	Hand  *Object
	Slots [SlotCount]*Object
	// Moves counts the moves made, for the walk animation; Idle the updates since the last.
	Moves, Idle int

	// want is the direction a walk was asked for since the last update, or −1.
	want int
	next int64
	// vision is the cell visibility is computed from.
	vision content.Cell
	// statusTicks counts status ticks, for what comes every fourth one.
	statusTicks int
}

// Cell returns the cell the player stands in.
func (p *Player) Cell() content.Cell { return cellOf(p.U, p.V) }

// Walking reports whether the player shows its walk rather than its idle frame.
func (p *Player) Walking() bool { return p.Moves > 0 && p.Idle < idleUpdates }

// weight is what the player adds to a plate.
func (p *Player) weight() int {
	if p.Template.Weight > 0 {
		return p.Template.Weight
	}
	return playerWeight
}

// Pack returns the container in the right hand, whose contents are the pack slots, or nil.
func (p *Player) Pack(pk *content.Pack) *Object {
	o := p.Slots[SlotRightHand]
	if o == nil || pk.Items[o.Item] == nil || pk.Items[o.Item].Container == nil {
		return nil
	}
	return o
}

// Slot returns the object in slot i, pack slots included, or nil.
func (p *Player) Slot(pk *content.Pack, i int) *Object {
	switch {
	case i >= 0 && i < SlotCount:
		return p.Slots[i]
	case i >= SlotPack:
		if bag := p.Pack(pk); bag != nil && i-SlotPack < len(bag.Contents) {
			return bag.Contents[i-SlotPack]
		}
	}
	return nil
}

// Player returns the player, or nil when the pack has none.
func (w *World) Player() *Player { return w.player }

// newPlayer makes the player from the pack's player template, or returns nil.
func (w *World) newPlayer() *Player {
	tmpl := w.pack.Things[w.pack.Manifest.Player]
	if tmpl == nil {
		return nil
	}
	p := &Player{Template: tmpl, Facing: tmpl.Facing, want: -1, Attributes: map[string]int{}}
	for k, v := range tmpl.Stats {
		if k == StatIntervalBonus {
			p.IntervalBonus = v
			continue
		}
		p.Attributes[k] = v
	}
	p.MaxHitPoints = w.roll(tmpl.HitPoints)
	p.MaxMagicPoints = w.roll(tmpl.MagicPoints)
	p.HitPoints, p.MagicPoints = p.MaxHitPoints, p.MaxMagicPoints
	return p
}

// place puts the player at the center of cell c of the current level: things in its way are
// crushed, its weight goes on the cell's plate, and the cell becomes its vision cell. A plate
// pressed by the arrival runs its hooks unless quiet.
func (w *World) place(c content.Cell, quiet bool) {
	p, l := w.player, w.Current()
	p.U, p.V = cellCenter(c)
	for _, t := range append([]*Thing(nil), l.Things...) {
		if overlaps(p.U, p.V, p.Template.Size(), t.U, t.V, t.Template.Size()) {
			w.removeThing(l, t, "crushed")
		}
	}
	if quiet {
		l.addWeight(c, p.weight())
	} else {
		w.weigh(l, c, p.weight(), 0)
	}
	p.vision = c
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: c, Thing: PlayerID, Detail: "placed"})
	w.see()
}

// travel takes the player to a level and cell: it leaves its plate, the level changes, and it
// arrives (docs/spec/rules.md, The player and the features).
func (w *World) travel(to *content.Destination) {
	p, l := w.player, w.Current()
	if _, err := w.load(to.Level); err != nil {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Detail: err.Error()})
		return
	}
	w.weigh(l, p.Cell(), -p.weight(), 0)
	if err := w.enter(to.Level); err != nil {
		return
	}
	w.place(to.Cell, false)
}

// hurtPlayer takes n hit points from the player, saying what hurt it.
func (w *World) hurtPlayer(n int, by string) {
	p := w.player
	if p == nil || n <= 0 {
		return
	}
	p.HitPoints = max(0, p.HitPoints-n)
	w.emit(Event{Kind: EventThing, Level: w.current, Cell: p.Cell(), Thing: PlayerID, Detail: "hurt by " + by})
}

// updatePlayer is the player's update: it walks if a walk was asked for (docs/spec/entities.md,
// The player's update).
func (w *World) updatePlayer() {
	p := w.player
	p.next = w.step + w.PlayerInterval()
	d := p.want
	p.want = -1
	if d < 0 {
		if p.Idle < idleUpdates {
			if p.Idle++; p.Idle == idleUpdates {
				w.emit(Event{Kind: EventThing, Level: w.current, Cell: p.Cell(), Thing: PlayerID, Detail: "idle"})
			}
		}
		return
	}
	turned := p.Facing != d
	p.Facing = d
	if !w.walk(d) && turned {
		w.emit(Event{Kind: EventThing, Level: w.current, Cell: p.Cell(), Thing: PlayerID, Detail: "turned"})
	}
}

// Carried returns the weight the player carries: the hand and every slot, with their contents
// (docs/spec/items.md, Weight and containers).
func (p *Player) Carried(pk *content.Pack) int {
	w := 0
	if p.Hand != nil {
		w += p.Hand.Weight(pk)
	}
	for _, o := range p.Slots {
		if o != nil {
			w += o.Weight(pk)
		}
	}
	return w
}

// Agility returns the player's agility as its load leaves it, and the ticks a heavy load adds
// besides (docs/spec/player.md, Speed).
func (p *Player) Agility(pk *content.Pack) (agility, extra int) {
	agility = p.Attributes[AttrAgility]
	w, limit := p.Carried(pk), 10*p.Attributes[AttrStrength]
	switch {
	case w < limit/10:
		agility++
	case w >= limit:
		pen := (w-limit)/20 + 1
		agility -= pen
		extra = pen + 1
	}
	return agility, extra
}

// IntervalTicks returns the ticks the player's speed adds to its template's base interval:
// load and agility, and low endurance. A player without attributes adds none.
func (p *Player) IntervalTicks(pk *content.Pack) int {
	if len(p.Attributes) == 0 {
		return p.IntervalBonus
	}
	agility, n := p.Agility(pk)
	n += p.IntervalBonus
	switch {
	case agility >= 15:
		n--
	case agility < 10:
		n += 10 - agility
	}
	if end := p.Attributes[AttrEndurance]; end < 10 {
		n += (10-end)/2 + 1
	}
	return n
}

// PlayerInterval returns the steps between the player's updates.
func (w *World) PlayerInterval() int64 {
	p := w.player
	ms := float64(p.Template.IntervalMS) + float64(p.IntervalTicks(w.pack))*1000/ticksPerSecond
	return max(1, int64(ms*StepsPerSecond/1000+0.5))
}

// walk moves the player a step along direction d and reports whether it moved
// (docs/spec/entities.md, Moving).
func (w *World) walk(d int) bool {
	p, l := w.player, w.Current()
	speed := p.Template.Speed
	if d == 1 || d == 5 {
		speed /= 2
	}
	du, dv := Dirs[d][0]*speed, Dirs[d][1]*speed

	// Terrain: each axis alone, from where the player stands.
	su, sv := du, dv
	for ; su != 0; su -= sign(su) {
		ok, f := w.probe(l, p.U+su, p.V, d)
		if f != nil {
			w.takeFeature(f, d)
			return true
		}
		if ok {
			break
		}
	}
	for ; sv != 0; sv -= sign(sv) {
		ok, f := w.probe(l, p.U, p.V+sv, d)
		if f != nil {
			w.takeFeature(f, d)
			return true
		}
		if ok {
			break
		}
	}

	// Things: the player walks out of anything it already overlaps.
	n := p.Template.Size()
	skip := map[*Thing]bool{}
	for _, t := range l.Things {
		if overlaps(p.U, p.V, n, t.U, t.V, t.Template.Size()) {
			skip[t] = true
		}
	}
	var hit *body
	for su != 0 || sv != 0 {
		b := w.blocker(l, nil, true, p.U+su, p.V+sv, n, skip)
		if b == nil {
			break
		}
		hit = b
		su, sv = toward0(su, backOff), toward0(sv, backOff)
	}
	if su == 0 && sv == 0 && hit != nil && hit.thing != nil {
		t := hit.thing
		switch {
		case pushable(t):
			from := [2]int{t.U, t.V}
			if w.push(l, t, du, dv, 0) {
				su, sv = t.U-from[0], t.V-from[1]
				p.next += Steps(pushTicks)
				w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "pushed"})
			}
		case t.Template.Kind == content.ThingBoulder:
			if t.Rolling != nil && t.Rolling.Direction != d {
				w.hurtPlayer(w.roll(content.Dice{Count: 2, Sides: 3}), "boulder")
			}
			if d%2 == 0 {
				w.startBoulder(t, d, false)
			}
		}
	}
	if su == 0 && sv == 0 && hit == nil && d%2 == 0 {
		su, sv = w.slideOff(l, du, dv)
	}
	if su == 0 && sv == 0 {
		return false
	}
	w.movePlayerBy(l, su, sv)
	return true
}

// slideOff returns the slide along the diagonal wall ahead of a step (du, dv) along one axis, if
// the step walks into its filled corner (docs/spec/entities.md, Moving).
func (w *World) slideOff(l *Level, du, dv int) (int, int) {
	p := w.player
	c := cellOf(p.U+du, p.V+dv)
	if !l.InBounds(c) || l.Wall[c[0]][c[1]] == content.NoFrame {
		return 0, 0
	}
	set := w.pack.Walls[l.Def.Walls]
	if set == nil {
		return 0, 0
	}
	shape := set.Shape(l.Wall[c[0]][c[1]])
	if shape == nil {
		return 0, 0
	}
	cu, cv, ok := content.CornerSigns(shape.Diagonal)
	var su, sv int
	switch {
	case !ok:
		return 0, 0
	case dv == 0 && sign(du) == cu:
		sv = -cv * slide
	case du == 0 && sign(dv) == cv:
		su = -cu * slide
	default:
		return 0, 0
	}
	if ok, _ := w.probe(l, p.U+su, p.V+sv, -1); !ok {
		return 0, 0
	}
	return su, sv
}

// probe reports whether the player's ground point may stand at (u, v) walking along d (−1 for
// no walk), and returns a feature that takes the player instead: an exit in a wall walked into
// along an axis, or a pit or open trapdoor.
func (w *World) probe(l *Level, u, v, d int) (bool, *Feature) {
	c := cellOf(u, v)
	if !l.InBounds(c) {
		return false, nil
	}
	f := l.byCell[c]
	if l.Wall[c[0]][c[1]] != content.NoFrame {
		if f != nil && f.Kind == content.FeatureExit && d >= 0 && d%2 == 0 {
			return false, f
		}
		return false, nil
	}
	if l.Blocked(c) {
		return false, nil
	}
	if f != nil && d >= 0 && (f.Kind == content.FeaturePit || f.Kind == content.FeatureTrapdoor && f.Open) {
		return false, f
	}
	return true, nil
}

// takeFeature takes the player through an exit, or down a pit or open trapdoor walked into
// along direction d.
func (w *World) takeFeature(f *Feature, d int) {
	if f.To == nil {
		return
	}
	if f.Kind != content.FeatureExit {
		w.fallPlayer(f, d)
		return
	}
	w.travel(f.To)
}

// fallPlayer drops the player through pit or trapdoor f, hurting it 4d5 unless it steps down
// carefully, along directions 1–3.
func (w *World) fallPlayer(f *Feature, d int) {
	if d < 1 || d > 3 {
		w.hurtPlayer(w.roll(content.Dice{Count: 4, Sides: 5}), "falling")
	}
	w.travel(f.To)
}

// movePlayerBy moves the player by (du, dv) and runs what its new cell does.
func (w *World) movePlayerBy(l *Level, du, dv int) {
	p := w.player
	from := p.Cell()
	p.U, p.V = p.U+du, p.V+dv
	p.Moves++
	p.Idle = 0
	to := p.Cell()
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: to, Thing: PlayerID, Detail: "moved"})
	if ft := l.FloorType(to); ft != nil && ft.Depth == 4 && w.chance(1, 4) {
		w.hurtPlayer(1, "deep water")
	}
	if from != to {
		w.weigh(l, from, -p.weight(), 0)
		w.weigh(l, to, p.weight(), 0)
		w.playerEntered(l, to, 0)
	}
}

// playerEntered runs what cell c does to the player entering it: its floor's effect, a floor
// exit, a teleporter (docs/spec/rules.md, The player and the features).
func (w *World) playerEntered(l *Level, c content.Cell, depth int) {
	if depth >= maxDepth || w.current != l.ID {
		return
	}
	if ft := l.FloorType(c); ft != nil && ft.Enter != nil {
		w.floorEffect(ft.Enter)
	}
	f := l.byCell[c]
	switch {
	case f == nil:
	case f.Kind == content.FeatureExit && f.To != nil:
		w.travel(f.To)
	case f.Kind == content.FeatureTeleporter && f.Armed:
		w.teleportPlayer(l, f, depth)
	case f.Kind == content.FeaturePit || f.Kind == content.FeatureTrapdoor && f.Open:
		w.fallPlayer(f, -1)
	}
}

// teleportPlayer moves the player through teleporter f, keeping its place within the cell.
func (w *World) teleportPlayer(l *Level, f *Feature, depth int) {
	p := w.player
	from := p.Cell()
	to := w.jump(l, f)
	w.emit(Event{Kind: EventSound, Level: l.ID, Cell: f.Cell, Detail: soundTeleport})
	p.U, p.V = CellUnits*to[0]+p.U-CellUnits*from[0], CellUnits*to[1]+p.V-CellUnits*from[1]
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: to, Thing: PlayerID, Detail: "teleported"})
	w.weigh(l, from, -p.weight(), depth)
	w.weigh(l, to, p.weight(), depth)
	w.playerEntered(l, to, depth+1)
}

// floorEffect applies a floor's effect to the player.
func (w *World) floorEffect(e *content.Effect) {
	p := w.player
	if e.Damage != (content.Dice{}) {
		by := "the floor"
		if e.Type != "" {
			by = e.Type
		}
		w.hurtPlayer(w.roll(e.Damage), by)
	}
	if e.DrainMagic > 0 {
		p.MagicPoints = max(0, p.MagicPoints-e.DrainMagic)
		w.emit(Event{Kind: EventThing, Level: w.current, Cell: p.Cell(), Thing: PlayerID, Detail: "drained"})
	}
}

// statusTick runs the player's status tick: the floor's tick effect, deep water, and a magic
// point every fourth tick (docs/spec/player.md, Status tick).
func (w *World) statusTick() {
	p, l := w.player, w.Current()
	if ft := l.FloorType(p.Cell()); ft != nil {
		if ft.Tick != nil {
			w.floorEffect(ft.Tick)
		}
		if ft.Depth == 4 {
			w.hurtPlayer(1, "deep water")
		}
	}
	if p.statusTicks++; p.statusTicks%4 == 0 && p.MagicPoints < p.MaxMagicPoints {
		p.MagicPoints++
	}
}

// pickUp takes the top object of the player's cell into the hand, or the first empty shelf
// (docs/spec/items.md, Engine model).
func (w *World) pickUp() {
	p, l := w.player, w.Current()
	c := p.Cell()
	objs := l.Objects[c]
	if len(objs) == 0 {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: c, Detail: "nothing to pick up"})
		return
	}
	o := objs[len(objs)-1]
	slot := -1
	if p.Hand != nil {
		for i := range Shelves {
			if p.Slots[i] == nil {
				slot = i
				break
			}
		}
		if slot < 0 {
			w.message(l, c, content.MsgNoRoom, o.Item)
			return
		}
	}
	w.takeFloorObject(l, c)
	if slot < 0 {
		p.Hand = o
	} else {
		p.Slots[slot] = o
	}
}

// takeFloorObject takes the top object off cell c's floor, lifting its weight off a plate.
func (w *World) takeFloorObject(l *Level, c content.Cell) *Object {
	objs := l.Objects[c]
	o := objs[len(objs)-1]
	l.Objects[c] = objs[:len(objs)-1]
	if len(l.Objects[c]) == 0 {
		delete(l.Objects, c)
	}
	w.emit(Event{Kind: EventObject, Level: l.ID, Cell: c, Item: o.Item, Detail: "taken"})
	w.weigh(l, c, -o.Weight(w.pack), 0)
	return o
}

// putDown puts the hand's object down on the player's cell; a refusal keeps it in hand.
func (w *World) putDown() {
	p, l := w.player, w.Current()
	if p.Hand == nil {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: p.Cell(), Detail: "nothing in hand"})
		return
	}
	if w.drop(l, p.Cell(), p.Hand, byPlayer, 0) {
		p.Hand = nil
	}
}

// swap exchanges the hand with slot i (docs/spec/items.md, Engine model).
func (w *World) swap(i int) {
	p, l := w.player, w.Current()
	refuse := func(why string) {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: p.Cell(), Detail: why})
	}
	switch {
	case i >= 0 && i < Shelves, i == SlotLeftHand, i == SlotRightHand:
		p.Hand, p.Slots[i] = p.Slots[i], p.Hand
	case i >= Shelves && i < SlotCount:
		refuse("worn slots come with Fight")
	case i >= SlotPack:
		bag := p.Pack(w.pack)
		if bag == nil {
			refuse("no bag in the right hand")
			return
		}
		box := w.pack.Items[bag.Item].Container
		k := i - SlotPack
		if k >= box.Capacity {
			refuse("the bag has no such place")
			return
		}
		if p.Hand != nil && w.pack.Items[p.Hand.Item].Size > box.Largest {
			refuse("too big for the bag")
			return
		}
		switch {
		case k < len(bag.Contents) && p.Hand != nil:
			p.Hand, bag.Contents[k] = bag.Contents[k], p.Hand
		case k < len(bag.Contents):
			p.Hand = bag.Contents[k]
			bag.Contents = append(bag.Contents[:k], bag.Contents[k+1:]...)
		case p.Hand != nil:
			bag.Contents = append(bag.Contents, p.Hand)
			p.Hand = nil
		}
	default:
		refuse("no such slot")
		return
	}
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: p.Cell(), Thing: PlayerID, Detail: "swapped"})
}

// activate uses the lever in the player's cell, or else the nearest mouth in reach
// (docs/spec/rules.md, The player and the features).
func (w *World) activate() {
	p, l := w.player, w.Current()
	c := p.Cell()
	if f := l.byCell[c]; f != nil && f.Kind == content.FeatureLever {
		w.use(l, c)
		return
	}
	var best *Feature
	bestD := 0
	for _, f := range l.Features {
		if f.Kind != content.FeatureMouth || abs(f.Cell[0]-c[0]) > 1 || abs(f.Cell[1]-c[1]) > 1 {
			continue
		}
		mu, mv := cellCenter(f.Cell)
		if d := abs(mu-p.U) + abs(mv-p.V); best == nil || d < bestD {
			best, bestD = f, d
		}
	}
	if best == nil {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: c, Detail: "nothing to use"})
		return
	}
	w.use(l, best.Cell)
}
