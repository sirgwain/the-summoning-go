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

// rightClick is a right click: the object in hand goes to the first empty shelf; with an empty
// hand, a click on the player takes the top object underfoot straight to that shelf. With no
// empty shelf it does nothing.
func (v *Viewer) rightClick(onPlayer bool) []input.Command {
	p := v.World.Player()
	shelf := firstEmptyShelf(p)
	switch {
	case shelf < 0:
		return nil
	case p.Hand != nil:
		return []input.Command{input.Swap{Slot: shelf}}
	case onPlayer && len(v.World.Current().Objects[p.Cell()]) > 0:
		return []input.Command{input.PickUp{}}
	}
	return nil
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
