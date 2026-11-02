package sim

import (
	"slices"
	"strconv"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The player's magic (docs/spec/spells.md): the icons it knows, its spellbook, its memorizations,
// each school's selected spell, and the gesture playing.

// Magic constants (docs/spec/spells.md).
const (
	maxMemorized = 9
	// noPower asks a cast to roll the spell's own power.
	noPower = -1
)

// Magic is the player's magic.
type Magic struct {
	// Icons are the gesture icons it knows, as letters, in order.
	Icons string `json:"icons,omitempty"`
	// Spellbook is the spells it has learned, in the order learned.
	Spellbook []string `json:"spellbook,omitempty"`
	// Memorized counts each spell's memorizations; Selected is each school's selected spell.
	Memorized map[string]int          `json:"memorized,omitempty"`
	Selected  [content.Schools]string `json:"selected"`
	// Gesture is the gesture playing, nil when none is.
	Gesture *Gesture `json:"gesture,omitempty"`
}

// Gesture is a cast's gesture playing on the gesture panel before the spell takes effect.
type Gesture struct {
	Spell string `json:"spell"`
	// Shown is how many of the gesture's icons have shown; the spell takes effect after the last.
	Shown int `json:"shown"`
}

// Icon returns the icon showing, as a letter, and false between casts.
func (m *Magic) Icon(p *content.Pack) (byte, bool) {
	g := m.Gesture
	if g == nil || g.Shown == 0 {
		return 0, false
	}
	sp := p.Spells[g.Spell]
	if sp == nil || g.Shown > len(sp.Gesture) {
		return 0, false
	}
	return sp.Gesture[g.Shown-1], true
}

// Knows reports whether the player knows icon c.
func (m *Magic) Knows(c byte) bool { return slices.Contains([]byte(m.Icons), c) }

// teach marks the icons of gesture known; letters that are not icons teach nothing.
func (m *Magic) teach(gesture string) {
	for i := 0; i < len(gesture); i++ {
		if c := gesture[i]; c >= 'A' && c < 'A'+content.GestureIcons && !m.Knows(c) {
			b := []byte(m.Icons + string(c))
			slices.Sort(b)
			m.Icons = string(b)
		}
	}
}

// Learned reports whether spell is in the spellbook.
func (m *Magic) Learned(spell string) bool { return slices.Contains(m.Spellbook, spell) }

// memorizeLimit is how many memorizations school can hold.
func (w *World) memorizeLimit(school int) int {
	p := w.player
	n := floorDiv(p.Attributes[AttrTalent]-10, 2) + p.Skills[SkillSchool+school].Level/2
	return max(1, min(n, maxMemorized))
}

// schoolTotal is how many memorizations school holds.
func (w *World) schoolTotal(school int) int {
	n := 0
	for id, k := range w.player.Magic.Memorized {
		if sp := w.pack.Spells[id]; sp != nil && sp.School == school {
			n += k
		}
	}
	return n
}

// memorize memorizes spell once more and selects it, forgetting one of its school's at random
// first when the school is full.
func (w *World) memorize(sp *content.Spell) {
	p, l := w.player, w.Current()
	m := &p.Magic
	if m.Memorized == nil {
		m.Memorized = map[string]int{}
	}
	if total := w.schoolTotal(sp.School); total > 0 && total >= w.memorizeLimit(sp.School) {
		pick := w.rng.IntN(total)
		for _, id := range sortedKeys(m.Memorized) {
			if s := w.pack.Spells[id]; s == nil || s.School != sp.School {
				continue
			}
			if pick -= m.Memorized[id]; pick < 0 {
				if m.Memorized[id]--; m.Memorized[id] == 0 {
					delete(m.Memorized, id)
				}
				w.message(l, p.Cell(), content.MsgForgot, "")
				w.message(l, p.Cell(), content.MsgForgotMore, "")
				break
			}
		}
	}
	m.Memorized[sp.ID]++
	m.Selected[sp.School] = sp.ID
	w.emit(Event{Kind: EventMagic, Level: l.ID, Cell: p.Cell(), Item: sp.ID, Detail: "memorized"})
}

// learn enters a gesture: one matching a spell puts it in the spellbook and memorizes it; one
// matching nothing costs a magic point per icon.
func (w *World) learn(gesture string) {
	p, l := w.player, w.Current()
	m := &p.Magic
	switch {
	case gesture == "" || len(gesture) > content.MaxGesture:
		w.refuse("a gesture is 1 to " + strconv.Itoa(content.MaxGesture) + " icons")
		return
	}
	w.cue(p.Cell(), content.CueGesture)
	for i := 0; i < len(gesture); i++ {
		if !m.Knows(gesture[i]) {
			w.refuse("the icon " + gesture[i:i+1] + " is not known")
			return
		}
		if i > 0 && gesture[i] == gesture[i-1] {
			w.refuse("an icon cannot follow itself")
			return
		}
	}
	for _, id := range sortedKeys(w.pack.Spells) {
		sp := w.pack.Spells[id]
		if sp.Gesture != gesture {
			continue
		}
		if !m.Learned(id) {
			m.Spellbook = append(m.Spellbook, id)
			w.emit(Event{Kind: EventMagic, Level: l.ID, Cell: p.Cell(), Item: id, Detail: "learned"})
		}
		w.memorize(sp)
		return
	}
	p.MagicPoints = max(0, p.MagicPoints-len(gesture))
	w.emit(Event{Kind: EventMagic, Level: l.ID, Cell: p.Cell(), Detail: "no spell has that gesture"})
}

// memorizeLearned memorizes a spell of the spellbook.
func (w *World) memorizeLearned(id string) {
	sp := w.pack.Spells[id]
	if sp == nil || !w.player.Magic.Learned(id) {
		w.refuse(id + " is not in the spellbook")
		return
	}
	w.memorize(sp)
}

// schoolName returns the text key of school's name.
func (w *World) schoolName(school int) string {
	if m := w.pack.Magic; m != nil && school < len(m.Schools) {
		return m.Schools[school]
	}
	return ""
}

// nextSpell selects school's next memorized spell, in spellbook order.
func (w *World) nextSpell(school int) {
	p, l := w.player, w.Current()
	m := &p.Magic
	if school < 0 || school >= content.Schools {
		w.refuse("no such school")
		return
	}
	var spells []string
	for _, id := range m.Spellbook {
		if sp := w.pack.Spells[id]; sp != nil && sp.School == school && m.Memorized[id] > 0 {
			spells = append(spells, id)
		}
	}
	at := slices.Index(spells, m.Selected[school])
	if len(spells) == 0 || len(spells) == 1 && at == 0 {
		key := content.MsgNotMemorized
		if len(spells) == 1 {
			key = content.MsgOnlySpell
		}
		w.emit(Event{Kind: EventMessage, Level: l.ID, Cell: p.Cell(), Text: w.messageText(key), Detail: key, Item: w.schoolName(school)})
		return
	}
	m.Selected[school] = spells[(at+1)%len(spells)]
	w.emit(Event{Kind: EventMagic, Level: l.ID, Cell: p.Cell(), Item: m.Selected[school], Detail: "selected"})
}

// cast starts casting school's selected spell: its checks, its cost and a memorization, then its
// gesture plays (docs/spec/spells.md, Casting).
func (w *World) cast(school int) {
	p, l := w.player, w.Current()
	m := &p.Magic
	if school < 0 || school >= content.Schools {
		w.refuse("no such school")
		return
	}
	if p.Dead {
		w.refuse("the player is dead")
		return
	}
	if w.deepWater() {
		w.message(l, p.Cell(), content.MsgCannotCast, "")
		return
	}
	id := m.Selected[school]
	sp := w.pack.Spells[id]
	if sp == nil || m.Memorized[id] == 0 {
		w.emit(Event{Kind: EventMessage, Level: l.ID, Cell: p.Cell(), Text: w.messageText(content.MsgNotMemorized),
			Detail: content.MsgNotMemorized, Item: w.schoolName(school)})
		return
	}
	if m.Gesture != nil {
		w.refuse("a gesture is playing")
		return
	}
	if !w.spellNeeds(sp) {
		return
	}
	if sp.Cost > p.MagicPoints {
		w.message(l, p.Cell(), content.MsgNoMagic, "")
		return
	}
	w.cue(p.Cell(), content.CueCast)
	p.MagicPoints -= sp.Cost
	if m.Memorized[id]--; m.Memorized[id] == 0 {
		delete(m.Memorized, id)
	}
	m.Gesture = &Gesture{Spell: id}
	p.gesture = w.begin(len(sp.Gesture) + 1)
	w.emit(Event{Kind: EventMagic, Level: l.ID, Cell: p.Cell(), Item: id, Detail: "casting"})
}

// playGesture shows the next icon of the gesture playing, or after the last casts its spell; with
// none of the spell memorized any more, the school's next is selected (docs/spec/spells.md).
func (w *World) playGesture() {
	p := w.player
	m := &p.Magic
	g := m.Gesture
	sp := w.pack.Spells[g.Spell]
	if sp != nil && g.Shown < len(sp.Gesture) {
		g.Shown++
		return
	}
	m.Gesture = nil
	if sp == nil {
		return
	}
	if m.Memorized[sp.ID] == 0 {
		for _, id := range m.Spellbook {
			if s := w.pack.Spells[id]; s != nil && s.School == sp.School && m.Memorized[id] > 0 {
				m.Selected[sp.School] = id
				break
			}
		}
	}
	w.castSpell(w.playerCaster(), sp, noPower)
}

// caster is what casts a spell: the player, a creature, or a cell's shooter.
type caster struct {
	player bool
	thing  *Thing
	// u, v are where it casts from; dir the table direction it faces.
	u, v, dir int
}

func (w *World) playerCaster() caster {
	p := w.player
	return caster{player: true, u: p.U, v: p.V, dir: p.Facing}
}

// spellPower rolls spell sp's power for the player: its dice and a bonus die for each point of
// power attribute and school skill.
func (w *World) spellPower(sp *content.Spell, c caster) int {
	n := w.roll(sp.Power)
	if !c.player || sp.PowerBonus == 0 {
		return n
	}
	p := w.player
	extra := max(0, floorDiv(p.Attributes[AttrPower]-10, 2)+p.Skills[SkillSchool+sp.School].Level)
	return n + w.roll(content.Dice{Count: extra, Sides: sp.PowerBonus})
}

// castSpell makes spell sp take effect, cast by c with power (noPower to roll it). The player's
// casts practice its school.
func (w *World) castSpell(c caster, sp *content.Spell, power int) {
	if power == noPower {
		power = w.spellPower(sp, c)
	}
	l := w.Current()
	cell := cellOf(c.u, c.v)
	w.emit(Event{Kind: EventMagic, Level: l.ID, Cell: cell, Item: sp.ID, Detail: "cast", Damage: power})
	if c.player {
		w.practice(SkillSchool+sp.School, 1+sp.Practice)
	}
	for _, e := range sp.Effects {
		w.afterEffect(func() { w.spellEffect(l, c, sp, e, power) })
	}
}

// ahead returns the point dist units ahead of c along its direction.
func (c caster) ahead(dist int) (int, int) {
	return c.u + Dirs[c.dir][0]*dist, c.v + Dirs[c.dir][1]*dist
}

// spellNeeds reports whether spell sp's condition holds for the player, showing why not when it
// does not.
func (w *World) spellNeeds(sp *content.Spell) bool {
	n := sp.Needs
	if n == nil {
		return true
	}
	p, l := w.player, w.Current()
	c := w.playerCaster()
	ok, key := true, ""
	switch {
	case n.FloorAhead > 0:
		u, v := c.ahead(n.FloorAhead)
		ok, key = !l.Blocked(cellOf(u, v)), content.MsgNoRoomForWall
	case n.Mark != "":
		ok, key = w.mark(n.Mark) != nil, content.MsgNoDestination
	case n.SpellNear > 0:
		ok, key = len(w.spellsNear(l, n.SpellNear)) > 0, content.MsgNoSpellNear
	case len(n.Holding) > 0:
		ok, key = w.holding(n.Holding), content.MsgNothingToMend
	case n.CreatureNear > 0:
		ok, key = w.nearestCreature(n.CreatureNear) != nil, content.MsgNoCreatureNear
	case n.Poisoned:
		ok, key = p.Poisoned, content.MsgNotPoisoned
	case n.Fatigued:
		ok, key = p.Fatigue > 0, content.MsgNotFatigued
	case n.Hurt:
		ok, key = p.HitPoints < p.MaxHitPoints, content.MsgFullHealth
	}
	if !ok {
		if n.Else != "" {
			key = n.Else
		}
		w.message(l, p.Cell(), key, "")
	}
	return ok
}

// holding reports whether each of items is in one of the player's hands, a hand each.
func (w *World) holding(items []string) bool {
	p := w.player
	hands := []*Object{p.Slots[SlotRightHand], p.Slots[SlotLeftHand]}
	for _, it := range items {
		found := false
		for i, o := range hands {
			if o != nil && o.Item == it {
				hands[i], found = nil, true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// handHolding returns the hand (0 right, 1 left) holding item, or −1.
func (w *World) handHolding(item string) int {
	for h := range 2 {
		if o := w.player.Slots[SlotRightHand+h]; o != nil && o.Item == item {
			return h
		}
	}
	return -1
}

// nearestCreature returns the nearest living creature within dist of the player (half the
// Manhattan distance), or nil.
func (w *World) nearestCreature(dist int) *Thing {
	p, l := w.player, w.Current()
	var best *Thing
	bestD := 0
	for _, t := range l.Things {
		if !alive(t) {
			continue
		}
		if d := (abs(t.U-p.U) + abs(t.V-p.V)) / 2; d <= dist && (best == nil || d < bestD) {
			best, bestD = t, d
		}
	}
	return best
}

// spellsNear returns the spells in flight over l within dist of the player.
func (w *World) spellsNear(l *Level, dist int) []*Flight {
	p := w.player
	var out []*Flight
	for _, f := range l.Flights {
		if f.Spell != nil && f.Spell.ending == 0 && (abs(f.U-p.U)+abs(f.V-p.V))/2 < dist {
			out = append(out, f)
		}
	}
	return out
}

// spellEffect runs one of spell sp's effects, cast by c with power. Effects other than missiles
// and fired objects act on the player, and only a player's cast runs them.
func (w *World) spellEffect(l *Level, c caster, sp *content.Spell, e content.Action, power int) {
	switch {
	case e.Missile != nil:
		w.castMissile(l, c, e.Missile, power)
		return
	case e.FireItem != nil:
		if w.pack.Items[*e.FireItem] == nil {
			return
		}
		o := w.newObject(*e.FireItem)
		w.launch(l, o, c.u, c.v, c.dir, firedRange, w.missileDamage(o, true), c.player, true, c.thing)
		return
	case !c.player || w.player == nil:
		return
	}
	p := w.player
	at := p.Cell()
	switch {
	case e.Timed != nil:
		w.timedSpell(*e.Timed, power)
	case e.Conjure != nil:
		w.conjure(l, c, sp, e.Conjure, power)
	case e.Portal != nil:
		w.openPortal(l, c, e.Portal)
	case e.TeleportToMark != nil:
		if f := w.mark(*e.TeleportToMark); f != nil {
			w.weigh(l, at, -p.weight(), 0)
			w.place(f.Cell, false)
			w.emit(Event{Kind: EventThing, Level: l.ID, Cell: p.Cell(), Thing: PlayerID, Detail: "rippled"})
		}
	case e.Dispel != nil:
		for _, f := range w.spellsNear(l, *e.Dispel) {
			w.endSpell(l, f)
		}
	case e.Join != nil:
		if w.holding(e.Join.Items) {
			for h := range 2 {
				if o := p.Slots[SlotRightHand+h]; o != nil {
					p.Slots[SlotRightHand+h] = nil
					w.takeOff(o)
				}
			}
			o := w.newObject(e.Join.Into)
			p.Slots[SlotRightHand] = o
			w.putOn(o)
			w.emit(Event{Kind: EventThing, Level: l.ID, Cell: at, Thing: PlayerID, Item: e.Join.Into, Detail: "joined"})
		}
	case e.Freeze != nil:
		w.message(l, at, content.MsgFrozen, "")
		for _, t := range l.Things {
			t.next += Steps(power * *e.Freeze)
		}
	case e.ShapeChange != nil:
		if t := w.nearestCreature(*e.ShapeChange); t != nil {
			w.shapeChange(t.Template.Sprite)
		}
	case e.Transmute != nil:
		w.transmute(l, e.Transmute, power)
	case e.CurePoison:
		p.Poisoned = false
		w.message(l, at, content.MsgCured, "")
	case e.Restore:
		n := min(power, p.Fatigue)
		p.Fatigue -= n
		p.Attributes[AttrEndurance] += n
	case e.Heal != nil:
		w.heal(w.roll(*e.Heal))
	case e.Effect != nil:
		w.playerEffect(*e.Effect)
	case e.Sparkle != nil:
		w.playerSparkle(*e.Sparkle)
	case e.SetCounter != nil:
		w.setCounter(e.SetCounter.Counter, e.SetCounter.Value)
	case e.Converse != nil:
		if t := w.pack.Things[*e.Converse]; t != nil {
			w.startTalk(t, nil)
		}
	}
}

// timedSpell starts or extends a timed effect for power status ticks (docs/spec/spells.md,
// Effects).
func (w *World) timedSpell(effect string, power int) {
	p, l := w.player, w.Current()
	at := p.Cell()
	say := func(key string) { w.message(l, at, key, "") }
	switch effect {
	case content.TimedBattleRage:
		if p.Timed[TimedBattleRage] == 0 {
			p.Attributes[AttrStrength] += 2
			p.Attributes[AttrEndurance] += 2
			p.Attributes[AttrAgility] -= 2
			say(content.MsgBattleRage)
		} else {
			say(content.MsgBattleRageExtended)
		}
		p.addTimed(TimedBattleRage, power)
	case content.TimedSight:
		p.addTimed(TimedSight, power)
	case content.TimedHaste:
		p.addTimed(TimedHaste, power)
		say(content.MsgHaste)
	case content.TimedInvisibility:
		p.addTimed(TimedInvisibility, power)
	case content.TimedFireShield:
		p.addTimed(TimedFireShield, power)
		say(content.MsgFireShield)
	case content.TimedLightningShield:
		p.addTimed(TimedLightningShield, power)
		say(content.MsgLightningShield)
	case content.TimedWeaponSkills, content.TimedMagicSkills:
		timed, first, up, more := TimedWeaponSkills, SkillWeapon, content.MsgWeaponSkillsUp, content.MsgWeaponSkillsExtended
		if effect == content.TimedMagicSkills {
			timed, first, up, more = TimedMagicSkills, SkillSchool, content.MsgMagicSkillsUp, content.MsgMagicSkillsExtended
		}
		if p.Timed[timed] == 0 {
			for i := range 4 {
				w.raiseSkill(first+i, 1, true)
			}
			say(up)
		} else {
			say(more)
		}
		p.addTimed(timed, power)
	}
}

// conjure puts a new conjured thing half a cell ahead of the caster, lasting power of its updates
// and looking by the school's skill; with no room for it, message wall_failed.
func (w *World) conjure(l *Level, c caster, sp *content.Spell, cj *content.Conjure, power int) {
	tmpl := w.pack.Things[cj.Thing]
	u, v := c.ahead(cj.Distance)
	if tmpl == nil || l.Blocked(cellOf(u, v)) || w.blocker(l, nil, false, u, v, tmpl.Size(), nil) != nil {
		w.message(l, cellOf(c.u, c.v), content.MsgWallFailed, "")
		return
	}
	t := w.newThing(w.newThingID(l, tmpl.Kind), tmpl)
	t.U, t.V, t.Lasts = u, v, power
	if cj.LookPerSkill > 0 {
		frames := 1
		if b := w.pack.Sprites[tmpl.Sprite]; b != nil {
			frames = b.Frames
		}
		f := min(w.player.Skills[SkillSchool+sp.School].Level/cj.LookPerSkill, frames-1)
		t.Frame = &f
	}
	t.next = w.step + stepsMS(tmpl.IntervalMS)
	l.Things = append(l.Things, t)
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "conjured"})
	w.weigh(l, t.Cell(), t.weight(), 0)
	w.entered(l, t, 0)
}

// openPortal opens a portal a distance ahead of the caster, closing the level's other portals.
func (w *World) openPortal(l *Level, c caster, pt *content.Portal) {
	tmpl := w.pack.Things[pt.Thing]
	u, v := c.ahead(pt.Distance)
	for _, t := range append([]*Thing(nil), l.Things...) {
		if t.Template.Kind == content.ThingPortal {
			w.removeThing(l, t, "closed")
		}
	}
	if tmpl == nil || l.Blocked(cellOf(u, v)) || w.blocker(l, nil, true, u, v, tmpl.Size(), nil) != nil {
		w.message(l, cellOf(c.u, c.v), content.MsgNoRoomForPortal, "")
		return
	}
	t := w.newThing(w.newThingID(l, tmpl.Kind), tmpl)
	t.U, t.V = u, v
	to := pt.To
	t.Portal = &to
	l.Things = append(l.Things, t)
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "portal opened"})
}

// enterPortal takes the player through portal t: it closes, the player's level and cell are kept
// for the way back, and the player goes where it leads.
func (w *World) enterPortal(l *Level, t *Thing) {
	p := w.player
	to := *t.Portal
	w.removeThing(l, t, "closed")
	p.Return = &content.Destination{Level: l.ID, Cell: p.Cell()}
	w.travel(&to)
}

// shapeChange gives the player the look of sprite until it next moves.
func (w *World) shapeChange(sprite string) {
	p := w.player
	p.Look, p.Status.Transformed = sprite, true
	if c := w.pack.Manifest.Counters; c != nil && c.Transformed != "" && c.TransformedLook == sprite {
		w.setCounter(c.Transformed, 1)
	}
	w.emit(Event{Kind: EventThing, Level: w.current, Cell: p.Cell(), Thing: PlayerID, Detail: "transformed"})
}

// endShapeChange gives the player its own look back.
func (w *World) endShapeChange() {
	p := w.player
	if !p.Status.Transformed {
		return
	}
	p.Look, p.Status.Transformed = "", false
	if c := w.pack.Manifest.Counters; c != nil && c.Transformed != "" {
		w.setCounter(c.Transformed, 0)
	}
	w.emit(Event{Kind: EventThing, Level: w.current, Cell: p.Cell(), Thing: PlayerID, Detail: "looks itself again"})
}

// transmute fills the holder in one hand from the other hand's object (docs/spec/spells.md,
// Liquify).
func (w *World) transmute(l *Level, tr *content.Transmute, power int) {
	p := w.player
	hand := w.handHolding(tr.Holder)
	if hand < 0 {
		w.message(l, p.Cell(), content.MsgNoBottle, "")
		return
	}
	other := p.Slots[SlotRightHand+1-hand]
	made, listed := tr.Otherwise, false
	if other != nil {
		if into, ok := tr.Table[other.Item]; ok {
			made, listed = into, true
		}
	}
	o := w.newObject(made)
	if !listed {
		o.Value = power
	}
	old := p.Slots[SlotRightHand+hand]
	p.Slots[SlotRightHand+hand] = o
	w.takeOff(old)
	w.putOn(o)
	if listed {
		p.Slots[SlotRightHand+1-hand] = nil
		w.takeOff(other)
	}
	w.message(l, p.Cell(), content.MsgFilled, made)
}

// read looks at the object in slot (input.SlotHand for the hand): a scroll teaches its icons.
func (w *World) read(slot int) {
	p, l := w.player, w.Current()
	o := p.Hand
	if slot >= 0 {
		o = p.Slot(w.pack, slot)
	}
	if o == nil {
		w.refuse("nothing to read")
		return
	}
	it := w.pack.Items[o.Item]
	if it == nil {
		return
	}
	if it.Teaches != "" {
		p.Magic.teach(it.Teaches)
	}
	w.emit(Event{Kind: EventMagic, Level: l.ID, Cell: p.Cell(), Item: o.Item, Detail: "read"})
}

// shootSpell casts spell id from cell c along direction d with power dice, as a level procedure
// does: from the cell's center, set back half a cell against the direction.
func (w *World) shootSpell(l *Level, cell content.Cell, id string, d int, power content.Dice) {
	sp := w.pack.Spells[id]
	if sp == nil || d < 0 || d >= len(Dirs) {
		return
	}
	u, v := cellCenter(cell)
	w.castSpellOn(l, caster{u: u, v: v, dir: d}, sp, w.roll(power))
}

// castSpellOn casts on level l, which need not be the current one: only missiles fly there.
func (w *World) castSpellOn(l *Level, c caster, sp *content.Spell, power int) {
	if l == w.Current() {
		w.castSpell(c, sp, power)
		return
	}
	for _, e := range sp.Effects {
		if e.Missile != nil {
			w.castMissile(l, c, e.Missile, power)
		}
	}
}

// Creatures casting (docs/spec/creatures.md, Walkers and Stationary casters).
const (
	// castChance is the 1 in n chance a walker's step casts its spell.
	castChance = 10
	// A stationary caster counts castCycle to castCycle+castCycleRange updates between casts,
	// turns at turnAt left and casts at castAt left.
	castCycle      = 15
	castCycleRange = 10
	turnAt         = 10
	castAt         = 1
)

// creatureCaster is creature t casting along its facing.
func creatureCaster(t *Thing) caster { return caster{thing: t, u: t.U, v: t.V, dir: t.Facing} }

// walkerCasts is a walker that has just stepped casting its spell, 1 step in 10: a hostile one
// faces the player first when chasing it, and casts only then; others face a random way. It
// then rests.
func (w *World) walkerCasts(l *Level, t *Thing) {
	sp := w.pack.Spells[t.Template.Spell]
	if sp == nil || !w.chance(1, castChance) {
		return
	}
	p := w.player
	if t.Hostile {
		if t.walkCount != 1 || p == nil {
			return
		}
		t.Facing = 2 * towards(t.U, t.V, p.U, p.V)
	} else {
		t.Facing = 2 * w.rng.IntN(4)
	}
	t.Pose = PoseAttack
	w.castSpellOn(l, creatureCaster(t), sp, w.roll(t.Template.SpellPower))
	t.walkDir, t.walkCount = restDir, 10+w.rng.IntN(5)
}

// stationaryCasts counts a stationary caster's updates: it turns, and later casts its spell.
func (w *World) stationaryCasts(l *Level, t *Thing) {
	sp := w.pack.Spells[t.Template.Spell]
	if sp == nil || w.player == nil || w.current != l.ID {
		return
	}
	if t.casting == 0 {
		t.casting = castCycle + w.rng.IntN(castCycleRange)
		return
	}
	switch t.casting--; t.casting {
	case turnAt:
		t.Facing = (t.Facing + 4) % 8
		w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "turned"})
	case castAt:
		w.castSpellOn(l, creatureCaster(t), sp, w.roll(t.Template.SpellPower))
	}
}

// wearConjured takes one from what conjured thing t lasts, removing it at none.
func (w *World) wearConjured(l *Level, t *Thing) {
	if t.Lasts--; t.Lasts > 0 {
		return
	}
	w.removeThing(l, t, "faded")
}
