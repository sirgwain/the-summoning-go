package summoning

import "testing"

// TestWindUp checks the wind-up's formula (docs/spec/combat.md, Wind-up): fewer ms for more hit
// dice and for a shorter interval, within 300..800.
func TestWindUp(t *testing.T) {
	for _, c := range []struct{ dice, interval, want int }{
		{0, 6, 800},
		{2, 6, 700},
		{3, 3, 325},
		{0, 3, 400},
		{0, 2, 300},
		{3, 8, 800},
		{10, 8, 400},
		{10, 4, 300},
		{255, 3, 300},
	} {
		if got := windUp(c.dice, c.interval); got != c.want {
			t.Errorf("%d hit dice every %d ticks: %d ms, want %d", c.dice, c.interval, got, c.want)
		}
	}
}
