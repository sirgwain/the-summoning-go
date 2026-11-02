package devpack

import (
	"image"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/sim"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// shown is the flipbook of the first effect or sparkle event, and its kind.
func shown(events []sim.Event) (kind, id string) {
	for _, e := range events {
		if e.Kind == sim.EventEffect || e.Kind == sim.EventSparkle {
			return e.Kind, e.Detail
		}
	}
	return "", ""
}

// TestEffectsPlay sets off each of the dev pack's view effects and sparkles.
func TestEffectsPlay(t *testing.T) { forPacks(t, testEffectsPlay) }

func testEffectsPlay(t *testing.T, build func() *content.Pack) {
	walkInto := func(from content.Cell, d int) func(w *sim.World) []sim.Event {
		return func(w *sim.World) []sim.Event {
			w.Update([]input.Command{input.PlacePlayer{Cell: from}})
			var all []sim.Event
			for range 60 {
				all = append(all, w.Update([]input.Command{input.Walk{Direction: d}})...)
				if k, _ := shown(all); k != "" {
					break
				}
			}
			return all
		}
	}
	for _, c := range []struct {
		name, level string
		do          func(w *sim.World) []sim.Event
		kind, want  string
	}{
		{"pit, stepping down", Hall, walkInto(content.Cell{Pit[0], Pit[1] - 1}, 2), sim.EventEffect, FlipbookClimbDown},
		{"pit, falling", Hall, walkInto(content.Cell{Pit[0], Pit[1] + 1}, 6), sim.EventEffect, FlipbookFall},
		{"exit D", Hall, func(w *sim.World) []sim.Event {
			return w.Update([]input.Command{input.PlacePlayer{Cell: ExitD}})
		}, sim.EventEffect, FlipbookThrough},
		{"socket", Hall, func(w *sim.World) []sim.Event {
			w.Update([]input.Command{input.Drop{Cell: Socket, Item: ItemRedGem}})
			return w.Update([]input.Command{input.Drop{Cell: Socket, Item: ItemBlueGem}})
		}, sim.EventEffect, FlipbookSocket},
		{"sealed door", Hall, func(w *sim.World) []sim.Event {
			return w.Update([]input.Command{input.Drop{Cell: SealedDoor, Item: ItemSphere}})
		}, sim.EventSparkle, SparkleSeal},
		{"heal", Study, func(w *sim.World) []sim.Event {
			w.Player().Magic.Icons = "ABCDEFGHIJKL"
			w.Player().HitPoints = 1
			w.Update([]input.Command{input.Learn{Gesture: "FHBEHL"}})
			all := w.Update([]input.Command{input.Cast{School: 3}})
			for range 300 {
				if k, _ := shown(all); k != "" {
					break
				}
				all = append(all, w.Update(nil)...)
			}
			return all
		}, sim.EventSparkle, SparkleHeal},
		{"the sage on fireballs", Study, func(w *sim.World) []sim.Event {
			w.Update([]input.Command{input.PlacePlayer{Cell: content.Cell{SageCell[0] + 1, SageCell[1]}}})
			var all []sim.Event
			for _, cmd := range []input.Command{input.Talk{Thing: "sage"}, input.Continue{Ask: "magic"},
				input.Continue{Ask: "fireball"}} {
				all = append(all, w.Update([]input.Command{cmd})...)
			}
			for range 10 {
				if k, _ := shown(all); k != "" {
					break
				}
				all = append(all, w.Update([]input.Command{input.Continue{}})...)
			}
			return all
		}, sim.EventEffect, FlipbookTalk},
		{"the target dies", Study, func(w *sim.World) []sim.Event {
			w.Current().Thing("target").HitPoints = 1
			w.Update([]input.Command{input.PlacePlayer{Cell: StudyStart}})
			w.Update([]input.Command{input.Learn{Gesture: "DJ"}})
			w.Player().Facing = 2
			var all []sim.Event
			for range 300 {
				all = append(all, w.Update([]input.Command{input.Cast{School: 0}})...)
				if k, _ := shown(all); k != "" {
					break
				}
			}
			return all
		}, sim.EventEffect, FlipbookCaster},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, err := sim.New(build(), 1)
			if err != nil {
				t.Fatal(err)
			}
			w.Update([]input.Command{input.GoTo{Level: c.level}})
			kind, id := shown(c.do(w))
			if kind != c.kind || id != c.want {
				t.Errorf("played %s %q, want %s %q", kind, id, c.kind, c.want)
			}
		})
	}
}

// TestPitEffectInViewer steps into the hall's pit through the viewer: the climb-down flipbook
// plays with the world held, and the cellar follows when it ends.
func TestPitEffectInViewer(t *testing.T) { forPacks(t, testPitEffectInViewer) }

func testPitEffectInViewer(t *testing.T, build func() *content.Pack) {
	v, err := viewer.New(build(), 1, Hall, 2, false, image.Pt(1280, 800))
	if err != nil {
		t.Fatal(err)
	}
	v.Play = true
	v.World.Update([]input.Command{input.PlacePlayer{Cell: content.Cell{Pit[0], Pit[1] - 1}}})
	var played string
	for range 600 {
		in := viewer.Input{}
		if v.Showing == nil && played == "" {
			in.Held = map[viewer.Key]bool{viewer.KeyPageDown: true} // down and right: direction 2
		}
		v.Update(in)
		if v.Showing != nil {
			played = v.Showing.Run.Flipbook.ID
			if v.World.Current().ID != Hall {
				t.Fatal("the level changed while the effect played")
			}
		}
		if played != "" && !v.Holding() && v.World.Current().ID == Cellar {
			break
		}
	}
	if played != FlipbookClimbDown || v.World.Current().ID != Cellar {
		t.Errorf("played %q, now in %s", played, v.World.Current().ID)
	}
}
