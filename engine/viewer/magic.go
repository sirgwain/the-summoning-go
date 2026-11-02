package viewer

import (
	"fmt"
	"image"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
)

// The spell panel: the gesture icons the player knows, the gesture being entered, and the
// spellbook (docs/spec/spells.md, Gestures, learning and memorizing). Entering a gesture and
// choosing Learn sends it; choosing a spell of the spellbook memorizes it once more.

// Spell panel layout, in panel pixels, scaled together like the hotbar's.
const (
	iconStep    = 34
	iconColumns = 6
	// letterHeight is the room under each icon of the grid for the letter that enters it.
	letterHeight = 12
	panelPad     = 4
	buttonWidth  = 60
	buttonHeight = 20
	bookColumns  = 2
	bookWidth    = 136
	bookHeight   = 18
	panelWidthPx = panelPad*2 + content.MaxGesture*iconStep
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

// BookEntry is a spell of the spellbook: choosing it memorizes it.
type BookEntry struct {
	Rect  image.Rectangle
	Spell string
	Icon  *content.Image
	Label string
}

// SpellPanel is the spell panel laid out on the screen. Rect is on the screen, Scale its screen
// pixels per panel pixel; every other rectangle is in panel pixels from Rect's corner.
type SpellPanel struct {
	Rect  image.Rectangle
	Scale int
	Icons []IconButton
	// Entered are the icons entered so far.
	Entered []IconButton
	Buttons []PanelButton
	Book    []BookEntry
}

// OpenSpells opens the spell panel, when playing.
func (v *Viewer) OpenSpells() {
	if v.Play && v.World.Player() != nil {
		v.SpellsOpen, v.Entered = true, ""
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
	for i, id := range p.Magic.Spellbook {
		s := v.Pack.Spells[id]
		if s == nil {
			continue
		}
		x, ey := panelPad+(i%bookColumns)*bookWidth, y+(i/bookColumns)*bookHeight
		label := fmt.Sprintf("%s %s %d", inspect.Text(v.Pack, s.Name), s.Gesture, s.Cost)
		if n := p.Magic.Memorized[id]; n > 0 {
			label += fmt.Sprintf(" x%d", n)
		}
		sp.Book = append(sp.Book, BookEntry{Rect: image.Rect(x, ey, x+bookWidth-2, ey+bookHeight-2), Spell: id, Icon: s.Icon, Label: label})
	}
	rows := (len(p.Magic.Spellbook) + bookColumns - 1) / bookColumns
	size := image.Pt(panelWidthPx, y+rows*bookHeight+panelPad)
	bottom := v.Screen.Y - Margin
	if h := v.Hotbar(); h != nil {
		bottom = h.Rect.Min.Y - Margin
		sp.Scale = h.Scale
	}
	sp.Scale = max(1, min(sp.Scale, v.Screen.X/size.X, bottom/max(size.Y, 1)))
	at := image.Pt((v.Screen.X-size.X*sp.Scale)/2, bottom-size.Y*sp.Scale)
	sp.Rect = image.Rectangle{at, at.Add(size.Mul(sp.Scale))}
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

// spellKeys are the keys while the spell panel is open: A–L enter icons, Backspace takes the last
// off, Enter learns, Escape, F10 or R closes.
func (v *Viewer) spellKeys(in Input) []input.Command {
	switch {
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
		if at.In(e.Rect) {
			return []input.Command{input.Memorize{Spell: e.Spell}}
		}
	}
	return nil
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
