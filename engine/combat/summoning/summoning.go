// Package summoning is The Summoning's combat rule set (docs/spec/combat.md, docs/spec/player.md).
package summoning

import (
	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Rules is the Summoning's rule set.
type Rules struct{}

var _ combat.Rules = Rules{}

// hitScore is the score that hits.
const hitScore = 12

func d20(r combat.Roller) int { return 1 + r.IntN(20) }

// PlayerStrikes rolls the player's to-hit score and damage (docs/spec/combat.md, The player
// attacks).
func (Rules) PlayerStrikes(r combat.Roller, s combat.PlayerStrike) combat.Result {
	score := (s.Agility-s.CreatureAccuracy)/2 + d20(r)
	if s.Endurance < 10 {
		score += s.Endurance - 10
	}
	score += (s.Accuracy - 12) / 2
	if s.Status.Accurate {
		score += 2
	}
	if s.Classed {
		score += s.Skill - 1
	}
	if s.Status.BattleRage {
		score += 4
	}
	if s.Behind {
		score += 4
	}
	score = max(score, 1)
	if score < hitScore {
		return combat.Result{Score: score}
	}
	d := combat.Roll(r, s.Damage)
	if d <= 0 {
		return combat.Result{Score: score, Hit: true}
	}
	if s.Classed {
		d += s.Skill
	}
	if s.Behind {
		d *= 2
	}
	if s.Status.BattleRage {
		d += d / 2
	}
	if s.Status.Accurate {
		d += 2
	}
	if s.Vulnerable {
		d += d / 4
	}
	if s.Weapon {
		d += s.Strength - 10
	}
	if s.PhysicalOnly {
		d = 0
		if s.Hurts != nil {
			d = combat.Roll(r, *s.Hurts)
		}
	}
	return combat.Result{Score: score, Hit: true, Damage: max(0, d-s.Resistance)}
}

// CreatureStrikes rolls a creature's to-hit score and damage (docs/spec/combat.md, Creatures
// attack).
func (Rules) CreatureStrikes(r combat.Roller, s combat.CreatureStrike) combat.Result {
	roll := d20(r)
	score := (s.Accuracy-s.PlayerAgility)/2 + s.Bonus + roll
	if s.Status.BattleRage {
		score += 4
	}
	if s.Status.Invisible {
		score -= 4
	}
	if s.Status.BlowProtection {
		score -= 4
	}
	if s.Behind {
		score += 4
	}
	score = max(score, 1)
	if score < hitScore {
		return combat.Result{Score: score}
	}
	d := combat.Roll(r, s.Damage)
	if s.Status.BlowProtection && roll != 20 {
		d -= d / 4
	}
	if s.Behind {
		d *= 2
	}
	return combat.Result{Score: score, Hit: true, Damage: max(0, d)}
}

// HurtPlayer works damage through the player's defenses (docs/spec/combat.md, Hurting the
// player). Charges in held objects come with statuses.
func (Rules) HurtPlayer(d int, k combat.Kind, def combat.PlayerDefense) (int, int) {
	st := def.Status
	if st.BattleRage {
		d *= 2
	}
	if st.ResistQuarter {
		d -= d / 4
	}
	if st.ResistForty {
		d -= 4 * d / 10
	}
	d = max(d, 1)
	if st.Shields.Has(k) {
		d = 1
	}
	if k != combat.Physical && def.Absorb != nil {
		if d = def.Absorb(d); d <= 0 {
			return 0, 0
		}
	}
	wear := 0
	if def.BodyArmor && def.ArmorClass > 0 {
		wear = min(d, def.ArmorClass)
		d = max(d-def.ArmorClass, 1)
	}
	return d, wear
}

// HurtCreature works damage through a creature's defenses (docs/spec/combat.md, Hurting a
// creature).
func (Rules) HurtCreature(d int, k combat.Kind, def combat.CreatureDefense) int {
	switch {
	case def.Invulnerable, def.Ignores.Has(k):
		return 0
	}
	if def.Resists.Has(k) {
		d /= 4
	}
	if def.Weak.Has(k) {
		d += d / 4
	}
	return d
}

// Wear takes 3n when n is more than 1 (docs/spec/combat.md, Wear).
func (Rules) Wear(n int) int {
	if n > 1 {
		return 3 * n
	}
	return n
}

// levels are each level's hit and magic point dice (docs/spec/player.md, Experience and levels).
var levels = [...]struct{ hit, magic content.Dice }{
	{d(1, 10), d(1, 10)}, {d(1, 10), d(2, 5)}, {d(2, 10), d(2, 10)}, {d(2, 10), d(4, 5)},
	{d(3, 10), d(3, 10)}, {d(3, 10), d(6, 5)}, {d(3, 10), d(6, 5)}, {d(4, 10), d(4, 10)},
	{d(4, 10), d(8, 5)}, {d(4, 10), d(8, 5)}, {d(5, 10), d(5, 10)},
}

func d(n, sides int) content.Dice { return content.Dice{Count: n, Sides: sides} }

// LevelThreshold is 500 for the first level, each doubling the last, with 512000 becoming
// 600000 (and so the next 1200000).
func (Rules) LevelThreshold(level int) int {
	t := 500
	for range level {
		if t *= 2; t == 512000 {
			t = 600000
		}
	}
	return t
}

// MaxLevel is 11.
func (Rules) MaxLevel() int { return len(levels) }

// LevelGain returns level's dice, level counting from 1.
func (Rules) LevelGain(level int) (content.Dice, content.Dice) {
	l := levels[min(max(level, 1), len(levels))-1]
	return l.hit, l.magic
}

// SkillThreshold doubles, up to 1400h.
func (Rules) SkillThreshold(t int) int { return min(2*t, 0x1400) }

// SkillExperience is 57.
func (Rules) SkillExperience() int { return 57 }
