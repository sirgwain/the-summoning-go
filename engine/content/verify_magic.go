package content

import (
	"reflect"
	"strconv"
	"strings"
)

// Which actions each kind of action list may use: a level's procedures, a spell's effects, a
// conversation's clauses, by their JSON names.
var (
	levelOps = opSet("on", "off", "toggle", "set_hook", "set_threshold", "set_accepts", "set_message",
		"roll", "stop", "shoot_item", "shoot_spell", "create", "set_floor", "set_wall")
	spellOps = opSet("missile", "fire_item", "timed", "conjure", "portal", "teleport_to_mark", "dispel",
		"join", "freeze", "shape_change", "transmute", "cure_poison", "restore", "heal", "converse", "set_counter")
	talkOps = opSet("say", "say_as", "offer", "forget_offers", "take", "give", "set_counter",
		"add_counter", "end", "remove_speaker", "hostile", "set_frame", "heal_fully", "run", "teach",
		"set_portrait", "effect", "travel", "return", "jump", "ending", "menu", "transcript")
)

func opSet(names ...string) map[string]bool {
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}
	return m
}

// Ops returns the JSON names of the operations set in a; a well-formed action has one.
func (a Action) Ops() []string {
	var ops []string
	v, t := reflect.ValueOf(a), reflect.TypeOf(a)
	for i := range t.NumField() {
		if !v.Field(i).IsZero() {
			ops = append(ops, strings.Split(t.Field(i).Tag.Get("json"), ",")[0])
		}
	}
	return ops
}

// actionKind reports whether a sets exactly one operation, and one that allowed lets it use.
func (v *verifier) actionKind(owner string, a Action, allowed map[string]bool) bool {
	ops := a.Ops()
	switch {
	case len(ops) != 1:
		v.fail("%s: %d operations %v, want 1", owner, len(ops), ops)
		return false
	case !allowed[ops[0]]:
		v.fail("%s: %s cannot be used here", owner, ops[0])
		return false
	}
	return true
}

func (v *verifier) spellRef(owner, id string) {
	if _, ok := v.p.Spells[id]; !ok {
		v.fail("%s: spell %q not found", owner, id)
	}
}

func (v *verifier) conversationRef(owner, id string) {
	if _, ok := v.p.Conversations[id]; !ok {
		v.fail("%s: conversation %q not found", owner, id)
	}
}

func (v *verifier) thingRef(owner, id string) *Thing {
	t, ok := v.p.Things[id]
	if !ok {
		v.fail("%s: thing %q not found", owner, id)
	}
	return t
}

// icons checks a string of gesture icons.
func (v *verifier) icons(owner, icons string) {
	for _, c := range icons {
		if c < 'A' || c >= 'A'+GestureIcons {
			v.fail("%s: gesture icon %q", owner, c)
		}
	}
}

func (v *verifier) pointers() {
	pt := v.p.Pointers
	if pt == nil {
		return
	}
	v.frame("pointers.json arrow", pt.Arrow.Sprite, pt.Arrow.Frame)
	for _, im := range []*Image{pt.Talk, pt.Wait} {
		if im != nil {
			v.frame("pointers.json", im.Sprite, im.Frame)
		}
	}
}

func (v *verifier) magic() {
	m := v.p.Magic
	if m == nil {
		return
	}
	if len(m.Icons) != GestureIcons {
		v.fail("magic.json: %d icons, want %d", len(m.Icons), GestureIcons)
	}
	for _, im := range m.Icons {
		v.frame("magic.json icon", im.Sprite, im.Frame)
	}
	if m.Panel != nil {
		v.frame("magic.json panel", m.Panel.Sprite, m.Panel.Frame)
	}
	if len(m.Schools) != Schools {
		v.fail("magic.json: %d schools, want %d", len(m.Schools), Schools)
	}
	for _, s := range m.Schools {
		v.text("magic.json", s)
	}
}

func (v *verifier) spell(s *Spell) {
	owner := "spell " + s.ID
	v.text(owner, s.Name)
	if s.School < 0 || s.School >= Schools {
		v.fail("%s: school %d", owner, s.School)
	}
	if s.Cost < 0 || s.PowerBonus < 0 || s.Practice < 0 {
		v.fail("%s: negative cost, bonus or practice", owner)
	}
	if len(s.Gesture) > 9 {
		v.fail("%s: a gesture of %d icons", owner, len(s.Gesture))
	}
	v.icons(owner, s.Gesture)
	if s.Icon != nil {
		v.frame(owner+" icon", s.Icon.Sprite, s.Icon.Frame)
	}
	if n := s.Needs; n != nil {
		set := 0
		for _, ok := range []bool{n.FloorAhead > 0, n.Mark != "", n.SpellNear > 0, len(n.Holding) > 0, n.CreatureNear > 0, n.Poisoned, n.Fatigued, n.Hurt} {
			if ok {
				set++
			}
		}
		if set != 1 {
			v.fail("%s: needs %d conditions, want 1", owner, set)
		}
		for _, it := range n.Holding {
			v.item(owner, it)
		}
	}
	for i, a := range s.Effects {
		v.spellEffect(owner+" effect "+itoa(i), a)
	}
}

func (v *verifier) spellEffect(owner string, a Action) {
	if !v.actionKind(owner, a, spellOps) {
		return
	}
	switch {
	case a.Missile != nil:
		m := a.Missile
		v.frames(owner, m.Sprite)
		v.effect(owner+" sound", m.Sound)
		if m.Cloud != "" {
			v.frames(owner+" cloud", m.Cloud)
		}
		switch m.Hurts {
		case "", HurtsFire, HurtsLightning, HurtsPoison:
		default:
			v.fail("%s: unknown damage %q", owner, m.Hurts)
		}
		if m.Bane != "" {
			v.thingRef(owner, m.Bane)
		}
	case a.FireItem != nil:
		v.item(owner, *a.FireItem)
	case a.Timed != nil:
		if !timedEffects[*a.Timed] {
			v.fail("%s: unknown timed effect %q", owner, *a.Timed)
		}
	case a.Conjure != nil:
		if t := v.thingRef(owner, a.Conjure.Thing); t != nil && t.Kind != ThingConjured {
			v.fail("%s: %s is not a conjured thing", owner, t.ID)
		}
	case a.Portal != nil:
		if t := v.thingRef(owner, a.Portal.Thing); t != nil && t.Kind != ThingPortal {
			v.fail("%s: %s is not a portal", owner, t.ID)
		}
		v.destination(owner, a.Portal.To)
	case a.Join != nil:
		if len(a.Join.Items) != 2 {
			v.fail("%s: joins %d items, want 2", owner, len(a.Join.Items))
		}
		for _, it := range append(a.Join.Items, a.Join.Into) {
			v.item(owner, it)
		}
	case a.Converse != nil:
		if t := v.thingRef(owner, *a.Converse); t != nil && t.Conversation == "" {
			v.fail("%s: thing %s holds no conversation", owner, t.ID)
		}
	case a.Transmute != nil:
		v.item(owner, a.Transmute.Holder)
		v.item(owner, a.Transmute.Otherwise)
		for _, from := range sortedKeys(a.Transmute.Table) {
			v.item(owner, from)
			v.item(owner, a.Transmute.Table[from])
		}
	}
}

func (v *verifier) destination(owner string, d Destination) {
	l, ok := v.p.Levels[d.Level]
	switch {
	case !ok:
		v.fail("%s: level %q not found", owner, d.Level)
	case !l.InBounds(d.Cell):
		v.fail("%s: cell %v outside %s", owner, d.Cell, d.Level)
	}
}

func (v *verifier) keyword(owner, id string) {
	if v.p.Keywords != nil {
		for _, k := range v.p.Keywords.Keywords {
			if k.ID == id {
				return
			}
		}
	}
	v.fail("%s: keyword %q not found", owner, id)
}

func (v *verifier) keywords() {
	k := v.p.Keywords
	if k == nil {
		return
	}
	seen := map[string]bool{}
	for _, kw := range k.Keywords {
		if seen[kw.ID] || kw.ID == "" {
			v.fail("keywords.json: duplicate or empty id %q", kw.ID)
		}
		seen[kw.ID] = true
		v.text("keywords.json", kw.Word)
		if kw.Answer != "" {
			v.text("keywords.json", kw.Answer)
		}
	}
}

func (v *verifier) conversation(c *Conversation) {
	owner := "conversation " + c.ID
	for i, cl := range c.Clauses {
		what := owner + " clause " + itoa(i)
		v.condition(what, cl.If)
		for j, a := range cl.Do {
			v.talkAction(what+" action "+itoa(j), a, len(c.Clauses))
		}
	}
}

func (v *verifier) condition(owner string, c Condition) {
	set := 0
	for _, ok := range []bool{c.FirstMeeting, c.Greeting, len(c.Asked) > 0, c.Never, len(c.Carries) > 0,
		c.Counter != nil, c.Dead != "", c.FullHealth, c.CarriesCount != nil, c.SlotEmpty != nil, c.SlotHolds != nil, c.Always} {
		if ok {
			set++
		}
	}
	if set != 1 {
		v.fail("%s: %d conditions, want 1", owner, set)
	}
	for _, k := range c.Asked {
		v.keyword(owner, k)
	}
	for _, it := range c.Carries {
		v.item(owner, it)
	}
	if t := c.Counter; t != nil {
		n := 0
		for _, x := range []*int{t.Above, t.Equals, t.Below} {
			if x != nil {
				n++
			}
		}
		if n != 1 || t.Counter == "" {
			v.fail("%s: a counter test wants a counter and one comparison", owner)
		}
	}
	if c.Dead != "" {
		v.thingRef(owner, c.Dead)
	}
	if c.CarriesCount != nil {
		v.item(owner, c.CarriesCount.Item)
	}
	if c.SlotHolds != nil {
		v.item(owner, c.SlotHolds.Item)
	}
}

func (v *verifier) talkAction(owner string, a Action, clauses int) {
	if !v.actionKind(owner, a, talkOps) {
		return
	}
	portrait := func(id string) {
		if _, ok := v.p.Portraits[id]; id != "" && !ok {
			v.fail("%s: portrait %q not found", owner, id)
		}
	}
	switch {
	case a.Say != nil:
		v.text(owner, *a.Say)
	case a.SayAs != nil:
		v.text(owner, a.SayAs.Text)
		portrait(a.SayAs.Portrait)
	case a.Offer != nil:
		v.keyword(owner, *a.Offer)
	case a.Take != nil:
		v.item(owner, *a.Take)
	case a.Give != nil:
		v.item(owner, a.Give.Item)
		for _, c := range a.Give.Contents {
			v.item(owner, c.Item)
		}
	case a.SetPortrait != nil:
		portrait(*a.SetPortrait)
	case a.Travel != nil:
		v.destination(owner, *a.Travel)
	case a.Teach != nil:
		v.icons(owner, *a.Teach)
	case a.Jump != nil:
		if *a.Jump < 0 || *a.Jump >= clauses {
			v.fail("%s: jump to clause %d of %d", owner, *a.Jump, clauses)
		}
	case a.Menu != nil:
		if !menus[*a.Menu] {
			v.fail("%s: unknown menu %q", owner, *a.Menu)
		}
	case a.Transcript != nil:
		if *a.Transcript != TranscriptHeading && *a.Transcript != TranscriptContinued {
			v.fail("%s: unknown transcript heading %q", owner, *a.Transcript)
		}
	}
}

func itoa(i int) string { return strconv.Itoa(i) }
