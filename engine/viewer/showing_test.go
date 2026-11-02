package viewer

import (
	"image"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// showingViewer is a viewer of the dev pack with a framed flipbook "flipbook.t.box" and an
// unframed "flipbook.t.sparkle", each a cancellable 100 ms step and a plain 100 ms one.
func showingViewer(t *testing.T) *Viewer {
	t.Helper()
	v := newViewer(t)
	sprite := v.Pack.Things[v.Pack.Manifest.Player].Sprite
	for id, framed := range map[string]bool{"flipbook.t.box": true, "flipbook.t.sparkle": false} {
		v.Pack.Flipbooks[id] = &content.Flipbook{ID: id, Sprite: sprite, Framed: framed,
			Steps: []content.FlipbookStep{{Frame: 0, MS: 100, Cancel: true}, {Frame: 1, MS: 100}}}
	}
	v.Flipbooks = sortedIDs(v.Pack.Flipbooks)
	return v
}

// The world waits while a flipbook plays, and goes on when it ends.
func TestShowingHoldsTheWorld(t *testing.T) {
	v := showingViewer(t)
	v.queueShows([]sim.Event{{Kind: sim.EventEffect, Level: v.World.Current().ID, Detail: "flipbook.t.box"}})
	step := v.World.Step()
	updates := 0
	for v.Holding() {
		v.Update(Input{})
		if updates++; updates > 100 {
			t.Fatal("the flipbook never ended")
		}
	}
	if v.World.Step() != step {
		t.Errorf("the world ran %d steps during the flipbook", v.World.Step()-step)
	}
	// 200 ms at 60 updates a second.
	if updates != 12 {
		t.Errorf("played for %d updates, want 12", updates)
	}
	v.Update(Input{})
	if v.World.Step() != step+1 {
		t.Error("the world did not go on after the flipbook")
	}
}

func TestShowingCancels(t *testing.T) {
	v := showingViewer(t)
	esc := Input{Pressed: map[Key]bool{KeyEscape: true}}
	v.PlayFlipbook("flipbook.t.box")
	v.Update(Input{})
	v.Update(esc)
	if v.Holding() {
		t.Fatal("Esc did not cut the flipbook short in its cancellable step")
	}
	v.PlayFlipbook("flipbook.t.box")
	for range 7 {
		v.Update(Input{})
	}
	if v.Showing == nil || v.Showing.Run.Step != 1 {
		t.Fatal("not in the second step")
	}
	v.Update(esc)
	if !v.Holding() {
		t.Error("Esc cut short a step that is not cancellable")
	}
}

func TestShowingPlacement(t *testing.T) {
	v := showingViewer(t)
	level := v.World.Current().ID
	start := func(e sim.Event) *Showing {
		t.Helper()
		v.Showing, v.shows = nil, nil
		e.Level = level
		v.queueShows([]sim.Event{e})
		v.Update(Input{})
		return v.Showing
	}
	size := v.Pack.Sprites[v.Pack.Flipbooks["flipbook.t.box"].Sprite].FrameSize

	// No creature: row 10, column 100 of the view, the column rounded down to a multiple of 8.
	if s := start(sim.Event{Kind: sim.EventEffect, Detail: "flipbook.t.box"}); s == nil || s.At != v.Origin.Add(image.Pt(96, 10)) || !s.Framed {
		t.Errorf("no creature: %+v", s)
	}
	// By a cell near the view's top-left: 16 up and left of it, kept 8 inside the view.
	c := content.Cell{2, 2}
	v.Origin = cellCenter(c).Sub(image.Pt(3, 3))
	if s := start(sim.Event{Kind: sim.EventEffect, Detail: "flipbook.t.box", Feature: "f", Cell: c}); s == nil || s.At != v.Origin.Add(image.Pt(8, 8)) {
		t.Errorf("by the top-left: %+v, want %v", s, v.Origin.Add(image.Pt(8, 8)))
	}
	// Near the bottom-right, kept 8 inside.
	view := v.viewRect()
	v.Origin = cellCenter(c).Sub(view.Size()).Add(image.Pt(3, 3))
	view = v.viewRect()
	want := image.Pt(view.Dx()-size[0]-8, view.Dy()-size[1]-8)
	want.X -= want.X % 8
	if s := start(sim.Event{Kind: sim.EventEffect, Detail: "flipbook.t.box", Feature: "f", Cell: c}); s == nil || s.At != v.Origin.Add(want) {
		t.Errorf("by the bottom-right: %+v, want %v", s, v.Origin.Add(want))
	}

	// A player's sparkle: 12 rows up and 10 columns left of its point.
	p := v.World.Player()
	if s := start(sim.Event{Kind: sim.EventSparkle, Detail: "flipbook.t.sparkle", Thing: sim.PlayerID}); s == nil ||
		s.At != drawlist.RefPoint(p.U, p.V).Add(image.Pt(-10, -12)) || s.Framed {
		t.Errorf("player sparkle: %+v", s)
	}
	// A feature's: 20 rows up, and only when on screen.
	v.Recenter()
	c = p.Cell()
	if s := start(sim.Event{Kind: sim.EventSparkle, Detail: "flipbook.t.sparkle", Feature: "door", Cell: c}); s == nil ||
		s.At != cellCenter(c).Add(image.Pt(-10, -20)) {
		t.Errorf("door sparkle: %+v", s)
	}
	v.Origin = cellCenter(c).Add(image.Pt(1000, 1000))
	if s := start(sim.Event{Kind: sim.EventSparkle, Detail: "flipbook.t.sparkle", Feature: "door", Cell: c}); s != nil || v.Holding() {
		t.Errorf("a sparkle off screen played: %+v", s)
	}
	// Another level's events are not shown.
	v.Showing, v.shows = nil, nil
	v.queueShows([]sim.Event{{Kind: sim.EventEffect, Level: "level.other", Detail: "flipbook.t.box"}})
	if v.Holding() {
		t.Error("queued another level's effect")
	}
}

// The tools' keys choose a flipbook and play it.
func TestFlipbookKeys(t *testing.T) {
	v := showingViewer(t)
	v.Update(Input{Pressed: map[Key]bool{"B": true}})
	if v.Flipbooks[v.Flipbook] == "" {
		t.Fatal("no flipbook chosen")
	}
	v.Update(Input{Pressed: map[Key]bool{"K": true}})
	if !v.Holding() {
		t.Error("K did not play the flipbook")
	}
}
