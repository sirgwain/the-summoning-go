package sim

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// TestHold learns a spell and takes an object onto the hand in a held world: both are done, no
// step passes, and a walk is refused.
func TestHold(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2)
	p := w.Player()
	c := p.Cell()
	do(w, input.Drop{Cell: c, Item: devpack.ItemKey})
	step, u, v := w.Step(), p.U, p.V

	ev := w.Hold([]input.Command{input.Learn{Gesture: "DJ"}, input.PickUp{Hand: true}, input.Walk{Direction: 2}})
	if !p.Magic.Learned(devpack.SpellFlameArrow) {
		t.Errorf("held, learning DJ did nothing: %v", ev)
	}
	if p.Hand == nil || p.Hand.Item != devpack.ItemKey {
		t.Errorf("held, the hand holds %+v", p.Hand)
	}
	if !has(ev, EventRefused, "the world is held") {
		t.Errorf("held, the walk was not refused: %v", ev)
	}
	for range 50 {
		w.Hold([]input.Command{input.Walk{Direction: 2}})
	}
	if w.Step() != step || p.U != u || p.V != v {
		t.Errorf("held, the step went %d to %d and the player %d,%d to %d,%d", step, w.Step(), u, v, p.U, p.V)
	}
	do(w)
	if w.Step() != step+1 {
		t.Errorf("an update after holding ran step %d, want %d", w.Step(), step+1)
	}
}
