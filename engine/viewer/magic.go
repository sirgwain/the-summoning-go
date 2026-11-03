package viewer

import (
	"fmt"
	"image"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/frame"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
)

// The spell panel: the gesture icons the player knows, the gesture being entered, and the
// spellbook (docs/spec/spells.md, Gestures, learning and memorizing). Entering a gesture and
// choosing Learn sends it. The spellbook is a scrolling list by school, each school headed by its
// memorizations and limit, each spell a row with a Memorize button that memorizes it once more.

// Spell panel layout, in panel pixels, scaled together like the hotbar's.
const (
	iconStep    = 34
	iconColumns = 6
	// letterHeight is the room under each icon of the grid for the letter that enters it.
	letterHeight = 12
	panelPad     = 4
	buttonWidth  = 60
	buttonHeight = 20
	panelWidthPx = panelPad*2 + content.MaxGesture*iconStep
	// bookRows is how many rows of the spellbook show at once, each rowHeight high; the scroll
	// bar beside them is barWidth wide and each row's Memorize button memorizeWidth.
	bookRows      = 8
	rowHeight     = 20
	barWidth      = 8
	memorizeWidth = 64
	bookWidth     = panelWidthPx - panelPad*2 - barWidth - 2
)

// IconButton is a gesture icon on the panel; one the player does not know is shown empty. A known
// icon of the grid shows the Letter that enters it in LetterRect, under it.
type IconButton struct {
	Rect       image.Rectangle
	LetterRect image.Rectangle
	Letter     byte
	Icon       content.Image
	Known      bool
}

// PanelButton is a labeled button on a panel.
type PanelButton struct {
	Rect  image.Rectangle
	Name  string
	Label string
}

// BookEntry is a spell row of the spellbook showing on the panel: its icon in IconRect, its
// name, gesture, cost and memorizations as Label, and Button the Memorize button.
type BookEntry struct {
	Rect     image.Rectangle
	IconRect image.Rectangle
	Button   image.Rectangle
	Spell    string
	Icon     *content.Image
	Label    string
}

// BookHeader is a school's heading row: its name and its memorizations of its limit.
type BookHeader struct {
	Rect   image.Rectangle
	School int
	Label  string
}

// bookRow is a row of the whole spellbook list: a school's heading, or a spell when Spell is set.
type bookRow struct {
	school int
	spell  string
}

// SpellPanel is the spell panel laid out on the screen. Rect is on the screen, Scale its screen
// pixels per panel pixel; every other rectangle is in panel pixels from Rect's corner.
type SpellPanel struct {
	// Rect is the panel on the screen, Frame it with its frame round it.
	Rect, Frame image.Rectangle
	Scale       int
	Icons       []IconButton
	// Entered are the icons entered so far.
	Entered []IconButton
	Buttons []PanelButton
	// Book and Headers are the spellbook's rows that show, List the rectangle they show in,
	// Rows how many rows the whole list has and Scroll the first that shows. Bar is the scroll
	// bar and Thumb the part of it for the rows showing, empty when every row shows.
	Book    []BookEntry
	Headers []BookHeader
	List    image.Rectangle
	Rows    int
	Scroll  int
	Bar     image.Rectangle
	Thumb   image.Rectangle
}

// bookList is the spellbook as rows: each school with learned spells, in school order, a heading
// then its spells in the order learned.
func (v *Viewer) bookList() []bookRow {
	p := v.World.Player()
	var rows []bookRow
	for school := range content.Schools {
		head := false
		for _, id := range p.Magic.Spellbook {
			if s := v.Pack.Spells[id]; s == nil || s.School != school {
				continue
			}
			if !head {
				rows, head = append(rows, bookRow{school: school}), true
			}
			rows = append(rows, bookRow{school: school, spell: id})
		}
	}
	return rows
}

// OpenSpells opens the spell panel, when playing.
func (v *Viewer) OpenSpells() {
	if v.Play && v.World.Player() != nil {
		v.SpellsOpen, v.Entered, v.spellbook.wheel = true, "", 0
		v.cue(content.CueGesture)
	}
}

// SpellPanel returns the spell panel as it is now, or nil when it is closed.
func (v *Viewer) SpellPanel() *SpellPanel {
	p := v.World.Player()
	if !v.SpellsOpen || p == nil {
		return nil
	}
	sp := &SpellPanel{}
	icon := func(c byte) (content.Image, bool) {
		if m := v.Pack.Magic; m != nil && int(c-'A') < len(m.Icons) {
			return m.Icons[c-'A'], true
		}
		return content.Image{}, false
	}
	for i := range content.GestureIcons {
		c := byte('A' + i)
		x, y := panelPad+(i%iconColumns)*iconStep, panelPad+(i/iconColumns)*(iconStep+letterHeight)
		img, _ := icon(c)
		sp.Icons = append(sp.Icons, IconButton{
			Rect: image.Rect(x, y, x+iconStep-2, y+iconStep-2), LetterRect: image.Rect(x, y+iconStep-2, x+iconStep-2, y+iconStep+letterHeight-2),
			Letter: c, Icon: img, Known: p.Magic.Knows(c),
		})
	}
	bx := panelPad + iconColumns*iconStep + panelPad
	for i, b := range []PanelButton{{Name: "learn", Label: "Learn"}, {Name: "back", Label: "Back"}, {Name: "close", Label: "Close"}} {
		y := panelPad + i*(buttonHeight+2)
		b.Rect = image.Rect(bx, y, bx+buttonWidth, y+buttonHeight)
		sp.Buttons = append(sp.Buttons, b)
	}
	y := panelPad + 2*(iconStep+letterHeight) + panelPad
	for i := 0; i < len(v.Entered); i++ {
		img, _ := icon(v.Entered[i])
		x := panelPad + i*iconStep
		sp.Entered = append(sp.Entered, IconButton{Rect: image.Rect(x, y, x+iconStep-2, y+iconStep-2), Letter: v.Entered[i], Icon: img, Known: true})
	}
	y += iconStep + panelPad
	rows := v.bookList()
	shown := max(1, min(len(rows), bookRows))
	sp.Rows = len(rows)
	sp.Scroll = max(0, min(v.spellbook.scroll, len(rows)-shown))
	sp.List = image.Rect(panelPad, y, panelPad+bookWidth, y+shown*rowHeight)
	if len(rows) > shown {
		sp.Bar = image.Rect(sp.List.Max.X+2, y, sp.List.Max.X+2+barWidth, sp.List.Max.Y)
		h := sp.Bar.Dy()
		top, bot := sp.Scroll*h/len(rows), (sp.Scroll+shown)*h/len(rows)
		sp.Thumb = image.Rect(sp.Bar.Min.X, y+top, sp.Bar.Max.X, y+max(bot, top+4))
	}
	for i, r := range rows[sp.Scroll : sp.Scroll+min(shown, len(rows))] {
		ry := y + i*rowHeight
		rect := image.Rect(sp.List.Min.X, ry, sp.List.Max.X, ry+rowHeight-2)
		if r.spell == "" {
			label := fmt.Sprintf("School %d", r.school+1)
			if m := v.Pack.Magic; m != nil && r.school < len(m.Schools) {
				label = inspect.Text(v.Pack, m.Schools[r.school])
			}
			label += fmt.Sprintf("  %d/%d", v.World.SchoolTotal(r.school), v.World.MemorizeLimit(r.school))
			sp.Headers = append(sp.Headers, BookHeader{Rect: rect, School: r.school, Label: label})
			continue
		}
		s := v.Pack.Spells[r.spell]
		label := fmt.Sprintf("%s %s %d", inspect.Text(v.Pack, s.Name), s.Gesture, s.Cost)
		if n := p.Magic.Memorized[r.spell]; n > 0 {
			label += fmt.Sprintf(" x%d", n)
		}
		sp.Book = append(sp.Book, BookEntry{
			Rect: rect, IconRect: image.Rect(rect.Min.X, ry, rect.Min.X+rowHeight-2, ry+rowHeight-2),
			Button: image.Rect(rect.Max.X-memorizeWidth, ry, rect.Max.X, ry+rowHeight-2),
			Spell:  r.spell, Icon: s.Icon, Label: label,
		})
	}
	size := image.Pt(panelWidthPx, sp.List.Max.Y+panelPad)
	bottom := v.Screen.Y - Margin
	if h := v.Hotbar(); h != nil {
		bottom = h.Frame.Min.Y - Margin
		sp.Scale = h.Scale
	}
	outer := size.Add(image.Pt(2*frame.Width, 2*frame.Width))
	sp.Scale = max(1, min(sp.Scale, v.Screen.X/outer.X, bottom/outer.Y))
	at := image.Pt((v.Screen.X-size.X*sp.Scale)/2, bottom-(size.Y+frame.Width)*sp.Scale)
	sp.Rect = image.Rectangle{at, at.Add(size.Mul(sp.Scale))}
	sp.Frame = framed(sp.Rect, sp.Scale)
	return sp
}

// Screen returns the screen rectangle of panel rectangle r.
func (sp *SpellPanel) Screen(r image.Rectangle) image.Rectangle {
	return image.Rectangle{sp.Rect.Min.Add(r.Min.Mul(sp.Scale)), sp.Rect.Min.Add(r.Max.Mul(sp.Scale))}
}

// enter adds icon c to the gesture being entered: a known icon, at most eight, never the same
// twice in a row.
func (v *Viewer) enter(c byte) {
	p := v.World.Player()
	n := len(v.Entered)
	if !p.Magic.Knows(c) || n >= content.MaxGesture || n > 0 && v.Entered[n-1] == c {
		return
	}
	v.Entered += string(c)
	v.cue(content.CueGesture)
}

// takeOffEntered takes the last icon off the gesture being entered.
func (v *Viewer) takeOffEntered() {
	if n := len(v.Entered); n > 0 {
		v.Entered = v.Entered[:n-1]
		v.cue(content.CueGesture)
	}
}

// learnEntered sends the gesture entered and clears it.
func (v *Viewer) learnEntered() []input.Command {
	if v.Entered == "" {
		return nil
	}
	g := v.Entered
	v.Entered = ""
	return []input.Command{input.Learn{Gesture: g}}
}

// scrollBook scrolls the spellbook list by n rows, no further than its ends.
func (v *Viewer) scrollBook(n int) {
	rows := len(v.bookList())
	v.spellbook.scroll = max(0, min(v.spellbook.scroll+n, rows-min(rows, bookRows)))
}

// spellKeys are the keys while the spell panel is open: A–L enter icons, Backspace takes the last
// off, Enter learns, Page Up and Page Down scroll the spellbook, Escape, F10 or R closes.
func (v *Viewer) spellKeys(in Input) []input.Command {
	switch {
	case in.pressed(KeyPageUp):
		v.scrollBook(-(bookRows - 1))
		return nil
	case in.pressed(KeyPageDown):
		v.scrollBook(bookRows - 1)
		return nil
	case in.pressed(KeyEscape, KeyF10, "R"):
		v.SpellsOpen = false
		return nil
	case in.pressed(KeyBackspace):
		v.takeOffEntered()
		return nil
	case in.pressed(KeyEnter):
		return v.learnEntered()
	}
	for i := range content.GestureIcons {
		if in.pressed(Key(rune('A' + i))) {
			v.enter(byte('A' + i))
		}
	}
	return nil
}

// spellClick is a click on the spell panel at screen point pt.
func (v *Viewer) spellClick(sp *SpellPanel, pt image.Point, right bool) []input.Command {
	at := pt.Sub(sp.Rect.Min).Div(sp.Scale)
	if right {
		v.SpellsOpen = false
		return nil
	}
	for _, b := range sp.Icons {
		if at.In(b.Rect) {
			v.enter(b.Letter)
			return nil
		}
	}
	for _, b := range sp.Buttons {
		if !at.In(b.Rect) {
			continue
		}
		switch b.Name {
		case "learn":
			return v.learnEntered()
		case "back":
			if n := len(v.Entered); n > 0 {
				v.Entered = v.Entered[:n-1]
			}
		case "close":
			v.SpellsOpen = false
		}
		return nil
	}
	for _, e := range sp.Book {
		if at.In(e.Button) {
			return []input.Command{input.Memorize{Spell: e.Spell}}
		}
	}
	if at.In(sp.Bar) && !at.In(sp.Thumb) {
		if at.Y < sp.Thumb.Min.Y {
			v.scrollBook(-(bookRows - 1))
		} else {
			v.scrollBook(bookRows - 1)
		}
	}
	return nil
}

// spellWheel scrolls the spellbook by the wheel while the cursor is over the spell panel, a row
// per whole wheel unit, the wheel turned away scrolling down the list.
func (v *Viewer) spellWheel(sp *SpellPanel, in Input) {
	if in.Wheel == 0 || !in.Cursor.In(sp.Frame) {
		return
	}
	v.spellbook.wheel += in.Wheel
	for ; v.spellbook.wheel >= 1; v.spellbook.wheel-- {
		v.scrollBook(-1)
	}
	for ; v.spellbook.wheel <= -1; v.spellbook.wheel++ {
		v.scrollBook(1)
	}
}

// castKeys are the play keys that cast: F1–F4 or 1–4 cast a school's spell, with Shift select its
// next; F10 or R opens the spell panel.
func (v *Viewer) castKeys(in Input) []input.Command {
	var cmds []input.Command
	for s := range content.Schools {
		if in.pressed(Key(fmt.Sprintf("F%d", s+1)), Key(fmt.Sprintf("Digit%d", s+1))) {
			if in.held(KeyShift) {
				cmds = append(cmds, input.NextSpell{School: s})
			} else {
				cmds = append(cmds, input.Cast{School: s})
			}
		}
	}
	if in.pressed(KeyF10, "R") {
		v.OpenSpells()
	}
	return cmds
}
