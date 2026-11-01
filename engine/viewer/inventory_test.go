package viewer

import (
	"image"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// TestInventoryPanel takes mail onto the pointer, opens the panel with I, puts the mail on by clicking the body's box with the mail
// on the pointer, and fills the pack of a bag in the left hand by clicking its first box; the
// figure shows the mail, and the details describe what is under the pointer.
func TestInventoryPanel(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	p := v.World.Player()
	c := p.Cell()
	v.World.Update([]input.Command{input.Drop{Cell: c, Item: devpack.ItemMail}})
	if err := NewRunner(mustScript(t, `[
		{"at": `+itoa(v.Ticks)+`, "click_player": true},
		{"at": `+itoa(v.Ticks+1)+`, "keys": ["I"]},
		{"at": `+itoa(v.Ticks+2)+`, "slot": 26}
	]`), t.TempDir()).Run(v); err != nil {
		t.Fatal(err)
	}
	ip := v.Inventory()
	if ip == nil {
		t.Fatal("I did not open the panel")
	}
	if o := p.Slots[sim.SlotBody]; o == nil || o.Item != devpack.ItemMail || p.Hand != nil {
		t.Fatalf("the body holds %+v, the hand %+v", o, p.Hand)
	}
	if p.ArmorClass < 4 {
		t.Errorf("armor class %d with the mail on", p.ArmorClass)
	}
	if !slices.ContainsFunc(ip.Parts, func(d DollPart) bool { return d.Image.Frame == 1 }) {
		t.Errorf("the figure does not show the mail: %+v", ip.Parts)
	}
	// Hovering the body's box describes the mail.
	r, _ := ip.SlotRect(sim.SlotBody)
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: r.Min.Add(r.Max).Div(2)})
	if !hasLine(v.Inventory(), "Chain Mail") || !hasLine(v.Inventory(), "Armor 4") {
		t.Errorf("details: %+v", v.Inventory().Lines)
	}

	// A bag in the left hand opens the pack; its first box takes what is on the pointer.
	p.Slots[sim.SlotLeftHand] = &sim.Object{Item: devpack.ItemSack}
	p.Hand = &sim.Object{Item: devpack.ItemKey}
	ip = v.Inventory()
	pr, ok := ip.SlotRect(sim.SlotPack)
	if !ok {
		t.Fatal("no pack boxes with a bag in the left hand")
	}
	tapAt(v, pr.Min.Add(pr.Max).Div(2))
	if bag := p.Slots[sim.SlotLeftHand]; len(bag.Contents) != 1 || bag.Contents[0].Item != devpack.ItemKey || p.Hand != nil {
		t.Errorf("the pack holds %+v, the hand %+v", bag.Contents, p.Hand)
	}
	// The stats show the attributes and the skills' ratings.
	if !hasLine(ip, "STR:18") || !hasLine(ip, "Beginner 1") {
		t.Errorf("stats: %+v", ip.Lines)
	}
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{KeyEscape: true}, Cursor: image.Pt(-1, -1)})
	if v.Inventory() != nil {
		t.Error("Escape did not close the panel")
	}
}

// TestHandActionBoxes puts the sword in the right hand: its hand shows three boxes, its two
// actions with their icons and throwing, and the third throws it.
func TestHandActionBoxes(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	p := v.World.Player()
	p.Slots[sim.SlotRightHand] = &sim.Object{Item: devpack.ItemSword, Value: 100}
	var right []ActionButton
	for _, a := range v.Hotbar().Actions {
		if a.Hand == 0 {
			right = append(right, a)
		}
	}
	if len(right) != 3 || right[0].Icon == nil || right[1].Icon == nil || right[2].Action != ThrowAction || right[2].Icon == nil {
		t.Fatalf("the right hand's boxes %+v", right)
	}
	if left := v.Hotbar().Actions[len(v.Hotbar().Actions)-1]; left.Hand != 1 || left.Label != "strike" || left.Picture == nil {
		t.Errorf("the empty left hand's box %+v", left)
	}
	if err := NewRunner(mustScript(t, `[{"at": `+itoa(v.Ticks)+`, "action": [0, 2]}]`), t.TempDir()).Run(v); err != nil {
		t.Fatal(err)
	}
	if p.Slots[sim.SlotRightHand] != nil {
		t.Error("the throw box did not throw the sword")
	}
}

func hasLine(ip *InventoryPanel, text string) bool {
	return slices.ContainsFunc(ip.Lines, func(l TextLine) bool { return strings.Contains(l.Text, text) })
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
