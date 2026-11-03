package viewer

import (
	"image"
	"strings"
	"unicode"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The conversation window (docs/spec/dialogue.md): the line said, beside its portrait, with its
// keywords marked; once it has been read, the offered keywords as buttons and a field to type a
// word. While a conversation is open the world waits, and the window takes all input.

// Conversation window layout, in window pixels, which the window scales by up to talkScale.
const (
	talkScale      = 2
	talkWidth      = 520
	talkPad        = 8
	talkPortrait   = 48
	talkFieldWidth = 240
	maxField       = 39
	// DefaultName is the player's name in texts when the player has none.
	DefaultName = "Stranger"
)

// Word is a word of a line laid out on the screen; Keyword marks one that can be asked.
type Word struct {
	Rect    image.Rectangle
	Text    string
	Keyword bool
}

// TalkView is the open conversation laid out on the screen. Rect is on the screen and Scale its
// screen pixels per window pixel; every other rectangle and point is in window pixels from
// Rect's corner.
type TalkView struct {
	Rect  image.Rectangle
	Scale int
	// Portrait is the portrait beside the line, drawn at PortraitAt at PortraitScale, if Has.
	Portrait      content.Image
	PortraitAt    image.Point
	PortraitScale int
	HasPortrait   bool
	Mirrored      bool
	// Words are the line's words; Icons the gesture icons a line teaches, at IconsAt.
	Words   []Word
	Icons   []content.Image
	IconsAt []image.Point
	// Waiting is true while a line waits to be read; otherwise Buttons are the offered keywords
	// (or Bye) and Field the field a word is typed in.
	Waiting bool
	Buttons []PanelButton
	Field   image.Rectangle
	Typed   string
	// Selected is the highlighted button, or len(Buttons) for the field: what Enter takes.
	Selected int
}

// Talk returns the open conversation laid out, or nil.
func (v *Viewer) Talk() *TalkView {
	t := v.World.Talk()
	if t == nil {
		return nil
	}
	w := talkWidth
	at := image.Point{}
	tv := &TalkView{Waiting: t.Line != nil, Typed: v.Typed, PortraitScale: 2}
	textX := at.X + talkPad + talkPortrait*tv.PortraitScale + talkPad
	y := at.Y + talkPad
	if l := t.Line; l != nil {
		portrait := l.Portrait
		if portrait == "" && l.Kind == sim.LineSay {
			portrait = v.World.Player().Portrait
		}
		if pt := v.Pack.Portraits[portrait]; pt != nil {
			tv.Portrait, tv.HasPortrait, tv.Mirrored = pt.Image, true, pt.Mirrored && l.Portrait == ""
			tv.PortraitAt = image.Pt(at.X+talkPad, y)
		}
		switch l.Kind {
		case sim.LineTeach:
			x := textX
			for i := 0; i < len(l.Icons); i++ {
				if m := v.Pack.Magic; m != nil && int(l.Icons[i]-'A') < len(m.Icons) {
					tv.Icons = append(tv.Icons, m.Icons[l.Icons[i]-'A'])
					tv.IconsAt = append(tv.IconsAt, image.Pt(x, y))
					x += 34 * 2
				}
			}
			y += 32*2 + talkPad
		default:
			tv.Words = layoutWords(v.lineText(l.Text), textX, y, at.X+w-talkPad)
			if n := len(tv.Words); n > 0 {
				y = tv.Words[n-1].Rect.Max.Y + talkPad
			}
		}
		y = max(y, at.Y+talkPad+talkPortrait*tv.PortraitScale+talkPad)
	} else {
		x := at.X + talkPad
		for i, k := range t.Offered {
			if k == "" {
				continue
			}
			label := inspect.KeywordWord(v.Pack, k)
			bw := len(label)*CharWidth + 2*talkPad
			if x+bw > at.X+w-talkPad {
				x, y = at.X+talkPad, y+LineHeight+talkPad
			}
			tv.Buttons = append(tv.Buttons, PanelButton{Rect: image.Rect(x, y, x+bw, y+LineHeight+4), Name: "offer." + string(rune('0'+i)), Label: label})
			x += bw + talkPad
		}
		if len(tv.Buttons) == 0 {
			tv.Buttons = append(tv.Buttons, PanelButton{Rect: image.Rect(x, y, x+5*CharWidth+2*talkPad, y+LineHeight+4), Name: "bye", Label: "Bye"})
		}
		y += LineHeight + 4 + talkPad
		tv.Field = image.Rect(at.X+talkPad, y, at.X+talkPad+talkFieldWidth, y+LineHeight+4)
		y = tv.Field.Max.Y + talkPad
		tv.Selected = max(0, min(v.TalkSelected, len(tv.Buttons)))
	}
	// Centered in the view above the hotbar, as large as fits up to talkScale.
	size := image.Pt(w, y)
	bottom := v.Screen.Y
	if h := v.Hotbar(); h != nil {
		bottom = h.Frame.Min.Y - Margin
	}
	tv.Scale = max(1, min(talkScale, (v.Screen.X-2*Margin)/size.X, max(bottom-2*Margin, 1)/size.Y))
	corner := image.Pt((v.Screen.X-size.X*tv.Scale)/2, max(Margin, (bottom-size.Y*tv.Scale)/2))
	tv.Rect = image.Rectangle{corner, corner.Add(size.Mul(tv.Scale))}
	return tv
}

// Screen returns the screen rectangle of window rectangle r.
func (tv *TalkView) Screen(r image.Rectangle) image.Rectangle {
	return image.Rectangle{tv.Rect.Min.Add(r.Min.Mul(tv.Scale)), tv.Rect.Min.Add(r.Max.Mul(tv.Scale))}
}

// at returns the window pixel under screen point pt.
func (tv *TalkView) at(pt image.Point) image.Point { return pt.Sub(tv.Rect.Min).Div(tv.Scale) }

// lineText returns text key's text with the player's name in place of @.
func (v *Viewer) lineText(key string) string {
	return strings.ReplaceAll(inspect.Text(v.Pack, key), "@", v.playerName())
}

// playerName returns the player's name, or DefaultName while it has none.
func (v *Viewer) playerName() string {
	if p := v.World.Player(); p != nil && p.Name != "" {
		return p.Name
	}
	return DefaultName
}

// layoutWords lays text's words out from (x, y), wrapping before right. A word marked with a
// leading * is a keyword, without its punctuation.
func layoutWords(text string, x, y, right int) []Word {
	var out []Word
	left := x
	for _, f := range strings.Fields(text) {
		kw := strings.HasPrefix(f, "*")
		f = strings.TrimPrefix(f, "*")
		w := len([]rune(f)) * CharWidth
		if x+w > right && x > left {
			x, y = left, y+LineHeight
		}
		word := Word{Rect: image.Rect(x, y, x+w, y+LineHeight), Text: f, Keyword: kw}
		out = append(out, word)
		x += w + CharWidth
	}
	return out
}

// keywordOf returns a marked word's keyword text, without the punctuation around it.
func keywordOf(w Word) string {
	return strings.TrimFunc(w.Text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// talkInput turns input into conversation commands. While a line waits: Escape or a right click
// leaves, a click on a marked word or its initial asks it next, and anything else goes on. While
// waiting for a word, one button or the field is highlighted, the first button at first: Up and
// Down (and Left and Right among the buttons) move the highlight, pointing at a button or the field highlights it, and typing fills the
// field and highlights it. Enter takes the highlighted button's keyword (or Bye), or the typed
// word with the field highlighted; clicking a button takes it. "Bye", Escape or a right click
// leaves.
func (v *Viewer) talkInput(in Input) []input.Command {
	tv := v.Talk()
	left := in.Left.Released
	cursor := tv.at(in.Cursor)
	if in.pressed(KeyEscape) || in.Right.Released {
		v.Typed, v.TalkSelected = "", 0
		return []input.Command{input.Leave{}}
	}
	if tv.Waiting {
		v.TalkSelected = 0
		if left {
			for _, w := range tv.Words {
				if w.Keyword && cursor.In(w.Rect) {
					return []input.Command{input.Continue{Ask: keywordOf(w)}}
				}
			}
			return []input.Command{input.Continue{}}
		}
		for _, r := range in.Chars {
			for _, w := range tv.Words {
				if k := keywordOf(w); w.Keyword && k != "" && unicode.ToUpper([]rune(k)[0]) == unicode.ToUpper(r) {
					return []input.Command{input.Continue{Ask: k}}
				}
			}
		}
		if in.anyPressed() {
			return []input.Command{input.Continue{}}
		}
		return nil
	}
	ask := func(b PanelButton) []input.Command {
		v.Typed, v.TalkSelected = "", 0
		if b.Name == "bye" {
			return []input.Command{input.Leave{}}
		}
		return []input.Command{input.Ask{Word: b.Label}}
	}
	field := len(tv.Buttons)
	sel := tv.Selected
	if in.Cursor != v.talkCursor {
		v.talkCursor = in.Cursor
		for i, b := range tv.Buttons {
			if cursor.In(b.Rect) {
				sel = i
			}
		}
		if cursor.In(tv.Field) {
			sel = field
		}
	}
	if left {
		for _, b := range tv.Buttons {
			if cursor.In(b.Rect) {
				return ask(b)
			}
		}
		v.TalkSelected = sel
		return nil
	}
	switch {
	case in.pressed(KeyArrowUp) || in.pressed(KeyArrowLeft) && sel < field:
		sel = max(0, sel-1)
	case in.pressed(KeyArrowDown) || in.pressed(KeyArrowRight) && sel < field:
		sel = min(field, sel+1)
	}
	if in.pressed(KeyBackspace) && v.Typed != "" {
		r := []rune(v.Typed)
		v.Typed = string(r[:len(r)-1])
	}
	for _, r := range in.Chars {
		if unicode.IsPrint(r) && len([]rune(v.Typed)) < maxField {
			v.Typed += string(r)
			sel = field
		}
	}
	v.TalkSelected = sel
	if in.pressed(KeyEnter) {
		if sel < field {
			return ask(tv.Buttons[sel])
		}
		word := strings.TrimSpace(v.Typed)
		v.Typed = ""
		switch {
		case word == "":
			return nil
		case strings.EqualFold(word, "bye"):
			v.TalkSelected = 0
			return []input.Command{input.Leave{}}
		}
		v.TalkSelected = 0
		return []input.Command{input.Ask{Word: word}}
	}
	return nil
}
