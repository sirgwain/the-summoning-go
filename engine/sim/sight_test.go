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
