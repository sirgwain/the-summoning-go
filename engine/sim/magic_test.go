package sim

import (
	"slices"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// studyWorld returns a world of the dev pack with the player in the study at cell c, facing d.
func studyWorld(t *testing.T, c cell, d int, edit ...func(p *content.Pack)) *World {
	t.Helper()
	p := devpack.Build()
	for _, e := range edit {
		e(p)
	}
	if err := p.Verify(); err != nil {
		t.Fatal(err)
	}
	w, err := New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	do(w, input.GoTo{Level: devpack.Study})
	do(w, input.PlacePlayer{Cell: c})
	w.Player().Facing = d
	return w
}

// until runs updates until done reports true, at most n, and returns the events.
func until(w *World, n int, done func() bool) []Event {
	var events []Event
	for range n {
		if done() {
			break
		}
		events = append(events, w.Update(nil)...)
	}
	return events
}

func hasEvent(events []Event, kind, detail string) bool {
	return slices.ContainsFunc(events, func(e Event) bool { return e.Kind == kind && e.Detail == detail })
}

func TestLearnMemorizeAndCast(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2)
	p := w.Player()
	if p.Magic.Icons != "DJ" {
		t.Fatalf("knows %q, want the template's DJ", p.Magic.Icons)
	}
	// An icon not known, and an icon twice in a row, are refused; a gesture of no spell costs a
	// magic point an icon.
	if ev := do(w, input.Learn{Gesture: "IBF"}); !has(ev, EventRefused, "the icon I is not known") {
		t.Errorf("learning unknown icons: %v", ev)
	}
	if ev := do(w, input.Learn{Gesture: "DD"}); !has(ev, EventRefused, "an icon cannot follow itself") {
		t.Errorf("an icon twice: %v", ev)
	}
	mp := p.MagicPoints
	do(w, input.Learn{Gesture: "JD"})
	if p.MagicPoints != mp-2 || len(p.Magic.Spellbook) != 0 {
		t.Errorf("a gesture of no spell: %d magic points, spellbook %v", p.MagicPoints, p.Magic.Spellbook)
	}
	ev := do(w, input.Learn{Gesture: "DJ"})
	if !hasEvent(ev, EventMagic, "learned") || !p.Magic.Learned(devpack.SpellFlameArrow) ||
		p.Magic.Memorized[devpack.SpellFlameArrow] != 1 || p.Magic.Selected[0] != devpack.SpellFlameArrow {
		t.Fatalf("learning DJ: %v, magic %+v", ev, p.Magic)
	}

	mp = p.MagicPoints
	do(w, input.Cast{School: 0})
	if p.MagicPoints != mp-5 || p.Magic.Memorized[devpack.SpellFlameArrow] != 0 || p.Magic.Gesture == nil {
		t.Fatalf("casting: %d magic points, magic %+v", p.MagicPoints, p.Magic)
	}
	// While the gesture plays its icons show, one a player update; then the arrow flies to the
	// target four cells ahead and kills it.
	sawIcon := false
	target := w.Current().Thing("target")
	until(w, 2000, func() bool {
		if c, ok := p.Magic.Icon(w.Pack()); ok && c == 'J' {
			sawIcon = true
		}
		return target.HitPoints == 0
	})
	if !sawIcon {
		t.Error("the gesture's last icon never showed")
	}
	if target.HitPoints != 0 {
		t.Fatalf("the target has %d hit points", target.HitPoints)
	}
	if s := p.Skills[SkillSchool]; s.Practice == 0 && s.Level == 1 {
		t.Errorf("casting and hitting practiced nothing: %+v", s)
	}
	if ev := do(w, input.Cast{School: 0}); !has(ev, EventMessage, content.MsgNotMemorized) {
		t.Errorf("casting with nothing memorized: %v", ev)
	}
}

func TestMemorizeLimitForgets(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2)
	do(w, input.Learn{Gesture: "DJ"})
	// Talent 16 and skill 1 hold (16 − 10) / 2 + 1 / 2 = 3.
	for range 3 {
		do(w, input.Memorize{Spell: devpack.SpellFlameArrow})
	}
	if n := w.SchoolTotal(0); n != 3 {
		t.Errorf("school 0 holds %d, want 3", n)
	}
	if ev := do(w, input.Memorize{Spell: devpack.SpellFireball}); !has(ev, EventRefused, devpack.SpellFireball+" is not in the spellbook") {
		t.Errorf("memorizing an unlearned spell: %v", ev)
	}
}

func TestSpellNeedsAndRefusals(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2)
	p := w.Player()
	p.Magic.teach("FHBEL")
	do(w, input.Learn{Gesture: "FHBEHL"})
	if p.Magic.Selected[3] != devpack.SpellHeal {
		t.Fatalf("heal not learned: %+v", p.Magic)
	}
	if ev := do(w, input.Cast{School: 3}); !has(ev, EventMessage, content.MsgFullHealth) || p.Magic.Gesture != nil {
		t.Errorf("healing at full health: %v", ev)
	}
	p.HitPoints = 10
	p.MagicPoints = 3
	if ev := do(w, input.Cast{School: 3}); !has(ev, EventMessage, content.MsgNoMagic) {
		t.Errorf("casting without the magic points: %v", ev)
	}
	p.MagicPoints = 50
	do(w, input.Cast{School: 3})
	until(w, 2000, func() bool { return p.Magic.Gesture == nil })
	if p.HitPoints != p.MaxHitPoints {
		t.Errorf("heal left %d of %d hit points", p.HitPoints, p.MaxHitPoints)
	}
}

func TestKanoOpensDoor(t *testing.T) {
	w := studyWorld(t, devpack.KanoFrom, 4)
	p := w.Player()
	p.Magic.teach("DB")
	do(w, input.Learn{Gesture: "DB"})
	do(w, input.Cast{School: 1})
	door := w.Current().Feature("door.kano")
	until(w, 2000, func() bool { return door.State == content.DoorOpen })
	if door.State != content.DoorOpen {
		t.Fatalf("the door is %s", door.State)
	}
}

func TestMagicWallStandsThenFades(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2)
	p := w.Player()
	p.Magic.teach("BIKG")
	do(w, input.Learn{Gesture: "BIKG"})
	do(w, input.Cast{School: 1})
	l := w.Current()
	wall := func() *Thing {
		for _, t := range l.Things {
			if t.Template.Kind == content.ThingConjured {
				return t
			}
		}
		return nil
	}
	until(w, 2000, func() bool { return wall() != nil })
	mw := wall()
	if mw == nil {
		t.Fatal("no wall conjured")
	}
	if mw.Lasts <= 0 || mw.V <= p.V {
		t.Errorf("wall lasts %d at (%d, %d), player at (%d, %d)", mw.Lasts, mw.U, mw.V, p.U, p.V)
	}
	until(w, 100000, func() bool { return wall() == nil })
	if wall() != nil {
		t.Error("the wall never faded")
	}
}

func TestReadScrollTeachesIcons(t *testing.T) {
	w := studyWorld(t, devpack.StudyScroll, 2)
	p := w.Player()
	do(w, input.PickUp{Hand: true})
	if ev := do(w, input.Read{Slot: input.SlotHand}); !hasEvent(ev, EventMagic, "read") {
		t.Fatalf("reading: %v", ev)
	}
	if p.Magic.Icons != "BDFIJ" {
		t.Errorf("knows %q after reading", p.Magic.Icons)
	}
	do(w, input.Learn{Gesture: "IBF"})
	if !p.Magic.Learned(devpack.SpellFireball) {
		t.Error("the scroll's gesture did not learn fireball")
	}
}

func TestCreatureCasts(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2, func(p *content.Pack) {
		p.Things[devpack.ThingTarget].Spell = devpack.SpellFireball
		p.Things[devpack.ThingTarget].SpellPower = content.Dice{Bonus: 3}
		p.Things[devpack.ThingTarget].HitPoints = content.Dice{Bonus: 100}
	})
	p := w.Player()
	tgt := w.Current().Thing("target")
	tgt.Facing = 6
	hp := p.HitPoints
	until(w, 5000, func() bool { return p.HitPoints < hp })
	if p.HitPoints != hp-3 {
		t.Errorf("the caster's fireball left %d of %d hit points", p.HitPoints, hp)
	}
}

func TestObjectCastsAndTalks(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2, func(p *content.Pack) {
		p.Items[devpack.ItemSphere].Casts = &content.Cast{Spell: devpack.SpellFireball, Power: &content.Dice{Bonus: 50}}
		p.Items[devpack.ItemSphere].Actions = []content.ItemAction{{Kind: content.ActionUseUp, Delay: 3}}
		p.Items[devpack.ItemRock].Conversation = devpack.ThingSage
		p.Items[devpack.ItemRock].UseResult = true
		p.Items[devpack.ItemRock].Actions = []content.ItemAction{{Kind: content.ActionUseUp, Delay: 3}}
		p.Manifest.Counters.UseResult = "var.used"
	})
	p := w.Player()
	p.Slots[SlotRightHand] = w.newObject(devpack.ItemSphere)
	do(w, input.Strike{Hand: 0})
	if p.Slots[SlotRightHand] != nil {
		t.Error("the sphere was not used up")
	}
	tgt := w.Current().Thing("target")
	until(w, 500, func() bool { return tgt.HitPoints == 0 })
	if tgt.HitPoints != 0 {
		t.Errorf("the sphere's fireball left the target %d", tgt.HitPoints)
	}
	// The rock talks; it is used only when the conversation sets the result counter.
	rock := w.newObject(devpack.ItemRock)
	p.Slots[SlotRightHand] = rock
	p.Delays[0] = 0
	do(w, input.Strike{Hand: 0})
	if w.Talk() == nil {
		t.Fatal("the rock did not start a conversation")
	}
	do(w, input.Leave{})
	if p.Slots[SlotRightHand] != rock {
		t.Error("the rock was used though the counter is 0")
	}
	p.Delays[0] = 0
	do(w, input.Strike{Hand: 0})
	w.setCounter("var.used", 1)
	do(w, input.Leave{})
	if p.Slots[SlotRightHand] == rock {
		t.Error("the rock was not used though the counter is 1")
	}
}

// line returns the open conversation's line's text key, or "".
func line(w *World) string {
	if t := w.Talk(); t != nil && t.Line != nil {
		return t.Line.Text
	}
	return ""
}

func offered(w *World) []string {
	var out []string
	for _, k := range w.Talk().Offered {
		if k != "" {
			out = append(out, k)
		}
	}
	return out
}

func TestTalkToSage(t *testing.T) {
	w := studyWorld(t, cell{1, 3}, 2)
	p := w.Player()
	ev := do(w, input.Talk{})
	if w.Talk() == nil || !hasEvent(ev, EventTalk, "started") {
		t.Fatalf("talking: %v", ev)
	}
	if line(w) != "talk.sage.hello" || w.Talk().Portrait != devpack.PortraitSage {
		t.Errorf("first line %q, portrait %q", line(w), w.Talk().Portrait)
	}
	step := w.Step()
	do(w, input.Walk{Direction: 2})
	if w.Step() != step {
		t.Error("the world ran during the conversation")
	}
	do(w, input.Continue{})
	if got := offered(w); !slices.Equal(got, []string{"keyword.name", "keyword.magic"}) {
		t.Errorf("offered %v", got)
	}
	// Asking says the player's question first, then the sage's answer, then teaches.
	do(w, input.Ask{Word: "Magic!"})
	if l := w.Talk().Line; l == nil || l.Text != "keyword.magic.answer" || l.Portrait != "" {
		t.Fatalf("asking magic: %+v", l)
	}
	do(w, input.Continue{})
	if line(w) != "talk.sage.magic" {
		t.Fatalf("answer %q", line(w))
	}
	do(w, input.Continue{})
	if l := w.Talk().Line; l == nil || l.Kind != LineTeach || l.Icons != "IBF" || p.Magic.Icons != "BDFIJ" {
		t.Fatalf("teaching: %+v, knows %q", l, p.Magic.Icons)
	}
	do(w, input.Continue{})
	if got := offered(w); !slices.Equal(got, []string{"keyword.name", "keyword.fireball"}) {
		t.Errorf("offered %v after magic", got)
	}
	// A word that is not a keyword is refused; one no clause answers says nothing.
	if ev := do(w, input.Ask{Word: "xyzzy"}); !has(ev, EventRefused, "not a keyword: xyzzy") {
		t.Errorf("a word that is not a keyword: %v", ev)
	}
	// The gift jumps past never to the clause that gives, once.
	do(w, input.Ask{Word: "gift"})
	do(w, input.Continue{})
	if line(w) != "talk.sage.gift" {
		t.Fatalf("gift: %q", line(w))
	}
	do(w, input.Continue{})
	if p.Slots[0] == nil || p.Slots[0].Item != devpack.ItemPotion || w.Counters()[devpack.CounterGift] != 1 {
		t.Errorf("the gift: shelf %+v, counters %v", p.Slots[0], w.Counters())
	}
	do(w, input.Ask{Word: "gift"})
	do(w, input.Continue{})
	if line(w) != "talk.sage.nomore" {
		t.Errorf("a second gift: %q", line(w))
	}
	do(w, input.Leave{})
	if w.Talk() != nil || !w.Records()[devpack.ThingSage].Talked {
		t.Fatal("leaving did not end the conversation")
	}
	// The next time, the sage greets the player again.
	do(w, input.Talk{Thing: "sage"})
	if line(w) != "talk.sage.again" {
		t.Errorf("greeting again: %q", line(w))
	}
	// Going on from a line with a marked word asks it next.
	do(w, input.Continue{Ask: "magic"})
	if line(w) != "keyword.magic.answer" {
		t.Errorf("asking a marked word: %q", line(w))
	}
}

func TestFindKeyword(t *testing.T) {
	p := devpack.Build()
	text := p.Texts["text.en"].Strings
	for word, want := range map[string]string{
		"magic": "keyword.magic", "MAGIC": "keyword.magic", "magician": "keyword.magic",
		"magi!": "keyword.magic", "gift?": "keyword.gift", "ma": "", "": "", "fire": "keyword.fireball",
	} {
		k, ok := FindKeyword(p, text, word)
		if ok != (want != "") || k.ID != want {
			t.Errorf("%q matched %q (%v), want %q", word, k.ID, ok, want)
		}
	}
}

// A gesture plays two updates a picture: the empty panel, then each icon, with the empty panel
// between two icons when the second holds the first; the spell takes effect as the last icon
// first shows, which stays four updates.
func TestGesturePictures(t *testing.T) {
	holds := &content.Magic{Holds: map[string]string{"B": "A"}}
	for _, c := range []struct {
		gesture, want string
		effect        int
	}{
		{"AB", "..AA..BBBB", 7},
		{"BA", "..BBAAAA", 5},
		{"C", "..CCCC", 3},
	} {
		pictures, effect := gesturePictures(c.gesture, holds)
		got := strings.ReplaceAll(string(pictures), "\x00", ".")
		if got != c.want || effect != c.effect {
			t.Errorf("%s: %q, effect %d; want %q, %d", c.gesture, got, effect, c.want, c.effect)
		}
	}
}
