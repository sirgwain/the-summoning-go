package viewer

import (
	"image"
	"image/color"
	"math"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// Transitions and lost objects (docs/spec/scenes.md, Level transitions; docs/pack.md, Effects).
// A teleport or a change of level the pack gives a transition shows the old view giving way to
// the new by its style, holding the world while it plays. An object that breaks or runs out in
// one of the player's slots shows going where the slot is drawn. The viewer works out what each
// frame shows; the window draws it.

// Transition is a transition playing: the old view, as last drawn, giving way to the new.
type Transition struct {
	Def *content.Transition
	// From and To are the player's middle on the screen in the old view and in the new.
	From, To image.Point
	// Seq counts the transitions started, so the window knows a new one from the last.
	Seq int
	// ms is the time shown, from updates.
	ms, updates int
}

// Done reports whether the transition has shown all its time.
func (t *Transition) Done() bool { return t.ms >= t.Def.MS }

// Progress is how far the transition has gone, 0 to 1, in its steps when it moves in steps.
func (t *Transition) Progress() float64 {
	p := min(1, float64(t.ms)/float64(t.Def.MS))
	if n := t.Def.Steps; n > 0 && p < 1 {
		p = math.Floor(p*float64(n+1)) / float64(n+1)
	}
	return p
}

// TransitionFrame is what one frame of a transition shows over the screen.
type TransitionFrame struct {
	// With Panels, the old view and the new are each drawn from their Src part, in screen
	// pixels of that view, stretched over their Dst part of the screen; the rest is Color.
	Panels                         bool
	OldDst, OldSrc, NewDst, NewSrc image.Rectangle
	// Otherwise one view shows: the new one when New, else the old. Its rows shift sideways by up
	// to Shift pixels, in a zigzag each of whose zags is Rows rows; Amount of Color lies over it;
	// and outside the circle of Radius about Center, when Radius is not negative, it is Color.
	New         bool
	Shift, Rows float64
	Amount      float64
	Center      image.Point
	Radius      float64
	Color       color.RGBA
}

// Ripple defaults, in view pixels.
const (
	rippleSize = 8
	rippleRows = 16
)

// Frame is what the transition shows now on a screen of size, at scale screen pixels per view
// pixel.
func (t *Transition) Frame(size image.Point, scale int) TransitionFrame {
	d := t.Def
	p := t.Progress()
	f := TransitionFrame{Color: d.RGBA(), Radius: -1}
	full := image.Rectangle{Max: size}
	switch d.Style {
	case content.StyleSlide, content.StyleSqueeze:
		f.Panels = true
		from := d.From
		if from == "" {
			from = content.SideRight
			if d.Style == content.StyleSqueeze {
				from = content.SideBottom
			}
		}
		across := size.X
		if from == content.SideTop || from == content.SideBottom {
			across = size.Y
		}
		in := int(math.Round(p * float64(across)))
		f.NewDst, f.OldDst, f.NewSrc, f.OldSrc = split(full, from, in)
		if d.Style == content.StyleSqueeze {
			f.NewSrc, f.OldSrc = full, full
		}
	case content.StyleIris:
		hold := float64(d.HoldMS) / float64(d.MS)
		half := (1 - hold) / 2
		f.Radius = 0
		switch {
		case p < half:
			f.Center = t.From
			f.Radius = reach(t.From, size) * (1 - p/half)
		case p >= half+hold:
			f.New, f.Center = true, t.To
			f.Radius = reach(t.To, size) * min(1, (p-half-hold)/half)
		default:
			f.Center = t.From
		}
	case content.StyleRipple:
		f.New = p >= 0.5
		size, rows := d.Size, d.Rows
		if size == 0 {
			size = rippleSize
		}
		if rows == 0 {
			rows = rippleRows
		}
		f.Shift = float64(size*scale) * math.Sin(math.Pi*p)
		f.Rows = float64(rows * scale)
	case content.StyleFade:
		f.New = p >= 0.5
		f.Amount = 1 - math.Abs(2*p-1)
	case content.StyleCurtains:
		f.Panels = true
		hold := float64(d.HoldMS) / float64(d.MS)
		half := (1 - hold) / 2
		// How far each side has closed across its half of the screen, 0 to 1.
		shut := 1.0
		switch {
		case p < half:
			shut = smooth(p / half)
		case p >= half+hold:
			shut = 1 - smooth(min(1, (p-half-hold)/half))
		}
		w := size.X
		in := int(math.Round(shut * float64(w) / 2))
		gap := image.Rect(in, 0, w-in, size.Y)
		if p < half {
			f.OldDst, f.OldSrc = gap, gap
		} else {
			f.NewDst, f.NewSrc = gap, gap
		}
	default:
		f.New = true
	}
	return f
}

// smooth eases x, 0 to 1, in and out.
func smooth(x float64) float64 { return x * x * (3 - 2*x) }

// split cuts r where a view coming in from side has come in by in pixels: the parts of the
// screen the new view and the old take, and the parts of each view that show unstretched.
func split(r image.Rectangle, side string, in int) (newDst, oldDst, newSrc, oldSrc image.Rectangle) {
	w, h := r.Dx(), r.Dy()
	switch side {
	case content.SideLeft:
		return image.Rect(0, 0, in, h), image.Rect(in, 0, w, h), image.Rect(w-in, 0, w, h), image.Rect(0, 0, w-in, h)
	case content.SideTop:
		return image.Rect(0, 0, w, in), image.Rect(0, in, w, h), image.Rect(0, h-in, w, h), image.Rect(0, 0, w, h-in)
	case content.SideBottom:
		return image.Rect(0, h-in, w, h), image.Rect(0, 0, w, h-in), image.Rect(0, 0, w, in), image.Rect(0, in, w, h)
	}
	return image.Rect(w-in, 0, w, h), image.Rect(0, 0, w-in, h), image.Rect(0, 0, in, h), image.Rect(in, 0, w, h)
}

// reach is how far from pt the screen's farthest corner is.
func reach(pt, size image.Point) float64 {
	dx := math.Max(float64(pt.X), float64(size.X-pt.X))
	dy := math.Max(float64(pt.Y), float64(size.Y-pt.Y))
	return math.Hypot(dx, dy)
}

// Transitioning reports whether a transition plays.
func (v *Viewer) Transitioning() bool { return v.Transition != nil }

// startTransition starts the transition of the first transition event of events the pack shows,
// from where the player was in the view as last drawn, from.
func (v *Viewer) startTransition(events []sim.Event, from image.Point) {
	if !v.Play {
		return
	}
	for _, e := range events {
		if e.Kind != sim.EventTransition {
			continue
		}
		d := v.Pack.Transition(e.Detail)
		if d == nil {
			return
		}
		v.transitions++
		v.Transition = &Transition{Def: d, From: from, To: v.playerMiddle(), Seq: v.transitions}
		return
	}
}

// updateTransition advances the transition playing by an update, ending it when it is done.
func (v *Viewer) updateTransition() {
	t := v.Transition
	t.updates++
	t.ms = t.updates * 1000 / sim.StepsPerSecond
	if t.Done() {
		v.Transition = nil
	}
}

// playerMiddle is the middle of the player's sprite on the screen.
func (v *Viewer) playerMiddle() image.Point {
	p := v.World.Player()
	if p == nil {
		return v.Screen.Div(2)
	}
	return v.ScreenPoint(v.spriteMiddle(p.Template, v.thingPoint(p.Template, p.U, p.V)))
}

// spriteMiddle is the middle of template tmpl's sprite box whose top-left is at.
func (v *Viewer) spriteMiddle(tmpl *content.Thing, at image.Point) image.Point {
	if tmpl != nil {
		if sp := v.Pack.Sprites[tmpl.Sprite]; sp != nil {
			return at.Add(image.Pt(sp.FrameSize[0]/2, sp.FrameSize[1]/2))
		}
	}
	return at
}

// lostItem is an object lost from one of the player's slots, still showing going.
type lostItem struct {
	image content.Image
	slot  int
	// at is the update it went on.
	at int64
}

// LostView is a lost object showing going: its image in a box on the screen, drawn at Scale
// screen pixels per art pixel, Age of the way through its style.
type LostView struct {
	Image content.Image
	Rect  image.Rectangle
	Scale int
	Age   float64
	Style string
}

// loseItems notes the objects events say broke or ran out in the player's slots.
func (v *Viewer) loseItems(events []sim.Event) {
	if !v.Play || v.Pack.Lost() == nil {
		return
	}
	for _, e := range events {
		if e.Kind != sim.EventObject || e.Detail != "worn_out" {
			continue
		}
		if it := v.Pack.Items[e.Item]; it != nil {
			v.lost = append(v.lost, lostItem{image: it.IconFor(0), slot: e.Slot, at: v.Ticks})
			if v.Pack.Lost().Inventory && !v.InventoryOpen {
				v.InventoryOpen, v.lostOpened = true, true
			}
		}
	}
}

// lostHolds reports whether a lost object going on the inventory panel holds the world.
func (v *Viewer) lostHolds() bool {
	def := v.Pack.Lost()
	return def != nil && def.Inventory && len(v.lost) > 0
}

// lostAge is how far lost object l has gone, 0 to 1 and beyond.
func (v *Viewer) lostAge(l lostItem, def *content.LostItem) float64 {
	return float64(v.Ticks-l.at) * 1000 / sim.StepsPerSecond / float64(def.MS)
}

// dropLost forgets the lost objects that have finished going.
func (v *Viewer) dropLost() {
	def := v.Pack.Lost()
	out := v.lost[:0]
	for _, l := range v.lost {
		if def != nil && v.lostAge(l, def) < 1 {
			out = append(out, l)
		}
	}
	v.lost = out
	if len(v.lost) == 0 && v.lostOpened {
		v.InventoryOpen, v.lostOpened = false, false
	}
}

// LostItems are the lost objects showing going: in the slot's box on the inventory panel while
// it is open, else on the hotbar, else over the player's head.
func (v *Viewer) LostItems() []LostView {
	def := v.Pack.Lost()
	if def == nil || len(v.lost) == 0 || !v.Play {
		return nil
	}
	h, ip := v.Hotbar(), v.Inventory()
	var out []LostView
	for _, l := range v.lost {
		lv := LostView{Image: l.image, Age: v.lostAge(l, def), Style: def.Style, Scale: v.Scale}
		if lv.Age >= 1 {
			continue
		}
		placed := false
		if ip != nil {
			for _, b := range ip.Slots {
				if b.Slot == l.slot {
					lv.Rect, lv.Scale, placed = ip.Screen(b.Rect), ip.Scale, true
				}
			}
		}
		if !placed && h != nil {
			lv.Rect, placed = h.SlotRect(l.slot)
			lv.Scale = h.Scale
		}
		if !placed {
			if h != nil {
				lv.Scale = h.Scale
			}
			side := SlotSize * lv.Scale
			p := v.World.Player()
			top := v.ScreenPoint(v.thingPoint(p.Template, p.U, p.V))
			mid := v.playerMiddle()
			lv.Rect = image.Rect(mid.X-side/2, top.Y-side, mid.X+side/2, top.Y)
		}
		out = append(out, lv)
	}
	return out
}
