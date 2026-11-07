package sim

import (
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Damage kinds are the pack's (docs/pack.md, Combat): combat.Kind is a place in its list.

// kind returns damage kind id's place in the pack's list, the first kind's for one it lacks.
func (w *World) kind(id string) combat.Kind { return combat.Kind(w.pack.KindOr(id)) }

// isKind reports whether k is damage kind id, which the pack has.
func (w *World) isKind(k combat.Kind, id string) bool {
	i, ok := w.pack.Kind(id)
	return ok && combat.Kind(i) == k
}

// kindID returns damage kind k's id.
func (w *World) kindID(k combat.Kind) string {
	kinds := w.pack.CombatRules().Kinds
	if k < 0 || int(k) >= len(kinds) {
		return ""
	}
	return kinds[k].ID
}

// kinds returns the set of damage kinds ids name, leaving out those the pack lacks.
func (w *World) kinds(ids []string) combat.Kinds {
	var s combat.Kinds
	for _, id := range ids {
		if k, ok := w.pack.Kind(id); ok {
			s = s.With(combat.Kind(k))
		}
	}
	return s
}

// legacyResists are the behaviors that resist a damage kind, by kind id.
var legacyResists = map[string]string{
	content.KindFire: content.BehaviorResistsFire, content.KindLightning: content.BehaviorResistsLight,
	content.KindPoison: content.BehaviorResistsPoison,
}

// creatureDefense is what creature t's template does to damage: its lists of kinds, with the
// behaviors that stand for them (resists_fire for resisting fire; physical_only for ignoring
// every kind but the first).
func (w *World) creatureDefense(t *Thing) combat.CreatureDefense {
	tm := t.Template
	if d, ok := w.defenses[tm]; ok {
		return d
	}
	d := combat.CreatureDefense{
		Invulnerable: tm.Has(content.BehaviorInvulnerable),
		Resists:      w.kinds(tm.Resists), Ignores: w.kinds(tm.Ignores), Weak: w.kinds(tm.WeakTo),
	}
	for id, b := range legacyResists {
		if tm.Has(b) {
			d.Resists |= w.kinds([]string{id})
		}
	}
	if tm.Has(content.BehaviorPhysicalOnly) {
		for k := 1; k < len(w.pack.CombatRules().Kinds); k++ {
			d.Ignores = d.Ignores.With(combat.Kind(k))
		}
	}
	if w.defenses == nil {
		w.defenses = map[*content.Thing]combat.CreatureDefense{}
	}
	w.defenses[tm] = d
	return d
}

// resistsPoison reports whether creature t cannot be poisoned: it resists or ignores poison.
func (w *World) resistsPoison(t *Thing) bool {
	if t.Template.Has(content.BehaviorResistsPoison) {
		return true
	}
	k, ok := w.pack.Kind(content.KindPoison)
	if !ok {
		return false
	}
	d := w.creatureDefense(t)
	return d.Resists.Has(combat.Kind(k)) || d.Ignores.Has(combat.Kind(k))
}

// timedByName are the timed effects by their names in the pack.
var timedByName = map[string]int{
	content.TimedSight: TimedSight, content.TimedFireShield: TimedFireShield,
	content.TimedLightningShield: TimedLightningShield, content.TimedBattleRage: TimedBattleRage,
	content.TimedHaste: TimedHaste, content.TimedInvisibility: TimedInvisibility,
	content.TimedWeaponSkills: TimedWeaponSkills, content.TimedMagicSkills: TimedMagicSkills,
}

// combatStatus is what the player's statuses do to a fight, its shields those of the pack's
// combat record that are running.
func (w *World) combatStatus() combat.Status {
	p := w.player
	s := combat.Status{
		BattleRage: p.Timed[TimedBattleRage] > 0, Accurate: p.Status.Accurate,
		Invisible:      p.Status.Invisible || p.Timed[TimedInvisibility] > 0,
		BlowProtection: p.Status.BlowProtection, ResistQuarter: p.Status.ResistQuarter, ResistForty: p.Status.ResistForty,
	}
	for name, id := range w.pack.CombatRules().Shields {
		if i, ok := timedByName[name]; ok && p.Timed[i] > 0 {
			s.Shields |= w.kinds([]string{id})
		}
	}
	return s
}

// Defenses learned for kinds a creature's lists name: "resists:fire" and so on.
const (
	DefenseResists = "resists:"
	DefenseIgnores = "ignores:"
	DefenseWeak    = "weak:"
)

// meetKind learns the defenses of creature t that being hurt by kind k shows: what its behaviors
// say of it, and what its lists say. A weakness of a creature with a vulnerability shows as the
// vulnerability.
func (w *World) meetKind(l *Level, t *Thing, k combat.Kind) {
	tm, id := t.Template, w.kindID(k)
	if b, ok := legacyResists[id]; ok {
		w.meetDefense(l, t, b)
	}
	if k != combat.Physical {
		w.meetDefense(l, t, content.BehaviorPhysicalOnly)
	}
	learn := func(d string) {
		if l != w.Current() {
			return
		}
		if e := w.learnCreature(t, StageFought); e != nil && !e.Knows(d) {
			e.Defenses = append(e.Defenses, d)
		}
	}
	switch {
	case slices.Contains(tm.Resists, id):
		learn(DefenseResists + id)
	case slices.Contains(tm.Ignores, id):
		learn(DefenseIgnores + id)
	}
	if slices.Contains(tm.WeakTo, id) {
		if tm.Vulnerability != nil {
			w.meetDefense(l, t, DefenseVulnerable)
		} else {
			learn(DefenseWeak + id)
		}
	}
}

// shotAim is how far off a straight line, in units, the player may stand for a creature to
// shoot at it.
const shotAim = 12

// creatureShoots has hostile creature t with a ranged attack shoot at the player when its
// interval has passed and the player stands within reach along one of the eight directions
// (docs/pack.md, Things): it faces that way, takes its attack pose and launches its shot, which
// vanishes where it lands. It reports whether t shot.
func (w *World) creatureShoots(l *Level, t *Thing) bool {
	s, p := t.Template.Shoots, w.player
	if s == nil || p == nil || p.Dead || p.hidden() || t.Alert == 0 || w.current != l.ID || w.step < t.ShotNext {
		return false
	}
	speed := s.Speed
	if speed <= 0 {
		speed = flightSpeed
	}
	du, dv := p.U-t.U, p.V-t.V
	best, bestOff := -1, shotAim+1
	for d, dir := range Dirs {
		along := du*dir[0] + dv*dir[1]
		if along <= 0 || along > s.Range*speed*max(1, abs(dir[0])+abs(dir[1])) {
			continue
		}
		off := abs(du*dir[1] - dv*dir[0])
		if dir[0] != 0 && dir[1] != 0 {
			off /= 2
		}
		if off < bestOff {
			best, bestOff = d, off
		}
	}
	if best < 0 {
		return false
	}
	t.Facing, t.Pose = best, PoseAttack
	t.ShotNext = w.step + stepsMS(s.IntervalMS)
	k := w.kind(s.Kind)
	w.launch(l, w.newObject(s.Item), t.U, t.V, best, s.Range, w.roll(s.Damage), false, true, t,
		shotOptions{kind: &k, speed: s.Speed, vanishes: true})
	return true
}
