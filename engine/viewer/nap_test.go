package viewer

import (
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
)

// napInput holds N, pressed when press.
func napInput(press bool) Input {
	return Input{Held: map[Key]bool{KeyNap: true}, Pressed: map[Key]bool{KeyNap: press}}
}

// TestNap holds N: the world runs napFactor updates per update until N is released, and starts
// again only on a new press; full magic points refuse a nap and end one.
func TestNap(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	p := v.World.Player()
	l := v.World.Current()
	things := l.Things
	l.Things = nil
	p.MagicPoints = 0
	step := v.World.Step()
	v.Update(napInput(true))
	v.Update(napInput(false))
	if !v.Napping || v.World.Step()-step != 2*napFactor {
		t.Fatalf("napping %v after %d steps, notice %q, mp %d/%d", v.Napping, v.World.Step()-step, v.Notice, p.MagicPoints, p.MaxMagicPoints)
	}
	if !strings.HasPrefix(v.Notice, "Waiting: ") {
		t.Errorf("notice %q", v.Notice)
	}
	// Another key stops it, and holding N on does not start it again.
	in := napInput(false)
	in.Pressed["I"] = true
	v.Update(in)
	step = v.World.Step()
	v.Update(napInput(false))
	if v.Napping || v.World.Step()-step != 1 {
		t.Fatalf("napping %v after another key, %d steps", v.Napping, v.World.Step()-step)
	}
	// Released and pressed again, it naps until the magic points are full.
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}})
	p.MagicPoints = p.MaxMagicPoints - 1
	v.Update(napInput(true))
	for i := 0; v.Napping && i < 1000; i++ {
		v.Update(napInput(false))
	}
	if v.Napping || p.MagicPoints < p.MaxMagicPoints || v.Notice != "Your magic points are full." {
		t.Fatalf("napping %v at %d/%d magic points, notice %q", v.Napping, p.MagicPoints, p.MaxMagicPoints, v.Notice)
	}
	// Full, a new press is refused.
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}})
	step = v.World.Step()
	v.Update(napInput(true))
	if v.Napping || v.World.Step()-step != 1 {
		t.Errorf("a nap at full magic points: napping %v, %d steps", v.Napping, v.World.Step()-step)
	}
	// A creature near refuses one.
	l.Things = things
	if !v.World.CreatureNear() {
		t.Skip("no creature near the study's start to refuse a nap")
	}
	p.MagicPoints = 0
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}})
	v.Update(napInput(true))
	if v.Napping || v.Notice != "A creature is near." {
		t.Errorf("a nap with a creature near: napping %v, notice %q", v.Napping, v.Notice)
	}
}
