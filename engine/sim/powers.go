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
	powerShowMap = 0x22
	// powerSmoke and powerOpenAllDoors only show their words.
	powerSmoke        = 0x18
	powerOpenAllDoors = 0x42
	maxTimed          = 255
	confusedOver      = 25
	confusionBump     = 6
	// The confusion a teleporter, a ripple teleport and a change of sex add.
	teleporterConfusion = 30
	rippleConfusion     = 40
	sexChangeConfusion  = 50
	regenerateChance    = 30
	boostAmount         = 2
	boostTicks          = 10
	shieldTicks         = 10
	freezeTicks         = 100
	nearCreatures       = wakeDistance
	smiteDamage         = 2000
)

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

// gainMagic adds n magic points, at most the maximum, and says so when more than 2
// (docs/spec/player.md, Derived values).
func (w *World) gainMagic(n int) {
	p := w.player
	n = max(0, min(n, p.MaxMagicPoints-p.MagicPoints))
	p.MagicPoints += n
	if n > 2 {
		w.messageNumbers(w.Current(), p.Cell(), content.MsgMagicRestored, 0, n)
	}
}

// changeAttribute changes attribute i by n, and its maximum too when withMax, saying so unless
// quiet (docs/spec/player.md, Derived values).
func (w *World) changeAttribute(i, n int, withMax, quiet bool) {
	p := w.player
	a := attributeNames[i]
	p.Attributes[a] += n
	if withMax {
		p.MaxAttributes[a] += n
	}
	if quiet || n == 0 {
		return
	}
	key := content.MsgAttributeUp
	if n < 0 {
		key = content.MsgAttributeDown
	}
	w.message(w.Current(), p.Cell(), key, "attr."+a)
}

// recoverFatigue moves up to n fatigue back to endurance, saying how much (docs/spec/player.md,
// Derived values).
func (w *World) recoverFatigue(n int) {
	p := w.player
	if p.Fatigue == 0 {
		return
	}
	n = min(n, p.Fatigue)
	p.Fatigue -= n
	p.Attributes[AttrEndurance] += n
	p.Exertion = p.Attributes[AttrEndurance] + 3
	if n == 1 {
		w.message(w.Current(), p.Cell(), content.MsgEnduranceRestored, "")
	} else {
		w.messageNumbers(w.Current(), p.Cell(), content.MsgEndurancePointsRestored, 0, n)
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
	w.runDownTimed()
	for i := range p.Boosts {
		b := &p.Boosts[i]
		if b.Ticks > 0 {
			if b.Ticks--; b.Ticks == 0 {
				w.changeAttribute(i, -b.Amount, false, false)
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
		w.gainMagic(w.roll(content.Dice{Count: 2, Sides: 2}))
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

// CreatureNear reports whether a living hostile creature is near the player, as near as power
// 12h reaches, on a cell the player can see (docs/spec/player.md, Napping).
func (w *World) CreatureNear() bool {
	if w.player == nil {
		return false
	}
	l := w.Current()
	for _, t := range w.creaturesNear() {
		if t.Hostile && l.Visible(t.Cell()) {
			return true
		}
	}
	return false
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
	switch {
	case power >= 0x30 && power <= 0x41:
		n := power - 0x30
		w.changeAttribute(n/3, (n%3+1)*sign, false, true)
		return true
	case power >= 0x2A && power <= 0x2F && !ending:
		b := &p.Boosts[power-0x2A]
		if b.Ticks > 0 {
			w.message(l, p.Cell(), content.MsgBoostExtended, "")
		} else {
			b.Amount = boostAmount
			w.changeAttribute(power-0x2A, boostAmount, false, false)
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
	case 0x0C:
		p.Status.ResistQuarter = !ending
	case 0x0D:
		p.Status.ResistForty = !ending
	case 0x0E:
		p.Status.BlowProtection = !ending
	case 0x10:
		w.raiseSkill(SkillWeapon+3, 3*sign, true)
	case 0x1F:
		w.raiseSkill(SkillWeapon+1, 2*sign, true)
	case powerCharges:
		p.Status.Charges = !ending
	case 0x43:
		p.Status.Accurate = !ending
	case 0x4F:
		p.Status.FireImmune = !ending
		p.IntervalBonus -= 2 * sign
	case 0x5B:
		p.Status.DeathBlowWard = !ending
	case 0x16:
	}
	if ending {
		return true
	}
	switch power {
	case 0x01:
		w.gainMagic(w.roll(content.Dice{Count: 2, Sides: 6}))
	case 0x02:
		w.gainMagic(w.roll(content.Dice{Count: 4, Sides: 6}))
	case 0x03:
		w.gainMagic(3000)
	case 0x04:
		p.Confusion += 75
		w.loseHitPoints(w.roll(content.Dice{Count: 2, Sides: 8}), true, "confusion")
		w.poisonPlayer()
	case 0x11, 0x26, 0x1A:
		w.weigh(l, p.Cell(), -p.weight(), 0)
		w.place(w.mark(runeOf(power)).Cell, false)
		w.emit(Event{Kind: EventSound, Level: l.ID, Cell: p.Cell(), Detail: soundTeleport})
		w.transition(content.TransitionRipple)
		w.addConfusion(rippleConfusion)
	case 0x12:
		for _, t := range w.creaturesNear() {
			w.hurtThing(l, t, smiteDamage, combat.Physical)
		}
	case 0x14:
		if w.chance(1, 2) {
			w.changeSex()
			w.message(l, p.Cell(), content.MsgSexChanged, "")
			p.Confusion += sexChangeConfusion
		} else {
			n := 1
			if w.chance(1, 2) {
				n = -1
			}
			w.changeAttribute(w.rng.IntN(6), n, true, false)
		}
	case 0x15:
		p.Poisoned, p.Confusion = false, 0
		w.playerSparkle(w.pack.Effect(content.SparkleCure))
	case 0x1C:
		p.addTimed(TimedFireShield, shieldTicks)
		w.message(l, p.Cell(), content.MsgFireShield, "")
	case 0x27:
		p.addTimed(TimedLightningShield, shieldTicks)
		w.message(l, p.Cell(), content.MsgLightningShield, "")
	case 0x1D:
		w.heal(w.roll(content.Dice{Count: 6, Sides: 6}))
		w.playerSparkle(w.pack.Effect(content.SparkleHeal))
	case 0x28:
		w.heal(w.roll(content.Dice{Count: 5, Sides: 10}))
	case 0x45:
		w.heal(3000)
		w.playerSparkle(w.pack.Effect(content.SparkleHeal))
	case 0x46:
		w.heal(w.roll(content.Dice{Count: 20, Sides: 4}))
	case 0x20:
		w.raiseSkill(SkillSchool+w.rng.IntN(4), 1, false)
	case 0x25:
		w.raiseSkill(SkillWeapon+w.rng.IntN(4), 1, false)
	case 0x21:
		w.freeze(l, freezeTicks)
		w.message(l, p.Cell(), content.MsgFrozen, "")
	case 0x24:
		p.Poisoned = false
	case 0x44:
		w.recoverFatigue(1 + w.rng.IntN(3))
	case powerSmoke:
		w.message(l, p.Cell(), content.MsgCastSmoke, "")
	case powerOpenAllDoors:
		w.message(l, p.Cell(), content.MsgOpenAllDoors, "")
	case 0x4D:
		p.Status.DeathProtection = true
		w.message(l, p.Cell(), content.MsgProtected, "")
	case 0x50:
		w.addConfusion(150)
	case 0x0F, 0x17, 0x4C, 0x47, 0x16, powerSpeed1, powerSpeed2, powerSpeed2Lasting,
		0x0C, 0x0D, 0x0E, 0x10, 0x1F, powerCharges, 0x43, 0x4F, 0x5B:
	default:
		w.emit(Event{Kind: EventNotYet, Level: l.ID, Cell: p.Cell(), Detail: "power " + strconv.Itoa(power)})
		return false
	}
	return true
}

// useObject uses object o, in a hand, on a shelf or in the pack, by a use or use-up action: its power starts unless it
// refuses, or it casts its spell or starts its conversation; then a use takes a charge, and a
// use up leaves its spent object (docs/spec/items.md, Using up).
func (w *World) useObject(o *Object, a content.ItemAction) {
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
		w.learnItem(o.Item, StageUsed)
		w.castSpell(w.playerCaster(), sp, power)
	case it.Conversation != "":
		tmpl := w.pack.Things[it.Conversation]
		if tmpl == nil {
			return
		}
		t := w.startTalk(tmpl, nil)
		if t != nil {
			w.learnItem(o.Item, StageUsed)
		}
		if t == nil || !it.UseResult {
			return
		}
		t.use = func() {
			if c := w.pack.Manifest.Counters; c != nil && w.counters[c.UseResult] != 0 {
				w.usedUp(o, it, a)
			}
		}
		return
	case it.Power == 0:
		return
	case it.Power == powerRandomize:
		// The object becomes a random object where it is kept, rather than being used up
		// (docs/spec/items.md, Powers).
		if len(w.randomItems) == 0 {
			return
		}
		w.learnItem(o.Item, StageUsed)
		w.sound(w.player.Cell(), a.Sound)
		w.replaceKept(o, w.newObject(w.randomItems[w.rng.IntN(len(w.randomItems))]))
		return
	default:
		// Trying it teaches what it does, even when it refuses (docs/spec/player.md, The book).
		w.learnItem(o.Item, StageUsed)
		if !w.usable(it.Power) || !w.startPower(it.Power, 1) {
			return
		}
	}
	w.usedUp(o, it, a)
}

// runDownTimed counts down the timed objects in worn slots: those whose power starts worn and
// lasts. Each loses one from its value, or its item's shared charges, as a 16-bit number; one
// reaching 0 comes off, is announced and is gone (docs/spec/items.md, Timed objects).
func (w *World) runDownTimed() {
	p := w.player
	for slot := Shelves; slot < SlotCount; slot++ {
		o := p.Slots[slot]
		if o == nil {
			continue
		}
		it := w.pack.Items[o.Item]
		if it == nil || !it.PowerWorn || !content.PowerLasts(it.Power) {
			continue
		}
		left := 0
		if it.SharedCharges {
			w.shared[o.Item] = (w.shared[o.Item] - 1) & 0xFFFF
			left = w.shared[o.Item]
		} else {
			o.Value = (o.Value - 1) & 0xFFFF
			left = o.Value
		}
		if left != 0 {
			continue
		}
		p.Slots[slot] = nil
		w.takeOff(o)
		w.message(w.Current(), p.Cell(), content.MsgVanishes, o.Item)
		w.emit(Event{Kind: EventObject, Level: w.current, Cell: p.Cell(), Item: o.Item, Detail: "used"})
		w.emit(Event{Kind: EventObject, Level: w.current, Cell: p.Cell(), Item: o.Item, Detail: "worn_out", Slot: slot})
	}
}

// usedUp uses o up after action a, wherever the player keeps it: a charged object loses a charge
// and stays; another leaves its place to its spent object, or empty (docs/spec/items.md, Using
// up). An object the player no longer keeps is left alone.
func (w *World) usedUp(o *Object, it *content.Item, a content.ItemAction) {
	p := w.player
	slot := p.slotOf(w.pack, o)
	if slot < 0 {
		return
	}
	if a.Kind == content.ActionUse {
		w.useOne(o)
		return
	}
	w.sound(p.Cell(), a.Sound)
	var spent *Object
	if it.Spent != "" {
		spent = w.newObject(it.Spent)
	}
	w.replaceKept(o, spent)
}

// replaceKept puts object by in the place where the player keeps o, or empties the place when by
// is nil, and announces o as used. An object the player no longer keeps is left alone.
func (w *World) replaceKept(o, by *Object) {
	p := w.player
	slot := p.slotOf(w.pack, o)
	switch {
	case slot < 0:
		return
	case slot >= SlotPack:
		bag := p.Pack(w.pack)
		i := slot - SlotPack
		if by != nil {
			bag.Contents[i] = by
		} else {
			bag.Contents = append(bag.Contents[:i], bag.Contents[i+1:]...)
		}
	case slot < Shelves:
		p.Slots[slot] = by
	default:
		p.Slots[slot] = by
		w.takeOff(o)
		if by != nil {
			w.putOn(by)
		}
	}
	w.emit(Event{Kind: EventObject, Level: w.current, Cell: p.Cell(), Item: o.Item, Detail: "used"})
}

// slotOf returns the slot holding o, a fixed slot or a pack slot, or −1.
func (p *Player) slotOf(pk *content.Pack, o *Object) int {
	for i, s := range p.Slots {
		if s == o {
			return i
		}
	}
	if bag := p.Pack(pk); bag != nil {
		for i, c := range bag.Contents {
			if c == o {
				return SlotPack + i
			}
		}
	}
	return -1
}

// Confused reports whether the player is confused: its confusion over 25.
func (p *Player) Confused() bool { return p.Confusion > confusedOver }

// HasStatus reports whether status (content.Status…) holds for the player, as its status icon
// shows it (docs/spec/scenes.md, The game screen).
func (p *Player) HasStatus(status string) bool {
	switch status {
	case content.StatusConfused:
		return p.Confused()
	case content.StatusPoisoned:
		return p.Poisoned
	case content.StatusFireShield:
		return p.Timed[TimedFireShield] > 0
	case content.StatusLightningShield:
		return p.Timed[TimedLightningShield] > 0
	case content.StatusBattleRage:
		return p.Timed[TimedBattleRage] > 0
	case content.StatusHaste:
		return p.Timed[TimedHaste] > 0
	case content.StatusInvisible:
		return p.Timed[TimedInvisibility] > 0
	case content.StatusSight:
		return p.Timed[TimedSight] > 0
	case content.StatusRegenerating:
		return p.Status.Regenerating
	case content.StatusManaRegenerating:
		return p.Status.ManaRegenerating
	case content.StatusLevitating:
		return p.Status.Levitating
	case content.StatusDeathProtection:
		return p.Status.DeathProtection
	}
	return false
}

// addConfusion adds n to the player's confusion, saying so when it makes the player confused
// (docs/spec/player.md, Confusion).
func (w *World) addConfusion(n int) {
	p := w.player
	was := p.Confused()
	p.Confusion += n
	if !was && p.Confused() {
		w.message(w.Current(), p.Cell(), content.MsgConfused, "")
	}
}

// poisonPlayer poisons the player, saying so when it was not poisoned already.
func (w *World) poisonPlayer() {
	p := w.player
	if p == nil || p.Dead || p.Poisoned {
		return
	}
	p.Poisoned = true
	w.message(w.Current(), p.Cell(), content.MsgPoisoned, "")
}

// newGameCounted reports whether the pack's new-game counter is set.
func (w *World) newGameCounted() bool {
	cr := w.pack.Creation
	return cr != nil && cr.Counter != "" && w.counters[cr.Counter] != 0
}

// changeSex changes the player's sex, and its portrait to its portrait's other-sex one.
func (w *World) changeSex() {
	p := w.player
	p.Status.Female = !p.Status.Female
	if pt := w.pack.Portraits[p.Portrait]; pt != nil && pt.Other != "" {
		p.Portrait = pt.Other
	}
}

// Magnified reports whether objects on the floor are drawn magnified: while the player's timed
// sight lasts or the pack's magnify counter is set (docs/spec/items.md, Floor).
func (w *World) Magnified() bool {
	if p := w.player; p != nil && p.Timed[TimedSight] > 0 {
		return true
	}
	c := w.pack.Manifest.Counters
	return c != nil && c.Magnify != "" && w.counters[c.Magnify] != 0
}
