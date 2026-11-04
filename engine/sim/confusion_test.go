package sim

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// updates runs the player's next n updates, gathering the events.
func updates(w *World, n int) []Event {
	var all []Event
	p := w.Player()
	for range n {
		next := p.next
		for p.next == next {
			all = append(all, w.Update(nil)...)
		}
	}
	return all
}

func TestConfusionFallsWithEveryUpdate(t *testing.T) {
	w := walkWorld(t,
		"#####",
		"#@..#",
		"#####")
	p := w.Player()
	w.Update(nil)
	p.Confusion = 75
	updates(w, 10)
	if p.Confusion != 65 {
		t.Errorf("standing 10 updates left confusion %d, want 65", p.Confusion)
	}
	updates(w, 100)
	if p.Confusion != 0 || p.Confused() {
		t.Errorf("confusion %d after it ran out", p.Confusion)
	}
}

func TestDrinkingIsa(t *testing.T) {
	w := fightWorld(t, nil,
		"#####",
		"#@..#",
		"#####")
	p := w.Player()
	p.Slots[SlotBody] = w.newObject("item.mail")
	w.putOn(p.Slots[SlotBody])
	w.Update(nil)
	hp := p.HitPoints
	w.events = nil
	w.startPower(0x04, 1)
	ev := w.events
	lost := hp - p.HitPoints
	if p.Confusion != 75 {
		t.Errorf("confusion %d", p.Confusion)
	}
	// 2d8 past the armor: at least 2, though the mail's 4 would leave a roll of 2 at 1.
	if lost < 2 || lost > 16 {
		t.Errorf("lost %d hit points", lost)
	}
	if !p.Poisoned || !has(ev, EventMessage, content.MsgPoisoned) {
		t.Errorf("poisoned %v, events %v", p.Poisoned, ev)
	}
	if has(ev, EventMessage, content.MsgConfused) {
		t.Error("power 04h said the player is confused")
	}
}

func TestBecomingConfusedIsSaid(t *testing.T) {
	w := walkWorld(t,
		"#####",
		"#@..#",
		"#####")
	p := w.Player()
	w.Update(nil)
	p.Confusion = 10
	var ev []Event
	w.events = nil
	w.startPower(0x50, 1)
	ev = w.events
	if !has(ev, EventMessage, content.MsgConfused) {
		t.Errorf("power 50h: %v", ev)
	}
	w.events = nil
	w.startPower(0x50, 1)
	if has(w.events, EventMessage, content.MsgConfused) {
		t.Error("said again while already confused")
	}
}

func TestWallBumpConfuses(t *testing.T) {
	for _, tc := range []struct {
		start int
		said  bool
	}{{20, false}, {22, true}, {26, false}} {
		w := walkWorld(t,
			"####",
			"#@.#",
			"####")
		p := w.Player()
		// Up walks into the wall above: press against it first.
		for range 10 {
			w.Update([]input.Command{input.Walk{Direction: 0}})
		}
		p.Confusion = tc.start
		var ev []Event
		next := p.next
		for p.next == next {
			ev = append(ev, w.Update([]input.Command{input.Walk{Direction: 0}})...)
		}
		if p.Confusion != tc.start-1+confusionBump {
			t.Errorf("from %d: confusion %d", tc.start, p.Confusion)
		}
		if has(ev, EventMessage, content.MsgConfused) != tc.said {
			t.Errorf("from %d: said %v", tc.start, !tc.said)
		}
	}
}

func TestTeleporterConfuses(t *testing.T) {
	w := walkWorld(t,
		"#######",
		"#@t...#",
		"#.....#",
		"#######")
	ev := walk(w, 2, 2)
	if c := w.Player().Confusion; c < teleporterConfusion-2 || c > teleporterConfusion {
		t.Errorf("confusion %d after a teleporter", c)
	}
	if !has(ev, EventMessage, content.MsgConfused) {
		t.Errorf("no confused message: %v", ev)
	}

	w = walkWorld(t,
		"#######",
		"#@t...#",
		"#.....#",
		"#######")
	five := 5
	w.Current().Def.TeleporterConfusion = &five
	walk(w, 2, 2)
	if c := w.Player().Confusion; c > 5 {
		t.Errorf("confusion %d on a level giving 5", c)
	}
}

func TestThrownIsaPoisons(t *testing.T) {
	edit := func(p *content.Pack) {
		p.Items["item.isa"] = &content.Item{ID: "item.isa", Weight: 2, Size: 1, Power: 0x04, PoisonsThrown: true}
	}
	w := fightWorld(t, edit,
		"#########",
		"#@..G...#",
		"#########")
	p, l := w.Player(), w.Current()
	g := l.Thing("guard")
	p.Facing = 2
	p.Slots[SlotRightHand] = w.newObject("item.isa")
	w.Update([]input.Command{input.Throw{Hand: 0}})
	for range 200 {
		w.Update(nil)
	}
	// The throw does weight / 10 + 1; the poison may have taken more since.
	if g.HitPoints > 20-1 || !g.Poisoned {
		t.Errorf("the guard: %d hit points, poisoned %v", g.HitPoints, g.Poisoned)
	}
	if p.Confusion != 0 || p.Poisoned {
		t.Error("throwing Isa hurt the thrower")
	}
	if e := w.Book().Items["item.isa"]; e == nil || e.Stage != StageUsed {
		t.Errorf("the book's entry: %+v", e)
	}

	w = fightWorld(t, func(pk *content.Pack) {
		edit(pk)
		g := pk.Things["thing.guard"]
		g.Behaviors = append(g.Behaviors, content.BehaviorResistsPoison)
	},
		"#########",
		"#@..G...#",
		"#########")
	p, g = w.Player(), w.Current().Thing("guard")
	p.Facing = 2
	p.Slots[SlotRightHand] = w.newObject("item.isa")
	w.Update([]input.Command{input.Throw{Hand: 0}})
	for range 200 {
		w.Update(nil)
	}
	if g.Poisoned {
		t.Error("a guard resisting poison was poisoned")
	}
}

func TestHasStatus(t *testing.T) {
	p := &Player{}
	if p.HasStatus(content.StatusConfused) || p.HasStatus(content.StatusFireShield) {
		t.Error("a fresh player has statuses")
	}
	p.Confusion = 26
	p.Timed[TimedFireShield] = 3
	p.Poisoned = true
	for _, s := range []string{content.StatusConfused, content.StatusFireShield, content.StatusPoisoned} {
		if !p.HasStatus(s) {
			t.Errorf("%s does not hold", s)
		}
	}
	p.Confusion = 25
	if p.HasStatus(content.StatusConfused) {
		t.Error("confused at 25")
	}
}
