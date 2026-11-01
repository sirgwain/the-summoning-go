package sim

import (
	"strconv"

	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Status is the player's lasting conditions (docs/spec/player.md, Status).
type Status struct {
	Regenerating     bool `json:"regenerating,omitempty"`
	ManaRegenerating bool `json:"mana_regenerating,omitempty"`
	ResistQuarter    bool `json:"resist_quarter,omitempty"`
	ResistForty      bool `json:"resist_forty,omitempty"`
	Charges          bool `json:"charges,omitempty"`
	FireImmune       bool `json:"fire_immune,omitempty"`
	DeathBlowWard    bool `json:"death_blow_ward,omitempty"`
	Accurate         bool `json:"accurate,omitempty"`
	BlowProtection   bool `json:"blow_protection,omitempty"`
	Levitating       bool `json:"levitating,omitempty"`
	Invisible        bool `json:"invisible,omitempty"`
	Transformed      bool `json:"transformed,omitempty"`
	DeathProtection  bool `json:"death_protection,omitempty"`
	Female           bool `json:"female,omitempty"`
}

// Timed effects, counted down once per status tick (docs/spec/player.md, Status).
const (
	TimedSight = iota
	TimedFireShield
	TimedLightningShield
	TimedBattleRage
	TimedHaste
	TimedInvisibility
	TimedWeaponSkills
	TimedMagicSkills
	TimedCount
)

// Boost is an attribute's timed boost.
type Boost struct {
	Amount int `json:"amount"`
	Ticks  int `json:"ticks"`
}

// attributeNames are the attributes in the game's order.
var attributeNames = [6]string{AttrStrength, AttrAgility, AttrEndurance, AttrAccuracy, AttrTalent, AttrPower}

// Status and effect constants (docs/spec/player.md, items.md).
const (
	// powerShowMap shows the level map.
	powerShowMap     = 0x22
	maxTimed         = 255
	confusedOver     = 25
	confusionBump    = 6
	regenerateChance = 30
	boostAmount      = 2
	boostTicks       = 10
	shieldTicks      = 10
	freezeTicks      = 100
	nearCreatures    = wakeDistance
	smiteDamage      = 2000
)

// combatStatus is what the player's statuses do to a fight.
func (p *Player) combatStatus() combat.Status {
	return combat.Status{
		BattleRage: p.Timed[TimedBattleRage] > 0, Accurate: p.Status.Accurate,
		Invisible:      p.Status.Invisible || p.Timed[TimedInvisibility] > 0,
		BlowProtection: p.Status.BlowProtection, ResistQuarter: p.Status.ResistQuarter, ResistForty: p.Status.ResistForty,
		FireShield: p.Timed[TimedFireShield] > 0, LightningShield: p.Timed[TimedLightningShield] > 0,
	}
}

// hidden reports whether creatures cannot see the player to chase it.
func (p *Player) hidden() bool {
	return p.Status.Invisible || p.Timed[TimedInvisibility] > 0 || p.Status.Transformed
}

// addTimed adds n status ticks to timed effect i, saturating at 255.
func (p *Player) addTimed(i, n int) { p.Timed[i] = min(p.Timed[i]+n, maxTimed) }

// exert adds exertion n (at least 1); past END + 3 it costs a point of endurance as fatigue
// (docs/spec/player.md, Exertion and fatigue).
func (w *World) exert(n int) {
	p := w.player
	p.Exertion += max(n, 1)
	if end := p.Attributes[AttrEndurance]; p.Exertion > end+3 && end > 1 {
		p.Attributes[AttrEndurance]--
		p.Fatigue++
		p.Exertion = 2
	}
}

// heal adds n hit points, at most the maximum, and says so when more than 1.
func (w *World) heal(n int) {
	p := w.player
	n = min(n, p.MaxHitPoints-p.HitPoints)
	p.HitPoints += n
	if n > 1 {
		w.messageNumbers(w.Current(), p.Cell(), content.MsgHealed, 0, n)
	}
}

// raiseSkill changes skill i's level by n, saying so unless quiet.
func (w *World) raiseSkill(i, n int, quiet bool) {
	p := w.player
	p.Skills[i].Level = max(0, p.Skills[i].Level+n)
	if !quiet && n > 0 {
		w.emit(Event{Kind: EventMessage, Level: w.current, Cell: p.Cell(), Text: w.messageText(content.MsgSkillUp),
			Detail: content.MsgSkillUp, Item: "skill." + strconv.Itoa(i)})
	}
}

// statusTickPlayer runs the status tick's parts that count down: timed effects, attribute boosts,
// exertion and fatigue (docs/spec/player.md, Status tick).
func (w *World) statusTickPlayer() {
	p, l := w.player, w.Current()
	for i := range p.Boosts {
		b := &p.Boosts[i]
		if b.Ticks > 0 {
			if b.Ticks--; b.Ticks == 0 {
				p.Attributes[attributeNames[i]] -= b.Amount
				b.Amount = 0
			}
		}
	}
	for i := range p.Timed {
		if p.Timed[i] == 0 {
			continue
		}
		if p.Timed[i]--; p.Timed[i] > 0 {
			continue
		}
		switch i {
		case TimedBattleRage:
			p.Attributes[AttrStrength] -= 2
			p.Attributes[AttrEndurance] -= 2
			p.Attributes[AttrAgility] += 2
		case TimedHaste:
			w.message(l, p.Cell(), content.MsgSlowed, "")
		case TimedWeaponSkills:
			for c := range 4 {
				w.raiseSkill(SkillWeapon+c, -1, true)
			}
			w.message(l, p.Cell(), content.MsgLostWeaponSkills, "")
		case TimedMagicSkills:
			for c := range 4 {
				w.raiseSkill(SkillSchool+c, -1, true)
			}
			w.message(l, p.Cell(), content.MsgLostMagicSkills, "")
		}
	}
	switch {
	case p.Exertion > 0:
		p.Exertion--
	case p.Fatigue > 0:
		p.Fatigue--
		p.Attributes[AttrEndurance]++
		p.Exertion = p.Attributes[AttrEndurance] + 3
	}
}

// everyMove runs the chances rolled on each of the player's updates: regenerating, mana
// regenerating and poison (docs/spec/player.md, Every move).
func (w *World) everyMove() {
	p := w.player
	if p.Status.Regenerating && w.chance(1, regenerateChance) {
		w.heal(w.roll(content.Dice{Count: 2, Sides: 2}))
	}
	if p.Status.ManaRegenerating && w.chance(1, regenerateChance) {
		p.MagicPoints = min(p.MagicPoints+w.roll(content.Dice{Count: 2, Sides: 2}), p.MaxMagicPoints)
	}
	if p.Poisoned && w.chance(1, poisonChance) {
		w.hurtPlayer(1+w.rng.IntN(2), combat.Physical, "poison")
	}
}

// usable reports whether power can do something now, showing why not when it cannot
// (docs/spec/items.md, Powers).
func (w *World) usable(power int) bool {
	p, l := w.player, w.Current()
	no := func(key string) bool {
		w.message(l, p.Cell(), key, "")
		return false
	}
	switch power {
	case 0x01, 0x02, 0x03:
		if p.MagicPoints >= p.MaxMagicPoints {
			return no(content.MsgFullMagic)
		}
	case 0x11, 0x26, 0x1A:
		if w.mark(runeOf(power)) == nil {
			return no(content.MsgNoDestination)
		}
	case 0x12:
		if len(w.creaturesNear()) == 0 {
			return no(content.MsgNoCreatures)
		}
	case 0x15:
		if !p.Poisoned && p.Confusion <= confusedOver {
			return no(content.MsgNotPoisonedOrConfused)
		}
	case 0x1D, 0x28, 0x45, 0x46:
		if p.HitPoints >= p.MaxHitPoints {
			return no(content.MsgFullHealth)
		}
	case 0x24:
		if !p.Poisoned {
			return no(content.MsgNotPoisoned)
		}
	case 0x44:
		if p.Fatigue == 0 {
			return no(content.MsgNotFatigued)
		}
	case 0x4D:
		if p.Status.DeathProtection {
			return no(content.MsgAlreadyHave)
		}
	case powerShowMap:
		// The map shows, and the object is never used up.
		w.showMap()
		return false
	case powerEnding:
		if w.endingChoice() == nil {
			return no(content.MsgNoEnding)
		}
		w.chooseEnding()
		return false
	}
	return true
}

// runeOf is the rune of the mark a teleporting power goes to.
func runeOf(power int) string {
	switch power {
	case 0x26:
		return "2"
	case 0x1A:
		return "3"
	}
	return "1"
}

// mark returns the current level's mark of rune r, or nil.
func (w *World) mark(r string) *Feature {
	for _, f := range w.Current().Features {
		if f.Kind == content.FeatureMark && f.Rune == r {
			return f
		}
	}
	return nil
}

// creaturesNear returns the living creatures near the player.
func (w *World) creaturesNear() []*Thing {
	p, l := w.player, w.Current()
	var out []*Thing
	for _, t := range l.Things {
		if alive(t) && (abs(t.U-p.U)+abs(t.V-p.V))/2 <= nearCreatures {
			out = append(out, t)
		}
	}
	return out
}

// startPower starts a power, worn (sign 1) or used; with sign −1 it ends a worn power, undoing
// what lasts while worn (docs/spec/items.md, Powers). It reports false for a power that waits
// for a later milestone.
func (w *World) startPower(power, sign int) bool {
	p, l := w.player, w.Current()
	ending := sign < 0
	attr := func(i, n int) { p.Attributes[attributeNames[i]] += n * sign }
	switch {
	case power >= 0x30 && power <= 0x41:
		n := power - 0x30
		attr(n/3, n%3+1)
		return true
	case power >= 0x2A && power <= 0x2F && !ending:
		b := &p.Boosts[power-0x2A]
		if b.Ticks > 0 {
			w.message(l, p.Cell(), content.MsgBoostExtended, "")
		} else {
			b.Amount = boostAmount
			p.Attributes[attributeNames[power-0x2A]] += boostAmount
		}
		b.Ticks += boostTicks
		return true
	case power >= 0x05 && power <= 0x08 && !ending:
		w.raiseSkill(SkillSchool+power-0x05, 1, false)
		return true
	case power >= 0x48 && power <= 0x4B && !ending:
		w.raiseSkill(SkillWeapon+power-0x48, 1, false)
		return true
	}
	switch power {
	case powerSpeed1:
		p.IntervalBonus -= sign
	case powerSpeed2, powerSpeed2Lasting:
		p.IntervalBonus -= 2 * sign
	case 0x0F:
		if p.Status.Levitating != !ending {
			full := p.baseWeight()
			p.Status.Levitating = !ending
			if !ending {
				w.weigh(l, p.Cell(), -full, 0)
			} else {
				w.weigh(l, p.Cell(), full, 0)
			}
		}
	case 0x17:
		p.Status.Regenerating = !ending
	case 0x4C:
		p.Status.ManaRegenerating = !ending
	case 0x47:
		p.Status.Invisible = !ending
	case 0x16:
	}
	if ending {
		return true
	}
	switch power {
	case 0x01:
		p.MagicPoints = min(p.MagicPoints+w.roll(content.Dice{Count: 2, Sides: 6}), p.MaxMagicPoints)
	case 0x02:
		p.MagicPoints = min(p.MagicPoints+w.roll(content.Dice{Count: 4, Sides: 6}), p.MaxMagicPoints)
	case 0x03:
		p.MagicPoints = p.MaxMagicPoints
	case 0x04:
		p.Confusion += 75
		w.hurtPlayer(w.roll(content.Dice{Count: 2, Sides: 8}), combat.Physical, "confusion")
	case 0x0C:
		p.Status.ResistQuarter = true
	case 0x0D:
		p.Status.ResistForty = true
	case 0x0E:
		p.Status.BlowProtection = true
	case 0x10:
		w.raiseSkill(SkillWeapon+3, 3, true)
	case 0x1F:
		w.raiseSkill(SkillWeapon+1, 2, true)
	case 0x11, 0x26, 0x1A:
		w.weigh(l, p.Cell(), -p.weight(), 0)
		w.place(w.mark(runeOf(power)).Cell, false)
		w.emit(Event{Kind: EventSound, Level: l.ID, Cell: p.Cell(), Detail: soundTeleport})
	case 0x12:
		for _, t := range w.creaturesNear() {
			w.hurtThing(l, t, smiteDamage, combat.Physical)
		}
	case 0x14:
		if w.chance(1, 2) {
			p.Status.Female = !p.Status.Female
		} else {
			n := 1
			if w.chance(1, 2) {
				n = -1
			}
			a := attributeNames[w.rng.IntN(6)]
			p.Attributes[a] += n
			p.MaxAttributes[a] += n
		}
	case 0x15:
		p.Poisoned, p.Confusion = false, 0
	case 0x1C:
		p.addTimed(TimedFireShield, shieldTicks)
	case 0x27:
		p.addTimed(TimedLightningShield, shieldTicks)
	case 0x1D:
		w.heal(w.roll(content.Dice{Count: 6, Sides: 6}))
	case 0x28:
		w.heal(w.roll(content.Dice{Count: 5, Sides: 10}))
	case 0x45:
		w.heal(3000)
	case 0x46:
		w.heal(w.roll(content.Dice{Count: 20, Sides: 4}))
	case 0x20:
		w.raiseSkill(SkillSchool+w.rng.IntN(4), 1, false)
	case 0x25:
		w.raiseSkill(SkillWeapon+w.rng.IntN(4), 1, false)
	case 0x21:
		for _, t := range l.Things {
			t.next += Steps(freezeTicks)
		}
	case 0x23:
		p.Status.Charges = true
	case 0x24:
		p.Poisoned = false
	case 0x43:
		p.Status.Accurate = true
	case 0x44:
		n := min(1+w.rng.IntN(3), p.Fatigue)
		p.Fatigue -= n
		p.Attributes[AttrEndurance] += n
	case 0x4D:
		p.Status.DeathProtection = true
		w.message(l, p.Cell(), content.MsgProtected, "")
	case 0x4F:
		p.Status.FireImmune = true
		p.IntervalBonus -= 2
	case 0x50:
		p.Confusion += 150
	case 0x5B:
		p.Status.DeathBlowWard = true
	case 0x0F, 0x17, 0x4C, 0x47, 0x16, powerSpeed1, powerSpeed2, powerSpeed2Lasting:
	default:
		w.emit(Event{Kind: EventNotYet, Level: l.ID, Cell: p.Cell(), Detail: "power " + strconv.Itoa(power)})
		return false
	}
	return true
}

// useObject uses the object in hand by a use or use-up action: its power starts unless it
// refuses, or it casts its spell or starts its conversation; then a used-up object, or one out of
// charges, becomes its spent object (docs/spec/items.md, Using objects).
func (w *World) useObject(hand int, o *Object, a content.ItemAction) {
	it := w.pack.Items[o.Item]
	switch {
	case it.Casts != nil:
		sp := w.pack.Spells[it.Casts.Spell]
		if sp == nil {
			return
		}
		if it.Casts.Message != "" {
			w.message(w.Current(), w.player.Cell(), it.Casts.Message, "")
		}
		power := noPower
		if it.Casts.Power != (content.Dice{}) {
			power = w.roll(it.Casts.Power)
		}
		w.castSpell(w.playerCaster(), sp, power)
	case it.Conversation != "":
		tmpl := w.pack.Things[it.Conversation]
		if tmpl == nil {
			return
		}
		t := w.startTalk(tmpl, nil)
		if t == nil || !it.UseResult {
			return
		}
		t.use = func() {
			if c := w.pack.Manifest.Counters; c != nil && w.counters[c.UseResult] != 0 {
				w.usedUp(hand, o, it, a)
			}
		}
		return
	case it.Power == 0 || !w.usable(it.Power) || !w.startPower(it.Power, 1):
		return
	}
	w.usedUp(hand, o, it, a)
}

// usedUp takes a use from object o in hand: one charge for a use action, else all of it; an
// object used up becomes its spent object.
func (w *World) usedUp(hand int, o *Object, it *content.Item, a content.ItemAction) {
	slot := SlotRightHand + hand
	if w.player.Slots[slot] != o {
		return
	}
	if a.Kind == content.ActionUse {
		if o.Value--; o.Value > 0 {
			return
		}
	}
	w.player.Slots[slot] = nil
	w.takeOff(o)
	if it.Spent != "" {
		spent := w.newObject(it.Spent)
		w.player.Slots[slot] = spent
		w.putOn(spent)
	}
	w.emit(Event{Kind: EventObject, Level: w.current, Cell: w.player.Cell(), Item: o.Item, Detail: "used"})
}
