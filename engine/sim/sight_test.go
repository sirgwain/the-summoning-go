package sim

import "testing"

// A long room is visible end to end, but only the cells near the player are seen, and walking
// down it sees the rest (docs/spec/world.md, Engine model).
func TestSeenWithinSightWindow(t *testing.T) {
	w := walkWorld(t,
		"######################",
		"#@...................#",
		"######################")
	l := w.Current()
	far := cell{1, 20}
	if !l.Visible(far) {
		t.Fatal("the far end of the room is not visible")
	}
	if l.Seen(far) {
		t.Error("the far end of the room is seen from the start")
	}
	for _, c := range []cell{{1, 1}, {1, 8}, {0, 7}, {2, 6}} {
		if !l.Seen(c) {
			t.Errorf("%v near the player is not seen", c)
		}
	}
	if l.Seen(cell{1, 10}) {
		t.Error("(1, 10), nine tile columns away, is seen")
	}
	w.place(cell{1, 15}, true)
	if !l.Seen(far) {
		t.Error("the far end is not seen once the player is near it")
	}
	w.place(cell{1, 1}, true)
	if !l.Seen(far) {
		t.Error("the far end was forgotten as seen")
	}
}

// The player's ground point in a wall's cell, as a diagonal step into an inside corner can leave
// it, does not move the vision cell, so the room beyond that wall stays dark (docs/spec/world.md,
// Visibility).
func TestVisionStaysOffWalls(t *testing.T) {
	w := walkWorld(t,
		"######",
		"#@#..#",
		"#.####",
		"######")
	l, p := w.Current(), w.Player()
	p.U, p.V = cellCenter(cell{1, 2})
	w.see()
	if p.vision != (cell{1, 1}) {
		t.Errorf("vision cell %v, want the floor (1, 1)", p.vision)
	}
	if l.Visible(cell{1, 3}) {
		t.Error("the room beyond the wall is visible")
	}
}
