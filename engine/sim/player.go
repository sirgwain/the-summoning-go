package sim

import (
	"strconv"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// PlayerID is the id events and draw lists give the player.
const PlayerID = "player"

// Inventory slots (docs/spec/items.md, Inventory).
const (
	// Shelves are slots 0 .. Shelves−1.
	Shelves       = 20
	SlotHead      = 20
	SlotArms      = 21
	SlotRightHand = 22
	SlotLeftHand  = 23
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
	// skillThreshold is every skill's first practice threshold.
	skillThreshold = 10
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
	Template *content.Thing `json:"-"`
	// U and V are its ground point on the current level, in world units.
	U, V int
	// Facing is a table direction.
	Facing                      int
	HitPoints, MaxHitPoints     int
	MagicPoints, MaxMagicPoints int
	// Attributes are the player's attributes by name, as they are now; MaxAttributes their
	// maxima, from which fatigue, boosts and worn objects move them.
	Attributes, MaxAttributes map[string]int
	// IntervalBonus is the ticks worn objects and powers add to the interval.
	IntervalBonus int
	// Portrait is the player's portrait, which gives its figure's colors.
	Portrait string
	// Hand is the object held on the pointer; Slots the inventory's fixed slots.
	Hand  *Object
	Slots [SlotCount]*Object
	// Opened is one more than the fixed slot whose container is open as the pack, 0 for none:
	// then the bag in the left hand is the pack.
	Opened int
	// Moves counts the moves made, for the walk animation; Idle the updates since the last.
	Moves, Idle int
	// StepCount counts walk steps for the footsteps, 0..5.
	StepCount int
	// Skills are the weapon classes' and schools' levels; Experience, Level and NextLevel the
	// player's experience, level and the experience the next level needs.
	Skills                       [Skills]Skill
	Experience, Level, NextLevel int
	// ArmorClass is the armor the player wears; Delays each hand's updates until it acts again.
	ArmorClass int
	Delays     [2]int
	// Poisoned and Dead are the player's conditions; Status its lasting statuses, Timed its timed
	// effects in status ticks, and Boosts each attribute's timed boost.
	Poisoned, Dead bool
	Status         Status
	Timed          [TimedCount]int
	Boosts         [6]Boost
	// Exertion, Fatigue and Confusion are counts (docs/spec/player.md).
	Exertion, Fatigue, Confusion int
	// Shown is the damage shown over the player, until the step ShownUntil.
	Shown      int
	ShownUntil int64
	// Pose is a swing pose ("attack", "attack.left", "attack.both"), held poseLeft more updates.
	Pose     string
	poseLeft int
	// bowRelease is the step after a bow shot was fired, 0 for none: the bow sounds at the
	// player's first update from then, unless the player walks first.
	bowRelease int64
	// Magic is the player's spells (docs/spec/spells.md).
	Magic Magic
	// Look is the sprite a Shape Change gives the player, empty for its own.
	Look string
	// Return is where a portal took the player from, for the way back.
	Return *content.Destination
	// WeaponCount is the damage counted weapons have dealt (combat.md, Special weapons).
	WeaponCount int
	// Name is the character's name, which texts give for @.
	Name string

	// want is the direction a walk was asked for since the last update, or −1.
	want int
	// last and next are the steps of the player's last update and its next.
	last, next int64
	// delays and gesture are the hands' delays and the gesture playing as they began, for
	// HandWait and GestureWait.
	delays  [2]wait
	gesture wait
	// vision is the cell visibility is computed from.
	vision content.Cell
	// statusTicks counts status ticks, for what comes every fourth one.
	statusTicks int
}

// Cell returns the cell the player stands in.
func (p *Player) Cell() content.Cell { return cellOf(p.U, p.V) }

// Walking reports whether the player shows its walk rather than its idle frame.
func (p *Player) Walking() bool { return p.Moves > 0 && p.Idle < idleUpdates }

// weight is what the player adds to a plate: nothing while levitating.
func (p *Player) weight() int {
	if p.Status.Levitating {
		return 0
	}
	return p.baseWeight()
}

// baseWeight is the player's weight on its feet.
func (p *Player) baseWeight() int {
	if p.Template.Weight > 0 {
		return p.Template.Weight
	}
	return playerWeight
}

// Pack returns the container whose contents are the pack slots, or nil: the one opened while it
// is still in its slot, else the one in the left hand (docs/spec/items.md, Opening containers).
func (p *Player) Pack(pk *content.Pack) *Object {
	if s, ok := p.OpenSlot(pk); ok {
		return p.Slots[s]
	}
	if o := p.Slots[SlotLeftHand]; isContainer(pk, o) {
		return o
	}
	return nil
}

// OpenSlot returns the fixed slot of the container opened as the pack, and false when none is
// open or it has left its slot.
func (p *Player) OpenSlot(pk *content.Pack) (int, bool) {
	s := p.Opened - 1
	if s < 0 || s >= SlotCount || !isContainer(pk, p.Slots[s]) {
		return 0, false
	}
	return s, true
}

func isContainer(pk *content.Pack, o *Object) bool {
	return o != nil && pk.Items[o.Item] != nil && pk.Items[o.Item].Container != nil
}

// open opens the container in fixed slot i as the pack, or closes it when it is open
// (docs/spec/items.md, Opening containers).
func (w *World) open(i int) {
	p, l := w.player, w.Current()
	if s, ok := p.OpenSlot(w.pack); ok && s == i {
		p.Opened = 0
		w.emit(Event{Kind: EventThing, Level: l.ID, Cell: p.Cell(), Thing: PlayerID, Detail: "closed"})
		return
	}
	if i < 0 || i >= SlotCount || !isContainer(w.pack, p.Slots[i]) {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: p.Cell(), Detail: "nothing to open there"})
		return
	}
	p.Opened = i + 1
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: p.Cell(), Thing: PlayerID, Detail: "opened"})
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
	p := &Player{Template: tmpl, Portrait: tmpl.Portrait, Facing: tmpl.Facing, want: -1, Attributes: map[string]int{}, StepCount: 1}
	p.Magic.teach(tmpl.Icons)
	for i := range p.Skills {
		p.Skills[i].Threshold = skillThreshold
	}
	for k, v := range tmpl.Stats {
		switch {
		case k == StatIntervalBonus:
			p.IntervalBonus = v
		case strings.HasPrefix(k, "weapon.") || strings.HasPrefix(k, "school."):
			i, err := strconv.Atoi(k[len("weapon."):])
			if err == nil && i >= 0 && i < 4 {
				if k[0] == 's' {
					i += SkillSchool
				}
				p.Skills[i].Level = v
			}
		default:
			p.Attributes[k] = v
		}
	}
	p.MaxAttributes = map[string]int{}
	for _, a := range attributeNames {
		if v, ok := p.Attributes[a]; ok {
			p.MaxAttributes[a] = v
		}
	}
	p.NextLevel = w.rules.LevelThreshold(0)
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
// arrives (docs/spec/rules.md, The player and the features). Cue is the transition that shows it,
// "" for none. It reports whether the player arrived.
func (w *World) travel(to *content.Destination, cue string) bool {
	p, l := w.player, w.Current()
	if _, err := w.load(to.Level); err != nil {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Detail: err.Error()})
		return false
	}
	w.weigh(l, p.Cell(), -p.weight(), 0)
	if err := w.enter(to.Level); err != nil {
		return false
	}
	w.place(to.Cell, false)
	w.transition(cue)
	return true
}

// transition reports that the player moved by transition cue, "" for none.
func (w *World) transition(cue string) {
	if cue != "" && w.player != nil {
		w.emit(Event{Kind: EventTransition, Level: w.current, Cell: w.player.Cell(), Thing: PlayerID, Detail: cue})
	}
}

// hurtPlayer hurts the player by n of kind k through its defenses, saying what hurt it; worn body
// armor wears, and at 0 hit points the player dies (docs/spec/combat.md, Hurting the player).
func (w *World) hurtPlayer(n int, k combat.Kind, by string) {
	w.foughtBy(by)
	p := w.player
	if p == nil || n <= 0 || p.Dead {
		return
	}
	armor := p.Slots[SlotBody]
	def := combat.PlayerDefense{Status: p.combatStatus(), ArmorClass: p.ArmorClass, BodyArmor: armor != nil}
	absorbed := false
	if p.Status.Charges {
		def.Absorb = func(d int) int {
			rest := w.absorb(d)
			absorbed = rest < d
			return rest
		}
	}
	d, wear := w.rules.HurtPlayer(n, k, def)
	if wear > 0 && armor.Value < unwearing {
		armor.Value = max(0, armor.Value-wear)
		w.wornChanged()
	}
	p.HitPoints = max(0, p.HitPoints-d)
	if !absorbed {
		p.Shown, p.ShownUntil = min(d, maxShown), w.step+6*w.PlayerInterval()
	}
	w.emit(Event{Kind: EventThing, Level: w.current, Cell: p.Cell(), Thing: PlayerID, Detail: "hurt by " + by, Damage: d})
	if p.HitPoints == 0 && p.Status.DeathProtection {
		p.Status.DeathProtection = false
		p.HitPoints, p.MagicPoints = p.MaxHitPoints, p.MaxMagicPoints
		w.emit(Event{Kind: EventThing, Level: w.current, Cell: p.Cell(), Thing: PlayerID, Detail: "revived"})
	}
	if p.HitPoints == 0 {
		w.playerDies(by)
	}
}

// updatePlayer is the player's update: it walks if a walk was asked for (docs/spec/entities.md,
// The player's update).
func (w *World) updatePlayer() {
	p := w.player
	p.last, p.next = w.step, w.step+w.PlayerInterval()
	if p.Dead {
		return
	}
	for i := range p.Delays {
		p.Delays[i] = max(0, p.Delays[i]-1)
	}
	if p.bowRelease != 0 && w.step >= p.bowRelease {
		p.bowRelease = 0
		if p.want < 0 {
			w.cue(p.Cell(), content.CueBow)
		}
	}
	// A swing pose holds through poseUpdates updates, which do not walk; the next stands.
	posing := p.Pose != ""
	if posing {
		if p.poseLeft > 0 {
			p.poseLeft--
		} else {
			p.Pose, p.StepCount = "", 1
			w.emit(Event{Kind: EventThing, Level: w.current, Cell: p.Cell(), Thing: PlayerID, Detail: "stands"})
		}
	}
	w.everyMove()
	if p.Magic.Gesture != nil {
		w.playGesture()
		if p.Dead || w.talk != nil {
			return
		}
	}
	d := p.want
	p.want = -1
	if p.Confusion > 0 {
		if d >= 0 && p.Confusion > confusedOver && w.rng.IntN(100) < p.Confusion {
			d = (d + 7 + 2*w.rng.IntN(2)) % 8
		}
		p.Confusion--
	}
	if posing {
		return
	}
	if d < 0 || p.Dead {
		if p.Idle < idleUpdates {
			if p.Idle++; p.Idle == idleUpdates {
				p.StepCount = 1
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

// wornLookSlots are the slots whose objects' looks the figure shows: the worn slots but the hands.
var wornLookSlots = []int{SlotHead, SlotArms, SlotFeet, SlotQuiver, SlotBody, SlotNeck}

// FigureLooks returns the sprites of the player's figure layers, bottom up: each layer's look from
// what is worn, else its own, for the player's sex (docs/spec/player.md, Worn looks). It is nil
// when the template draws no layers.
func (p *Player) FigureLooks(pk *content.Pack) []string {
	layers := p.Template.Figure
	if len(layers) == 0 {
		return nil
	}
	female := false
	if pt := pk.Portraits[p.Portrait]; pt != nil {
		female = pt.Female
	}
	out := make([]string, len(layers))
	for i, l := range layers {
		look := l.FigureLook
		for _, s := range wornLookSlots {
			if o := p.Slots[s]; o != nil {
				if it := pk.Items[o.Item]; it != nil {
					if lk, ok := it.Looks[l.Name]; ok {
						look = lk
					}
				}
			}
		}
		out[i] = look.For(female)
	}
	return out
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
	if p.Timed[TimedHaste] > 0 {
		n--
	}
	if p.Timed[TimedBattleRage] > 0 {
		n--
	}
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
		case pushable(t) && !(p.Status.Levitating && t.Template.Kind == content.ThingCreature):
			from := [2]int{t.U, t.V}
			if w.push(l, t, du, dv, 0) {
				su, sv = t.U-from[0], t.V-from[1]
				p.next += Steps(pushTicks)
				w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "pushed"})
			}
		case t.Template.Kind == content.ThingPortal && t.Portal != nil:
			w.enterPortal(l, t)
			return true
		case alive(t) && t.Template.Has(content.BehaviorHurtsOnContact):
			w.hurtPlayer(1, combat.Physical, t.ID)
			if t.Template.Has(content.BehaviorPoisonous) {
				w.player.Poisoned = true
			}
		case t.Template.Kind == content.ThingBoulder:
			if t.Rolling != nil && t.Rolling.Direction != d {
				w.hurtPlayer(w.roll(content.Dice{Count: 2, Sides: 3}), combat.Physical, "boulder")
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
		if hit == nil {
			p.Confusion += confusionBump
		}
		return false
	}
	w.movePlayerBy(l, su, sv)
	w.stride()
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
	shape := set.Shape(l.Shape(c))
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
	if f != nil && d >= 0 && !w.player.Status.Levitating && (f.Kind == content.FeaturePit || f.Kind == content.FeatureTrapdoor && f.Open) {
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
	w.goThrough(f.Effect, f, content.TransitionExitWall)
}

// fallPlayer drops the player through pit or trapdoor f, hurting it 4d5 unless it steps down
// carefully, along directions 1–3. Stepping down plays the climb, falling the fall.
func (w *World) fallPlayer(f *Feature, d int) {
	cue := content.EffectClimbDown
	if d < 1 || d > 3 {
		cue = content.EffectFall
		w.hurtPlayer(w.roll(content.Dice{Count: 4, Sides: 5}), combat.Physical, "falling")
	}
	w.goThrough(w.pack.Effect(cue), f, content.TransitionFall)
}

// goThrough plays view effect id at the player, then takes the player through exit, pit or
// trapdoor f of the current level once it ends, shown by transition cue, and records the link,
// and for an exit the way back beside the arrival.
func (w *World) goThrough(id string, f *Feature, cue string) {
	l := w.Current()
	w.playerEffect(id)
	w.afterEffect(func() {
		if w.travel(f.To, cue) {
			w.recordLink(l, f)
			if f.Kind == content.FeatureExit {
				w.recordReturn(l, f)
			}
		}
	})
}

// movePlayerBy moves the player by (du, dv) and runs what its new cell does.
func (w *World) movePlayerBy(l *Level, du, dv int) {
	p := w.player
	from := p.Cell()
	p.U, p.V = p.U+du, p.V+dv
	w.endShapeChange()
	p.Moves++
	p.Idle = 0
	to := p.Cell()
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: to, Thing: PlayerID, Detail: "moved"})
	if ft := l.FloorType(to); ft != nil && ft.Depth == 4 && !p.Status.Levitating && w.chance(1, 4) {
		w.hurtPlayer(1, combat.Physical, "deep water")
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
	if ft := l.FloorType(c); ft != nil && ft.Enter != nil && !w.player.Status.Levitating {
		w.floorEffect(ft.Enter)
	}
	f := l.byCell[c]
	switch {
	case f == nil:
	case f.Kind == content.FeatureExit && f.To != nil:
		w.goThrough(f.Effect, f, content.TransitionExitFloor)
	case f.Kind == content.FeatureTeleporter && f.Armed:
		w.teleportPlayer(l, f, depth)
	case (f.Kind == content.FeaturePit || f.Kind == content.FeatureTrapdoor && f.Open) && !w.player.Status.Levitating:
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
	w.recordLink(l, f)
	w.transition(content.TransitionTeleporter)
	w.weigh(l, from, -p.weight(), depth)
	w.weigh(l, to, p.weight(), depth)
	w.playerEntered(l, to, depth+1)
}

// floorEffect applies a floor's effect to the player.
func (w *World) floorEffect(e *content.Effect) {
	p := w.player
	if e.Damage != (content.Dice{}) {
		by, k := "the floor", combat.Physical
		if e.Type != "" {
			by = e.Type
		}
		if e.Type == "fire" {
			if p.Status.FireImmune || p.Status.Levitating {
				return
			}
			k = combat.Fire
		}
		if e.Cue != "" {
			w.cue(p.Cell(), e.Cue)
		}
		w.hurtPlayer(w.roll(e.Damage), k, by)
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
		if ft.Depth == 4 && !p.Status.Levitating {
			w.hurtPlayer(1, combat.Physical, "deep water")
		}
	}
	if p.statusTicks++; p.statusTicks%4 == 0 && p.MagicPoints < p.MaxMagicPoints {
		p.MagicPoints++
	}
	w.statusTickPlayer()
}

// pickUp takes the top object of the player's cell: onto the pointer when toHand and the hand is
// empty, else an arrow into a worn quiver with room and anything else to the first empty shelf,
// with its message (docs/spec/items.md, Engine model).
func (w *World) pickUp(toHand bool) {
	p, l := w.player, w.Current()
	c := p.Cell()
	objs := l.Objects[c]
	if len(objs) == 0 {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: c, Detail: "nothing to pick up"})
		w.message(l, c, content.MsgNothingToTake, "")
		return
	}
	o := objs[len(objs)-1]
	toHand = toHand && p.Hand == nil
	if !toHand && w.intoQuiver(o) {
		w.takeFloorObject(l, c)
		w.cue(c, content.CueTake)
		w.message(l, c, content.MsgIntoQuiver, o.Item)
		return
	}
	slot := -1
	if !toHand {
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
	w.cue(c, content.CueTake)
	w.message(l, c, content.MsgTaken, o.Item)
	if slot < 0 {
		p.Hand = o
		w.read(input.SlotHand)
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
		w.cue(p.Cell(), content.CueDrop)
	}
}

// swap exchanges the hand with slot i (docs/spec/items.md, Engine model).
func (w *World) swap(i int) {
	p, l := w.player, w.Current()
	refuse := func(why string) {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: p.Cell(), Detail: why})
	}
	exchanged := p.Hand != nil || p.Slot(w.pack, i) != nil
	if why := w.exchange(i); why != "" {
		refuse(why)
		return
	}
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: p.Cell(), Thing: PlayerID, Detail: "swapped"})
	if exchanged {
		w.cue(p.Cell(), content.CueTake)
	}
	if p.Hand != nil {
		w.read(input.SlotHand)
	}
}

// exchange exchanges the hand with slot i, putting on and taking off what goes in and out of a
// worn slot, and returns why not when it cannot.
func (w *World) exchange(i int) string {
	p := w.player
	switch {
	case i >= 0 && i < Shelves:
		p.Hand, p.Slots[i] = p.Slots[i], p.Hand
	case i == SlotQuiver && p.Hand != nil && w.intoQuiver(p.Hand):
		p.Hand = nil
	case i >= Shelves && i < SlotCount:
		if wear := wearOf(i); wear != "" && p.Hand != nil {
			if it := w.pack.Items[p.Hand.Item]; it == nil || it.Wear != wear {
				return "that is not worn there"
			}
		}
		off := p.Slots[i]
		p.Hand, p.Slots[i] = off, p.Hand
		if off != nil {
			w.takeOff(off)
		}
		if on := p.Slots[i]; on != nil {
			w.putOn(on)
		}
	case i >= SlotPack:
		bag := p.Pack(w.pack)
		if bag == nil {
			return "no container open"
		}
		box := w.pack.Items[bag.Item].Container
		k := i - SlotPack
		if k >= box.Capacity {
			return "the bag has no such place"
		}
		if p.Hand != nil && w.pack.Items[p.Hand.Item].Size > box.Largest {
			return "too big for the bag"
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
		return "no such slot"
	}
	return ""
}

// activate uses the lever in the player's cell, or puts the key a keyhole there takes into it,
// or else uses the nearest mouth in reach (docs/spec/rules.md, The player and the features).
func (w *World) activate() {
	p, l := w.player, w.Current()
	c := p.Cell()
	if f := l.byCell[c]; f != nil && f.Kind == content.FeatureLever {
		w.use(l, c)
		return
	}
	if f := l.byCell[c]; f != nil && f.Kind == content.FeatureSlot && f.Keyhole {
		w.useKey(l, f)
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

// startInventory gives the player the objects its template starts with; worn ones start as if
// put on (docs/spec/player.md, Starting inventory).
func (w *World) startInventory() {
	p := w.player
	for _, io := range p.Template.Inventory {
		o := w.object(io.Object)
		if o == nil || io.Slot < 0 || io.Slot >= SlotCount {
			continue
		}
		p.Slots[io.Slot] = o
		if io.Slot >= Shelves {
			w.putOn(o)
		}
	}
}

// useKey puts the key keyhole f takes into it from what the player carries: the hand, the
// shelves, the hands, then the bag in the left hand. A key the keyhole refuses stays where it was.
func (w *World) useKey(l *Level, f *Feature) {
	p := w.player
	take, ok := w.findCarried(f.Accepts)
	if !ok {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: f.Cell, Detail: "no " + f.Accepts + " for the keyhole"})
		return
	}
	o, putBack := take()
	if !w.drop(l, f.Cell, o, byPlayer, 0) {
		putBack()
		return
	}
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: p.Cell(), Thing: PlayerID, Detail: "used a key"})
}

// findCarried finds an object of item among what the player carries, in the order useKey gives,
// and returns a function that takes it out and returns it with a function that puts it back.
func (w *World) findCarried(item string) (func() (*Object, func()), bool) {
	p := w.player
	if p.Hand != nil && p.Hand.Item == item {
		return func() (*Object, func()) {
			o := p.Hand
			p.Hand = nil
			return o, func() { p.Hand = o }
		}, true
	}
	order := make([]int, 0, Shelves+2)
	for i := range Shelves {
		order = append(order, i)
	}
	order = append(order, SlotRightHand, SlotLeftHand)
	for _, i := range order {
		if o := p.Slots[i]; o != nil && o.Item == item {
			return func() (*Object, func()) {
				p.Slots[i] = nil
				if i >= Shelves {
					w.takeOff(o)
				}
				return o, func() {
					p.Slots[i] = o
					if i >= Shelves {
						w.putOn(o)
					}
				}
			}, true
		}
	}
	if bag := p.Pack(w.pack); bag != nil {
		for k, o := range bag.Contents {
			if o.Item == item {
				return func() (*Object, func()) {
					bag.Contents = append(bag.Contents[:k:k], bag.Contents[k+1:]...)
					return o, func() { bag.Contents = append(bag.Contents[:k:k], append([]*Object{o}, bag.Contents[k:]...)...) }
				}, true
			}
		}
	}
	return nil, false
}

// moveSlot moves the object in slot from to a place of kind to without the pointer: the pack's
// next place, the first empty shelf or its worn slot, whatever that slot held taking its old
// place (docs/spec/items.md, Quick moves).
func (w *World) moveSlot(from int, to string) {
	p, l := w.player, w.Current()
	refuse := func(why string) {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: p.Cell(), Detail: why})
	}
	o := p.Slot(w.pack, from)
	switch {
	case p.Hand != nil:
		refuse("something in hand")
		return
	case o == nil:
		refuse("nothing there")
		return
	}
	var dest int
	switch to {
	case input.ToPack:
		bag := p.Pack(w.pack)
		switch {
		case bag == nil:
			refuse("no container open")
			return
		case bag == o:
			refuse("that is the pack")
			return
		case from >= SlotPack:
			refuse("already in the pack")
			return
		}
		dest = SlotPack + len(bag.Contents)
	case input.ToShelf:
		if dest = firstEmptyShelf(p); dest < 0 {
			w.message(l, p.Cell(), content.MsgNoRoom, o.Item)
			return
		}
	case input.ToWorn:
		var ok bool
		if dest, ok = w.wornSlot(o); !ok {
			refuse("that is not worn")
			return
		}
		if dest == from {
			refuse("already worn")
			return
		}
	default:
		refuse("no such place")
		return
	}
	if why := w.exchange(from); why != "" {
		refuse(why)
		return
	}
	if why := w.exchange(dest); why != "" {
		w.exchange(from)
		refuse(why)
		return
	}
	if p.Hand != nil {
		w.exchange(from)
	}
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: p.Cell(), Thing: PlayerID, Detail: "moved"})
	w.cue(p.Cell(), content.CueTake)
}

// wornSlot returns the slot object o is worn in: its wear's, the left hand for a shield (a hand
// item with armor and no weapon class), else the right hand; false when it is not worn.
func (w *World) wornSlot(o *Object) (int, bool) {
	it := w.pack.Items[o.Item]
	switch {
	case it == nil || it.Wear == "":
		return 0, false
	case it.Wear == content.WearHand && it.Armor > 0 && it.WeaponClass == nil:
		return SlotLeftHand, true
	case it.Wear == content.WearHand:
		return SlotRightHand, true
	}
	s, ok := wearSlots[it.Wear]
	return s, ok
}

// firstEmptyShelf returns the player's first empty shelf, or −1.
func firstEmptyShelf(p *Player) int {
	for i := range Shelves {
		if p.Slots[i] == nil {
			return i
		}
	}
	return -1
}
