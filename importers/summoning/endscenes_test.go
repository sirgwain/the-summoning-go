package summoning

import (
	"math/rand/v2"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// TestEndScaling checks the scaling's counts add up to the scaled size, and keep the ends.
func TestEndScaling(t *testing.T) {
	for s := 1; s < 60; s++ {
		for den := 1; den <= 20; den++ {
			for n := 1; n <= den; n++ {
				sum := 0
				for _, c := range scaleCounts(s, n, den) {
					sum += c
				}
				if sum != scaledSize(s, n, den) {
					t.Fatalf("%d scaled %d/%d: %d pixels, size %d", s, n, den, sum, scaledSize(s, n, den))
				}
			}
		}
	}
	if c := scaleCounts(4, 1, 2); c[0]+c[1]+c[2]+c[3] != 2 {
		t.Errorf("4 halved: %v", c)
	}
}

// TestEndWander steps a figure along the axis farther from its target, then diagonally.
func TestEndWander(t *testing.T) {
	f := &endFigure{homeR: 10, homeC: 10, r: 0, c: 0, tr: 3, tc: 1, moving: true}
	var path [][2]int
	for f.moving {
		f.wander(nil)
		path = append(path, [2]int{f.r, f.c})
	}
	if want := [][2]int{{1, 0}, {2, 0}, {3, 1}}; len(path) != len(want) || path[0] != want[0] || path[1] != want[1] || path[2] != want[2] {
		t.Errorf("path %v, want %v", path, want)
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		f.wander(rng)
		if !f.moving {
			continue
		}
		if f.tr < 10 || f.tr > 24 || f.tc < 10 || f.tc > 24 {
			t.Fatalf("target (%d, %d) is not within 0–14 of home", f.tr, f.tc)
		}
	}
}

// TestEndBolt checks where a bolt meets a figure, and that a figure facing left is mirrored to
// end at its column.
func TestEndBolt(t *testing.T) {
	right := &endFigure{c: 18, width: 136, right: true}
	left := &endFigure{c: 299, width: 112, art: endArt{id: "sprite.f", w: 120}}
	figs := []*endFigure{right, left}
	for _, c := range []struct {
		x, facing int
		hit       bool
	}{
		{18, 1, true}, {122, 1, true}, {123, 1, false}, {17, 1, false},
		{219 - 24, -1, true}, {299 - 24, -1, true}, {218 - 24, -1, false}, {300 - 24, -1, false},
	} {
		if got := hit(c.x, figs, c.facing); got != c.hit {
			t.Errorf("a bolt at %d against facing %d: hit %v", c.x, c.facing, got)
		}
	}
	if p := left.place(2, 0); !p.Mirror || p.At != (content.Offset{299 - 120, 0}) || p.Frame != 7 {
		t.Errorf("left figure placed %+v", p)
	}
	left.held = [2]int{-1, -1}
	if p := left.place(0, 0); p.Frame != 0 {
		t.Errorf("holding nothing: frame %d", p.Frame)
	}
}
