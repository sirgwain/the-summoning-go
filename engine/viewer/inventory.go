package viewer

import (
	"fmt"
	"image"
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The inventory panel (docs/spec/items.md, The inventory panel): the figure with what it wears
// and a box for each worn slot beside it, the pack of the bag in the left hand, the details of the
// object under the pointer or on it, and the character's stats. Its boxes work as the hotbar's do,
// through the same slot code (slots.go).

// Inventory panel layout, in panel pixels, scaled together like the hotbar's.
const (
	invPad    = 6
	dollW     = 96
	dollH     = 128
	invGap    = 4
	invColumn = 8
	// invTextW is the width of the details and stats columns; detailLines the details' lines.
	invTextW    = 40 * CharWidth
	detailLines = 5
)

// wornLeft and wornRight are the worn slots in the boxes left and right of the figure, top down:
// the armor on the left, and on the right the quiver, the amulet and the hands, in the hotbar's
// order (right hand above left).
var (
	wornLeft  = []int{sim.SlotHead, sim.SlotBody, sim.SlotArms, sim.SlotFeet}
	wornRight = []int{sim.SlotQuiver, sim.SlotNeck, sim.SlotRightHand, sim.SlotLeftHand}
)

// DollPart is an image drawn on the figure, mirrored for the right hand.
type DollPart struct {
	Image    content.Image
	At       image.Point
	Mirrored bool
}

// TextLine is a line of text on a panel, with an icon before it if it has one.
type TextLine struct {
	At   image.Point
	Text string
	Icon *content.Image
}

// InventoryPanel is the inventory panel laid out on the screen. Rect is on the screen and Scale
// its screen pixels per panel pixel; every other rectangle and point is in panel pixels.
type InventoryPanel struct {
	Rect  image.Rectangle
	Scale int
	// Doll is the figure, drawn at DollAt, with Parts over it, if HasDoll.
	Doll    content.Image
	DollAt  image.Point
	HasDoll bool
	Parts   []DollPart
	// Slots are the worn slots' boxes and the pack's.
	Slots []SlotBox
	Lines []TextLine
}

// ToggleInventory opens or closes the inventory panel, when playing.
func (v *Viewer) ToggleInventory() {
	v.InventoryOpen = !v.InventoryOpen && v.Play && v.World.Player() != nil
}

// Inventory returns the inventory panel as it is now, or nil when it is closed.
func (v *Viewer) Inventory() *InventoryPanel {
	p := v.World.Player()
	if !v.InventoryOpen || !v.Play || p == nil {
		return nil
	}
	inv := v.Pack.Inventory
	if inv == nil {
		inv = &content.Inventory{}
	}
	ip := &InventoryPanel{}
	// The figure between two columns of worn slots.
	dollX := invPad + SlotSize + invGap
	ip.DollAt = image.Pt(dollX, invPad)
	if inv.Doll != nil {
		ip.Doll, ip.HasDoll = *inv.Doll, true
		ip.Parts = v.dollParts(p, inv)
	}
	step := dollH / len(wornLeft)
	for i, slot := range wornLeft {
		y := invPad + i*step + (step-SlotSize)/2
		ip.Slots = append(ip.Slots, SlotBox{Rect: image.Rect(invPad, y, invPad+SlotSize, y+SlotSize), Slot: slot})
	}
	rx := dollX + dollW + invGap
	for i, slot := range wornRight {
		y := invPad + i*step + (step-SlotSize)/2
		ip.Slots = append(ip.Slots, SlotBox{Rect: image.Rect(rx, y, rx+SlotSize, y+SlotSize), Slot: slot})
	}
	// The pack, two rows, then the details, beside the figure.
	x := rx + SlotSize + 2*invGap
	y := invPad
	if bag := p.Pack(v.Pack); bag != nil {
		box := v.Pack.Items[bag.Item].Container
		ip.Lines = append(ip.Lines, TextLine{At: image.Pt(x, y), Text: inspect.ItemName(v.Pack, bag.Item)})
		y += LineHeight
		cols := (box.Capacity + 1) / 2
		for k := range box.Capacity {
			bx, by := x+(k%cols)*SlotSize, y+(k/cols)*SlotSize
			ip.Slots = append(ip.Slots, SlotBox{Rect: image.Rect(bx, by, bx+SlotSize, by+SlotSize), Slot: sim.SlotPack + k})
		}
		y += 2*SlotSize + invGap
	}
	detailsAt := image.Pt(x, y)
	y += detailLines * LineHeight
	right := max(x+invTextW, x+invColumn*SlotSize)
	// The stats under the figure, across the panel.
	y = max(y, invPad+dollH) + invGap
	stats := v.stats(p, inv)
	for _, l := range stats {
		l.At = l.At.Add(image.Pt(invPad, y))
		ip.Lines = append(ip.Lines, l)
	}
	if n := len(stats); n > 0 {
		y += stats[n-1].At.Y + LineHeight
	}
	size := image.Pt(right+invPad, y+invPad)
	bottom := v.Screen.Y - Margin
	if h := v.Hotbar(); h != nil {
		bottom = h.Rect.Min.Y - Margin
		ip.Scale = h.Scale
	}
	ip.Scale = max(1, min(ip.Scale, (v.Screen.X-2*Margin)/size.X, max(bottom-Margin, 1)/size.Y))
	corner := image.Pt((v.Screen.X-size.X*ip.Scale)/2, bottom-size.Y*ip.Scale)
	ip.Rect = image.Rectangle{corner, corner.Add(size.Mul(ip.Scale))}
	// The details last: they describe the box under the pointer, which needs the panel placed.
	o := v.described(ip)
	for i, s := range v.details(o) {
		if i < detailLines {
			ip.Lines = append(ip.Lines, TextLine{At: detailsAt.Add(image.Pt(0, i*LineHeight)), Text: s})
		}
	}
	return ip
}

// Screen returns the screen rectangle of panel rectangle r.
func (ip *InventoryPanel) Screen(r image.Rectangle) image.Rectangle {
	return image.Rectangle{ip.Rect.Min.Add(r.Min.Mul(ip.Scale)), ip.Rect.Min.Add(r.Max.Mul(ip.Scale))}
}

// at returns the panel pixel under screen point pt.
func (ip *InventoryPanel) at(pt image.Point) image.Point { return pt.Sub(ip.Rect.Min).Div(ip.Scale) }

// SlotRect returns the screen rectangle of slot's box, and false when the panel has none.
func (ip *InventoryPanel) SlotRect(slot int) (image.Rectangle, bool) {
	for _, b := range ip.Slots {
		if b.Slot == slot {
			return ip.Screen(b.Rect), true
		}
	}
	return image.Rectangle{}, false
}

// dollParts are what the figure wears, in the order they are drawn: quiver, body, head, feet,
// neck, arms, then the hands, those with armor last (docs/spec/items.md, The inventory panel).
func (v *Viewer) dollParts(p *sim.Player, inv *content.Inventory) []DollPart {
	var parts []DollPart
	add := func(slot int, armor bool) {
		o := p.Slots[slot]
		if o == nil {
			return
		}
		it := v.Pack.Items[o.Item]
		if it == nil || (slot == sim.SlotRightHand || slot == sim.SlotLeftHand) && (it.Armor > 0) != armor {
			return
		}
		look := it.DollFor(len(o.Contents))
		if slot == sim.SlotQuiver && inv.QuiverSlot != nil {
			look = inv.QuiverSlot
		}
		if look == nil {
			return
		}
		at := image.Pt(look.At[0], look.At[1])
		mirrored := slot == sim.SlotRightHand
		if mirrored {
			if b := v.Pack.Sprites[look.Image.Sprite]; b != nil {
				at.X = dollW - at.X - b.FrameSize[0]
			}
		}
		parts = append(parts, DollPart{Image: look.Image, At: at, Mirrored: mirrored})
	}
	for _, s := range []int{sim.SlotQuiver, sim.SlotBody, sim.SlotHead, sim.SlotFeet, sim.SlotNeck, sim.SlotArms} {
		add(s, false)
	}
	for _, armor := range []bool{false, true} {
		add(sim.SlotRightHand, armor)
		add(sim.SlotLeftHand, armor)
	}
	return parts
}

// described returns the object the panel describes: the one under the pointer on a panel, or
// else the one on the pointer; nil for neither.
func (v *Viewer) described(ip *InventoryPanel) *sim.Object {
	if o := v.objectUnder(ip, v.Cursor); o != nil {
		return o
	}
	return v.World.Player().Hand
}

// details are the lines describing object o, the one the panel describes; with none, the
// character: its name, title and load.
func (v *Viewer) details(o *sim.Object) []string {
	p := v.World.Player()
	if o == nil {
		return []string{v.playerName(), v.levelTitle(p.Level), inspect.Label(v.Pack, content.LabelWeight, "Weight: ") + inspect.Weight(p.Carried(v.Pack))}
	}
	it := v.Pack.Items[o.Item]
	if it == nil {
		return nil
	}
	lines := []string{inspect.ItemName(v.Pack, o.Item), inspect.Label(v.Pack, content.LabelWeight, "Weight: ") + inspect.Weight(o.Weight(v.Pack))}
	if slices.Contains(it.Tags, "weapon") {
		most := 0
		for _, a := range it.Actions {
			most = max(most, a.Damage.Max())
		}
		if most > 1 {
			lines = append(lines, inspect.Label(v.Pack, content.LabelDamage, "Can do ")+fmt.Sprint(most)+inspect.Label(v.Pack, content.LabelDamageAfter, " damage points"))
		}
	}
	if it.Armor > 0 {
		armor := fmt.Sprint(it.Armor)
		if it.Wear == content.WearBody && it.Durability > 0 && o.Value < it.Durability {
			eff := max(it.Armor-(it.Durability-o.Value)/500, it.Armor/2)
			armor = fmt.Sprintf("%d (%d)", eff, it.Armor)
		}
		lines = append(lines, inspect.Label(v.Pack, content.LabelArmor, "AC bonus of ")+armor)
	}
	if slices.Contains(it.Tags, "two_handed") {
		lines = append(lines, inspect.Label(v.Pack, content.LabelTwoHanded, "Requires two hands"))
	}
	return lines
}

// levelTitle returns the title of the player's level, or "level n" without one.
func (v *Viewer) levelTitle(level int) string {
	if inv := v.Pack.Inventory; inv != nil && level < len(inv.LevelTitles) {
		return inspect.Text(v.Pack, inv.LevelTitles[level])
	}
	return fmt.Sprintf("level %d", level)
}

// stats are the character's lines: attributes (with the maximum after one that differs from it),
// hit and magic points, armor, level and experience, and each skill with a level, with its icon
// and rating.
func (v *Viewer) stats(p *sim.Player, inv *content.Inventory) []TextLine {
	short := []string{"STR", "AGI", "END", "ACC", "TAL", "POW"}
	names := []string{sim.AttrStrength, sim.AttrAgility, sim.AttrEndurance, sim.AttrAccuracy, sim.AttrTalent, sim.AttrPower}
	attrs := ""
	for i, n := range names {
		s := fmt.Sprintf("%s:%d", short[i], p.Attributes[n])
		if m, ok := p.MaxAttributes[n]; ok && m != p.Attributes[n] {
			s += fmt.Sprintf("/%d", m)
		}
		attrs += s + "  "
	}
	lines := []TextLine{
		{Text: attrs},
		{Text: fmt.Sprintf("HP %d/%d  MP %d/%d  AC %d  %s  EXP %d/%d", p.HitPoints, p.MaxHitPoints, p.MagicPoints, p.MaxMagicPoints,
			p.ArmorClass, v.levelTitle(p.Level), p.Experience, p.NextLevel), At: image.Pt(0, LineHeight)},
	}
	// The skills with a level, two to a line.
	n := 0
	for i, sk := range p.Skills {
		if sk.Level == 0 {
			continue
		}
		icons, ratings, k := inv.WeaponIcons, inv.WeaponRatings, i
		if i >= sim.SkillSchool {
			icons, ratings, k = inv.SchoolIcons, inv.SchoolRatings, i-sim.SkillSchool
		}
		text := inspect.SkillName(fmt.Sprint(i))
		if r := min(sk.Level, len(ratings)); r > 0 {
			text = inspect.Text(v.Pack, ratings[r-1])
		}
		l := TextLine{Text: fmt.Sprintf("%s %d (%d/%d)", text, sk.Level, sk.Practice, sk.Threshold),
			At: image.Pt((n%2)*(invTextW/2+3*SlotSize), (2+n/2)*LineHeight)}
		if k < len(icons) {
			l.Icon = &icons[k]
		}
		lines = append(lines, l)
		n++
	}
	return lines
}

// objectUnder returns the object in the box of the panel ip (if open) or the hotbar under screen
// point pt.
func (v *Viewer) objectUnder(ip *InventoryPanel, pt image.Point) *sim.Object {
	p := v.World.Player()
	if ip != nil && pt.In(ip.Rect) {
		if b, ok := slotAt(ip.Slots, ip.at(pt)); ok {
			return p.Slot(v.Pack, b.Slot)
		}
	}
	if h := v.Hotbar(); h != nil && pt.In(h.Rect) {
		if b, ok := slotAt(h.Slots, h.at(pt)); ok {
			return p.Slot(v.Pack, b.Slot)
		}
	}
	return nil
}

// inventoryClick is a click on the inventory panel at screen point pt: a box swaps with the hand
// (with Shift, reads its object); the right button opens a container or uses a usable object in
// a box, or sends the hand to a shelf.
func (v *Viewer) inventoryClick(ip *InventoryPanel, pt image.Point, right, shift bool) []input.Command {
	b, ok := slotAt(ip.Slots, ip.at(pt))
	if right {
		if ok {
			return v.boxRightClick(b.Slot)
		}
		return v.rightClick(false)
	}
	if !ok {
		return nil
	}
	if shift {
		return []input.Command{input.Read{Slot: b.Slot}}
	}
	return v.clickSlot(b.Slot)
}
