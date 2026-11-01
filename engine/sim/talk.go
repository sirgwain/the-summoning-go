package sim

import (
	"strings"
	"unicode"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// Conversations (docs/spec/dialogue.md): a conversation runs passes over its clauses, one with
// no word (the greeting) and one for each word the player asks. A pass stops at each text said,
// until the player goes on, so the presentation shows them one at a time.

// Conversation limits.
const (
	maxOffered = 6
	// fullHeal is what healing fully adds.
	fullHeal = 3000
	// counterMax is a counter's largest value; counters hold a byte.
	counterMax = 255
	// matchLetters is how many letters of a word match a keyword.
	matchLetters = 4
)

// Line kinds.
const (
	// LineSay is a text said beside a portrait: Portrait, or the player's when empty.
	LineSay = "say"
	// LineTeach shows gesture icons as they are taught.
	LineTeach = "teach"
	// LineMenu opens one of the game's menus (Menu, a content.Menu… name); going on returns
	// from it.
	LineMenu = "menu"
)

// Line is what a conversation shows and waits on.
type Line struct {
	Kind string `json:"kind"`
	// Text is a text key; Portrait the portrait beside it, empty for the player's.
	Text     string `json:"text,omitempty"`
	Portrait string `json:"portrait,omitempty"`
	// Icons are the icons taught.
	Icons string `json:"icons,omitempty"`
	Menu  string `json:"menu,omitempty"`
}

// Talk is an open conversation.
type Talk struct {
	Def *content.Conversation
	// Template is whose conversation it is; Speaker the thing spoken to, nil for none.
	Template *content.Thing
	Speaker  *Thing
	// Portrait is the portrait shown for the speaker.
	Portrait string
	// Offered are the keywords offered, "" for a free place.
	Offered [maxOffered]string
	// Line is the line waiting for the player to go on, nil when none is.
	Line *Line
	// Transcript is what has been asked and said, as text keys, "Q: " and a word for questions.
	Transcript []string

	talked bool
	// pass is the pass running, nil between passes; queued a word to ask when it ends.
	pass   *pass
	queued string
	added  int
	done   bool
	// after is where the player goes when the conversation ends; back goes back through a
	// portal's way; use finishes using the object that started it.
	after *content.Destination
	back  bool
	use   func()
	// ending is how play ends when the conversation does, for the ending power.
	ending *Ended
}

// pass is a pass running: the keyword it asks ("" for the greeting), the clause whose actions
// run and the next of them, or with no clause the clause to evaluate from.
type pass struct {
	keyword string
	clause  int
	action  int
	running bool
}

// Record is what the world keeps about a thing template: whether the player has talked to it,
// whether one of its creatures has died, and its conversation portrait if changed.
type Record struct {
	Talked   bool   `json:"talked,omitempty"`
	Dead     bool   `json:"dead,omitempty"`
	Portrait string `json:"portrait,omitempty"`
}

// Talk returns the open conversation, or nil.
func (w *World) Talk() *Talk { return w.talk }

// Records returns the world's records of thing templates, by template id.
func (w *World) Records() map[string]*Record { return w.records }

// Counters returns the world's counters, by name.
func (w *World) Counters() map[string]int { return w.counters }

// record returns template id's record, making it the first time.
func (w *World) record(id string) *Record {
	r := w.records[id]
	if r == nil {
		r = &Record{}
		w.records[id] = r
	}
	return r
}

// setCounter sets a counter to a byte's value.
func (w *World) setCounter(name string, n int) {
	n &= counterMax
	if n == 0 {
		delete(w.counters, name)
		return
	}
	w.counters[name] = n
}

// FindKeyword returns the keyword a typed word asks: the first whose first four letters it
// shares, ignoring case, up to a space or punctuation (docs/spec/dialogue.md, Keywords).
func FindKeyword(p *content.Pack, text map[string]string, word string) (content.Keyword, bool) {
	key := matchKey(word)
	if key == "" || p.Keywords == nil {
		return content.Keyword{}, false
	}
	for _, k := range p.Keywords.Keywords {
		if matchKey(text[k.Word]) == key {
			return k, true
		}
	}
	return content.Keyword{}, false
}

// matchKey is a word's first four letters in lower case, up to a space or punctuation.
func matchKey(word string) string {
	var b strings.Builder
	for _, r := range word {
		if b.Len() >= matchLetters || r == ' ' || unicode.IsPunct(r) {
			break
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// texts returns the pack's first language's strings, which keywords are matched in.
func (w *World) texts() map[string]string {
	for _, id := range sortedKeys(w.pack.Texts) {
		return w.pack.Texts[id].Strings
	}
	return nil
}

// talkTo starts a conversation with thing id of the current level, or hears mouth id, or with no
// id the nearest that talks, or the nearest mouth's message (docs/spec/dialogue.md, Talking).
func (w *World) talkTo(id string) {
	p, l := w.player, w.Current()
	if p.Dead {
		w.refuse("the player is dead")
		return
	}
	if f := l.Feature(id); id != "" && f != nil && f.Kind == content.FeatureMouth {
		w.use(l, f.Cell)
		return
	}
	if id != "" {
		t := l.Thing(id)
		if t == nil || !w.talks(t) {
			w.refuse(id + " does not talk")
			return
		}
		w.startTalk(t.Template, t)
		return
	}
	var best *Thing
	bestD := 0
	for _, t := range l.Things {
		if !w.talks(t) || !l.Visible(t.Cell()) {
			continue
		}
		if d := abs(t.U-p.U) + abs(t.V-p.V); best == nil || d < bestD {
			best, bestD = t, d
		}
	}
	var mouth *Feature
	for _, f := range l.Features {
		if f.Kind != content.FeatureMouth || !l.Visible(f.Cell) {
			continue
		}
		u, v := cellCenter(f.Cell)
		if d := abs(u-p.U) + abs(v-p.V); best == nil && mouth == nil || d < bestD {
			mouth, bestD = f, d
		}
	}
	switch {
	case mouth != nil:
		w.use(l, mouth.Cell)
	case best != nil:
		w.startTalk(best.Template, best)
	default:
		w.refuse("no one to talk to")
	}
}

// CanTalk reports whether thing t can be talked to now.
func (w *World) CanTalk(t *Thing) bool { return w.talks(t) }

// talks reports whether thing t can be talked to now.
func (w *World) talks(t *Thing) bool {
	return alive(t) && t.Template.Has(content.BehaviorTalks) && !t.Hostile && !t.Mute && w.pack.Conversations[t.Template.Conversation] != nil
}

// startTalk opens template tmpl's conversation, held by speaker (nil for none), and runs its
// greeting.
func (w *World) startTalk(tmpl *content.Thing, speaker *Thing) *Talk {
	def := w.pack.Conversations[tmpl.Conversation]
	if def == nil || w.talk != nil {
		return nil
	}
	rec := w.record(tmpl.ID)
	portrait := tmpl.TalkPortrait
	if rec.Portrait != "" {
		portrait = rec.Portrait
	}
	if p := w.player; p != nil && portrait != "" && portrait == p.Portrait {
		if pt := w.pack.Portraits[portrait]; pt != nil && pt.Talk != "" {
			portrait = pt.Talk
		}
	}
	t := &Talk{Def: def, Template: tmpl, Speaker: speaker, Portrait: portrait, talked: rec.Talked}
	w.talk = t
	w.emit(Event{Kind: EventTalk, Level: w.current, Thing: speakerID(speaker), Item: def.ID, Detail: "started"})
	t.pass = &pass{}
	w.runTalk()
	return t
}

func speakerID(t *Thing) string {
	if t == nil {
		return ""
	}
	return t.ID
}

// talkCommand carries out a command given while a conversation is open, and reports whether it
// was one.
func (w *World) talkCommand(cmd input.Command) bool {
	t := w.talk
	switch c := cmd.(type) {
	case input.Continue:
		if t.Line == nil {
			w.refuse("nothing to go on from")
			return true
		}
		t.Line = nil
		if c.Ask != "" {
			t.queued = c.Ask
		}
		w.runTalk()
	case input.Ask:
		if t.Line != nil || t.pass != nil {
			w.refuse("the conversation is not waiting for a word")
			return true
		}
		w.ask(c.Word)
	case input.Leave:
		t.done = true
		w.runTalk()
	default:
		return false
	}
	return true
}

// ask asks a typed word: a keyword starts a pass with it; anything else is refused.
func (w *World) ask(word string) {
	t := w.talk
	k, ok := FindKeyword(w.pack, w.texts(), word)
	if !ok {
		w.emit(Event{Kind: EventRefused, Level: w.current, Detail: "not a keyword: " + word})
		return
	}
	t.Transcript = append(t.Transcript, "Q: "+word)
	t.pass = &pass{keyword: k.ID}
	w.runTalk()
}

// runTalk runs the conversation on until it waits for the player or ends.
func (w *World) runTalk() {
	t := w.talk
	// The loop stops when the conversation is no longer the world's: play has ended.
	for t != nil && w.talk == t && !t.done && t.Line == nil {
		if t.pass == nil {
			if t.queued == "" {
				return
			}
			word := t.queued
			t.queued = ""
			w.ask(word)
			continue
		}
		w.stepPass()
	}
	if t != nil && w.talk == t && t.done {
		w.endTalk()
	}
}

// stepPass takes the running pass one step: it evaluates conditions from its clause until one
// holds, or runs that clause's next action.
func (w *World) stepPass() {
	t := w.talk
	ps := t.pass
	clauses := t.Def.Clauses
	if !ps.running {
		for ; ps.clause < len(clauses); ps.clause++ {
			holds, stop := w.holds(clauses[ps.clause].If, ps.keyword)
			if stop {
				t.pass = nil
				return
			}
			if holds {
				ps.running, ps.action = true, 0
				return
			}
		}
		t.pass = nil
		return
	}
	acts := clauses[ps.clause].Do
	if ps.action >= len(acts) {
		t.pass = nil
		return
	}
	a := acts[ps.action]
	ps.action++
	if a.Jump != nil {
		ps.clause, ps.running = *a.Jump, false
		return
	}
	w.talkAction(a)
}

// holds evaluates condition c for a pass asking keyword ("" for the greeting); stop is true for
// a condition that stops the pass. Asking a listed keyword shows the question, takes the listed
// keywords off the offered list and says the keyword's answer.
func (w *World) holds(c content.Condition, keyword string) (holds, stop bool) {
	t, p := w.talk, w.player
	greeting := keyword == ""
	switch {
	case c.FirstMeeting:
		return greeting && !t.talked, false
	case c.Greeting:
		return greeting, false
	case len(c.Asked) > 0:
		found := false
		for _, k := range c.Asked {
			found = found || k == keyword
		}
		if greeting || !found {
			return false, false
		}
		for _, k := range c.Asked {
			for i, o := range t.Offered {
				if o == k {
					t.Offered[i] = ""
				}
			}
		}
		if kw, ok := w.keyword(keyword); ok && kw.Answer != "" {
			if s := w.texts()[kw.Answer]; s != "" {
				w.say(Line{Kind: LineSay, Text: kw.Answer})
			}
		}
		return true, false
	case c.Never:
		return false, true
	case len(c.Carries) > 0:
		for _, it := range c.Carries {
			if w.countCarried(it) == 0 {
				return false, false
			}
		}
		return true, false
	case c.Counter != nil:
		n := w.counters[c.Counter.Counter]
		switch {
		case c.Counter.Above != nil:
			return n > *c.Counter.Above, false
		case c.Counter.Equals != nil:
			return n == *c.Counter.Equals, false
		case c.Counter.Below != nil:
			return n < *c.Counter.Below, false
		}
	case c.Dead != "":
		return w.record(c.Dead).Dead, false
	case c.FullHealth:
		return p != nil && p.HitPoints >= p.MaxHitPoints, false
	case c.CarriesCount != nil:
		return w.countCarried(c.CarriesCount.Item) >= c.CarriesCount.Count, false
	case c.SlotEmpty != nil:
		return p != nil && p.Slot(w.pack, *c.SlotEmpty) == nil, false
	case c.SlotHolds != nil:
		o := p.Slot(w.pack, c.SlotHolds.Slot)
		return o != nil && o.Item == c.SlotHolds.Item, false
	case c.Always:
		return true, false
	}
	return false, false
}

// keyword returns keyword id.
func (w *World) keyword(id string) (content.Keyword, bool) {
	if w.pack.Keywords != nil {
		for _, k := range w.pack.Keywords.Keywords {
			if k.ID == id {
				return k, true
			}
		}
	}
	return content.Keyword{}, false
}

// say shows a line and waits for the player to go on.
func (w *World) say(l Line) {
	t := w.talk
	t.Line = &l
	if l.Text != "" {
		t.Transcript = append(t.Transcript, l.Text)
	}
	w.emit(Event{Kind: EventTalk, Level: w.current, Text: l.Text, Detail: l.Kind})
}

// talkAction runs one action of a conversation.
func (w *World) talkAction(a content.Action) {
	t, p, l := w.talk, w.player, w.Current()
	switch {
	case a.Say != nil:
		w.say(Line{Kind: LineSay, Text: *a.Say, Portrait: t.Portrait})
	case a.SayAs != nil:
		w.say(Line{Kind: LineSay, Text: a.SayAs.Text, Portrait: a.SayAs.Portrait})
	case a.Offer != nil:
		w.offer(*a.Offer)
	case a.ForgetOffers:
		t.Offered = [maxOffered]string{}
	case a.Take != nil:
		w.takeCarried(*a.Take)
	case a.Give != nil:
		if o := w.object(*a.Give); o != nil {
			w.give(o)
		}
	case a.SetCounter != nil:
		w.setCounter(a.SetCounter.Counter, a.SetCounter.Value)
	case a.AddCounter != nil:
		w.setCounter(*a.AddCounter, w.counters[*a.AddCounter]+1)
	case a.End:
		t.done = true
	case a.RemoveSpeaker:
		if t.Speaker != nil && l.Thing(t.Speaker.ID) == t.Speaker {
			w.removeThing(l, t.Speaker, "removed")
		}
	case a.Hostile:
		if t.Speaker != nil {
			for _, x := range l.Things {
				if x.Template == t.Speaker.Template {
					w.turnHostile(x)
					x.Mute = true
				}
			}
		}
	case a.SetFrame != nil:
		if t.Speaker != nil {
			f := *a.SetFrame
			t.Speaker.Frame = &f
		}
	case a.HealFully:
		if p != nil {
			p.Poisoned, p.Confusion = false, 0
			w.heal(fullHeal)
		}
	case a.Run != nil:
		w.run(l, *a.Run, 0)
	case a.Teach != nil:
		if p != nil {
			p.Magic.teach(*a.Teach)
		}
		w.say(Line{Kind: LineTeach, Icons: *a.Teach})
	case a.SetPortrait != nil:
		t.Portrait = *a.SetPortrait
		w.record(t.Template.ID).Portrait = *a.SetPortrait
	case a.Effect != nil:
		cell := content.Cell{}
		if t.Speaker != nil {
			cell = t.Speaker.Cell()
		}
		w.emit(Event{Kind: EventEffect, Level: l.ID, Cell: cell, Thing: speakerID(t.Speaker), Detail: *a.Effect})
	case a.Travel != nil:
		d := *a.Travel
		t.after = &d
	case a.Return:
		t.back = true
	case a.Ending:
		w.closing()
	case a.Menu != nil:
		w.say(Line{Kind: LineMenu, Menu: *a.Menu})
	case a.Transcript != nil:
		w.emit(Event{Kind: EventNotYet, Level: l.ID, Detail: "printing the transcript"})
	}
}

// offer adds keyword k to the offered list: in its old place's stead, the first free place, or
// else over the places this pass has filled.
func (w *World) offer(k string) {
	t := w.talk
	for i, o := range t.Offered {
		if o == k {
			t.Offered[i] = ""
		}
	}
	for i, o := range t.Offered {
		if o == "" {
			t.Offered[i] = k
			return
		}
	}
	t.Offered[t.added%maxOffered] = k
	t.added++
}

// endTalk closes the conversation: the template is marked talked to, the after-conversation
// counter cleared, an object waiting on it used, and the player taken where it says.
func (w *World) endTalk() {
	t := w.talk
	w.talk = nil
	w.record(t.Template.ID).Talked = true
	if c := w.pack.Manifest.Counters; c != nil && c.AfterConversation != "" {
		w.setCounter(c.AfterConversation, 0)
	}
	w.emit(Event{Kind: EventTalk, Level: w.current, Thing: speakerID(t.Speaker), Item: t.Def.ID, Detail: "ended"})
	if t.use != nil {
		t.use()
	}
	if t.ending != nil {
		w.end(*t.ending)
		return
	}
	p := w.player
	switch {
	case p == nil:
	case t.back && p.Return != nil:
		to := *p.Return
		p.Return = nil
		w.travel(&to)
	case t.after != nil:
		w.travel(t.after)
	}
}

// countCarried counts the player's objects of item, in every slot and inside containers.
func (w *World) countCarried(item string) int {
	p := w.player
	if p == nil {
		return 0
	}
	var count func(o *Object) int
	count = func(o *Object) int {
		n := 0
		if o.Item == item {
			n++
		}
		for _, c := range o.Contents {
			n += count(c)
		}
		return n
	}
	n := 0
	for _, o := range p.Slots {
		if o != nil {
			n += count(o)
		}
	}
	return n
}

// takeCarried takes one object of item from what the player carries: the slots in order, each
// searched depth first.
func (w *World) takeCarried(item string) bool {
	p := w.player
	if p == nil {
		return false
	}
	var from func(list []*Object) ([]*Object, bool)
	from = func(list []*Object) ([]*Object, bool) {
		for i, o := range list {
			if o.Item == item {
				return append(list[:i:i], list[i+1:]...), true
			}
			if rest, ok := from(o.Contents); ok {
				o.Contents = rest
				return list, true
			}
		}
		return list, false
	}
	for i, o := range p.Slots {
		if o == nil {
			continue
		}
		if o.Item == item {
			p.Slots[i] = nil
			if i >= Shelves {
				w.takeOff(o)
			}
			return true
		}
		if rest, ok := from(o.Contents); ok {
			o.Contents = rest
			return true
		}
	}
	return false
}

// give gives the player object o: to the first empty shelf, or else down at its feet.
func (w *World) give(o *Object) {
	p, l := w.player, w.Current()
	if p == nil {
		return
	}
	for i := range Shelves {
		if p.Slots[i] == nil {
			p.Slots[i] = o
			w.emit(Event{Kind: EventObject, Level: l.ID, Cell: p.Cell(), Item: o.Item, Detail: "given"})
			return
		}
	}
	w.drop(l, p.Cell(), o, byWorld, 0)
}

// EventShowMap asks the presentation to show the level map.
const EventShowMap = "show_map"

// showMap shows the current level's map, or what the level shows in its stead: a message or a
// conversation (docs/spec/world.md, Level map).
func (w *World) showMap() {
	l := w.Current()
	m := l.Def.Map
	switch {
	case m == nil:
		w.emit(Event{Kind: EventShowMap, Level: l.ID})
	case m.Message != "":
		w.emit(Event{Kind: EventMessage, Level: l.ID, Text: m.Message, Detail: "map refused"})
	default:
		if t := w.pack.Things[m.Conversation]; t != nil {
			w.startTalk(t, nil)
		}
	}
}

// MapRefused reports whether the current level refuses its map, which hides the minimap.
func (w *World) MapRefused() bool { return w.Current().Def.Map != nil }
