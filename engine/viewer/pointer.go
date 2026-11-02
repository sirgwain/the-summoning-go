package viewer

import (
	"image"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The pointer while playing (docs/spec/dialogue.md, The pointer): the arrow over the view, the
// speech bubble over a creature that talks or a mouth (where a click talks to it), or the object
// in hand. A mouth's message shows beside the mouth.

// The box around a thing where the pointer talks to it: talkBox pixels square, from talkInset up
// and left of where the thing is drawn.
const (
	talkBox   = 32
	talkInset = 8
	// saidUpdates is how long a mouth's message shows unless the player acts first.
	saidUpdates = 6 * sim.StepsPerSecond
)

// Said is a mouth's message shown beside it.
type Said struct {
	Text string
	// At is the view pixel the message hangs from: the top middle of the mouth.
	At    image.Point
	Until int64
}

// TalkTarget returns the id of the creature that talks, or the mouth, under screen point pt,
// and false when there is none or the pointer is not over the view.
func (v *Viewer) TalkTarget(pt image.Point) (string, bool) {
	w := v.World
	if !v.Play || w.Player() == nil || !pt.In(image.Rectangle{Max: v.Screen}) || v.covered(pt) {
		return "", false
	}
	at := v.ViewPoint(pt)
	box := func(topLeft image.Point) image.Rectangle {
		corner := topLeft.Sub(image.Pt(talkInset, talkInset))
		return image.Rectangle{corner, corner.Add(image.Pt(talkBox, talkBox))}
	}
	l := w.Current()
	// The frontmost thing first: the draw list is back to front.
	for i := len(v.Cmds) - 1; i >= 0; i-- {
		c := v.Cmds[i]
		if c.Thing == "" || c.Thing == sim.PlayerID {
			continue
		}
		if t := l.Thing(c.Thing); t != nil && w.CanTalk(t) && at.In(box(image.Pt(c.X, c.Y))) {
			return t.ID, true
		}
	}
	for _, f := range l.Features {
		if f.Kind != content.FeatureMouth || !l.Visible(f.Cell) {
			continue
		}
		if tl, ok := v.mouthTopLeft(f); ok && at.In(box(tl)) {
			return f.ID, true
		}
	}
	return "", false
}

// mouthTopLeft returns the view pixel where mouth f's look is drawn from.
func (v *Viewer) mouthTopLeft(f *sim.Feature) (image.Point, bool) {
	look := v.Pack.Looks[f.Look]
	if look == nil {
		return image.Point{}, false
	}
	parts := look.States[f.LookState()]
	if len(parts) == 0 {
		return image.Point{}, false
	}
	return drawlist.TileBox(f.Cell).Add(image.Pt(parts[0].Offset[0], parts[0].Offset[1])), true
}

// Pointer returns the image drawn at the cursor and whether it is centered on it (an object in
// hand) rather than hung from its top-left; false draws the system's pointer.
func (v *Viewer) Pointer() (img content.Image, centered, ok bool) {
	if img, ok := v.HandImage(); ok {
		return img, true, true
	}
	pt := v.Pack.Pointers
	if !v.Play || pt == nil || v.World.Player() == nil {
		return content.Image{}, false, false
	}
	if v.World.Talk() == nil && !v.SpellsOpen && pt.Talk != nil {
		if _, ok := v.TalkTarget(v.Cursor); ok {
			return *pt.Talk, false, true
		}
	}
	return pt.Arrow, false, true
}

// hear shows what the events have mouths say beside them.
func (v *Viewer) hear(events []sim.Event) {
	l := v.World.Current()
	for _, e := range events {
		if e.Kind != sim.EventMessage || !v.isMouth(e) {
			continue
		}
		f := l.Feature(e.Feature)
		at := drawlist.TileBox(f.Cell).Add(image.Pt(16, 0))
		if tl, ok := v.mouthTopLeft(f); ok {
			at = tl.Add(image.Pt(16, 0))
		}
		v.Said = &Said{Text: v.lineText(e.Text), At: at, Until: v.Ticks + saidUpdates}
	}
}

// isMouth reports whether message event e is a mouth's of the current level, shown beside it.
func (v *Viewer) isMouth(e sim.Event) bool {
	l := v.World.Current()
	if e.Feature == "" || e.Level != l.ID {
		return false
	}
	f := l.Feature(e.Feature)
	return f != nil && f.Kind == content.FeatureMouth
}
