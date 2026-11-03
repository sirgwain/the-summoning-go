package viewer

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"unicode"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The book's window (docs/spec/player.md, The book): a tab each for creatures and items, the
// entries found down the left by name, and the chosen one's picture and what the player has
// learned of it on the right.

// Book tabs.
const (
	BookCreatures = iota
	BookItems
)

// Book window layout, in window pixels, which the window scales by up to talkScale.
const (
	bookWinWidth     = 640
	bookWinMinHeight = 300
	bookWinPad       = 8
	bookWinColumn    = 200
	bookWinRow       = SlotSize + 2
	bookWinTab       = 120
	bookWinButton    = 60
	bookWinField     = 220
	bookWinToggle    = 90
	// BookPictureScale is how many times their size the pane's pictures show; the list's icons
	// show at their own.
	BookPictureScale = 2
)

// Hints for what the next stage would show.
const (
	hintFight  = "Fight one to learn more."
	hintDefeat = "Defeat one to learn more."
	hintPickUp = "Pick one up to learn more."
	hintUse    = "Use one to learn more."
	hintWear   = "Wear one to learn more."
)

// BookView is the book laid out on the screen. Rect is on the screen and Scale its screen pixels
// per window pixel; every other rectangle and point is in window pixels from Rect's corner.
type BookView struct {
	Rect  image.Rectangle
	Scale int
	// Tabs are the Creatures and Items tabs, Tab the one open; Field is the filter field and
	// Typed what is typed in it; Close the button that closes. Full is the tools' full book, whose
	// Toggle switches between the current level's entries and all of them, as All tells.
	Tabs   [2]PanelButton
	Tab    int
	Field  image.Rectangle
	Typed  string
	Close  PanelButton
	Full   bool
	All    bool
	Toggle PanelButton
	// Rows are the entries that show down the left; First is the index of the first among them
	// all, Count how many the tab has, and Selected the chosen row among those shown, -1 for
	// none.
	Rows     []BookRow
	First    int
	Count    int
	Selected int
	// Pane is the right pane. Picture shows at PictureAt when HasPicture, at BookPictureScale
	// times its size as all the pane's pictures do; Lines are its text,
	// laid out below the picture.
	Pane       image.Rectangle
	Picture    content.Image
	HasPicture bool
	PictureAt  image.Point
	Lines      []BookLine
	// Empty is what the pane says when the tab has no entries.
	Empty string
}

// BookRow is an entry down the left: its name, beside its icon for an item. InLevel marks, in
// the full book, an entry on the current level.
type BookRow struct {
	Rect    image.Rectangle
	Label   string
	Icon    content.Image
	HasIcon bool
	InLevel bool
}

// BookLine is a line of the pane: the entry's name as a heading, a line of what is known, or the
// hint of what the next stage would show. In the full book a line may show pictures below its
// words, and a click on it may select a thing or a cell of the current level.
type BookLine struct {
	Words    []Word
	Heading  bool
	Hint     bool
	Pictures []BookPicture
	Thing    string
	Cell     *content.Cell
}

// BookPicture is a picture of the pane, its top-left corner at At.
type BookPicture struct {
	Image content.Image
	At    image.Point
}

// bookEntry is an entry of a tab: its template and the name it is listed by, and in the full
// book whether it is on the current level.
type bookEntry struct {
	id, name string
	inLevel  bool
}

// OpenBook opens the book, when playing, on the tab and entry last chosen.
func (v *Viewer) OpenBook() {
	if !v.Play || v.World.Player() == nil {
		return
	}
	v.BookOpen, v.BookFull, v.BookTyped, v.SpellsOpen = true, false, "", false
}

// CloseBook closes the book.
func (v *Viewer) CloseBook() { v.BookOpen, v.BookFull, v.BookTyped = false, false, "" }

// bookEntries returns tab's entries whose names hold the filter typed, ignoring case, by name.
func (v *Viewer) bookEntries(tab int) []bookEntry {
	b := v.World.Book()
	filter := strings.ToLower(strings.TrimSpace(v.BookTyped))
	var out []bookEntry
	add := func(id, name string) {
		if filter == "" || strings.Contains(strings.ToLower(name), filter) {
			out = append(out, bookEntry{id: id, name: name})
		}
	}
	if v.BookFull {
		for _, e := range v.fullBookEntries(tab) {
			if filter == "" || strings.Contains(strings.ToLower(e.name), filter) || strings.Contains(e.id, filter) {
				out = append(out, e)
			}
		}
	} else if tab == BookCreatures {
		for id := range b.Creatures {
			add(id, v.logName(id))
		}
	} else {
		for id := range b.Items {
			add(id, inspect.ItemName(v.Pack, id))
		}
	}
	slices.SortFunc(out, func(a, b bookEntry) int {
		if c := strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name)); c != 0 {
			return c
		}
		return strings.Compare(a.id, b.id)
	})
	return out
}

// bookChosen returns the index of the chosen entry of the open tab, the first when none is.
func (v *Viewer) bookChosen(entries []bookEntry) int {
	i := slices.IndexFunc(entries, func(e bookEntry) bool { return e.id == v.bookView.chosen[v.bookView.tab] })
	return max(i, 0)
}

// bookPicture returns creature t's picture: the first frame its animate shows, or the first of
// its idle animation for its starting facing.
func (v *Viewer) bookPicture(t *content.Thing) content.Image {
	img := content.Image{Sprite: t.Sprite}
	if t.Animate != nil && len(t.Animate.Frames) > 0 {
		img.Frame = t.Animate.Frames[0]
	} else if a, ok := v.Pack.Animation(t, fmt.Sprintf("idle.%d", t.Facing), "idle"); ok && len(a.Frames) > 0 {
		img.Frame = a.Frames[0]
	}
	return img
}

// levelName returns level id's name, or its id without one.
func (v *Viewer) levelName(id string) string {
	if l := v.Pack.Levels[id]; l != nil && l.Name != "" {
		return inspect.Text(v.Pack, l.Name)
	}
	return id
}

// weaponClassName returns weapon class n's name: the creation record's, or else the name of the
// weapon the class starts with.
func (v *Viewer) weaponClassName(n int) string {
	if c := v.Pack.Creation; c != nil && n >= 0 && n < len(c.WeaponClasses) {
		tr := c.WeaponClasses[n]
		if tr.Name != "" {
			return inspect.Text(v.Pack, tr.Name)
		}
		for _, o := range tr.Inventory {
			if it := v.Pack.Items[o.Item]; it != nil && it.WeaponClass != nil && *it.WeaponClass == n {
				return inspect.ItemName(v.Pack, o.Item)
			}
		}
	}
	return fmt.Sprintf("weapon class %d", n)
}

// spellName returns spell id's name.
func (v *Viewer) spellName(id string) string {
	if sp := v.Pack.Spells[id]; sp != nil && sp.Name != "" {
		return inspect.Text(v.Pack, sp.Name)
	}
	return id
}

// firstSeen says where and when entry e was first come across; the full book's entries the
// player has not come across have no level.
func (v *Viewer) firstSeen(e *sim.Entry) string {
	if e.Level == "" {
		return "Not come across yet."
	}
	return "First seen: " + v.levelName(e.Level) + ", " + LogTime(e.Step)
}

// CreatureLines returns what the book says of creature template id, as the entry e stands, and
// the hint of what the next stage would show, none for a creature that can be talked to.
func (v *Viewer) CreatureLines(id string, e *sim.Entry) (lines []string, hint string) {
	t := v.Pack.Things[id]
	if t == nil || e == nil {
		return nil, ""
	}
	talks := t.Has(content.BehaviorTalks)
	next := func(h string) string {
		if talks {
			return ""
		}
		return h
	}
	if t.Lore != "" {
		lines = append(lines, inspect.Text(v.Pack, t.Lore))
	}
	if talks {
		lines = append(lines, "Can be talked to.")
	}
	lines = append(lines, v.firstSeen(e))
	if e.Stage < sim.StageFought {
		return lines, next(hintFight)
	}
	if t.Attack != (content.Dice{}) {
		lines = append(lines, "Strikes for "+t.Attack.String()+".")
	}
	for _, b := range []struct{ behavior, text string }{
		{content.BehaviorPoisonous, "Its blows poison."},
		{content.BehaviorHurtsOnContact, "Touching it hurts."},
		{content.BehaviorDeathBlow, "One blow from it can kill."},
		{content.BehaviorMirrors, "Hurting it hurts you as much."},
	} {
		if t.Has(b.behavior) {
			lines = append(lines, b.text)
		}
	}
	if t.Spell != "" {
		lines = append(lines, "Casts "+v.spellName(t.Spell)+".")
	}
	for _, d := range e.Defenses {
		switch d {
		case sim.DefenseVulnerable:
			if t.Vulnerability != nil {
				lines = append(lines, "Vulnerable to: "+v.weaponClassName(*t.Vulnerability))
			}
		case content.BehaviorResistsFire:
			lines = append(lines, "Resists fire.")
		case content.BehaviorResistsLight:
			lines = append(lines, "Resists lightning.")
		case content.BehaviorResistsPoison:
			lines = append(lines, "Resists poison.")
		case content.BehaviorResistsTeleport:
			lines = append(lines, "Resists teleporting.")
		case content.BehaviorPhysicalOnly:
			lines = append(lines, "Magic does not hurt it, and of weapons only silver does.")
		}
	}
	if e.Stage < sim.StageKilled {
		return lines, next(hintDefeat)
	}
	if t.HitPoints != (content.Dice{}) {
		lines = append(lines, "Hit points: "+t.HitPoints.String())
	}
	if t.Experience != (content.Dice{}) {
		lines = append(lines, "Worth "+t.Experience.String()+" experience.")
	}
	lines = append(lines, fmt.Sprintf("Defeated: %d", e.Kills))
	if len(e.Drops) > 0 {
		names := make([]string, len(e.Drops))
		for i, d := range e.Drops {
			names[i] = inspect.ItemName(v.Pack, d)
		}
		lines = append(lines, "Drops: "+strings.Join(names, ", "))
	}
	return lines, ""
}

// wornOn says where an item of wear w is worn, "" for a hand.
var wornOn = map[string]string{
	content.WearHead: "Worn on the head.", content.WearBody: "Worn on the body.", content.WearFeet: "Worn on the feet.",
	content.WearArms: "Worn on the arms.", content.WearNeck: "Worn about the neck.", content.WearQuiver: "Worn as a quiver.",
}

// ItemLines returns what the book says of item id, as the entry e stands, and the hint of what
// the next stage would show.
func (v *Viewer) ItemLines(id string, e *sim.Entry) (lines []string, hint string) {
	it := v.Pack.Items[id]
	if it == nil || e == nil {
		return nil, ""
	}
	if it.Lore != "" {
		lines = append(lines, inspect.Text(v.Pack, it.Lore))
	}
	lines = append(lines, v.firstSeen(e))
	if it.Teaches != "" {
		return append(lines, v.scrollLines(it.Teaches)...), ""
	}
	if e.Stage < sim.StageHandled {
		return lines, hintPickUp
	}
	lines = append(lines, v.label(content.LabelWeight, "Weight: ")+weight(it.Weight))
	lines = append(lines, strikeLines(it)...)
	if it.Armor > 0 {
		lines = append(lines, v.label(content.LabelArmor, "AC bonus of ")+fmt.Sprint(it.Armor))
	}
	if s := wornOn[it.Wear]; s != "" {
		lines = append(lines, s)
	}
	if hasTag(it, "two_handed") {
		lines = append(lines, v.label(content.LabelTwoHanded, "Requires two hands"))
	}
	if c := it.Container; c != nil {
		lines = append(lines, fmt.Sprintf("Holds %d objects, of size up to %d.", c.Capacity, c.Largest))
		if c.Weightless {
			lines = append(lines, "What it holds weighs nothing.")
		}
	}
	if it.Charges > 1 {
		lines = append(lines, fmt.Sprintf("Charges: %d", it.Charges))
	}
	if e.Stage < sim.StageUsed {
		return lines, useHint(it)
	}
	if it.Effect != "" {
		lines = append(lines, inspect.Text(v.Pack, it.Effect))
	}
	if it.Casts != nil {
		lines = append(lines, "Casts "+v.spellName(it.Casts.Spell)+".")
	}
	return lines, ""
}

// useHint returns the hint for an item not yet used: to wear one whose power starts worn and that
// has no use action, to use one with a power, spell or conversation, and none for anything else.
func useHint(it *content.Item) string {
	if it.Power == 0 && it.Casts == nil && it.Conversation == "" {
		return ""
	}
	used := slices.ContainsFunc(it.Actions, func(a content.ItemAction) bool {
		return a.Kind == content.ActionUse || a.Kind == content.ActionUseUp
	})
	if it.PowerWorn && !used {
		return hintWear
	}
	return hintUse
}

// scrollLines describe what a scroll teaching gesture teaches: the spell with that gesture, by
// name and school, its cost and what it does; or, for no spell's, the gesture's icons.
func (v *Viewer) scrollLines(gesture string) []string {
	for _, id := range sortedIDs(v.Pack.Spells) {
		sp := v.Pack.Spells[id]
		if sp.Gesture != gesture {
			continue
		}
		teaches := "Teaches " + v.spellName(id)
		if m := v.Pack.Magic; m != nil && sp.School >= 0 && sp.School < len(m.Schools) {
			teaches += " (" + inspect.Text(v.Pack, m.Schools[sp.School]) + ")"
		}
		lines := []string{teaches + ".", fmt.Sprintf("Costs %d magic points.", sp.Cost)}
		if sp.Effect != "" {
			lines = append(lines, inspect.Text(v.Pack, sp.Effect))
		}
		return lines
	}
	return []string{"Teaches the gesture " + strings.Join(strings.Split(gesture, ""), " ") + "."}
}

// strikeLines describe item it's strike actions as the hotbar's boxes number them: a single one
// by its damage; with two, each with its damage, and the second whether it is slower and the
// skill it needs.
func strikeLines(it *content.Item) []string {
	var strikes []content.ItemAction
	for _, a := range it.Actions {
		if a.Attack() && a.Damage != (content.Dice{}) {
			strikes = append(strikes, a)
		}
	}
	if len(strikes) == 1 {
		return []string{"Strikes for " + strikes[0].Damage.String() + "."}
	}
	var out []string
	for i, a := range strikes {
		s := fmt.Sprintf("Strike %d: %s damage", i+1, a.Damage)
		if i > 0 && a.Delay > strikes[0].Delay {
			s += ", slower"
		}
		if a.Skill > 0 {
			s += fmt.Sprintf(", needs skill %d", a.Skill)
		}
		out = append(out, s+".")
	}
	return out
}

// Book returns the book laid out, or nil when it is closed.
func (v *Viewer) Book() *BookView {
	if !v.BookOpen {
		return nil
	}
	bottom := v.Screen.Y - Margin
	if h := v.Hotbar(); h != nil {
		bottom = h.Rect.Min.Y - Margin
	}
	avail := max(bottom-2*Margin, 1)
	bv := &BookView{Tab: v.bookView.tab, Typed: v.BookTyped, Selected: -1, Full: v.BookFull, All: v.BookAll}
	bv.Scale = max(1, min(talkScale, (v.Screen.X-2*Margin)/bookWinWidth, avail/bookWinMinHeight))
	size := image.Pt(bookWinWidth, max(bookWinMinHeight, avail/bv.Scale))
	corner := image.Pt((v.Screen.X-size.X*bv.Scale)/2, max(Margin, (bottom-size.Y*bv.Scale)/2))
	bv.Rect = image.Rectangle{corner, corner.Add(size.Mul(bv.Scale))}

	b := v.World.Book()
	tabs := [2][]bookEntry{v.bookEntries(BookCreatures), v.bookEntries(BookItems)}
	for i, name := range [2]string{"Creatures", "Items"} {
		x := bookWinPad + i*(bookWinTab+bookWinPad)
		bv.Tabs[i] = PanelButton{Rect: image.Rect(x, bookWinPad, x+bookWinTab, bookWinPad+LineHeight+4), Name: strings.ToLower(name),
			Label: fmt.Sprintf("%s (%d)", name, len(tabs[i]))}
	}
	bv.Close = PanelButton{Rect: image.Rect(size.X-bookWinPad-bookWinButton, bookWinPad, size.X-bookWinPad, bookWinPad+LineHeight+4), Name: "close", Label: "Close"}
	fieldEnd := bv.Close.Rect.Min.X - bookWinPad
	if bv.Full {
		label := "This level"
		if bv.All {
			label = "All"
		}
		bv.Toggle = PanelButton{Rect: image.Rect(fieldEnd-bookWinToggle, bookWinPad, fieldEnd, bookWinPad+LineHeight+4), Name: "toggle", Label: label}
		fieldEnd = bv.Toggle.Rect.Min.X - bookWinPad
	}
	fx := bv.Tabs[1].Rect.Max.X + bookWinPad + 6*CharWidth
	bv.Field = image.Rect(fx, bookWinPad, min(fx+bookWinField, fieldEnd), bookWinPad+LineHeight+4)
	top := bv.Tabs[0].Rect.Max.Y + bookWinPad
	body := image.Rect(bookWinPad, top, size.X-bookWinPad, size.Y-bookWinPad)
	bv.Pane = image.Rect(body.Min.X+bookWinColumn+bookWinPad, body.Min.Y, body.Max.X, body.Max.Y)

	entries := tabs[bv.Tab]
	bv.Count = len(entries)
	if len(entries) == 0 {
		switch {
		case v.BookTyped != "":
			bv.Empty = "Nothing matches."
		case bv.Full:
			bv.Empty = "None on this level. Tab shows all."
		case bv.Tab == BookCreatures:
			bv.Empty = "No creatures found yet."
		default:
			bv.Empty = "Nothing found yet."
		}
		return bv
	}
	chosen := v.bookChosen(entries)
	fits := max(1, body.Dy()/bookWinRow)
	first := max(0, min(v.bookView.first, len(entries)-fits))
	if chosen < first {
		first = chosen
	} else if chosen >= first+fits {
		first = chosen - fits + 1
	}
	v.bookView.first = first
	bv.First = first
	for i := first; i < len(entries) && i-first < fits; i++ {
		y := body.Min.Y + (i-first)*bookWinRow
		r := BookRow{Rect: image.Rect(body.Min.X, y, body.Min.X+bookWinColumn, y+bookWinRow-2), InLevel: entries[i].inLevel}
		room := bookWinColumn - 8
		if bv.Tab == BookItems {
			if it := v.Pack.Items[entries[i].id]; it != nil {
				r.Icon, r.HasIcon = it.Icon, true
				room -= SlotSize + 4
			}
		}
		r.Label = Clip(entries[i].name, room/CharWidth)
		bv.Rows = append(bv.Rows, r)
	}
	bv.Selected = chosen - first

	// The pane: the picture, the name, what is known, and the hint.
	id := entries[chosen].id
	var items []paneItem
	var hint string
	if bv.Tab == BookCreatures {
		if t := v.Pack.Things[id]; t != nil {
			bv.Picture, bv.HasPicture = v.bookPicture(t), true
		}
		if bv.Full {
			items = v.fullBookLines(bv.Tab, id)
		} else {
			var lines []string
			lines, hint = v.CreatureLines(id, b.Creatures[id])
			items = textItems(lines)
		}
	} else {
		if it := v.Pack.Items[id]; it != nil {
			bv.Picture, bv.HasPicture = it.Icon, true
		}
		if bv.Full {
			items = v.fullBookLines(bv.Tab, id)
		} else {
			var lines []string
			lines, hint = v.ItemLines(id, b.Items[id])
			items = textItems(lines)
		}
	}
	scrolled := false
	if bv.Full {
		// The full book's pane scrolls by lines, keeping at least the last one; scrolled, it
		// leaves out the picture and the name above them.
		v.bookView.scroll = max(0, min(v.bookView.scroll, len(items)-1))
		items, scrolled = items[v.bookView.scroll:], v.bookView.scroll > 0
		bv.HasPicture = bv.HasPicture && !scrolled
	}
	bv.PictureAt = bv.Pane.Min
	y := bv.Pane.Min.Y
	if bv.HasPicture {
		if s := v.Pack.Sprites[bv.Picture.Sprite]; s != nil {
			y += BookPictureScale*s.FrameSize[1] + bookWinPad
		}
	}
	add := func(it paneItem, heading, hint bool) {
		words := layoutWords(strings.ReplaceAll(it.text, "*", ""), bv.Pane.Min.X, y, bv.Pane.Max.X)
		if n := len(words); n > 0 {
			y = words[n-1].Rect.Max.Y
		} else if it.text == "" {
			y += LineHeight
		}
		line := BookLine{Words: words, Heading: heading, Hint: hint, Thing: it.thing, Cell: it.cell}
		// Pictures lie in a row below the text, each at its frame's size.
		x, tall := bv.Pane.Min.X+2*CharWidth, 0
		for _, img := range it.pictures {
			sp := v.Pack.Sprites[img.Sprite]
			if sp == nil || img.Frame < 0 || img.Frame >= sp.Frames {
				continue
			}
			w, h := BookPictureScale*sp.FrameSize[0], BookPictureScale*sp.FrameSize[1]
			if x > bv.Pane.Min.X+2*CharWidth && x+w > bv.Pane.Max.X {
				x, y, tall = bv.Pane.Min.X+2*CharWidth, y+tall+2, 0
			}
			line.Pictures = append(line.Pictures, BookPicture{Image: img, At: image.Pt(x, y+2)})
			x += w + bookWinPad
			tall = max(tall, h+4)
		}
		y += tall
		if y <= bv.Pane.Max.Y {
			bv.Lines = append(bv.Lines, line)
		}
	}
	if !scrolled {
		add(paneItem{text: entries[chosen].name}, true, false)
		y += LineHeight / 2
	}
	for _, it := range items {
		add(it, false, false)
	}
	if hint != "" {
		y += LineHeight / 2
		add(paneItem{text: hint}, false, true)
	}
	return bv
}

// textItems returns lines as pane items of text alone.
func textItems(lines []string) []paneItem {
	items := make([]paneItem, len(lines))
	for i, l := range lines {
		items[i] = paneItem{text: l}
	}
	return items
}

// Rect returns the line's text's rectangle, empty for a line without words.
func (l BookLine) Rect() image.Rectangle {
	var r image.Rectangle
	for _, w := range l.Words {
		r = r.Union(w.Rect)
	}
	return r
}

// at returns the window pixel under screen point pt.
func (bv *BookView) at(pt image.Point) image.Point { return pt.Sub(bv.Rect.Min).Div(bv.Scale) }

// bookChoose chooses entry i of the open tab.
func (v *Viewer) bookChoose(i int) {
	entries := v.bookEntries(v.bookView.tab)
	if len(entries) == 0 {
		return
	}
	i = max(0, min(len(entries)-1, i))
	if v.bookView.chosen[v.bookView.tab] != entries[i].id {
		v.bookView.scroll = 0
	}
	v.bookView.chosen[v.bookView.tab] = entries[i].id
}

// bookInput takes input while the book is open. Typing filters the entries by name, Backspace
// takes a letter off; Up, Down, the wheel or a click choose an entry, Page Up and Page Down move
// a page; Left and Right, or a click on a tab, switch tabs. Escape clears the filter, or closes
// the book, as a right click or Close does. In the full book, Tab or the toggle switches between
// the current level's entries and all, the wheel over the pane scrolls it, and a click on a line
// of where the entry is on the level closes the book and selects it there.
func (v *Viewer) bookInput(in Input) {
	bv := v.Book()
	if bv == nil {
		return
	}
	if in.pressed(KeyEscape) && v.BookTyped != "" {
		v.BookTyped = ""
		return
	}
	cursor := bv.at(in.Cursor)
	if in.pressed(KeyEscape) || in.Right.Released || in.Left.Released && cursor.In(bv.Close.Rect) {
		v.CloseBook()
		return
	}
	typed := false
	if in.pressed(KeyBackspace) && v.BookTyped != "" {
		r := []rune(v.BookTyped)
		v.BookTyped, typed = string(r[:len(r)-1]), true
	}
	for _, r := range in.Chars {
		if unicode.IsPrint(r) && len([]rune(v.BookTyped)) < maxField {
			v.BookTyped, typed = v.BookTyped+string(r), true
		}
	}
	if typed {
		v.bookView.first = 0
		return
	}
	if bv.Full {
		switch {
		case in.pressed(KeyTab) || in.Left.Released && cursor.In(bv.Toggle.Rect):
			v.BookAll = !v.BookAll
			v.bookView.first, v.bookView.scroll = 0, 0
			return
		case in.Wheel != 0 && cursor.In(bv.Pane):
			v.bookView.scroll = max(0, v.bookView.scroll-int(in.Wheel)*3)
			return
		case in.Left.Released:
			for _, l := range bv.Lines {
				if (l.Thing != "" || l.Cell != nil) && cursor.In(l.Rect()) {
					v.bookGo(l.Thing, l.Cell)
					return
				}
			}
		}
	}
	if in.Left.Released {
		for i, t := range bv.Tabs {
			if cursor.In(t.Rect) {
				v.bookView.tab, v.bookView.scroll = i, 0
				return
			}
		}
		for i, r := range bv.Rows {
			if cursor.In(r.Rect) {
				v.bookChoose(bv.First + i)
				return
			}
		}
	}
	chosen := bv.First + bv.Selected
	page := max(1, len(bv.Rows)-1)
	switch {
	case in.pressed(KeyArrowLeft):
		v.bookView.tab, v.bookView.scroll = BookCreatures, 0
	case in.pressed(KeyArrowRight):
		v.bookView.tab, v.bookView.scroll = BookItems, 0
	case in.pressed(KeyArrowUp):
		v.bookChoose(chosen - 1)
	case in.pressed(KeyArrowDown):
		v.bookChoose(chosen + 1)
	case in.pressed(KeyPageUp):
		v.bookChoose(chosen - page)
	case in.pressed(KeyPageDown):
		v.bookChoose(chosen + page)
	case in.Wheel > 0:
		v.bookChoose(chosen - 1)
	case in.Wheel < 0:
		v.bookChoose(chosen + 1)
	}
}
