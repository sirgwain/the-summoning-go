package viewer

import (
	"image"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
)

// idle runs n updates with no input.
func idle(v *Viewer, n int) {
	for range n {
		v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: image.Pt(-1, -1)})
	}
}

// TestPanelsHoldTheWorld opens the spell panel and the inventory panel in turn: while either is
// open no step passes, though a gesture entered on the spell panel is still learned; closed, the
// world runs again.
func TestPanelsHoldTheWorld(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	step := v.World.Step()

	pressKey(v, "R")
	if !v.SpellsOpen {
		t.Fatal("R did not open the spell panel")
	}
	idle(v, 20)
	for _, k := range []Key{"D", "J", KeyEnter} {
		pressKey(v, k)
	}
	if !v.World.Player().Magic.Learned(devpack.SpellFlameArrow) {
		t.Error("the spell panel held the world but did not learn DJ")
	}
	if got := v.World.Step(); got != step {
		t.Errorf("with the spell panel open the step went %d to %d", step, got)
	}
	pressKey(v, KeyEscape)
	idle(v, 5)
	if got := v.World.Step(); got <= step {
		t.Fatalf("with the spell panel closed the step stayed %d", got)
	}

	step = v.World.Step()
	pressKey(v, "I")
	if v.Inventory() == nil {
		t.Fatal("I did not open the inventory panel")
	}
	idle(v, 20)
	if got := v.World.Step(); got != step {
		t.Errorf("with the inventory panel open the step went %d to %d", step, got)
	}
	pressKey(v, KeyEscape)
	idle(v, 5)
	if got := v.World.Step(); got <= step {
		t.Errorf("with the inventory panel closed the step stayed %d", got)
	}
}
