package summoning

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// TestTactics checks the tactics the importer gives walkers (docs/spec/creatures.md, Tactics,
// Import).
func TestTactics(t *testing.T) {
	for _, c := range []struct {
		spell, poisonous bool
		attack, interval int
		keep             int
		after            string
	}{
		{true, false, 0x45, 6, 4, content.AfterBlowBackOff},
		{true, true, 0, 2, 4, content.AfterBlowBackOff},
		{false, true, 0x33, 6, 0, content.AfterBlowBackOff},
		{false, true, 0, 6, 0, ""},
		{false, false, 0x33, 3, 0, content.AfterBlowFlank},
		{false, false, 0x33, 2, 0, content.AfterBlowFlank},
		{false, false, 0, 2, 0, ""},
		{false, false, 0x33, 4, 0, ""},
	} {
		keep, after := tactics(c.spell, c.poisonous, c.attack, c.interval)
		if keep != c.keep || after != c.after {
			t.Errorf("%+v: keep %d, after %q", c, keep, after)
		}
	}
}
