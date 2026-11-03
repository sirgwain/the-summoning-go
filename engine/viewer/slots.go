package viewer

import (
	"image"
	"image/color"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// What the hotbar and the inventory panel share: boxes that show the player's slots, and how the
// mouse moves objects between the hand on the pointer, the slots and the floor
// (docs/spec/items.md, The hand and the mouse). A panel lays out its boxes; clicks on them, and on
// the player, go through here.

// SlotBox is a box on a panel showing one of the player's slots.
type SlotBox struct {
	// Rect is the box in the panel's pixels; Slot the slot it shows (sim.Slot*, or a shelf).
	Rect image.Rectangle
	Slot int
}

// slotAt returns the box of boxes holding panel pixel pt.
func slotAt(boxes []SlotBox, pt image.Point) (SlotBox, bool) {
	for _, b := range boxes {
		if pt.In(b.Rect) {
			return b, true
		}
	}
	return SlotBox{}, false
}

// clickSlot is a left click on a slot's box: the hand and the slot swap, and an object coming onto
// the pointer is read.
func (v *Viewer) clickSlot(slot int) []input.Command { return []input.Command{input.Swap{Slot: slot}} }

// clickPlayer is a left click on the player: with an empty hand it takes the top object
// underfoot onto the pointer; with an object in hand it puts it down on the player's cell, where
// a slot, socket, pit or plate takes it as the game says.
func (v *Viewer) clickPlayer() []input.Command {
	if v.World.Player().Hand != nil {
		return []input.Command{input.PutDown{}}
	}
	return []input.Command{input.PickUp{Hand: true}}
}

// rightClick is a right click: the object in hand goes to the first empty shelf, if there is
// one; with an empty hand, a click in the view takes the top object underfoot as the take key
// does, with its message when there is nothing there or no room.
func (v *Viewer) rightClick(inView bool) []input.Command {
	p := v.World.Player()
	shelf := firstEmptyShelf(p)
	switch {
	case p.Hand != nil && shelf >= 0:
		return []input.Command{input.Swap{Slot: shelf}}
	case p.Hand == nil && inView:
		return []input.Command{input.PickUp{}}
	}
	return nil
}

// openClick is a right click on a slot's box with an empty pointer: a container there opens as
// the pack, shown on the inventory panel, which opens too; clicked again it closes. It reports
// false when the click is not one.
func (v *Viewer) openClick(slot int) ([]input.Command, bool) {
	p := v.World.Player()
	if p.Hand != nil || slot < 0 || slot >= sim.SlotCount {
		return nil, false
	}
	o := p.Slots[slot]
	if o == nil || v.Pack.Items[o.Item] == nil || v.Pack.Items[o.Item].Container == nil {
		return nil, false
	}
	if s, open := p.OpenSlot(v.Pack); !open || s != slot {
		v.InventoryOpen = true
	}
	return []input.Command{input.Open{Slot: slot}}, true
}

// useClick is a right click on slot's box with nothing on the pointer: an object there that can
// be used (a use or use-up action, a potion's drink) is used where it lies (docs/spec/items.md,
// Using from a slot). It returns false for anything else.
func (v *Viewer) useClick(slot int) ([]input.Command, bool) {
	p := v.World.Player()
	if p.Hand != nil {
		return nil, false
	}
	o := p.Slot(v.Pack, slot)
	if o == nil || v.Pack.Items[o.Item] == nil {
		return nil, false
	}
	for _, a := range v.Pack.Items[o.Item].Actions {
		if a.Kind == content.ActionUse || a.Kind == content.ActionUseUp {
			return []input.Command{input.UseSlot{Slot: slot}}, true
		}
	}
	return nil, false
}

// boxRightClick is a right click on slot's box: a container opens, a usable object is used, and
// otherwise it is a right click as anywhere else.
func (v *Viewer) boxRightClick(slot int) []input.Command {
	if cmds, ok := v.openClick(slot); ok {
		return cmds
	}
	if cmds, ok := v.useClick(slot); ok {
		return cmds
	}
	return v.rightClick(false)
}

// moveClick is Shift and a left click on slot's box (docs/spec/items.md, Quick moves): with
// nothing on the pointer, a pack object goes to a shelf; with the inventory panel showing a pack,
// anything else goes into it; else a worn object goes to a shelf and a wearable one is put on.
// Anything else, or with an object on the pointer, the box's object is read.
func (v *Viewer) moveClick(slot int) []input.Command {
	p := v.World.Player()
	o := p.Slot(v.Pack, slot)
	move := func(to string) []input.Command { return []input.Command{input.MoveSlot{Slot: slot, To: to}} }
	if p.Hand != nil || o == nil {
		return []input.Command{input.Read{Slot: slot}}
	}
	switch bag := p.Pack(v.Pack); {
	case slot >= sim.SlotPack:
		return move(input.ToShelf)
	case v.Inventory() != nil && bag != nil && bag != o:
		return move(input.ToPack)
	case slot >= sim.Shelves && slot < sim.SlotCount:
		return move(input.ToShelf)
	case v.Pack.Items[o.Item] != nil && v.Pack.Items[o.Item].Wear != "":
		return move(input.ToWorn)
	}
	return []input.Command{input.Read{Slot: slot}}
}

// firstEmptyShelf returns the player's first empty shelf, or −1.
func firstEmptyShelf(p *sim.Player) int {
	for i := range sim.Shelves {
		if p.Slots[i] == nil {
			return i
		}
	}
	return -1
}

// Where a click counts as on the player (docs/spec/items.md, The mouse): within reachDistance view
// pixels, across plus down, of the point reachAt from its sprite's top-left.
var reachAt = image.Pt(8, 2)

const reachDistance = 32

// onPlayer reports whether view pixel pt is on the player: near its figure (as the game reaches
// it), on its figure, or on its cell's ground.
func (v *Viewer) onPlayer(pt image.Point) bool {
	p := v.World.Player()
	if p == nil {
		return false
	}
	if b := v.Pack.Sprites[p.Template.Sprite]; b != nil {
		anchor, _ := b.AnchorOf(0)
		d := pt.Sub(drawlist.RefPoint(p.U, p.V).Sub(image.Pt(anchor[0], anchor[1])).Add(reachAt))
		if abs(d.X)+abs(d.Y) < reachDistance {
			return true
		}
	}
	if id, _, ok := drawlist.ThingAt(v.Pack, v.Cmds, pt); ok && id == sim.PlayerID {
		return true
	}
	return drawlist.CellAt(pt) == p.Cell()
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// HandImage returns the image of the object on the pointer, if any.
func (v *Viewer) HandImage() (content.Image, bool) {
	p := v.World.Player()
	if !v.Play || p == nil || p.Hand == nil {
		return content.Image{}, false
	}
	it := v.Pack.Items[p.Hand.Item]
	if it == nil {
		return content.Image{}, false
	}
	return it.IconFor(len(p.Hand.Contents)), true
}

// FixedColors returns the colors fixed draw commands (the player's figure and what it holds) are
// looked up in: the palette's own colors, with the figure colors its portrait gives
// (docs/spec/player.md, Portrait).
func (v *Viewer) FixedColors(pal *content.Palette) [256]color.RGBA {
	c := pal.Colors
	if p := v.World.Player(); p != nil {
		if pt := v.Pack.Portraits[p.Portrait]; pt != nil {
			for i, k := range pt.FigureColors {
				if content.FigureColor+i < len(c) {
					c[content.FigureColor+i] = pal.Colors[k]
				}
			}
		}
	}
	return c
}
