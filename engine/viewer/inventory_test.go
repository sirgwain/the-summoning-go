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

// TestRightClickOpens right-clicks a sack on a shelf with nothing on the pointer: it opens as the
// pack, the inventory panel opens with its boxes, and a click on the first takes its key; a
// second right click closes it.
func TestRightClickOpens(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	p := v.World.Player()
	sack := &sim.Object{Item: devpack.ItemSack, Contents: []*sim.Object{{Item: devpack.ItemKey}}}
	p.Slots[5] = sack
	r, _ := v.Hotbar().SlotRect(5)
	at := r.Min.Add(r.Max).Div(2)
	rightAt := func(pt image.Point) {
		v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: pt, Right: Button{Pressed: true, Released: true}})
	}
	rightAt(at)
	if p.Pack(v.Pack) != sack || !v.InventoryOpen {
		t.Fatalf("right click: pack %+v, panel open %v", p.Pack(v.Pack), v.InventoryOpen)
	}
	pr, ok := v.Inventory().SlotRect(sim.SlotPack)
	if !ok {
		t.Fatal("no pack boxes for the opened sack")
	}
	tapAt(v, pr.Min.Add(pr.Max).Div(2))
	if p.Hand == nil || p.Hand.Item != devpack.ItemKey || len(sack.Contents) != 0 {
		t.Fatalf("took %+v from the opened sack", p.Hand)
	}
	p.Hand = nil
	r, _ = v.Hotbar().SlotRect(5)
	rightAt(r.Min.Add(r.Max).Div(2))
	if p.Pack(v.Pack) != nil {
		t.Errorf("a second right click left %+v open", p.Pack(v.Pack))
	}
}

// TestShiftClickMoves shift-clicks a sword on a shelf with the panel closed: it goes to the right
// hand. With a sack opened as the pack, shift-clicking the sword leaves it in hand (it is too
// big for the sack), shift-clicking boots puts them in the sack, and shift-clicking them in
// the sack takes them back to the first empty shelf.
func TestShiftClickMoves(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	p := v.World.Player()
	for i := range sim.Shelves {
		p.Slots[i] = nil
	}
	sword, sack := &sim.Object{Item: devpack.ItemSword}, &sim.Object{Item: devpack.ItemSack}
	p.Slots[5], p.Slots[6], p.Slots[sim.SlotRightHand] = sword, sack, nil
	shiftAt := func(r image.Rectangle) {
		v.Update(Input{Held: map[Key]bool{KeyShift: true}, Pressed: map[Key]bool{}, Cursor: r.Min.Add(r.Max).Div(2),
			Left: Button{Pressed: true, Released: true}})
	}
	r, _ := v.Hotbar().SlotRect(5)
	shiftAt(r)
	if p.Slots[sim.SlotRightHand] != sword || p.Slots[5] != nil || p.Hand != nil {
		t.Fatalf("shift click on the shelf: right hand %+v, shelf %+v, pointer %+v", p.Slots[sim.SlotRightHand], p.Slots[5], p.Hand)
	}
	r, _ = v.Hotbar().SlotRect(6)
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: r.Min.Add(r.Max).Div(2), Right: Button{Pressed: true, Released: true}})
	if p.Pack(v.Pack) != sack || v.Inventory() == nil {
		t.Fatal("the sack did not open on the panel")
	}
	r, ok := v.Inventory().SlotRect(sim.SlotRightHand)
	if !ok {
		t.Fatal("no right hand box on the panel")
	}
	shiftAt(r)
	if p.Slots[sim.SlotRightHand] != sword || len(sack.Contents) != 0 {
		t.Fatalf("a sword too big for the sack moved: hand %+v, sack %+v", p.Slots[sim.SlotRightHand], sack.Contents)
	}
	boots := &sim.Object{Item: devpack.ItemBoots}
	p.Slots[7] = boots
	r, _ = v.Hotbar().SlotRect(7)
	shiftAt(r)
	if p.Slots[7] != nil || len(sack.Contents) != 1 || sack.Contents[0] != boots {
		t.Fatalf("shift click on the boots: shelf %+v, sack %+v", p.Slots[7], sack.Contents)
	}
	r, _ = v.Inventory().SlotRect(sim.SlotPack)
	shiftAt(r)
	if len(sack.Contents) != 0 || p.Slots[0] != boots {
		t.Fatalf("shift click in the sack: sack %+v, first shelf %+v", sack.Contents, p.Slots[0])
	}
}

// A click on the figure picks a worn slot by its region; with an object on the pointer, the slot
// the object is worn in.
func TestDollSlot(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	p := v.World.Player()
	for i := sim.Shelves; i < sim.SlotCount; i++ {
		p.Slots[i] = nil
	}
	p.Hand = nil
	for _, c := range []struct {
		col, row, want int
	}{
		{10, 10, sim.SlotQuiver}, {60, 10, sim.SlotHead}, {45, 40, sim.SlotBody},
		{40, 110, sim.SlotFeet}, {40, 90, sim.SlotBody}, {20, 60, sim.SlotBody},
	} {
		if got := v.dollSlot(p, c.col, c.row); got != c.want {
			t.Errorf("empty, (%d, %d): slot %d, want %d", c.col, c.row, got, c.want)
		}
	}
	p.Slots[sim.SlotRightHand] = &sim.Object{Item: devpack.ItemSword}
	p.Slots[sim.SlotNeck] = &sim.Object{Item: devpack.ItemRock}
	for _, c := range []struct {
		col, row, want int
	}{
		{20, 60, sim.SlotRightHand}, {20, 40, sim.SlotRightHand}, {60, 60, sim.SlotBody},
		{45, 40, sim.SlotNeck},
	} {
		if got := v.dollSlot(p, c.col, c.row); got != c.want {
			t.Errorf("holding, (%d, %d): slot %d, want %d", c.col, c.row, got, c.want)
		}
	}
	// The mail on the pointer goes to the body wherever it is clicked; the sword to a hand by the
	// side clicked.
	p.Hand = &sim.Object{Item: devpack.ItemMail}
	if got := v.dollSlot(p, 60, 10); got != sim.SlotBody {
		t.Errorf("mail on the pointer: slot %d", got)
	}
	p.Hand = &sim.Object{Item: devpack.ItemSword}
	if l, r := v.dollSlot(p, 60, 100), v.dollSlot(p, 20, 10); l != sim.SlotLeftHand || r != sim.SlotRightHand {
		t.Errorf("sword on the pointer: %d and %d", l, r)
	}
}

// The armor gauge's armor part grows with the armor class, never quite full or empty between.
func TestArmorGauge(t *testing.T) {
	for ac, want := range map[int]int{0: 0, 1: 2, 15: 21, 29: 40, 30: 41, 40: 41, -3: 0} {
		if got := armorGauge(ac); got != want {
			t.Errorf("armor class %d: %d cells, want %d", ac, got, want)
		}
	}
}
