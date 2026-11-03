package viewer

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// transitionViewer is a playing viewer of the dev pack whose teleporter cue irises for 200 ms
// and whose lost objects shatter for 100 ms.
func transitionViewer(t *testing.T) *Viewer {
	t.Helper()
	v := newViewer(t)
	v.SetPlay(true)
	if v.Pack.Effects == nil {
		v.Pack.Effects = &content.EffectCues{ID: "effects"}
	}
	v.Pack.Effects.Transitions = map[string]*content.Transition{
		content.TransitionTeleporter: {Style: content.StyleIris, MS: 200, HoldMS: 100},
		content.TransitionFall:       {Style: content.StyleCut, MS: 200},
	}
	v.Pack.Effects.ItemLost = &content.LostItem{Style: content.LostShatter, MS: 100}
	return v
}

// A transition holds the world while it plays, and the world goes on when it ends.
func TestTransitionHoldsTheWorld(t *testing.T) {
	v := transitionViewer(t)
	v.took([]sim.Event{{Kind: sim.EventTransition, Level: v.Shown, Detail: content.TransitionTeleporter}})
	if v.Transition == nil || v.Transition.Seq != 1 {
		t.Fatalf("no transition started: %+v", v.Transition)
	}
	step := v.World.Step()
	updates := 0
	for v.Holding() {
		v.Update(Input{})
		if updates++; updates > 100 {
			t.Fatal("the transition never ended")
		}
	}
	if updates != 12 || v.World.Step() != step {
		t.Errorf("played %d updates (want 12), the world ran %d steps", updates, v.World.Step()-step)
	}
	// A cut, or a cue the pack leaves out, shows the new view at once.
	for _, cue := range []string{content.TransitionFall, content.TransitionRipple} {
		v.took([]sim.Event{{Kind: sim.EventTransition, Level: v.Shown, Detail: cue}})
		if v.Transition != nil {
			t.Errorf("%s started a transition", cue)
		}
	}
}

func TestTransitionFrames(t *testing.T) {
	size := image.Pt(100, 60)
	at := func(d content.Transition, ms int) TransitionFrame {
		tr := &Transition{Def: &d, From: image.Pt(10, 10), To: image.Pt(90, 50), ms: ms}
		return tr.Frame(size, 2)
	}
	// Iris: 1000 ms, 200 held closed, so it closes over 400 and opens over 400.
	iris := content.Transition{Style: content.StyleIris, MS: 1000, HoldMS: 200, Color: "#102030"}
	full := math.Hypot(90, 50)
	for _, tc := range []struct {
		ms     int
		new    bool
		center image.Point
		radius float64
	}{
		{0, false, image.Pt(10, 10), full},
		{200, false, image.Pt(10, 10), full / 2},
		{500, false, image.Pt(10, 10), 0},
		{800, true, image.Pt(90, 50), full / 2},
		{1000, true, image.Pt(90, 50), full},
	} {
		f := at(iris, tc.ms)
		if f.Panels || f.New != tc.new || f.Center != tc.center || math.Abs(f.Radius-tc.radius) > 1e-9 {
			t.Errorf("iris at %d ms: %+v, want new %v center %v radius %.1f", tc.ms, f, tc.new, tc.center, tc.radius)
		}
		if f.Color != (color.RGBA{0x10, 0x20, 0x30, 255}) {
			t.Errorf("iris color %v", f.Color)
		}
	}

	// Slide from the right, halfway: the new view's left half fills the screen's right half.
	f := at(content.Transition{Style: content.StyleSlide, MS: 100}, 50)
	if !f.Panels || f.NewDst != image.Rect(50, 0, 100, 60) || f.NewSrc != image.Rect(0, 0, 50, 60) ||
		f.OldDst != image.Rect(0, 0, 50, 60) || f.OldSrc != image.Rect(50, 0, 100, 60) {
		t.Errorf("slide: %+v", f)
	}
	// Squeeze from the top in 5 steps: at 40% it has taken 2 steps of 6, the views whole.
	f = at(content.Transition{Style: content.StyleSqueeze, From: content.SideTop, MS: 100, Steps: 5}, 40)
	if !f.Panels || f.NewDst != image.Rect(0, 0, 100, 20) || f.OldDst != image.Rect(0, 20, 100, 60) ||
		f.NewSrc != image.Rect(0, 0, 100, 60) || f.OldSrc != image.Rect(0, 0, 100, 60) {
		t.Errorf("squeeze: %+v", f)
	}
	// Ripple: largest at the middle, where the new view takes over, in screen pixels.
	f = at(content.Transition{Style: content.StyleRipple, MS: 100, Size: 4, Rows: 10}, 50)
	if !f.New || math.Abs(f.Shift-8) > 1e-9 || f.Rows != 20 {
		t.Errorf("ripple at the middle: %+v", f)
	}
	if f = at(content.Transition{Style: content.StyleRipple, MS: 100}, 25); f.New {
		t.Error("ripple shows the new view before the middle")
	}
	// Fade: all color at the middle.
	if f = at(content.Transition{Style: content.StyleFade, MS: 100}, 50); f.Amount != 1 || !f.New {
		t.Errorf("fade at the middle: %+v", f)
	}
	if f = at(content.Transition{Style: content.StyleFade, MS: 100}, 25); f.Amount != 0.5 || f.New {
		t.Errorf("fade at a quarter: %+v", f)
	}
	// Curtains of 100 ms holding 20: halfway closing (eased) the color covers a quarter of the
	// screen each side over the old view; held, nothing shows; halfway opening, the new view between.
	curtains := content.Transition{Style: content.StyleCurtains, MS: 100, HoldMS: 20}
	f = at(curtains, 20)
	if !f.Panels || f.OldDst != image.Rect(25, 0, 75, 60) || f.OldSrc != f.OldDst || !f.NewDst.Empty() {
		t.Errorf("curtains closing: %+v", f)
	}
	if f = at(curtains, 50); !f.OldDst.Empty() || !f.NewDst.Empty() {
		t.Errorf("curtains held: %+v", f)
	}
	if f = at(curtains, 80); f.NewDst != image.Rect(25, 0, 75, 60) || !f.OldDst.Empty() {
		t.Errorf("curtains opening: %+v", f)
	}
}

// An object lost from a slot shows going in its hotbar box, and over the player's head when the
// hotbar has none, until its time is up.
func TestLostItems(t *testing.T) {
	v := transitionViewer(t)
	var item string
	for id, it := range v.Pack.Items {
		if it.Icon.Sprite != "" {
			item = id
			break
		}
	}
	if item == "" {
		t.Skip("the dev pack has no item with an icon")
	}
	v.took([]sim.Event{
		{Kind: sim.EventObject, Item: item, Detail: "worn_out", Slot: sim.SlotRightHand},
		{Kind: sim.EventObject, Item: item, Detail: "worn_out", Slot: sim.SlotNeck},
	})
	lost := v.LostItems()
	if len(lost) != 2 {
		t.Fatalf("%d lost items showing, want 2", len(lost))
	}
	want, _ := v.Hotbar().SlotRect(sim.SlotRightHand)
	if lost[0].Rect != want || lost[0].Style != content.LostShatter {
		t.Errorf("the right hand's object goes at %v, want %v", lost[0].Rect, want)
	}
	if mid := v.playerMiddle(); lost[1].Rect.Max.Y > mid.Y || lost[1].Rect.Min.X > mid.X || lost[1].Rect.Max.X < mid.X {
		t.Errorf("the neck's object goes at %v, not over the player at %v", lost[1].Rect, mid)
	}
	for range 6 {
		v.Update(Input{})
	}
	if n := len(v.LostItems()); n != 0 {
		t.Errorf("%d lost items still showing after their time", n)
	}
}

// With inventory, a lost object opens the inventory panel and holds the world while it goes
// there, and the panel closes again after.
func TestLostOnInventory(t *testing.T) {
	v := transitionViewer(t)
	v.Pack.Effects.ItemLost.Inventory = true
	item := ""
	for id, it := range v.Pack.Items {
		if it.Icon.Sprite != "" {
			item = id
			break
		}
	}
	if item == "" {
		t.Skip("the dev pack has no item with an icon")
	}
	v.took([]sim.Event{{Kind: sim.EventObject, Item: item, Detail: "worn_out", Slot: sim.SlotNeck}})
	if !v.InventoryOpen || !v.Holding() {
		t.Fatalf("inventory open %v, holding %v", v.InventoryOpen, v.Holding())
	}
	lost := v.LostItems()
	ip := v.Inventory()
	want := image.Rectangle{}
	for _, b := range ip.Slots {
		if b.Slot == sim.SlotNeck {
			want = ip.Screen(b.Rect)
		}
	}
	if len(lost) != 1 || lost[0].Rect != want || lost[0].Scale != ip.Scale {
		t.Errorf("lost %+v, want the neck's box %v on the panel", lost, want)
	}
	step := v.World.Step()
	for range 5 {
		v.Update(Input{})
	}
	if !v.Holding() || v.World.Step() != step {
		t.Errorf("going: holding %v, world ran %d steps", v.Holding(), v.World.Step()-step)
	}
	v.Update(Input{})
	if v.InventoryOpen || v.Holding() {
		t.Errorf("after: inventory open %v, holding %v", v.InventoryOpen, v.Holding())
	}
	// A panel already open stays open.
	v.InventoryOpen = true
	v.took([]sim.Event{{Kind: sim.EventObject, Item: item, Detail: "worn_out", Slot: sim.SlotNeck}})
	for range 7 {
		v.Update(Input{})
	}
	if !v.InventoryOpen {
		t.Error("the panel the player had open closed")
	}
}
