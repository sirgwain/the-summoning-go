package viewer

import (
	"image"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The hotbar is the play view's bar along the bottom of the screen: the player's portrait, its
// gauges, its two hands with their actions, and its twenty shelves in two rows of ten. It works
// objects with the mouse as the inventory panel will, through the same slot boxes (slots.go);
// the panel adds the figure with the worn slots, the pack and the stats.

// Hotbar layout, in hotbar pixels: the art's own pixels, scaled together by the hotbar's scale.
const (
	ShelfColumns = 10
	ShelfRows    = sim.Shelves / ShelfColumns
	// SlotSize is a slot box's side: a 16-pixel icon with room around it.
	SlotSize     = 24
	portraitSize = 48
	gaugeWidth   = 6
	gap          = 4
	// actionWidth is an action button's width; two per hand, a slot high each.
	actionWidth = 40
	// hotbarWidthPer is the screen width that buys each step of hotbar scale.
	hotbarWidthPer = 600
	maxHotbarScale = 4
)

// The gauges, left to right.
const (
	GaugeHitPoints = iota
	GaugeMagicPoints
	GaugeLoad
	gaugeCount
)

// Gauge is a bar showing how full a quantity is.
type Gauge struct {
	Rect image.Rectangle
	// Value of Max; the bar fills Value / Max of its height, at most all of it.
	Value, Max int
}

// ActionButton is a button acting with a hand by one of its object's actions.
type ActionButton struct {
	Rect         image.Rectangle
	Hand, Action int
	// Label names the action; Ready is false while the hand waits its delay.
	Label string
	Ready bool
}

// Hotbar is the hotbar laid out on the screen.
type Hotbar struct {
	// Rect is the hotbar on the screen; Scale the screen pixels per hotbar pixel. Every other
	// rectangle is in hotbar pixels, from Rect's corner.
	Rect  image.Rectangle
	Scale int
	// Portrait is the player's portrait and where it goes; Mirrored draws it flipped.
	Portrait      content.Image
	PortraitRect  image.Rectangle
	Mirrored, Has bool
	Gauges        [gaugeCount]Gauge
	// Slots are the hands' boxes, right then left, and then the shelves'.
	Slots   []SlotBox
	Actions []ActionButton
}

// hotbarSize is the hotbar's size in hotbar pixels.
func hotbarSize() image.Point {
	w := portraitSize + gap + gaugeCount*(gaugeWidth+gap) + SlotSize + actionWidth*2 + gap + ShelfColumns*SlotSize
	return image.Pt(w, ShelfRows*SlotSize)
}

// Hotbar returns the hotbar as the player stands now, or nil when not playing.
func (v *Viewer) Hotbar() *Hotbar {
	p := v.World.Player()
	if !v.Play || p == nil {
		return nil
	}
	size := hotbarSize()
	s := max(1, min(v.Screen.X/hotbarWidthPer, maxHotbarScale, v.Screen.X/(size.X+2)))
	at := image.Pt((v.Screen.X-size.X*s)/2, v.Screen.Y-size.Y*s-Margin)
	h := &Hotbar{Rect: image.Rectangle{at, at.Add(size.Mul(s))}, Scale: s}

	h.PortraitRect = image.Rect(0, 0, portraitSize, portraitSize)
	if pt := v.Pack.Portraits[p.Portrait]; pt != nil {
		h.Portrait, h.Mirrored, h.Has = pt.Image, pt.Mirrored, true
	}
	x := portraitSize + gap
	limit := 10 * p.Attributes[sim.AttrStrength]
	values := [gaugeCount][2]int{
		{p.HitPoints, p.MaxHitPoints}, {p.MagicPoints, p.MaxMagicPoints}, {p.Carried(v.Pack), limit},
	}
	for i := range h.Gauges {
		h.Gauges[i] = Gauge{Rect: image.Rect(x, 0, x+gaugeWidth, size.Y), Value: values[i][0], Max: values[i][1]}
		x += gaugeWidth + gap
	}
	for hand := range 2 {
		y := hand * SlotSize
		h.Slots = append(h.Slots, SlotBox{Rect: image.Rect(x, y, x+SlotSize, y+SlotSize), Slot: sim.SlotRightHand + hand})
		labels := v.actionLabels(p, hand)
		for a, label := range labels {
			// One action takes both buttons' room.
			w := 2 * actionWidth / len(labels)
			ax := x + SlotSize + a*w
			h.Actions = append(h.Actions, ActionButton{
				Rect: image.Rect(ax, y, ax+w, y+SlotSize), Hand: hand, Action: a,
				Label: label, Ready: p.Delays[hand] == 0 && !p.Dead,
			})
		}
	}
	x += SlotSize + 2*actionWidth + gap
	for i := range sim.Shelves {
		bx, by := x+(i%ShelfColumns)*SlotSize, (i/ShelfColumns)*SlotSize
		h.Slots = append(h.Slots, SlotBox{Rect: image.Rect(bx, by, bx+SlotSize, by+SlotSize), Slot: i})
	}
	return h
}

// actionLabels names the actions of what hand holds: bare hands strike.
func (v *Viewer) actionLabels(p *sim.Player, hand int) []string {
	o := p.Slots[sim.SlotRightHand+hand]
	if o == nil {
		return []string{"strike"}
	}
	it := v.Pack.Items[o.Item]
	if it == nil {
		return nil
	}
	var labels []string
	for _, a := range it.Actions[:min(len(it.Actions), 2)] {
		labels = append(labels, map[string]string{
			content.ActionAttack: "strike", content.ActionAttackBoth: "swing", content.ActionFire: "fire",
			content.ActionUseUp: "use", content.ActionUse: "use",
		}[a.Kind])
	}
	return labels
}

// Screen returns the screen rectangle of hotbar rectangle r.
func (h *Hotbar) Screen(r image.Rectangle) image.Rectangle {
	return image.Rectangle{h.Rect.Min.Add(r.Min.Mul(h.Scale)), h.Rect.Min.Add(r.Max.Mul(h.Scale))}
}

// at returns the hotbar pixel under screen point pt.
func (h *Hotbar) at(pt image.Point) image.Point { return pt.Sub(h.Rect.Min).Div(h.Scale) }

// SlotRect returns the screen rectangle of slot's box, and false when the hotbar has none.
func (h *Hotbar) SlotRect(slot int) (image.Rectangle, bool) {
	for _, b := range h.Slots {
		if b.Slot == slot {
			return h.Screen(b.Rect), true
		}
	}
	return image.Rectangle{}, false
}

// click is a click at screen point pt on the hotbar: a slot swaps with the hand, an action acts
// (a right click on a slot does nothing more than on the view: the hand goes to a shelf).
func (h *Hotbar) click(v *Viewer, pt image.Point, right bool) []input.Command {
	if right {
		return v.rightClick(false)
	}
	at := h.at(pt)
	if b, ok := slotAt(h.Slots, at); ok {
		return v.clickSlot(b.Slot)
	}
	for _, a := range h.Actions {
		if at.In(a.Rect) {
			return []input.Command{input.Strike{Hand: a.Hand, Action: a.Action}}
		}
	}
	return nil
}

// HoverText names what is under the cursor on the hotbar: an object in a slot.
func (v *Viewer) HoverText(cursor image.Point) string {
	h := v.Hotbar()
	if h == nil || !cursor.In(h.Rect) {
		return ""
	}
	b, ok := slotAt(h.Slots, h.at(cursor))
	if !ok {
		return ""
	}
	if o := v.World.Player().Slot(v.Pack, b.Slot); o != nil {
		return inspect.ItemName(v.Pack, o.Item)
	}
	return ""
}

// SlotImage returns the icon of the object in slot, if any.
func (v *Viewer) SlotImage(slot int) (content.Image, bool) {
	o := v.World.Player().Slot(v.Pack, slot)
	if o == nil || v.Pack.Items[o.Item] == nil {
		return content.Image{}, false
	}
	return v.Pack.Items[o.Item].Icon, true
}
