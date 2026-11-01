package viewer

import (
	"image"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
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
	if left := v.Hotbar().Actions[len(v.Hotbar().Actions)-1]; left.Hand != 1 || left.Label != "strike" {
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

// TestClickNearPlayer puts down and picks up with a click near the figure, off its pixels and its
// cell's ground, as the game's reach allows; a click past the reach walks instead.
func TestClickNearPlayer(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	p := v.World.Player()
	b := v.Pack.Sprites[p.Template.Sprite]
	anchor, _ := b.AnchorOf(0)
	at := drawlist.RefPoint(p.U, p.V).Sub(image.Pt(anchor[0], anchor[1])).Add(reachAt)
	// Left of the reach point by most of the reach, and far off the player's cell.
	near := at.Add(image.Pt(-(reachDistance - 2), 0))
	if drawlist.CellAt(near) == p.Cell() {
		t.Fatalf("the near point %v is on the player's cell", near)
	}
	if id, _, ok := drawlist.ThingAt(v.Pack, v.Cmds, near); ok && id == sim.PlayerID {
		t.Fatalf("the near point %v is on the figure", near)
	}
	p.Hand = &sim.Object{Item: devpack.ItemKey}
	tapAt(v, v.ScreenPoint(near))
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: v.ScreenPoint(near)})
	if p.Hand != nil || len(v.World.Current().Objects[p.Cell()]) == 0 {
		t.Fatalf("a click near the player did not put the key down: hand %+v", p.Hand)
	}
	tapAt(v, v.ScreenPoint(near))
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: v.ScreenPoint(near)})
	if p.Hand == nil || p.Hand.Item != devpack.ItemKey {
		t.Fatalf("a click near the player did not pick the key up: hand %+v", p.Hand)
	}
	if far := at.Add(image.Pt(-reachDistance-40, 0)); v.onPlayer(far) {
		t.Errorf("a click at %v, past the reach, counts as on the player", far)
	}
}

// TestHoverContainer rests the pointer on a shelf holding a sack: the name, then what it holds,
// repeats counted; an empty sack says so, and a plain object shows its name alone.
func TestHoverContainer(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	p := v.World.Player()
	p.Slots[5] = &sim.Object{Item: devpack.ItemSack, Contents: []*sim.Object{
		{Item: devpack.ItemKey}, {Item: devpack.ItemMail}, {Item: devpack.ItemKey}}}
	r, ok := v.Hotbar().SlotRect(5)
	if !ok {
		t.Fatal("no shelf 5")
	}
	at := r.Min.Add(r.Max).Div(2)
	key, mail, sack := v.objectLines(&sim.Object{Item: devpack.ItemKey})[0], v.objectLines(&sim.Object{Item: devpack.ItemMail})[0],
		v.objectLines(&sim.Object{Item: devpack.ItemSack})[0]
	if got, want := v.HoverLines(at), []string{sack, "  " + key + " x2", "  " + mail}; !slices.Equal(got, want) {
		t.Errorf("hover %q, want %q", got, want)
	}
	p.Slots[5].Contents = nil
	if got := v.HoverLines(at); !slices.Equal(got, []string{sack, "  Empty"}) {
		t.Errorf("empty sack hover %q", got)
	}
	p.Slots[5] = &sim.Object{Item: devpack.ItemKey}
	if got := v.HoverLines(at); !slices.Equal(got, []string{key}) {
		t.Errorf("key hover %q", got)
	}
}
