package inspect

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// KeywordWord returns keyword id's word, or the id when the pack has no such keyword.
func KeywordWord(p *content.Pack, id string) string {
	if k := p.Keywords; k != nil {
		for _, kw := range k.Keywords {
			if kw.ID == id {
				return Text(p, kw.Word)
			}
		}
	}
	return id
}

// Condition describes a conversation clause's condition in words.
func Condition(p *content.Pack, c content.Condition) string {
	var parts []string
	add := func(format string, args ...any) { parts = append(parts, fmt.Sprintf(format, args...)) }
	items := func(ids []string) string {
		names := make([]string, len(ids))
		for i, id := range ids {
			names[i] = ItemName(p, id)
		}
		return strings.Join(names, " and ")
	}
	if c.FirstMeeting {
		add("first meeting")
	}
	if c.Greeting {
		add("greeting")
	}
	if len(c.Asked) > 0 {
		words := make([]string, len(c.Asked))
		for i, id := range c.Asked {
			words[i] = strconv.Quote(KeywordWord(p, id))
		}
		add("asked %s", strings.Join(words, " or "))
	}
	if c.Never {
		add("never (only jumped to)")
	}
	if len(c.Carries) > 0 {
		add("carries %s", items(c.Carries))
	}
	if t := c.Counter; t != nil {
		switch {
		case t.Above != nil:
			add("%s > %d", t.Counter, *t.Above)
		case t.Equals != nil:
			add("%s = %d", t.Counter, *t.Equals)
		case t.Below != nil:
			add("%s < %d", t.Counter, *t.Below)
		}
	}
	if c.Dead != "" {
		add("%s dead", c.Dead)
	}
	if c.FullHealth {
		add("full health")
	}
	if cc := c.CarriesCount; cc != nil {
		add("carries %d %s", cc.Count, ItemName(p, cc.Item))
	}
	if c.SlotEmpty != nil {
		add("%s empty", SlotName(*c.SlotEmpty))
	}
	if sh := c.SlotHolds; sh != nil {
		add("%s holds %s", SlotName(sh.Slot), ItemName(p, sh.Item))
	}
	if c.Always {
		add("always")
	}
	if len(parts) == 0 {
		return "nothing"
	}
	return strings.Join(parts, ", ")
}

// TalkAction describes a conversation action briefly, for a clause's summary.
func TalkAction(p *content.Pack, a content.Action) string {
	switch {
	case a.Say != nil:
		return "say"
	case a.SayAs != nil:
		return "say as " + a.SayAs.Portrait
	case a.Offer != nil:
		return "offer " + strconv.Quote(KeywordWord(p, *a.Offer))
	case a.Take != nil:
		return "take " + ItemName(p, *a.Take)
	case a.Give != nil:
		return "give " + ItemName(p, a.Give.Item)
	case a.SetCounter != nil:
		return fmt.Sprintf("%s := %d", a.SetCounter.Counter, a.SetCounter.Value)
	case a.AddCounter != nil:
		return *a.AddCounter + " += 1"
	case a.Run != nil:
		return "run " + *a.Run
	case a.Jump != nil:
		return "jump " + strconv.Itoa(*a.Jump)
	case a.Teach != nil:
		return "teach " + *a.Teach
	case a.Travel != nil:
		return fmt.Sprintf("travel %s (%d, %d)", a.Travel.Level, a.Travel.Cell[0], a.Travel.Cell[1])
	case a.Menu != nil:
		return "menu " + *a.Menu
	case a.SetFrame != nil:
		return "frame " + strconv.Itoa(*a.SetFrame)
	case a.SetPortrait != nil:
		return "portrait " + *a.SetPortrait
	case a.Effect != nil:
		return "effect " + *a.Effect
	}
	if ActionKind(a) != "" {
		return Action(p, a)
	}
	// The rest are flags or need no more than their name: end, hostile, return and the like.
	b, _ := json.Marshal(a)
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) == nil {
		for k := range m {
			return strings.ReplaceAll(k, "_", " ")
		}
	}
	return "?"
}

// Clause describes a conversation clause in a line: its condition, then its actions, a run of
// the same action counted once.
func Clause(p *content.Pack, c content.Clause) string {
	var acts []string
	n := 0
	for i, a := range c.Do {
		s := TalkAction(p, a)
		n++
		if i+1 < len(c.Do) && TalkAction(p, c.Do[i+1]) == s {
			continue
		}
		if n > 1 {
			s += fmt.Sprintf(" ×%d", n)
		}
		acts, n = append(acts, s), 0
	}
	return Condition(p, c.If) + ": " + strings.Join(acts, ", ")
}

// ConversationLines describes conversation id of the thing template named name: a heading, a line
// for each clause, numbered as jumps number them, and the counters the clauses test or change
// with their values in counters. talked tells whether the player has talked to the template.
func ConversationLines(p *content.Pack, id, name string, talked bool, counters map[string]int) []Line {
	conv := p.Conversations[id]
	if conv == nil {
		return []Line{{Text: "Talks: " + id + " (not in the pack)"}}
	}
	lines := []Line{{Text: fmt.Sprintf("Talks: %s (%s)%s", id, name, onOff(talked, ", talked to", ""))}}
	var names []string
	count := func(c string) {
		if !slices.Contains(names, c) {
			names = append(names, c)
		}
	}
	for i, c := range conv.Clauses {
		for _, a := range c.Do {
			if a.SetCounter != nil {
				count(a.SetCounter.Counter)
			}
			if a.AddCounter != nil {
				count(*a.AddCounter)
			}
		}
		if t := c.If.Counter; t != nil {
			count(t.Counter)
		}
		lines = append(lines, Line{Text: fmt.Sprintf("  %d %s", i, Clause(p, c))})
	}
	if len(names) > 0 {
		slices.Sort(names)
		vals := make([]string, len(names))
		for i, c := range names {
			vals[i] = fmt.Sprintf("%s = %d", c, counters[c])
		}
		lines = append(lines, Line{Text: "  counters: " + strings.Join(vals, ", ")})
	}
	return lines
}
