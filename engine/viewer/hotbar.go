package viewer

import (
	"fmt"
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
	// actionBoxes is how many action boxes each hand has, a slot square each.
	actionBoxes = 3
	// gestureSize is the gesture box's side: two slots high.
	gestureSize = 2 * SlotSize
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

// ActionButton is a box acting with a hand by one of its object's actions, or throwing it
// (Action ThrowAction).
type ActionButton struct {
	Rect         image.Rectangle
	Hand, Action int
	// Label names the action; Ready is false while the hand waits its delay, and Wait is how
	// much of the delay is still to run, from 1 just after acting to 0.
	Label string
	Ready bool
	Wait  float64
	// Icon is the action's icon, when the pack has one; the hand's box beside the actions shows
	// the object.
	Icon *content.Image
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
	// Spells are the schools' boxes, each showing its selected spell; Gesture shows the gesture
	// playing, and opens the spell panel.
	Spells  [content.Schools]SpellBox
	Gesture GestureBox
}

// SpellBox is a school's box on the hotbar: its selected spell and how often it is memorized.
type SpellBox struct {
	Rect   image.Rectangle
	School int
	// Icon is the selected spell's icon, if Has; Count its memorizations. Wait is how much of
	// the gesture playing is still to run before another spell can be cast, 0 with none.
	Icon  content.Image
	Has   bool
	Count int
	Wait  float64
}

// GestureBox shows the icon of the gesture playing, or the empty panel.
type GestureBox struct {
	Rect image.Rectangle
	Icon content.Image
	Has  bool
}

// hotbarSize is the hotbar's size in hotbar pixels.
func hotbarSize() image.Point {
	w := portraitSize + gap + gaugeCount*(gaugeWidth+gap) + SlotSize + actionBoxes*SlotSize + gap + ShelfColumns*SlotSize +
		gap + 2*SlotSize + gap + gestureSize
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
		for _, b := range v.actionButtons(p, hand) {
			ax := x + SlotSize + b.Action*SlotSize
			b.Rect = image.Rect(ax, y, ax+SlotSize, y+SlotSize)
			h.Actions = append(h.Actions, b)
		}
	}
	x += SlotSize + actionBoxes*SlotSize + gap
	for i := range sim.Shelves {
		bx, by := x+(i%ShelfColumns)*SlotSize, (i/ShelfColumns)*SlotSize
		h.Slots = append(h.Slots, SlotBox{Rect: image.Rect(bx, by, bx+SlotSize, by+SlotSize), Slot: i})
	}
	x += ShelfColumns*SlotSize + gap
	for s := range content.Schools {
		bx, by := x+(s/2)*SlotSize, (s%2)*SlotSize
		b := SpellBox{Rect: image.Rect(bx, by, bx+SlotSize, by+SlotSize), School: s, Wait: v.World.GestureWait()}
		if sp := v.Pack.Spells[p.Magic.Selected[s]]; sp != nil && sp.Icon != nil {
			b.Icon, b.Has, b.Count = *sp.Icon, true, p.Magic.Memorized[sp.ID]
		}
		h.Spells[s] = b
	}
	x += 2*SlotSize + gap
	h.Gesture.Rect = image.Rect(x, 0, x+gestureSize, gestureSize)
	if c, ok := p.Magic.Icon(v.Pack); ok && v.Pack.Magic != nil && int(c-'A') < len(v.Pack.Magic.Icons) {
		h.Gesture.Icon, h.Gesture.Has = v.Pack.Magic.Icons[c-'A'], true
	} else if m := v.Pack.Magic; m != nil && m.Panel != nil {
		h.Gesture.Icon, h.Gesture.Has = *m.Panel, true
	}
	return h
}

// ThrowAction is the action number of a hand's throw box.
const ThrowAction = 2

// actionButtons are the action boxes of what hand holds (docs/spec/items.md, The inventory
// panel): its actions, then throwing; bare hands only strike.
func (v *Viewer) actionButtons(p *sim.Player, hand int) []ActionButton {
	inv := v.Pack.Inventory
	if inv == nil {
		inv = &content.Inventory{}
	}
	ready, wait := p.Delays[hand] == 0 && !p.Dead, v.World.HandWait(hand)
	o := p.Slots[sim.SlotRightHand+hand]
	if o == nil {
		return []ActionButton{{Hand: hand, Action: 0, Label: "strike", Ready: ready, Wait: wait, Icon: inv.StrikeIcon}}
	}
	it := v.Pack.Items[o.Item]
	if it == nil {
		return nil
	}
	var out []ActionButton
	for a, act := range it.Actions[:min(len(it.Actions), 2)] {
		out = append(out, ActionButton{Hand: hand, Action: a, Ready: ready, Wait: wait, Icon: act.Icon,
			Label: map[string]string{
				content.ActionAttack: "strike", content.ActionAttackBoth: "swing", content.ActionFire: "fire",
				content.ActionUseUp: "use", content.ActionUse: "use",
			}[act.Kind]})
	}
	return append(out, ActionButton{Hand: hand, Action: ThrowAction, Label: "throw", Ready: ready, Wait: wait, Icon: inv.ThrowIcon})
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
		for _, b := range h.Spells {
			if h.at(pt).In(b.Rect) {
				return []input.Command{input.NextSpell{School: b.School}}
			}
		}
		if b, ok := slotAt(h.Slots, h.at(pt)); ok {
			if cmds, ok := v.openClick(b.Slot); ok {
				return cmds
			}
		}
		return v.rightClick(false)
	}
	at := h.at(pt)
	if b, ok := slotAt(h.Slots, at); ok {
		return v.clickSlot(b.Slot)
	}
	for _, b := range h.Spells {
		if at.In(b.Rect) {
			return []input.Command{input.Cast{School: b.School}}
		}
	}
	if at.In(h.Gesture.Rect) {
		v.OpenSpells()
		return nil
	}
	for _, a := range h.Actions {
		if at.In(a.Rect) {
			if a.Action == ThrowAction {
				return []input.Command{input.Throw{Hand: a.Hand}}
			}
			return []input.Command{input.Strike{Hand: a.Hand, Action: a.Action}}
		}
	}
	return nil
}

// HoverLines describe what is under the cursor on the hotbar or the inventory panel: the name of
// an object in a box, and for a container what it holds (docs/spec/items.md, The inventory panel).
func (v *Viewer) HoverLines(cursor image.Point) []string {
	if o := v.objectUnder(v.Inventory(), cursor); o != nil {
		return v.objectLines(o)
	}
	h := v.Hotbar()
	if h == nil || !cursor.In(h.Rect) {
		return nil
	}
	for _, s := range h.Spells {
		if h.at(cursor).In(s.Rect) {
			if sp := v.Pack.Spells[v.World.Player().Magic.Selected[s.School]]; sp != nil {
				return []string{inspect.Text(v.Pack, sp.Name)}
			}
		}
	}
	return nil
}

// objectLines are object o's name and, for a container, its contents by name, repeats counted,
// or "Empty".
func (v *Viewer) objectLines(o *sim.Object) []string {
	lines := []string{inspect.ItemName(v.Pack, o.Item)}
	if it := v.Pack.Items[o.Item]; it == nil || it.Container == nil {
		return lines
	}
	if len(o.Contents) == 0 {
		return append(lines, "  Empty")
	}
	var names []string
	count := map[string]int{}
	for _, c := range o.Contents {
		n := inspect.ItemName(v.Pack, c.Item)
		if count[n] == 0 {
			names = append(names, n)
		}
		count[n]++
	}
	for _, n := range names {
		if count[n] > 1 {
			n = fmt.Sprintf("%s x%d", n, count[n])
		}
		lines = append(lines, "  "+n)
	}
	return lines
}

// SlotImage returns the icon of the object in slot, if any.
func (v *Viewer) SlotImage(slot int) (content.Image, bool) {
	o := v.World.Player().Slot(v.Pack, slot)
	if o == nil || v.Pack.Items[o.Item] == nil {
		return content.Image{}, false
	}
	return v.Pack.Items[o.Item].IconFor(len(o.Contents)), true
}
