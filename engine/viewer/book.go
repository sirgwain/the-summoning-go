package viewer

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"unicode"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
)

// The book's window (docs/spec/player.md, The book): a tab each for creatures and items, the
// entries found down the left by name, and the chosen one's picture and what the player has
// learned of it on the right. The tools open the same window as the full book (fullbook.go). What
// the book says is inspect's; this lays it out and takes the input.

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

// bookState is the book's window: whether it is open, and as the tools' full book, listing all
// the pack's entries or the current level's; the filter typed; the tab open, the entry chosen on
// each tab, the first entry its list shows and how many lines the full book's pane is scrolled;
// and what the full book has worked out of the world.
type bookState struct {
	open, full, all bool
	typed           string
	tab             int
	chosen          [2]string
	first, scroll   int
	cache           bookCache
}

// bookCache is what the full book works out of the world, kept for the level and step it was
// worked out on: the level's places, and the pane of the entry last shown.
type bookCache struct {
	level  string
	step   int64
	ok     bool
	places inspect.Places
	tab    int
	id     string
	pane   []inspect.PaneItem
}

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
	// times its size as all the pane's pictures do; Lines are its text, laid out below the picture.
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

// bookSource is what a book lists and says: the player's book, or the tools' full book.
type bookSource interface {
	// entries are a tab's entries, before the filter.
	entries(tab int) []bookEntry
	// pane is what the pane says of a tab's entry id, and the hint of what the next stage would
	// show.
	pane(tab int, id string) (items []inspect.PaneItem, hint string)
	// empty is what a tab with no entries says, with no filter typed.
	empty(tab int) string
}

// gameBook is the player's book: what the world's book holds, as far as the player has learned.
type gameBook struct{ v *Viewer }

func (g gameBook) entries(tab int) []bookEntry {
	p, b := g.v.Pack, g.v.World.Book()
	var out []bookEntry
	if tab == BookCreatures {
		for id := range b.Creatures {
			out = append(out, bookEntry{id: id, name: inspect.ThingName(p, id)})
		}
	} else {
		for id := range b.Items {
			out = append(out, bookEntry{id: id, name: inspect.ItemName(p, id)})
		}
	}
	return out
}

func (g gameBook) pane(tab int, id string) ([]inspect.PaneItem, string) {
	p, b := g.v.Pack, g.v.World.Book()
	var lines []string
	var hint string
	if tab == BookCreatures {
		lines, hint = inspect.CreatureLines(p, id, b.Creatures[id])
	} else {
		lines, hint = inspect.ItemLines(p, id, b.Items[id], b.Recipes...)
	}
	items := make([]inspect.PaneItem, len(lines))
	for i, l := range lines {
		items[i] = inspect.PaneItem{Text: l}
	}
	return items, hint
}

func (gameBook) empty(tab int) string {
	if tab == BookCreatures {
		return "No creatures found yet."
	}
	return "Nothing found yet."
}

// source returns the book open, the full book or the player's.
func (v *Viewer) source() bookSource {
	if v.book.full {
		return fullBook{v}
	}
	return gameBook{v}
}

// OpenBook opens the book, when playing, on the tab and entry last chosen.
func (v *Viewer) OpenBook() {
	if !v.Play || v.World.Player() == nil {
		return
	}
	v.book.open, v.book.full, v.book.typed, v.SpellsOpen = true, false, "", false
}

// CloseBook closes the book.
func (v *Viewer) CloseBook() { v.book.open, v.book.full, v.book.typed = false, false, "" }

// bookEntries returns the open book's entries of tab whose names, or for the full book ids, hold
// the filter typed, ignoring case, by name.
func (v *Viewer) bookEntries(tab int) []bookEntry {
	filter := strings.ToLower(strings.TrimSpace(v.book.typed))
	out := v.source().entries(tab)
	if filter != "" {
		out = slices.DeleteFunc(out, func(e bookEntry) bool {
			return !strings.Contains(strings.ToLower(e.name), filter) && !(v.book.full && strings.Contains(e.id, filter))
		})
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
	i := slices.IndexFunc(entries, func(e bookEntry) bool { return e.id == v.book.chosen[v.book.tab] })
	return max(i, 0)
}

// Book returns the book laid out, or nil when it is closed.
func (v *Viewer) Book() *BookView {
	if !v.book.open {
		return nil
	}
	src := v.source()
	bottom := v.Screen.Y - Margin
	if h := v.Hotbar(); h != nil {
		bottom = h.Frame.Min.Y - Margin
	}
	avail := max(bottom-2*Margin, 1)
	bv := &BookView{Tab: v.book.tab, Typed: v.book.typed, Selected: -1, Full: v.book.full, All: v.book.all}
	bv.Scale = max(1, min(talkScale, (v.Screen.X-2*Margin)/bookWinWidth, avail/bookWinMinHeight))
	size := image.Pt(bookWinWidth, max(bookWinMinHeight, avail/bv.Scale))
	corner := image.Pt((v.Screen.X-size.X*bv.Scale)/2, max(Margin, (bottom-size.Y*bv.Scale)/2))
	bv.Rect = image.Rectangle{corner, corner.Add(size.Mul(bv.Scale))}

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
		bv.Empty = src.empty(bv.Tab)
		if v.book.typed != "" {
			bv.Empty = "Nothing matches."
		}
		return bv
	}
	chosen := v.bookChosen(entries)
	fits := max(1, body.Dy()/bookWinRow)
	first := max(0, min(v.book.first, len(entries)-fits))
	if chosen < first {
		first = chosen
	} else if chosen >= first+fits {
		first = chosen - fits + 1
	}
	v.book.first = first
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
	if bv.Tab == BookCreatures {
		if t := v.Pack.Things[id]; t != nil {
			bv.Picture, bv.HasPicture = inspect.BookPicture(v.Pack, t), true
		}
	} else if it := v.Pack.Items[id]; it != nil {
		bv.Picture, bv.HasPicture = it.Icon, true
	}
	items, hint := src.pane(bv.Tab, id)
	scrolled := false
	if bv.Full {
		// The full book's pane scrolls by lines, keeping at least the last one; scrolled, it
		// leaves out the picture and the name above them.
		v.book.scroll = max(0, min(v.book.scroll, len(items)-1))
		items, scrolled = items[v.book.scroll:], v.book.scroll > 0
		bv.HasPicture = bv.HasPicture && !scrolled
	}
	v.layPane(bv, items, entries[chosen].name, hint, scrolled)
	return bv
}

// layPane lays the pane's lines out in bv: the picture, then unless scrolled the name, the items
// and the hint.
func (v *Viewer) layPane(bv *BookView, items []inspect.PaneItem, name, hint string, scrolled bool) {
	bv.PictureAt = bv.Pane.Min
	y := bv.Pane.Min.Y
	if bv.HasPicture {
		if s := v.Pack.Sprites[bv.Picture.Sprite]; s != nil {
			y += BookPictureScale*s.FrameSize[1] + bookWinPad
		}
	}
	add := func(it inspect.PaneItem, heading, hint bool) {
		words := layoutWords(strings.ReplaceAll(it.Text, "*", ""), bv.Pane.Min.X, y, bv.Pane.Max.X)
		if n := len(words); n > 0 {
			y = words[n-1].Rect.Max.Y
		} else if it.Text == "" {
			y += LineHeight
		}
		line := BookLine{Words: words, Heading: heading, Hint: hint, Thing: it.Thing, Cell: it.Cell}
		// Pictures lie in a row below the text, each at its frame's size.
		x, tall := bv.Pane.Min.X+2*CharWidth, 0
		for _, img := range it.Pictures {
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
		add(inspect.PaneItem{Text: name}, true, false)
		y += LineHeight / 2
	}
	for _, it := range items {
		add(it, false, false)
	}
	if hint != "" {
		y += LineHeight / 2
		add(inspect.PaneItem{Text: hint}, false, true)
	}
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
	entries := v.bookEntries(v.book.tab)
	if len(entries) == 0 {
		return
	}
	i = max(0, min(len(entries)-1, i))
	if v.book.chosen[v.book.tab] != entries[i].id {
		v.book.scroll = 0
	}
	v.book.chosen[v.book.tab] = entries[i].id
}

// bookTab opens tab.
func (v *Viewer) bookTab(tab int) { v.book.tab, v.book.scroll = tab, 0 }

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
	if in.pressed(KeyEscape) && v.book.typed != "" {
		v.book.typed = ""
		return
	}
	cursor := bv.at(in.Cursor)
	if in.pressed(KeyEscape) || in.Right.Released || in.Left.Released && cursor.In(bv.Close.Rect) {
		v.CloseBook()
		return
	}
	typed := false
	if in.pressed(KeyBackspace) && v.book.typed != "" {
		r := []rune(v.book.typed)
		v.book.typed, typed = string(r[:len(r)-1]), true
	}
	for _, r := range in.Chars {
		if unicode.IsPrint(r) && len([]rune(v.book.typed)) < maxField {
			v.book.typed, typed = v.book.typed+string(r), true
		}
	}
	if typed {
		v.book.first = 0
		return
	}
	if bv.Full && v.fullBookInput(in, bv, cursor) {
		return
	}
	if in.Left.Released {
		for i, t := range bv.Tabs {
			if cursor.In(t.Rect) {
				v.bookTab(i)
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
		v.bookTab(BookCreatures)
	case in.pressed(KeyArrowRight):
		v.bookTab(BookItems)
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
