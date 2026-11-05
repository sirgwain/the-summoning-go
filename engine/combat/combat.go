// Package combat is the interface between the simulation and a combat rule set: to-hit, damage,
// hurting either side, and the numbers of experience and practice. The simulation finds who
// strikes whom and applies the results; a rule set only rolls and computes, from plain numbers
// (docs/design.md, Characters, stats and combat; docs/spec/combat.md, Engine model).
package combat

import "github.com/sirgwain/the-summoning-go/engine/content"

// Roller is the world's random numbers: IntN returns 0 .. n−1.
type Roller interface{ IntN(n int) int }

// Roll rolls dice.
func Roll(r Roller, d content.Dice) int {
	n := d.Bonus
	for range d.Count {
		if d.Sides > 0 {
			n += 1 + r.IntN(d.Sides)
		}
	}
	return n
}

// Kind is a kind of damage: its place in the pack's list of kinds (docs/pack.md, Combat). The
// first, Physical, is what blows, falls and boulders do.
type Kind int

// Physical is the pack's first damage kind.
const Physical Kind = 0

// Kinds is a set of damage kinds.
type Kinds uint64

// Has reports whether s holds kind k.
func (s Kinds) Has(k Kind) bool { return k >= 0 && k < 64 && s&(1<<k) != 0 }

// With returns s with kind k.
func (s Kinds) With(k Kind) Kinds {
	if k < 0 || k >= 64 {
		return s
	}
	return s | 1<<k
}

// Status is what the player's statuses do to a fight. Shields are the kinds the player's
// shields cut to 1 point.
type Status struct {
	BattleRage, Accurate, Invisible, BlowProtection bool
	ResistQuarter, ResistForty                      bool
	Shields                                         Kinds
}

// PlayerStrike is the player striking a creature with a hand.
type PlayerStrike struct {
	Agility, Endurance, Accuracy, Strength int
	// Weapon is false for bare hands; Skill is the weapon class's skill level.
	Weapon bool
	Skill  int
	Damage content.Dice
	Behind bool
	Status Status
	// The creature: its accuracy and resistance; Vulnerable when the weapon's class is its
	// vulnerability; PhysicalOnly when only weapons that hurt such creatures (Hurts) do.
	CreatureAccuracy, Resistance int
	Vulnerable, PhysicalOnly     bool
	// Hurts is the damage of a weapon that hurts physical-only creatures, nil for others.
	Hurts *content.Dice
}

// CreatureStrike is a creature striking the player.
type CreatureStrike struct {
	Accuracy, Bonus int
	Damage          content.Dice
	Behind          bool
	PlayerAgility   int
	Status          Status
}

// Result is a strike's outcome.
type Result struct {
	Score  int
	Hit    bool
	Damage int
}

// PlayerDefense is what stands between damage and the player's hit points.
type PlayerDefense struct {
	Status     Status
	ArmorClass int
	// BodyArmor is whether the player wears body armor, which the blow wears.
	BodyArmor bool
	// Absorb, when the player has the charges status, takes what charges absorb of damage d
	// and returns the rest.
	Absorb func(d int) int
}

// CreatureDefense is what a creature's template does to damage it takes: none at all when
// Invulnerable, and by kind a quarter (Resists), none (Ignores) or a quarter more (Weak).
type CreatureDefense struct {
	Invulnerable           bool
	Resists, Ignores, Weak Kinds
}

// Rules is a combat rule set.
type Rules interface {
	// PlayerStrikes rolls the player's strike.
	PlayerStrikes(r Roller, s PlayerStrike) Result
	// CreatureStrikes rolls a creature's strike.
	CreatureStrikes(r Roller, s CreatureStrike) Result
	// HurtPlayer returns the hit points damage d of kind k takes, and how much it wears the body
	// armor.
	HurtPlayer(d int, k Kind, def PlayerDefense) (damage, armorWear int)
	// HurtCreature returns the hit points damage d of kind k takes from a creature.
	HurtCreature(d int, k Kind, def CreatureDefense) int
	// Wear returns how much using an object by n wears it.
	Wear(n int) int
	// LevelThreshold returns the experience the player's level after level needs.
	LevelThreshold(level int) int
	// MaxLevel is the highest level.
	MaxLevel() int
	// LevelGain returns the hit and magic point dice reaching level adds.
	LevelGain(level int) (hit, magic content.Dice)
	// SkillThreshold returns the practice the next skill level needs after threshold.
	SkillThreshold(threshold int) int
	// SkillExperience is the experience a skill level gives.
	SkillExperience() int
}
