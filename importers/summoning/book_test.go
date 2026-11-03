package summoning

import "testing"

func TestPowerEffect(t *testing.T) {
	classes := [4]string{"Falchion", "War Axe", "Spear", "Bow"}
	schools := [4]string{"Wizardry", "Sorcery", "Enchantment", "Healing"}
	for _, c := range []struct {
		power int
		want  string
	}{
		{0x28, "Heals 5d10 hit points."},
		{0x06, "Raises Sorcery skill by 1."},
		{0x10, "Raises skill with weapons like the Bow by 3."},
		{0x34, "Raises agility by 2."},
		{0x2D, "Raises accuracy by 2 for a while."},
		{0x13, ""}, // a spell: the book names it
		{0x16, ""},
	} {
		if got := powerEffect(c.power, classes, schools); got != c.want {
			t.Errorf("power %02Xh: %q, want %q", c.power, got, c.want)
		}
	}
}
