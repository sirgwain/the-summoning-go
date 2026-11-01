package sim

import (
	"slices"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// music returns the music events' details.
func music(events []Event) []string {
	var out []string
	for _, e := range events {
		if e.Kind == EventMusic {
			out = append(out, e.Detail)
		}
	}
	return out
}

// footsteps walks n moves along d and returns the moves each footstep came on.
func footsteps(w *World, d, n int) []int {
	p := w.Player()
	var at []int
	for start := p.Moves; p.Moves-start < n; {
		for _, e := range walk(w, d, 1) {
			if e.Kind == EventSound && e.Detail == content.CueFootstep {
				at = append(at, p.Moves-start)
			}
		}
	}
	return at
}

func TestFootsteps(t *testing.T) {
	w := walkWorld(t,
		"####################",
		"#@.................#",
		"####################")
	if got := footsteps(w, 1, 7); !slices.Equal(got, []int{3, 6}) {
		t.Errorf("from the start: footsteps on moves %v, want [3 6]", got)
	}
	for range idleUpdates + 4 {
		w.Update(nil)
	}
	if got := footsteps(w, 5, 4); !slices.Equal(got, []int{3}) {
		t.Errorf("after standing: footsteps on moves %v, want [3]", got)
	}
	w.Player().Status.Levitating = true
	if got := footsteps(w, 1, 6); len(got) != 0 {
		t.Errorf("levitating: footsteps on moves %v", got)
	}
}

func TestTheme(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) { p.Things["thing.guard"].Theme = "music.8" },
		"#######",
		"#@...G#",
		"#######")
	if got := music(w.Update(nil)); !slices.Equal(got, []string{"music.8"}) {
		t.Fatalf("a themed creature in sight: music %q", got)
	}
	if got := music(w.Update(nil)); len(got) != 0 {
		t.Fatalf("still in sight: music %q", got)
	}
	w.Current().Thing("guard").HitPoints = 0
	if got := music(w.Update(nil)); !slices.Equal(got, []string{""}) {
		t.Fatalf("none in sight: music %q, want a fade", got)
	}
}

func TestLevelChangeFades(t *testing.T) {
	w := walkWorld(t, "###", "#@#", "###")
	w.events = nil
	if err := w.enter("level.below"); err != nil {
		t.Fatal(err)
	}
	if got := music(w.events); !slices.Equal(got, []string{""}) {
		t.Errorf("a change of level: music %q, want a fade", got)
	}
}

func TestNewGameMusic(t *testing.T) {
	p := devpack.Build()
	p.Manifest.Start.Music = devpack.MusicTune
	p.Creation.Opening = devpack.ThingSage
	w, err := NewGame(p, 1, Character{Name: "Mira", Portrait: devpack.PortraitPlayer})
	if err != nil {
		t.Fatal(err)
	}
	if got := music(w.Update(nil)); !slices.Equal(got, []string{devpack.MusicTune}) {
		t.Errorf("first update: music %q", got)
	}
	if got := music(w.Update(nil)); len(got) != 0 {
		t.Errorf("second update: music %q", got)
	}
}
