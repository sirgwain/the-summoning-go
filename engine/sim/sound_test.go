package sim

import (
	"slices"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// sounds returns what the sound events play: the cue, or the sound id.
func sounds(events []Event) []string {
	var out []string
	for _, e := range events {
		if e.Kind == EventSound {
			out = append(out, e.Detail+e.Item)
		}
	}
	return out
}

func TestBlowSounds(t *testing.T) {
	w := fightWorld(t, nil, "#######", "#@...G#", "#######")
	w.Player().Facing = 2
	if got := sounds(strike(w, 0, 0)); !slices.Equal(got, []string{content.CueMiss}) {
		t.Fatalf("a blow at nothing: %q", got)
	}
	besideGuard(w)
	hits := 0
	for range 40 {
		got := sounds(strike(w, 0, 0))
		if len(got) == 0 || got[0] != content.CueMiss && got[0] != content.CueHit {
			t.Fatalf("a blow at the guard: %q", got)
		}
		if got[0] == content.CueHit {
			hits++
		}
	}
	if hits == 0 {
		t.Error("no blow hit")
	}
}

func TestHurtSounds(t *testing.T) {
	w := walkWorld(t, "###", "#@#", "###")
	l := w.Current()
	l.Def.FloorTypes[0].Tick = &content.Effect{Damage: content.Dice{Bonus: 1}, Cue: content.CueHurt}
	w.events = nil
	w.statusTick()
	if got := sounds(w.events); !slices.Equal(got, []string{content.CueHurt}) {
		t.Errorf("a hurting floor: %q", got)
	}
	w.events = nil
	w.hurtPlayer(1, 0, "falling")
	if got := sounds(w.events); len(got) != 0 {
		t.Errorf("a quiet hurt: %q", got)
	}
}

func TestBowSound(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		p.Items["item.arrow"] = &content.Item{ID: "item.arrow", Weight: 1, Size: 1, Tags: []string{"arrows"},
			Missile: &content.Missile{Damage: content.Dice{Bonus: 5}}}
		p.Items["item.quiver"] = &content.Item{ID: "item.quiver", Weight: 1, Size: 2, Wear: content.WearQuiver,
			Container: &content.Container{Capacity: 4, Largest: 1}}
		bow := 3
		p.Items["item.bow"] = &content.Item{ID: "item.bow", Weight: 3, Size: 3, Wear: content.WearHand, WeaponClass: &bow,
			Actions: []content.ItemAction{{Kind: content.ActionFire, Delay: 3}}}
	}, "##########", "#@.......#", "##########")
	p := w.Player()
	p.Facing = 2
	p.Slots[SlotRightHand] = w.newObject("item.bow")
	p.Hand = w.newObject("item.quiver")
	w.Update([]input.Command{input.PutOn{}})
	shoot := func(walk bool) (first, later []string) {
		q := p.Slots[SlotQuiver]
		q.Contents = append(q.Contents, w.newObject("item.arrow"))
		p.Delays = [2]int{}
		first = sounds(w.Update([]input.Command{input.Strike{}}))
		for i := range 10 {
			var cmds []input.Command
			if walk && i == 0 {
				cmds = []input.Command{input.Walk{Direction: 6}}
			}
			later = append(later, sounds(w.Update(cmds))...)
		}
		return first, later
	}
	if first, later := shoot(false); len(first) != 0 || slices.Index(later, content.CueBow) < 0 {
		t.Errorf("a shot: %q, then %q", first, later)
	}
	if _, later := shoot(true); slices.Contains(later, content.CueBow) {
		t.Errorf("a shot walked away from: %q", later)
	}
}

func TestHandlingSounds(t *testing.T) {
	w := fightWorld(t, nil, "####", "#@.#", "####")
	p, l := w.Player(), w.Current()
	l.Objects[p.Cell()] = []*Object{w.newObject("item.sword")}
	if got := sounds(w.Update([]input.Command{input.PickUp{Hand: true}})); !slices.Equal(got, []string{content.CueTake}) {
		t.Errorf("picking up: %q", got)
	}
	if got := sounds(w.Update([]input.Command{input.PutDown{}})); !slices.Equal(got, []string{content.CueDrop}) {
		t.Errorf("putting down: %q", got)
	}
	if got := sounds(w.Update([]input.Command{input.Swap{Slot: 0}})); len(got) != 0 {
		t.Errorf("swapping nothing: %q", got)
	}
	p.Slots[0] = w.newObject("item.sword")
	if got := sounds(w.Update([]input.Command{input.Swap{Slot: 0}})); !slices.Equal(got, []string{content.CueTake}) {
		t.Errorf("swapping: %q", got)
	}
}

func TestUseUpSound(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		p.Items["item.bottle"] = &content.Item{ID: "item.bottle", Weight: 1, Size: 1}
		p.Items["item.potion"] = &content.Item{ID: "item.potion", Weight: 2, Size: 1, Power: 0x28, Spent: "item.bottle",
			Actions: []content.ItemAction{{Kind: content.ActionUseUp, Delay: 3, Sound: "sfx.drink"}}}
	}, "####", "#@.#", "####")
	p := w.Player()
	p.Slots[SlotRightHand] = w.newObject("item.potion")
	p.HitPoints = 5
	if got := sounds(w.Update([]input.Command{input.Strike{}})); !slices.Contains(got, "sfx.drink") {
		t.Errorf("drinking: %q", got)
	}
}

func TestSpellSounds(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2, func(p *content.Pack) {
		p.Spells[devpack.SpellFlameArrow].Effects[0].Missile.Sound = devpack.SFXBlip
	})
	if got := sounds(do(w, input.Learn{Gesture: "DJ"})); !slices.Equal(got, []string{content.CueGesture}) {
		t.Errorf("learning: %q", got)
	}
	if got := sounds(do(w, input.Cast{School: 0})); !slices.Equal(got, []string{content.CueCast}) {
		t.Errorf("casting: %q", got)
	}
	target := w.Current().Thing("target")
	if got := sounds(until(w, 2000, func() bool { return target.HitPoints == 0 })); !slices.Contains(got, devpack.SFXBlip) {
		t.Errorf("the arrow's hit: %q", got)
	}
}
