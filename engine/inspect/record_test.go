package inspect

import (
	"slices"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// TestFullBookAnimations shows a creature's animations as their frames: of those named for
// directions, the down-right one's, else the first direction's; the rest as they are.
func TestFullBookAnimations(t *testing.T) {
	p := content.NewPack(content.Manifest{})
	th := &content.Thing{ID: "thing.test", Kind: content.ThingCreature, Sprite: "sprite.none", Animations: map[string]content.Animation{
		"walk.0": {Frames: []int{0, 4}}, "walk.2": {Frames: []int{1, 5}, MS: content.Fixed(330)}, "walk.4": {Frames: []int{2, 6}},
		"attack.4": {Frames: []int{6}}, "attack.6": {Frames: []int{7}}, "dead": {Frames: []int{8}},
	}}
	got := map[string][]content.Image{}
	for _, it := range Animations(p, th) {
		got[strings.TrimSpace(it.Text)] = it.Pictures
	}
	img := func(fs ...int) []content.Image { return Frames("sprite.none", fs) }
	want := map[string][]content.Image{
		"animations:":                        nil,
		"attack (direction 4; 2 directions)": img(6),
		"dead":                               img(8),
		"walk (direction 2, down-right; 3 directions), 330 ms a frame": img(1, 5),
	}
	if len(got) != len(want) {
		t.Errorf("got %v", got)
	}
	for k, w := range want {
		if g, ok := got[k]; !ok || !slices.Equal(g, w) {
			t.Errorf("%q: %v, want %v (all %v)", k, g, w, got)
		}
	}
}

// TestLevelPlaces finds where the fixture's key is: in the sack on the floor, carried by the
// guard, and made by the plate's procedure; and the guard where it stands.
func TestLevelPlaces(t *testing.T) {
	w := testWorld(t)
	ps := LevelPlaces(w.Pack(), w)
	var texts []string
	for _, pl := range ps.Items["item.key"] {
		texts = append(texts, pl.Text)
	}
	want := []string{"in Sack on the floor at (0, 0)", "carried by guard", "p0: make Key at (1, 0)"}
	if !slices.Equal(texts, want) {
		t.Errorf("the key's places %q, want %q", texts, want)
	}
	if g := ps.Creatures["thing.guard"]; len(g) != 1 || g[0].Thing != "guard" {
		t.Errorf("the guard's places %+v", g)
	}
	if s := ps.Items["item.sack"]; len(s) != 1 || s[0].Cell == nil || *s[0].Cell != (content.Cell{0, 0}) {
		t.Errorf("the sack's places %+v", s)
	}
}
