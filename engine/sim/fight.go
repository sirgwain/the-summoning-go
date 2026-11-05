package sim

import (
	"strconv"

	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Fighting constants (docs/spec/creatures.md, docs/spec/combat.md).
const (
	// wakeDistance is how far from the player, in half the Manhattan distance, creatures move.
	wakeDistance = 320
	// restDir is the walk direction that rests.
	restDir = 4
	// maxShown is the most damage shown over the hurt.
	maxShown = 99
	// poseUpdates is how many of the player's updates a swing pose holds through; the next
	// turns the player back to standing.
	poseUpdates = 2
	// bareHandsDelay is an empty hand's delay, in player updates.
	bareHandsDelay = 6
	// poisonChance is the 1 in n chance each update that poison hurts.
	poisonChance = 40
	// unwearing is the durability that never wears.
	unwearing = 30000
	// bodyArmorValue is body armor's full value; its armor falls by 1 for each bodyArmorStep below.
	bodyArmorValue = 7500
	bodyArmorStep  = 500
)

// Skill is a weapon class's or school's level, its practice and the practice the next level needs.
type Skill struct {
	Level     int `json:"level"`
	Practice  int `json:"practice"`
	Threshold int `json:"threshold"`
}

// Skill indexes: weapon classes 0–3, then schools 0–3.
const (
	SkillWeapon = 0
	SkillSchool = 4
	Skills      = 8
)

// Thing poses.
const (
	PoseAttack = "attack"
)

// rulesFor returns the rule set pack p names; the Summoning's is the only one.
func rulesFor(*content.Pack) combat.Rules { return summoningRules }

// alive reports whether t is a living creature.
func alive(t *Thing) bool { return t.Template.Kind == content.ThingCreature && t.HitPoints > 0 }

// hurtThing hurts thing t by d of kind k: only living creatures are hurt, through their
// defenses, and one at 0 dies (docs/spec/combat.md, Hurting a creature). It returns the hit
// points taken.
func (w *World) hurtThing(l *Level, t *Thing, d int, k combat.Kind) int {
	if d > 0 && w.mirrored(t, d, k) {
		return 0
	}
	if t.Template.Kind == content.ThingConjured && d > 0 {
		w.wearConjured(l, t)
		return 0
	}
	if !alive(t) {
		return 0
	}
	if l == w.Current() {
		w.learnCreature(t, StageFought)
		w.meetKind(l, t, k)
	}
	d = w.rules.HurtCreature(d, k, w.creatureDefense(t))
	if d <= 0 {
		return 0
	}
	t.HitPoints = max(0, t.HitPoints-d)
	t.Shown, t.ShownUntil = min(d, maxShown), w.step+3*stepsMS(t.Template.IntervalMS)
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "hurt", Damage: d})
	if w.isKind(k, content.KindPoison) && !w.resistsPoison(t) {
		t.Poisoned = true
	}
	if t.HitPoints == 0 {
		w.creatureDies(l, t)
	}
	return d
}

// creatureDies drops what creature t carries, gives the player its experience, and leaves its
// corpse or removes it (docs/spec/creatures.md, Death).
func (w *World) creatureDies(l *Level, t *Thing) {
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "died"})
	if t.Template.Has(content.BehaviorStationary) {
		w.playEffect(l, w.pack.Effect(content.EffectCasterDies), t.Cell(), t.ID)
	}
	w.record(t.Template.ID).Dead = true
	t.Hostile, t.Poisoned, t.Pose = false, false, ""
	if w.player != nil {
		w.gainExperience(w.roll(t.Template.Experience))
	}
	if l == w.Current() {
		var drops []*Object
		if t.Template.Has(content.BehaviorLeavesCorpse) {
			drops = t.Items
		}
		w.killed(t, drops)
	}
	if !t.Template.Has(content.BehaviorLeavesCorpse) {
		w.removeThing(l, t, "removed")
		return
	}
	items := t.Items
	t.Items = nil
	for _, o := range items {
		w.drop(l, t.Cell(), o, byWorld, 0)
	}
}

// turnHostile makes creature t hostile, unless it never is.
func (w *World) turnHostile(t *Thing) {
	if t.Template.Has(content.BehaviorNeverHostile) || t.Template.Has(content.BehaviorInvulnerable) {
		return
	}
	t.Hostile = true
}

// updateCreatures runs the updates due of level l's walking creatures (docs/spec/creatures.md,
// Walkers).
func (w *World) updateCreatures(l *Level) {
	for _, t := range append([]*Thing(nil), l.Things...) {
		if w.spin(l, t) {
			continue
		}
		if t.Template.Kind == content.ThingConjured && w.step >= t.next && l.Thing(t.ID) == t {
			t.next = w.step + stepsMS(t.Template.IntervalMS)
			w.wearConjured(l, t)
			continue
		}
		if t.WindUpUntil > 0 {
			w.windingUp(l, t)
			continue
		}
		if !alive(t) || w.step < t.next || l.Thing(t.ID) != t {
			continue
		}
		t.next = w.step + stepsMS(t.Template.IntervalMS)
		t.glide.start(t.U, t.V, w.step, t.next)
		t.Moving = false
		if t.Pose != "" {
			t.Pose = ""
		}
		if t.Poisoned && w.chance(1, poisonChance) {
			w.hurtThing(l, t, 1+w.rng.IntN(2), combat.Physical)
			if !alive(t) {
				continue
			}
		}
		if w.creatureShoots(l, t) {
			continue
		}
		switch {
		case t.Template.Animate != nil:
			w.animate(l, t)
		case t.Template.Has(content.BehaviorStationary):
			w.stationaryCasts(l, t)
		case !t.Template.Has(content.BehaviorAnimated):
			w.walkCreature(l, t)
		}
	}
}

// walkCreature counts down creature t's walk, picks a new one when it runs out, and steps.
func (w *World) walkCreature(l *Level, t *Thing) {
	p := w.player
	if p == nil || w.current != l.ID || (abs(t.U-p.U)+abs(t.V-p.V))/2 > wakeDistance {
		return
	}
	if t.walkCount--; t.walkCount <= 0 {
		if t.Hostile && !p.Dead && !p.hidden() {
			t.walkDir, t.walkCount = towards(t.U, t.V, p.U, p.V), 1
		} else {
			t.walkDir, t.walkCount = w.rng.IntN(5), 5+w.rng.IntN(5)
			if t.walkDir == restDir {
				t.walkCount += 10
			}
		}
	}
	if t.walkDir == restDir {
		return
	}
	d := 2 * t.walkDir
	t.Facing = d
	speed := stepOf(t)
	du, dv := Dirs[d][0]*speed, Dirs[d][1]*speed
	for ; du != 0 && !w.creatureAccepts(l, t, t.U+du, t.V); du -= sign(du) {
	}
	for ; dv != 0 && !w.creatureAccepts(l, t, t.U, t.V+dv); dv -= sign(dv) {
	}
	var hit *body
	for du != 0 || dv != 0 {
		b := w.blocker(l, t, false, t.U+du, t.V+dv, t.Template.Bounds(), nil)
		if b == nil {
			break
		}
		hit = b
		du, dv = toward0(du, backOff), toward0(dv, backOff)
	}
	if hit == nil && (du != 0 || dv != 0) {
		w.moveThingBy(l, t, du, dv)
		t.Moving = true
		w.walkerCasts(l, t)
		return
	}
	if hit == nil {
		hit = w.blocker(l, t, false, t.U+Dirs[d][0]*speed, t.V+Dirs[d][1]*speed, t.Template.Bounds(), nil)
	}
	switch {
	case hit != nil && hit.thing == nil && t.Hostile && !p.Dead && t.Template.WindUpMS > 0:
		t.Facing = 2 * towards(t.U, t.V, p.U, p.V)
		t.WindUpFrom, t.WindUpUntil = w.step, w.step+stepsMS(t.Template.WindUpMS)
		w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "winds up"})
	case hit != nil && hit.thing == nil && t.Hostile && !p.Dead:
		t.Facing = 2 * towards(t.U, t.V, p.U, p.V)
		w.creatureStrikes(l, t)
		t.walkDir, t.walkCount = restDir, 10+w.rng.IntN(5)
	case hit != nil && hit.thing == nil && t.Template.Has(content.BehaviorHurtsOnContact):
		w.hurtPlayer(1, combat.Physical, t.ID)
		if t.Template.Has(content.BehaviorPoisonous) {
			w.poisonPlayer()
		}
	case t.Hostile:
		t.walkDir, t.walkCount = w.rng.IntN(4), 5+w.rng.IntN(5)
	default:
		t.walkDir, t.walkCount = (t.walkDir+2)%4, 5+w.rng.IntN(5)
	}
}

// windingUp runs creature t's wind-up: when its time has passed the blow lands on the player if
// the player is still a step ahead, else meets nothing (docs/spec/combat.md, Wind-up).
func (w *World) windingUp(l *Level, t *Thing) {
	p := w.player
	if !alive(t) || l.Thing(t.ID) != t {
		t.WindUpFrom, t.WindUpUntil = 0, 0
		return
	}
	if w.step < t.WindUpUntil {
		return
	}
	t.WindUpFrom, t.WindUpUntil = 0, 0
	t.next = w.step + stepsMS(t.Template.IntervalMS)
	t.glide.start(t.U, t.V, w.step, t.next)
	t.Moving = false
	if p == nil || p.Dead || w.current != l.ID {
		return
	}
	t.walkDir, t.walkCount = restDir, 10+w.rng.IntN(5)
	if w.playerAhead(l, t) {
		w.creatureStrikes(l, t)
		return
	}
	t.Pose = PoseAttack
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "strikes nothing"})
}

// stepOf is how far creature t steps: its own speed, else its template's, else 8.
func stepOf(t *Thing) int {
	switch {
	case t.Speed > 0:
		return t.Speed
	case t.Template.Speed > 0:
		return t.Template.Speed
	}
	return 8
}

// playerAhead reports whether the player's box overlaps creature t's moved one full step along
// its facing.
func (w *World) playerAhead(l *Level, t *Thing) bool {
	speed := stepOf(t)
	u, v := t.U+Dirs[t.Facing][0]*speed, t.V+Dirs[t.Facing][1]*speed
	for _, b := range w.bodies(l) {
		if b.thing == nil && overlaps(u, v, t.Template.Bounds(), b.u, b.v, b.box) {
			return true
		}
	}
	return false
}

// creatureAccepts reports whether creature t's ground point may stand at (u, v).
func (w *World) creatureAccepts(l *Level, t *Thing, u, v int) bool {
	c := cellOf(u, v)
	if l.Blocked(c) {
		return false
	}
	if ft := l.FloorType(c); ft != nil && ft.Avoided {
		return false
	}
	f := l.byCell[c]
	if f != nil && (f.Kind == content.FeaturePit || f.Kind == content.FeatureTrapdoor && f.Open) {
		return t.Template.Has(content.BehaviorResistsTeleport)
	}
	return true
}

// towards returns the four-way direction (0..3) from (u, v) that best closes the larger of the
// two axis gaps to (tu, tv).
func towards(u, v, tu, tv int) int {
	du, dv := tu-u, tv-v
	if abs(du) >= abs(dv) {
		if du < 0 {
			return 0
		}
		return 2
	}
	if dv > 0 {
		return 1
	}
	return 3
}

// behind reports whether a striker facing d strikes a target facing facing from behind.
func behind(d, facing int) bool { return d&^1 == facing&^1 }

// creatureStrikes is creature t striking the player (docs/spec/combat.md, Creatures attack).
func (w *World) creatureStrikes(l *Level, t *Thing) {
	p := w.player
	t.Pose = PoseAttack
	w.learnCreature(t, StageFought)
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "strikes"})
	if t.Template.Has(content.BehaviorDeathBlow) && !p.Status.DeathBlowWard {
		w.playerEffect(w.pack.Effect(content.EffectDeathBlow))
		w.killPlayer(t.ID)
		return
	}
	r := w.rules.CreatureStrikes(w.rng, combat.CreatureStrike{
		Accuracy: t.Template.Stats["accuracy"], Bonus: t.Template.Stats["bonus"], Damage: t.Template.Attack,
		Behind: behind(t.Facing, p.Facing), PlayerAgility: p.Attributes[AttrAgility], Status: w.combatStatus(),
	})
	if !r.Hit {
		w.messageNumbers(l, t.Cell(), content.MsgCreatureMiss, r.Score, 0)
		return
	}
	w.messageNumbers(l, t.Cell(), content.MsgCreatureHits, r.Score, r.Damage)
	if r.Damage > 0 {
		w.hurtPlayerAloud(r.Damage, combat.Physical, t.ID)
		if t.Template.Has(content.BehaviorPoisonous) && w.chance(2, 8) {
			w.poisonPlayer()
		}
	}
}

// messageNumbers shows engine message key with a strike's numbers.
func (w *World) messageNumbers(l *Level, c content.Cell, key string, score, damage int) {
	if w.quiet(key) {
		return
	}
	text := ""
	if m := w.pack.Messages; m != nil {
		text = m.Messages[key]
	}
	w.emit(Event{Kind: EventMessage, Level: l.ID, Cell: c, Text: text, Detail: key, Score: score, Damage: damage})
}

// strike is the player acting with hand (0 right, 1 left) by its object's action
// (docs/spec/items.md, Hands and actions).
func (w *World) strike(hand, action int) {
	p, l := w.player, w.Current()
	refuse := func(why string) {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: p.Cell(), Detail: why})
	}
	switch {
	case p.Dead:
		refuse("the player is dead")
		return
	case hand < 0 || hand > 1:
		refuse("no such hand")
		return
	case p.Delays[hand] > 0:
		refuse("the hand is not ready")
		return
	case w.HandBlocked(hand):
		refuse("the hand is blocked")
		return
	}
	o := p.Slots[SlotRightHand+hand]
	if o == nil {
		if action != 0 {
			refuse("bare hands have one action")
			return
		}
		if w.refuseRunning() {
			return
		}
		w.setDelay(hand, bareHandsDelay)
		w.exert(1)
		w.swing(hand, nil, content.ItemAction{Kind: content.ActionAttack, Damage: content.Dice{Count: 2, Sides: 2}})
		return
	}
	w.handAct(hand, o, action)
}

// useSlot uses the object in slot by its first use or use-up action, as the right hand would, or
// in a hand by that hand (docs/spec/items.md, Using from a slot).
func (w *World) useSlot(slot int) {
	p, l := w.player, w.Current()
	refuse := func(why string) {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: p.Cell(), Detail: why})
	}
	o := p.Slot(w.pack, slot)
	if o == nil || p.Hand == o {
		refuse("nothing to use there")
		return
	}
	action := -1
	for i, a := range w.pack.Items[o.Item].Actions {
		if a.Kind == content.ActionUse || a.Kind == content.ActionUseUp {
			action = i
			break
		}
	}
	switch {
	case action < 0:
		refuse("the " + o.Item + " cannot be used")
	case slot == SlotRightHand || slot == SlotLeftHand:
		w.strike(slot-SlotRightHand, action)
	case p.Dead:
		refuse("the player is dead")
	case p.Delays[0] > 0:
		refuse("the hand is not ready")
	default:
		w.handAct(0, o, action)
	}
}

// HandBlocked reports whether hand (0 right, 1 left) is blocked: it holds a two-handed object
// while the other hand holds anything, and then none of its actions can be taken
// (docs/spec/items.md, Hands and actions).
func (w *World) HandBlocked(hand int) bool {
	p := w.player
	if p == nil || hand < 0 || hand > 1 {
		return false
	}
	o := p.Slots[SlotRightHand+hand]
	if o == nil || p.Slots[SlotRightHand+1-hand] == nil {
		return false
	}
	it := w.pack.Items[o.Item]
	return it != nil && hasTag(it, "two_handed")
}

// handAct carries out action of object o as hand's (docs/spec/items.md, Hands and actions).
func (w *World) handAct(hand int, o *Object, action int) {
	p, l := w.player, w.Current()
	refuse := func(why string) {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: p.Cell(), Detail: why})
	}
	it := w.pack.Items[o.Item]
	if action < 0 || action >= len(it.Actions) {
		refuse("the " + o.Item + " has no such action")
		return
	}
	a := it.Actions[action]
	switch a.Kind {
	case content.ActionAttack, content.ActionAttackBoth, content.ActionFire:
		if w.refuseRunning() {
			return
		}
	default:
		if it.Casts != nil && w.refuseRunning() {
			return
		}
	}
	class := -1
	if it.WeaponClass != nil {
		class = *it.WeaponClass
	}
	if a.Kind != content.ActionFire && w.deepWater() {
		w.message(l, p.Cell(), content.MsgCannotAct, "")
		return
	}
	switch a.Kind {
	case content.ActionAttack, content.ActionAttackBoth:
		if class >= 0 && p.Skills[SkillWeapon+class].Level < a.Skill {
			refuse("not skilled enough")
			return
		}
	case content.ActionFire:
		if w.deepWater() {
			w.message(l, p.Cell(), content.MsgCannotAct, "")
			return
		}
		w.delay(hand, it, a)
		w.exert(o.Weight(w.pack) / 8)
		w.fire(o, a)
		return
	default:
		if a.Kind == content.ActionUse && w.Quantity(o) <= 0 {
			refuse("the " + o.Item + " has no charges left")
			return
		}
		w.delay(hand, it, a)
		w.useObject(o, a)
		if a.Swings {
			if hasTag(it, "weapon") {
				hand = 2
			}
			w.pose(hand)
		}
		return
	}
	w.delay(hand, it, a)
	w.exert(o.Weight(w.pack) / 8)
	if a.Kind == content.ActionAttackBoth {
		hand = 2
	}
	w.swing(hand, o, a)
}

// delay makes hand wait action a's delay, and the other hand too for a two-handed item
// (docs/spec/items.md, Hands and actions).
func (w *World) delay(hand int, it *content.Item, a content.ItemAction) {
	w.setDelay(hand, a.Delay)
	if hasTag(it, "two_handed") {
		w.setDelay(1-hand, a.Delay)
	}
}

// pose shows the player's swing pose of hand (2 for both) (docs/spec/combat.md, The player
// attacks).
func (w *World) pose(hand int) {
	w.player.Pose, w.player.poseLeft = []string{"attack", "attack.left", "attack.both"}[hand], poseUpdates
}

func hasTag(it *content.Item, tag string) bool {
	for _, t := range it.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// swing strikes with object o (nil for bare hands) by action a, swinging hand (2 for both): at
// the first creature one step ahead (docs/spec/combat.md, The player attacks).
// strikeAt returns what a blow by a body with box at (u, v) meets: the first living creature
// there, past any corpse or object in front of it, else the first thing there, else nil
// (docs/spec/combat.md, The player attacks).
func (w *World) strikeAt(l *Level, u, v int, box content.Box) *body {
	var first *body
	for _, b := range w.bodies(l) {
		if b.thing == nil || !overlaps(u, v, box, b.u, b.v, b.box) {
			continue
		}
		if alive(b.thing) {
			return &b
		}
		if first == nil {
			first = &b
		}
	}
	return first
}

func (w *World) swing(hand int, o *Object, a content.ItemAction) {
	p, l := w.player, w.Current()
	w.pose(hand)
	speed := p.Template.Speed
	if speed == 0 {
		speed = 8
	}
	d := p.Facing
	b := w.strikeAt(l, p.U+Dirs[d][0]*speed, p.V+Dirs[d][1]*speed, p.Template.Bounds())
	switch {
	case b == nil || b.thing == nil || !alive(b.thing):
		w.message(l, p.Cell(), content.MsgNotACreature, "")
		w.cue(p.Cell(), content.CueMiss)
		return
	case b.thing.Template.Has(content.BehaviorAnimated) && !b.thing.Template.Has(content.BehaviorMirrors):
		w.message(l, p.Cell(), content.MsgCannotHurt, "")
		w.cue(p.Cell(), content.CueMiss)
		return
	}
	t := b.thing
	var it *content.Item
	class := -1
	if o != nil {
		it = w.pack.Items[o.Item]
		if it.WeaponClass != nil {
			class = *it.WeaponClass
		}
	}
	s := combat.PlayerStrike{
		Agility: p.Attributes[AttrAgility], Endurance: p.Attributes[AttrEndurance],
		Accuracy: p.Attributes[AttrAccuracy], Strength: p.Attributes[AttrStrength],
		Weapon: o != nil, Damage: a.Damage, Behind: behind(d, t.Facing), Status: w.combatStatus(),
		CreatureAccuracy: t.Template.Stats["accuracy"], Resistance: t.Template.Stats["resistance"],
		Vulnerable:   class >= 0 && t.Template.Vulnerability != nil && *t.Template.Vulnerability == class,
		PhysicalOnly: t.Template.Has(content.BehaviorPhysicalOnly),
	}
	if class >= 0 {
		s.Skill = p.Skills[SkillWeapon+class].Level
	}
	if it != nil && hasTag(it, "silver") && len(it.Actions) > 1 {
		s.Hurts = &it.Actions[1].Damage
	}
	r := w.rules.PlayerStrikes(w.rng, s)
	if !r.Hit {
		w.messageNumbers(l, p.Cell(), content.MsgMiss, r.Score, 0)
		w.cue(p.Cell(), content.CueMiss)
		return
	}
	if s.Vulnerable {
		w.meetDefense(l, t, DefenseVulnerable)
	}
	w.meetDefense(l, t, content.BehaviorPhysicalOnly)
	if it != nil && it.Poisons {
		w.meetDefense(l, t, content.BehaviorResistsPoison)
	}
	w.messageNumbers(l, p.Cell(), content.MsgHit, r.Score, r.Damage)
	w.turnHostile(t)
	if r.Damage == 0 {
		w.cue(p.Cell(), content.CueMiss)
		return
	}
	w.cue(p.Cell(), content.CueHit)
	if it != nil {
		w.countWeapon(o, it, r.Damage)
	}
	if o != nil {
		w.wearWeapon(hand, o, r.Damage)
	}
	w.hurtThing(l, t, r.Damage, combat.Physical)
	if it != nil && it.Poisons && alive(t) && !w.resistsPoison(t) {
		t.Poisoned = true
	}
	practice := 1
	if t.Template.Kind == content.ThingCreature && t.HitPoints <= 0 {
		practice = w.killedBy(l, t)
	}
	if class >= 0 {
		w.practice(SkillWeapon+class, practice)
	}
}

// wearWeapon wears the weapon in hand by n; one worn past what is left breaks
// (docs/spec/combat.md, Wear).
func (w *World) wearWeapon(hand int, o *Object, n int) {
	if o.Value >= unwearing {
		return
	}
	n = w.rules.Wear(n)
	if n <= o.Value {
		o.Value -= n
		return
	}
	p, l := w.player, w.Current()
	for i := range 2 {
		if p.Slots[SlotRightHand+i] == o {
			p.Slots[SlotRightHand+i] = nil
			w.emit(Event{Kind: EventObject, Level: l.ID, Cell: p.Cell(), Item: o.Item, Detail: "worn_out", Slot: SlotRightHand + i})
		}
	}
	w.message(l, p.Cell(), content.MsgWeaponBreaks, o.Item)
	w.wornChanged()
}

// practice adds n practice to skill i, raising its level at the threshold (docs/spec/player.md,
// Skills).
func (w *World) practice(i, n int) {
	p := w.player
	s := &p.Skills[i]
	s.Practice += n
	if s.Practice < s.Threshold {
		return
	}
	s.Level++
	s.Practice -= s.Threshold
	s.Threshold = w.rules.SkillThreshold(s.Threshold)
	cue := content.CueWeaponSkillUp
	if i >= SkillSchool {
		cue = content.CueMagicSkillUp
	}
	w.emit(Event{Kind: EventSound, Level: w.current, Cell: p.Cell(), Detail: cue})
	w.emit(Event{Kind: EventMessage, Level: w.current, Cell: p.Cell(), Text: w.messageText(content.MsgSkillUp),
		Detail: content.MsgSkillUp, Item: "skill." + strconv.Itoa(i)})
	w.gainExperience(w.rules.SkillExperience())
}

func (w *World) messageText(key string) string {
	if m := w.pack.Messages; m != nil {
		return m.Messages[key]
	}
	return ""
}

// gainExperience adds experience, raising the player's level as far as it reaches
// (docs/spec/player.md, Experience and levels).
func (w *World) gainExperience(n int) {
	p := w.player
	if p == nil || n <= 0 {
		return
	}
	p.Experience += n
	for p.Level < w.rules.MaxLevel() && p.Experience >= p.NextLevel {
		p.Level++
		hit, magic := w.rules.LevelGain(p.Level)
		h, m := w.roll(hit), w.roll(magic)
		p.MaxHitPoints += h
		p.HitPoints += h
		p.MaxMagicPoints += m
		p.MagicPoints += m
		p.NextLevel = w.rules.LevelThreshold(p.Level)
		w.emit(Event{Kind: EventSound, Level: w.current, Cell: p.Cell(), Detail: content.CueLevelUp})
		w.message(w.Current(), p.Cell(), content.MsgLevelUp, "")
	}
}

// killPlayer kills the player outright.
func (w *World) killPlayer(by string) {
	p := w.player
	p.HitPoints = 0
	w.playerDies(by)
}

// playerDies marks the player dead (docs/spec/player.md, Dying).
func (w *World) playerDies(by string) {
	p := w.player
	if p.Dead {
		return
	}
	p.Dead, p.want = true, -1
	w.emit(Event{Kind: EventThing, Level: w.current, Cell: p.Cell(), Thing: PlayerID, Detail: "died, killed by " + by})
}

// armorOf returns the armor worn object o adds: body armor's falls by 1 for each 500 its value is
// below full, but not below half (docs/spec/items.md, Armor).
func (w *World) armorOf(o *Object) int {
	it := w.pack.Items[o.Item]
	if it == nil || it.Armor == 0 {
		return 0
	}
	if it.Wear != content.WearBody {
		return it.Armor
	}
	return max(it.Armor-(bodyArmorValue-min(o.Value, bodyArmorValue))/bodyArmorStep, it.Armor/2)
}

// wornChanged recomputes the player's armor class from what it wears and holds.
func (w *World) wornChanged() {
	p := w.player
	p.ArmorClass = 0
	for i := Shelves; i < SlotCount; i++ {
		if o := p.Slots[i]; o != nil {
			p.ArmorClass += w.armorOf(o)
		}
	}
}

// Speed powers (docs/spec/items.md, Powers).
const (
	powerSpeed1        = 0x09
	powerSpeed2        = 0x0A
	powerSpeed2Lasting = 0x0B
)

// putOn and takeOff add and take away what worn object o does: its armor, and its power when
// it starts worn (docs/spec/items.md, Inventory).
func (w *World) putOn(o *Object) {
	if it := w.pack.Items[o.Item]; it != nil && it.PowerWorn && it.Power != 0 {
		w.learnItem(o.Item, StageUsed)
		w.startPower(it.Power, 1)
	}
	w.wornChanged()
}

func (w *World) takeOff(o *Object) {
	if it := w.pack.Items[o.Item]; it != nil && it.PowerWorn && it.Power != 0 {
		w.startPower(it.Power, -1)
	}
	w.wornChanged()
}

// wearSlots are the worn slots by the wear an item names; hands are slots of their own.
var wearSlots = map[string]int{
	content.WearHead: SlotHead, content.WearArms: SlotArms, content.WearFeet: SlotFeet,
	content.WearQuiver: SlotQuiver, content.WearBody: SlotBody, content.WearNeck: SlotNeck,
}

// wearOf returns the wear slot name of slot i, or "".
func wearOf(i int) string {
	for name, s := range wearSlots {
		if s == i {
			return name
		}
	}
	return ""
}

// putOnHand puts the hand's object on: in its worn slot, or the right hand.
func (w *World) putOnHand() {
	p := w.player
	if p.Hand == nil {
		w.emit(Event{Kind: EventRefused, Level: w.current, Cell: p.Cell(), Detail: "nothing in hand"})
		return
	}
	slot := SlotRightHand
	if it := w.pack.Items[p.Hand.Item]; it != nil {
		if s, ok := wearSlots[it.Wear]; ok {
			slot = s
		}
	}
	w.swap(slot)
}
