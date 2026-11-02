package viewer

import (
	"image"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// Playing the world's view effects and sparkles (docs/spec/scenes.md, View effects and
// Sparkles). Each effect or sparkle event waits its turn in a queue; while one plays, the world
// and the tools wait.

// Placing a flipbook, in view pixels (scenes.md, Placing the box; Sparkles).
var (
	// effectBy is a view effect box's top-left from its creature's point; effectNone where it goes
	// with no creature on screen, from the view's top-left.
	effectBy   = image.Pt(-16, -16)
	effectNone = image.Pt(100, 10)
	// effectInside is how far within the view the box stays, and effectColumns what its column is
	// rounded down to a multiple of.
	effectInside  = 8
	effectColumns = 8
	// sparkleByPlayer and sparkleByCell are a sparkle's top-left from the player's point or a
	// feature's.
	sparkleByPlayer = image.Pt(-10, -12)
	sparkleByCell   = image.Pt(-10, -20)
)

// Showing is a flipbook playing over the view.
type Showing struct {
	Run *FlipbookRun
	// At is the flipbook's top-left in view pixels. Framed draws its border.
	At     image.Point
	Framed bool
	// ms counts the time the run has been given, at whole milliseconds, and updates the updates
	// that time came from.
	ms, updates int
}

// Size is the flipbook's size in view pixels.
func (s *Showing) Size(p *content.Pack) image.Point {
	if b := p.Sprites[s.Run.Flipbook.Sprite]; b != nil {
		return image.Pt(b.FrameSize[0], b.FrameSize[1])
	}
	return image.Point{}
}

// Frame is the image showing.
func (s *Showing) Frame() content.Image {
	return content.Image{Sprite: s.Run.Flipbook.Sprite, Frame: s.Run.Frame()}
}

// queueShows queues the effect and sparkle events of the shown level.
func (v *Viewer) queueShows(events []sim.Event) {
	for _, e := range events {
		if (e.Kind == sim.EventEffect || e.Kind == sim.EventSparkle) && e.Level == v.World.Current().ID {
			v.shows = append(v.shows, e)
		}
	}
}

// Holding reports whether a flipbook plays or waits to, holding the world.
func (v *Viewer) Holding() bool { return v.Showing != nil || len(v.shows) > 0 }

// updateShowing plays the flipbooks queued: Esc or the right button cuts one short where it
// allows.
func (v *Viewer) updateShowing(in Input) {
	for v.Showing == nil && len(v.shows) > 0 {
		e := v.shows[0]
		v.shows = v.shows[1:]
		v.Showing = v.place(e)
		if v.Showing != nil {
			v.Showing.Run.Update(0, v.enterStep)
		}
	}
	s := v.Showing
	if s == nil {
		return
	}
	if in.pressed(KeyEscape) || in.Right.Pressed {
		s.Run.Cancel()
	}
	if !s.Run.Done {
		s.updates++
		ms := s.updates * 1000 / sim.StepsPerSecond
		s.Run.Update(ms-s.ms, v.enterStep)
		s.ms = ms
	}
	if s.Run.Done {
		v.Showing = nil
	}
}

func (v *Viewer) enterStep(s *content.FlipbookStep) { PlayFlipbookStep(v.Audio, s) }

// place starts the flipbook an event names where it goes, or returns nil when it plays nothing:
// a flipbook the pack lacks, or a feature's sparkle off screen.
func (v *Viewer) place(e sim.Event) *Showing {
	f := v.Pack.Flipbooks[e.Detail]
	if f == nil {
		return nil
	}
	s := &Showing{Run: NewFlipbookRun(f), Framed: f.Framed}
	view := v.viewRect()
	pt, ok := v.eventPoint(e)
	if e.Kind == sim.EventSparkle {
		by := sparkleByPlayer
		if e.Thing == "" {
			by = sparkleByCell
			if !ok || !pt.In(view) {
				return nil
			}
		}
		s.At = pt.Add(by)
		return s
	}
	at := view.Min.Add(effectNone)
	if ok && pt.In(view) {
		at = pt.Add(effectBy)
	}
	size := s.Size(v.Pack)
	rel := at.Sub(view.Min)
	rel.X = max(effectInside, min(rel.X, view.Dx()-size.X-effectInside))
	rel.Y = max(effectInside, min(rel.Y, view.Dy()-size.Y-effectInside))
	rel.X -= rel.X % effectColumns
	s.At = view.Min.Add(rel)
	return s
}

// eventPoint is the view point an event is placed by: its thing's point while the thing is on
// the level, else its cell's, or false for an event placed by nothing.
func (v *Viewer) eventPoint(e sim.Event) (image.Point, bool) {
	if e.Thing == "" && e.Feature == "" {
		return image.Point{}, false
	}
	if e.Thing == sim.PlayerID {
		if p := v.World.Player(); p != nil {
			return drawlist.RefPoint(p.U, p.V), true
		}
	} else if t := v.World.Current().Thing(e.Thing); e.Thing != "" && t != nil {
		return drawlist.RefPoint(t.U, t.V), true
	}
	return cellCenter(e.Cell), true
}

// viewRect is the part of the view the screen shows, in view pixels, above the hotbar.
func (v *Viewer) viewRect() image.Rectangle {
	size := v.BufferSize()
	if h := v.Hotbar(); h != nil {
		size.Y = min(size.Y, (h.Rect.Min.Y-Margin)/v.Scale)
	}
	return image.Rectangle{v.Origin, v.Origin.Add(size)}
}

// PlayFlipbook plays a flipbook as the tools' test of it: a framed one as a view effect, any
// other as a sparkle, placed by the selected thing, else the selected cell, else the player.
func (v *Viewer) PlayFlipbook(id string) {
	f := v.Pack.Flipbooks[id]
	if f == nil {
		return
	}
	e := sim.Event{Kind: sim.EventSparkle, Level: v.World.Current().ID, Detail: id}
	if f.Framed {
		e.Kind = sim.EventEffect
	}
	switch {
	case v.Thing != "":
		e.Thing, e.Cell = v.Thing, *v.Selected
	case v.Selected != nil:
		e.Cell, e.Feature = *v.Selected, "selection"
	case v.World.Player() != nil:
		e.Thing, e.Cell = sim.PlayerID, v.World.Player().Cell()
	}
	v.shows = append(v.shows, e)
}

// flipbookLine is the panel's line for the flipbook the tools play.
func (v *Viewer) flipbookLine() string {
	if len(v.Flipbooks) == 0 {
		return "no flipbooks"
	}
	return "B flipbook: " + strings.TrimPrefix(v.Flipbooks[v.Flipbook], content.FlipbookPrefix) + "  K play it"
}
