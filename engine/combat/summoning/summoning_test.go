package summoning

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// fixed rolls n − 1 every time, so a d20 is 20 and each die its highest.
type fixed struct{}

func (fixed) IntN(n int) int { return n - 1 }

func TestPlayerStrikeRules(t *testing.T) {
	base := combat.PlayerStrike{Agility: 12, Endurance: 12, Accuracy: 12, Strength: 15, Weapon: true, Classed: true, Skill: 3,
		Damage: content.Dice{Count: 1, Sides: 6}}
	// d20 20 + (12 − 0) / 2 + 0 + 3 − 1 = 28; damage 6 + 3 + 15 − 10 = 14.
	if r := (Rules{}).PlayerStrikes(fixed{}, base); r.Score != 28 || r.Damage != 14 {
		t.Errorf("a weapon: %+v", r)
	}
	// A roll of 0 does nothing, whatever would be added.
	none := base
	none.Damage = content.Dice{}
	if r := (Rules{}).PlayerStrikes(fixed{}, none); !r.Hit || r.Damage != 0 {
		t.Errorf("no damage code: %+v", r)
	}
	// An object with no weapon class gets no skill, and still its strength.
	plain := base
	plain.Classed = false
	if r := (Rules{}).PlayerStrikes(fixed{}, plain); r.Score != 26 || r.Damage != 11 {
		t.Errorf("an object with no class: %+v", r)
	}
}
