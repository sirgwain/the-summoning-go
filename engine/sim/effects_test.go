package sim

import (
	"slices"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// Flipbook ids for the tests; the simulation only names them.
const (
	fbClimbDown = "flipbook.test.climb_down"
	fbFall      = "flipbook.test.fall"
	fbThrough   = "flipbook.test.through"
	fbCaster    = "flipbook.test.caster"
	fbDeathBlow = "flipbook.test.death_blow"
	fbHeal      = "flipbook.test.heal"
	fbCure      = "flipbook.test.cure"
	fbSocket    = "flipbook.test.socket"
	fbSeal      = "flipbook.test.seal"
)

func withEffectCues(p *content.Pack) {
	p.Effects = &content.EffectCues{ID: "effects", Cues: map[string]string{
		content.EffectClimbDown: fbClimbDown, content.EffectFall: fbFall,
		content.EffectCasterDies: fbCaster, content.EffectDeathBlow: fbDeathBlow,
		content.SparkleHeal: fbHeal, content.SparkleCure: fbCure,
	}}
}

func featureOf(p *content.Pack, level, id string) *content.Feature {
	for _, f := range p.Levels[level].Features {
		if f.ID == id {
			return f
		}
	}
	return nil
}

func eventOf(events []Event, kind string) (Event, bool) {
	i := slices.IndexFunc(events, func(e Event) bool { return e.Kind == kind })
	if i < 0 {
		return Event{}, false
	}
	return events[i], true
}

// walkUntil walks along d, an update a step, until an event of kind comes, and returns the
// update's events.
func walkUntil(t *testing.T, w *World, d int, kind string) []Event {
	t.Helper()
	for range 30 {
		if ev := walk(w, d, 1); has(ev, kind, "") {
			return ev
		}
	}
	t.Fatalf("no %s event", kind)
	return nil
}

// A level change after a view effect waits for it: the world names the effect, and changes
// level at the start of its next update.
func TestLevelChangeWaitsForEffect(t *testing.T) {
	for _, c := range []struct {
		name string
		rows []string
		d    int
		want string
		hurt bool
	}{
		{"exit marker", []string{"#####", "#@X.#", "#####"}, 2, fbThrough, false},
		{"stepping down", []string{"#####", "#@O.#", "#####"}, 2, fbClimbDown, false},
		{"falling", []string{"#####", "#.O@#", "#####"}, 6, fbFall, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := walkPack(c.rows...)
			withEffectCues(p)
			if f := featureOf(p, "level.walk", "exit"); f != nil {
				f.Effect = fbThrough
			}
			w, err := New(p, 1)
			if err != nil {
				t.Fatal(err)
			}
			ev := walkUntil(t, w, c.d, EventEffect)
			e, _ := eventOf(ev, EventEffect)
			if e.Detail != c.want || e.Thing != PlayerID || e.Level != "level.walk" {
				t.Errorf("effect %+v, want %s by the player", e, c.want)
			}
			if w.Current().ID != "level.walk" || has(ev, EventLevelChanged, "") || !w.Waiting() {
				t.Fatalf("the level changed with the effect: on %s", w.Current().ID)
			}
			if hurt := w.Player().HitPoints < 30; hurt != c.hurt {
				t.Errorf("hurt %v, want %v", hurt, c.hurt)
			}
			ev = w.Update(nil)
			if w.Current().ID != "level.below" || !has(ev, EventLevelChanged, "") || w.Waiting() {
				t.Errorf("after the effect: on %s, %v", w.Current().ID, ev)
			}
		})
	}
}

// With no flipbook to play, the level changes at once.
func TestLevelChangeWithoutEffect(t *testing.T) {
	w := walkWorld(t, "#####", "#@O.#", "#####")
	ev := walkUntil(t, w, 2, EventLevelChanged)
	if has(ev, EventEffect, "") || w.Waiting() {
		t.Errorf("an effect with no cue: %v", ev)
	}
}

func TestSealedDoorSparkles(t *testing.T) {
	w := newWorld(t, func(p *content.Pack) { featureOf(p, "level.00", "sealed").Sparkle = fbSeal })
	ev := do(w, input.Drop{Cell: cell{1, 1}, Item: "item.sphere"})
	e, ok := eventOf(ev, EventSparkle)
	if !ok || e.Detail != fbSeal || e.Feature != "sealed" || e.Cell != (cell{1, 1}) || e.Thing != "" {
		t.Errorf("sparkle %+v", e)
	}
	if has(ev, EventEffect, "") {
		t.Error("a sparkle held the world as a view effect")
	}
}

// The socket's effect plays before its wall opens.
func TestSocketEffectBeforeWallOpens(t *testing.T) {
	w := newWorld(t, func(p *content.Pack) { featureOf(p, "level.00", "socket").Effect = fbSocket })
	do(w, input.Drop{Cell: cell{0, 4}, Item: "item.gem2"})
	ev := do(w, input.Drop{Cell: cell{0, 4}, Item: "item.gem1"})
	if e, ok := eventOf(ev, EventEffect); !ok || e.Detail != fbSocket {
		t.Fatalf("no socket effect: %v", ev)
	}
	if w.Current().Wall[0][3] == content.NoFrame {
		t.Fatal("the wall opened with the effect")
	}
	ev = w.Update(nil)
	if w.Current().Wall[0][3] != content.NoFrame || !has(ev, EventMap, "") {
		t.Errorf("the wall did not open after the effect: %v", ev)
	}
}

// A spell's effect action holds the effects after it; its sparkle action does not.
func TestSpellEffectAndSparkle(t *testing.T) {
	w := walkWorld(t, "#####", "#@..#", "#####")
	sp := &content.Spell{ID: "spell.test", Effects: []content.Action{
		{Sparkle: ptr(fbHeal)},
		{SetCounter: &content.SetCounter{Counter: "var.a", Value: 1}},
		{Effect: ptr(fbSocket)},
		{SetCounter: &content.SetCounter{Counter: "var.b", Value: 1}},
	}}
	w.events = nil
	w.castSpell(w.playerCaster(), sp, 1)
	ev := w.events
	if s, ok := eventOf(ev, EventSparkle); !ok || s.Detail != fbHeal || s.Thing != PlayerID {
		t.Errorf("sparkle %+v", s)
	}
	if e, ok := eventOf(ev, EventEffect); !ok || e.Detail != fbSocket || e.Thing != PlayerID {
		t.Errorf("effect %+v", e)
	}
	if w.Counters()["var.a"] != 1 || w.Counters()["var.b"] != 0 {
		t.Fatalf("counters %v: the effect should hold only what follows it", w.Counters())
	}
	w.Update(nil)
	if w.Counters()["var.b"] != 1 {
		t.Errorf("counters %v after the effect", w.Counters())
	}
}

func TestPowerSparkles(t *testing.T) {
	p := walkPack("#####", "#@..#", "#####")
	withEffectCues(p)
	w, err := New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		power int
		want  string
	}{{0x15, fbCure}, {0x1D, fbHeal}, {0x45, fbHeal}, {0x24, ""}, {0x28, ""}, {0x44, ""}, {0x46, ""}} {
		w.events = nil
		w.startPower(c.power, 1)
		e, ok := eventOf(w.events, EventSparkle)
		if e.Detail != c.want || ok != (c.want != "") {
			t.Errorf("power %02Xh: sparkle %q, want %q", c.power, e.Detail, c.want)
		}
	}
}

func TestCasterDiesAndDeathBlow(t *testing.T) {
	p := walkPack("#####", "#@G.#", "#####")
	withEffectCues(p)
	w, err := New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	l := w.Current()
	g := l.Thing("guard")
	w.events = nil
	w.creatureDies(l, g)
	if e, ok := eventOf(w.events, EventEffect); !ok || e.Detail != fbCaster || e.Thing != "guard" || e.Cell != g.Cell() {
		t.Errorf("caster dies: %+v", e)
	}

	p = walkPack("#####", "#@G.#", "#####")
	withEffectCues(p)
	p.Things["thing.guard"].Behaviors = []string{content.BehaviorDeathBlow}
	if w, err = New(p, 1); err != nil {
		t.Fatal(err)
	}
	w.events = nil
	w.creatureStrikes(w.Current(), w.Current().Thing("guard"))
	e, ok := eventOf(w.events, EventEffect)
	if !ok || e.Detail != fbDeathBlow || e.Thing != PlayerID || !w.Player().Dead {
		t.Errorf("death blow: %+v, dead %v", e, w.Player().Dead)
	}
	if i, j := slices.IndexFunc(w.events, func(e Event) bool { return e.Kind == EventEffect }),
		slices.IndexFunc(w.events, func(e Event) bool { return e.Kind == EventThing && e.Thing == PlayerID }); i > j {
		t.Error("the player died before the death blow's effect")
	}
}

func TestTalkEffect(t *testing.T) {
	w := walkWorld(t, "#####", "#@G.#", "#####")
	l := w.Current()
	w.events = nil
	w.talk = &Talk{Speaker: l.Thing("guard")}
	w.talkAction(content.Action{Effect: ptr(fbSocket)})
	if e, ok := eventOf(w.events, EventEffect); !ok || e.Detail != fbSocket || e.Thing != "guard" {
		t.Errorf("talk effect %+v", e)
	}
}
