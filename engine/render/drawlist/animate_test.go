package drawlist

import (
	"slices"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// A still thing's timed idle animation steps by the clock; a walking one, or one whose frame the
// simulation chose, is left alone.
func TestAnimator(t *testing.T) {
	p, _ := testPack()
	p.Things["portal"] = &content.Thing{ID: "portal", Kind: content.ThingPortal, Sprite: "npc",
		Animations: map[string]content.Animation{"idle": {Frames: []int{0, 1, 2, 1}, MS: content.Fixed(100)}}}
	fixed := 2
	scene := func() Scene {
		return Scene{Things: []Thing{
			{ID: "a", Template: "portal"},
			{ID: "b", Template: "portal", Anim: "walk.0"},
			{ID: "c", Template: "portal", Frame: &fixed},
		}}
	}
	an := NewAnimator(1)
	var frames []int
	for range 10 {
		s := scene()
		an.Apply(p, &s)
		if s.Things[1].Frame != nil || *s.Things[2].Frame != 2 {
			t.Fatalf("animated what it should not: %+v", s.Things)
		}
		frames = append(frames, *s.Things[0].Frame)
		an.Advance(50)
	}
	if want := []int{0, 0, 1, 1, 2, 2, 1, 1, 0, 0}; !slices.Equal(frames, want) {
		t.Errorf("frames %v, want %v", frames, want)
	}
	// A thing gone is forgotten.
	an.Apply(p, &Scene{})
	if len(an.states) != 0 {
		t.Errorf("states left: %v", an.states)
	}
}
