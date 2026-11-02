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

// The conversation log window (docs/spec/dialogue.md, Engine model): the people talked with down
// the left, the chosen one's conversations newest first on the right, and a search field over
// them all. Typing searches; the left column then lists the matches, and choosing one shows its
// line in its conversation.

// Conversation log layout, in window pixels, which the window scales by up to talkScale.
const (
	logWidth     = 640
	logMinHeight = 300
	logPad       = 8
	logColumn    = 184
	logRowHeight = talkPortrait + 4
	logMatchRow  = LineHeight + 4
	logField     = 240
	logButton    = 60
)

// LogView is the conversation log laid out on the screen. Rect is on the screen and Scale its
// screen pixels per window pixel; every other rectangle is in window pixels from Rect's corner.
type LogView struct {
	Rect  image.Rectangle
	Scale int
	// Field is the search field and Typed what is typed in it; Close the button that closes.
	Field image.Rectangle
	Typed string
	Close PanelButton
	// Rows are the left column's: the people talked with, or the matches while a search is
	// typed. First is the index of the first row shown among them all, and Selected the chosen
	// row among those shown, -1 for none.
	Rows     []LogRow
	First    int
	Selected int
	// Pane is the right pane and Text its rows that show, scrolled by Scroll of PaneHeight.
	Pane       image.Rectangle
	Text       []LogText
	Scroll     int
	PaneHeight int
	// Empty is what the pane says when it shows nothing.
	Empty string
}

// LogRow is a row of the log's left column: a person, beside their latest portrait, or a match.
type LogRow struct {
	Rect        image.Rectangle
	Label       string
	Portrait    content.Image
	HasPortrait bool
	Mirrored    bool
}

// LogText is a row of the log's right pane: a conversation's heading, or a line laid out in words
// with its keywords marked. Focus marks the line a match chose.
type LogText struct {
	Words   []Word
	Heading bool
	Focus   bool
}

// logPerson is a template talked with, and the log's conversations with it, oldest first.
type logPerson struct {
	template string
	convs    []int
}

// logMatch is a line a search found: its conversation and entry in the world's log.
type logMatch struct {
	conv, entry int
	label       string
}

// logLine is a row of the pane before scrolling: its words, from the pane's top, and the line it
// shows (-1 for a heading).
type logLine struct {
	LogText
	conv, entry int
}

// OpenLog opens the conversation log, when playing, on the person talked with last.
func (v *Viewer) OpenLog() {
	if !v.Play || v.World.Player() == nil {
		return
	}
	v.LogOpen, v.LogTyped, v.logSelected, v.logScroll, v.logFocus = true, "", 0, 0, nil
	v.SpellsOpen = false
	if people := v.logPeople(); len(people) > 0 {
		v.logChosen = people[0].template
	}
}

// CloseLog closes the conversation log.
func (v *Viewer) CloseLog() { v.LogOpen, v.LogTyped, v.logFocus = false, "", nil }

// logPeople returns the templates talked with, the one talked with last first.
func (v *Viewer) logPeople() []logPerson {
	var out []logPerson
	at := map[string]int{}
	log := v.World.Log()
	for i := len(log) - 1; i >= 0; i-- {
		id := log[i].Template
		n, ok := at[id]
		if !ok {
			n = len(out)
			at[id] = n
			out = append(out, logPerson{template: id})
		}
		out[n].convs = append(out[n].convs, i)
	}
	for i := range out {
		slices.Reverse(out[i].convs)
	}
	return out
}

// logName returns template id's name, as the log shows it.
func (v *Viewer) logName(id string) string {
	if t := v.Pack.Things[id]; t != nil {
		if t.Name != "" {
			return inspect.Text(v.Pack, t.Name)
		}
		if c := v.Pack.Conversations[t.Conversation]; c != nil && c.Name != "" {
			return c.Name
		}
	}
	return id
}

// logEntryText returns entry e of conversation c as the log shows it: who says it, then the text
// with the player's name for @ and its keywords marked *.
func (v *Viewer) logEntryText(c *sim.Logged, e sim.LogEntry) string {
	switch e.Kind {
	case sim.LogQuestion:
		return v.playerName() + " asks: " + e.Text
	case sim.LogTeach:
		return "Taught the gesture " + strings.Join(strings.Split(e.Icons, ""), " ")
	}
	who := v.playerName()
	if e.Portrait != "" {
		who = v.logName(c.Template)
	}
	return who + ": " + v.lineText(e.Text)
}

// LogTime returns step as play time: hours, minutes and seconds.
func LogTime(step int64) string {
	s := step / sim.StepsPerSecond
	return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
}

// logHeading returns conversation c's heading: its level's name and when it began.
func (v *Viewer) logHeading(c *sim.Logged) string {
	name := c.Level
	if l := v.Pack.Levels[c.Level]; l != nil && l.Name != "" {
		name = inspect.Text(v.Pack, l.Name)
	}
	return name + ", " + LogTime(c.Step)
}

// logMatches returns the lines that hold the text typed, ignoring case, newest first.
func (v *Viewer) logMatches() []logMatch {
	query := strings.ToLower(strings.TrimSpace(v.LogTyped))
	if query == "" {
		return nil
	}
	var out []logMatch
	log := v.World.Log()
	for i := len(log) - 1; i >= 0; i-- {
		c := log[i]
		for j, e := range c.Entries {
			text := strings.ReplaceAll(v.logEntryText(c, e), "*", "")
			if strings.Contains(strings.ToLower(text), query) {
				out = append(out, logMatch{conv: i, entry: j, label: text})
			}
		}
	}
	return out
}

// logLines lays out the chosen person's conversations, newest first, wrapped to width.
func (v *Viewer) logLines(width int) ([]logLine, int) {
	var out []logLine
	y := 0
	for _, p := range v.logPeople() {
		if p.template != v.logChosen {
			continue
		}
		log := v.World.Log()
		for k := len(p.convs) - 1; k >= 0; k-- {
			i := p.convs[k]
			c := log[i]
			if len(out) > 0 {
				y += LineHeight / 2
			}
			h := Word{Rect: image.Rect(0, y, width, y+LineHeight), Text: v.logHeading(c)}
			out = append(out, logLine{LogText: LogText{Words: []Word{h}, Heading: true}, conv: i, entry: -1})
			y += LineHeight
			for j, e := range c.Entries {
				words := layoutWords(v.logEntryText(c, e), CharWidth, y, width)
				focus := v.logFocus != nil && *v.logFocus == [2]int{i, j}
				out = append(out, logLine{LogText: LogText{Words: words, Focus: focus}, conv: i, entry: j})
				if n := len(words); n > 0 {
					y = words[n-1].Rect.Max.Y
				}
			}
		}
	}
	return out, y
}

// Log returns the conversation log laid out, or nil when it is closed.
func (v *Viewer) Log() *LogView {
	if !v.LogOpen {
		return nil
	}
	bottom := v.Screen.Y - Margin
	if h := v.Hotbar(); h != nil {
		bottom = h.Rect.Min.Y - Margin
	}
	avail := max(bottom-2*Margin, 1)
	lv := &LogView{Typed: v.LogTyped, Selected: -1}
	lv.Scale = max(1, min(talkScale, (v.Screen.X-2*Margin)/logWidth, avail/logMinHeight))
	size := image.Pt(logWidth, max(logMinHeight, avail/lv.Scale))
	corner := image.Pt((v.Screen.X-size.X*lv.Scale)/2, max(Margin, (bottom-size.Y*lv.Scale)/2))
	lv.Rect = image.Rectangle{corner, corner.Add(size.Mul(lv.Scale))}

	lv.Close = PanelButton{Rect: image.Rect(size.X-logPad-logButton, logPad, size.X-logPad, logPad+LineHeight+4), Name: "close", Label: "Close"}
	fx := logPad + 6*CharWidth
	lv.Field = image.Rect(fx, logPad, fx+logField, logPad+LineHeight+4)
	top := lv.Field.Max.Y + logPad
	body := image.Rect(logPad, top, size.X-logPad, size.Y-logPad)

	// The left column: the matches while a search is typed, else the people.
	rowH := logRowHeight
	if v.LogTyped != "" {
		rowH = logMatchRow
	}
	fits := max(1, body.Dy()/rowH)
	var labels []string
	people := v.logPeople()
	if v.LogTyped != "" {
		matches := v.logMatches()
		for _, m := range matches {
			labels = append(labels, m.label)
		}
		lv.Selected = min(v.logSelected, len(labels)-1)
		if len(matches) == 0 {
			lv.Empty = "Nothing found."
		}
	} else {
		for i, p := range people {
			labels = append(labels, v.logName(p.template))
			if p.template == v.logChosen {
				lv.Selected = i
			}
		}
		if len(people) == 0 {
			lv.Empty = "No one has been talked with yet."
		}
	}
	first := max(0, lv.Selected-fits+1)
	lv.First = first
	for i := first; i < len(labels) && i-first < fits; i++ {
		y := body.Min.Y + (i-first)*rowH
		r := LogRow{Rect: image.Rect(body.Min.X, y, body.Min.X+logColumn, y+rowH-2), Label: Clip(labels[i], (logColumn-8)/CharWidth)}
		if v.LogTyped == "" {
			r.Label = Clip(labels[i], (logColumn-talkPortrait-12)/CharWidth)
			if pt := v.Pack.Portraits[v.logPortrait(people[i])]; pt != nil {
				r.Portrait, r.HasPortrait = pt.Image, true
			}
		}
		lv.Rows = append(lv.Rows, r)
	}
	if lv.Selected >= 0 {
		lv.Selected -= first
	}

	// The right pane: the chosen person's conversations, scrolled.
	lv.Pane = image.Rect(body.Min.X+logColumn+logPad, body.Min.Y, body.Max.X, body.Max.Y)
	lines, height := v.logLines(lv.Pane.Dx() - CharWidth)
	lv.PaneHeight = height
	lv.Scroll = max(0, min(v.logScroll, height-lv.Pane.Dy()))
	for _, l := range lines {
		t := l.LogText
		t.Words = slices.Clone(t.Words)
		in := true
		for i := range t.Words {
			r := t.Words[i].Rect.Add(image.Pt(lv.Pane.Min.X, lv.Pane.Min.Y-lv.Scroll))
			t.Words[i].Rect = r
			in = in && r.Min.Y >= lv.Pane.Min.Y && r.Max.Y <= lv.Pane.Max.Y
		}
		if in && len(t.Words) > 0 {
			lv.Text = append(lv.Text, t)
		}
	}
	return lv
}

// logPortrait returns the portrait the log shows for person p: the latest it was shown with.
func (v *Viewer) logPortrait(p logPerson) string {
	log := v.World.Log()
	return log[p.convs[len(p.convs)-1]].Portrait
}

// Screen returns the screen rectangle of window rectangle r.
func (lv *LogView) Screen(r image.Rectangle) image.Rectangle {
	return image.Rectangle{lv.Rect.Min.Add(r.Min.Mul(lv.Scale)), lv.Rect.Min.Add(r.Max.Mul(lv.Scale))}
}

// at returns the window pixel under screen point pt.
func (lv *LogView) at(pt image.Point) image.Point { return pt.Sub(lv.Rect.Min).Div(lv.Scale) }

// logChoose chooses row i of the left column: a person, or a match, whose line it shows.
func (v *Viewer) logChoose(i int) {
	if v.LogTyped == "" {
		if people := v.logPeople(); i >= 0 && i < len(people) {
			v.logChosen, v.logScroll, v.logFocus, v.logSelected = people[i].template, 0, nil, i
		}
		return
	}
	matches := v.logMatches()
	if i < 0 || i >= len(matches) {
		return
	}
	m := matches[i]
	v.logSelected = i
	v.logChosen = v.World.Log()[m.conv].Template
	focus := [2]int{m.conv, m.entry}
	v.logFocus = &focus
	// Scroll the line to a third of the way down the pane.
	if lv := v.Log(); lv != nil {
		lines, _ := v.logLines(lv.Pane.Dx() - CharWidth)
		for _, l := range lines {
			if l.conv == m.conv && l.entry == m.entry && len(l.Words) > 0 {
				v.logScroll = max(0, l.Words[0].Rect.Min.Y-lv.Pane.Dy()/3)
			}
		}
	}
}

// logInput takes input while the log is open. Typing searches, Backspace takes a letter off;
// Up and Down choose among the left column's rows, and Enter or a click chooses a match. The
// wheel, Page Up and Page Down scroll the pane. Escape clears the search, leaving the pane as it
// is, or closes the log, as a right click or Close does.
func (v *Viewer) logInput(in Input) {
	lv := v.Log()
	if lv == nil {
		return
	}
	if in.pressed(KeyEscape) && v.LogTyped != "" {
		v.LogTyped, v.logSelected = "", 0
		return
	}
	cursor := lv.at(in.Cursor)
	if in.pressed(KeyEscape) || in.Right.Released || in.Left.Released && cursor.In(lv.Close.Rect) {
		v.CloseLog()
		return
	}
	if in.Left.Released {
		for i, r := range lv.Rows {
			if cursor.In(r.Rect) {
				v.logChoose(lv.First + i)
				return
			}
		}
	}
	typed := false
	if in.pressed(KeyBackspace) && v.LogTyped != "" {
		r := []rune(v.LogTyped)
		v.LogTyped, typed = string(r[:len(r)-1]), true
	}
	for _, r := range in.Chars {
		if unicode.IsPrint(r) && len([]rune(v.LogTyped)) < maxField {
			v.LogTyped, typed = v.LogTyped+string(r), true
		}
	}
	if typed {
		v.logSelected = 0
		return
	}
	sel := v.logSelected
	if v.LogTyped == "" {
		sel = v.logChosenIndex()
	}
	switch {
	case in.pressed(KeyArrowUp):
		v.logStep(sel - 1)
	case in.pressed(KeyArrowDown):
		v.logStep(sel + 1)
	case in.pressed(KeyEnter) && v.LogTyped != "":
		v.logChoose(sel)
	case in.pressed(KeyPageDown):
		v.logScroll = lv.Scroll + lv.Pane.Dy()*3/4
	case in.pressed(KeyPageUp):
		v.logScroll = max(0, lv.Scroll-lv.Pane.Dy()*3/4)
	case in.Wheel != 0:
		v.logScroll = max(0, lv.Scroll-int(in.Wheel*3*LineHeight))
	}
	v.logScroll = max(0, min(v.logScroll, lv.PaneHeight-lv.Pane.Dy()))
}

// logChosenIndex returns the chosen person's place among the people, -1 for none.
func (v *Viewer) logChosenIndex() int {
	return slices.IndexFunc(v.logPeople(), func(p logPerson) bool { return p.template == v.logChosen })
}

// logStep moves the left column's choice to row i: a person is chosen at once, a match only
// highlighted until Enter.
func (v *Viewer) logStep(i int) {
	n := len(v.logPeople())
	if v.LogTyped != "" {
		n = len(v.logMatches())
	}
	if n == 0 {
		return
	}
	i = max(0, min(n-1, i))
	if v.LogTyped == "" {
		v.logChoose(i)
		return
	}
	v.logSelected = i
}
