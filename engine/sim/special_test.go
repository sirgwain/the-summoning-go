package sim

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// besideTarget returns a study world with the player facing the target from the cell before it,
// holding a sword in the right hand, and the target with hp hit points.
func besideTarget(t *testing.T, hp int, edit ...func(p *content.Pack)) (*World, *Thing) {
	t.Helper()
	edits := append([]func(p *content.Pack){func(p *content.Pack) {
		p.Things[devpack.ThingTarget].HitPoints = content.Dice{Bonus: hp}
		p.Things[devpack.ThingTarget].Stats = map[string]int{"accuracy": 0, "bonus": 0, "resistance": 0}
	}}, edit...)
	w := studyWorld(t, cell{devpack.TargetCell[0], devpack.TargetCell[1] - 1}, 2, edits...)
	p, tgt := w.Player(), w.Current().Thing("target")
	p.Slots[SlotRightHand] = w.newObject(devpack.ItemSword)
	// Within a step of the target: a blow reaches what the player's box would overlap a step
	// ahead. 14 units back along v the boxes are clear; a step of 8 overlaps them.
	p.U, p.V = tgt.U, tgt.V-14
	return w, tgt
}

// hitOnce strikes with the right hand until a blow lands, and returns that update's events.
func hitOnce(t *testing.T, w *World) []Event {
	t.Helper()
	for range 200 {
		w.Player().Delays[0] = 0
		if ev := do(w, input.Strike{Hand: 0}); has(ev, EventMessage, content.MsgHit) || hasEvent(ev, EventMessage, content.MsgHit) {
			return ev
		}
	}
	t.Fatal("no blow landed")
	return nil
}

func TestCountedWeaponGrows(t *testing.T) {
	w, _ := besideTarget(t, 1000, func(p *content.Pack) {
		p.Items[devpack.ItemSword].Grows = &content.Grows{Into: devpack.ItemSack, At: 5, First: devpack.ThingSage,
			Grown: devpack.ThingSage, Counter: "var.level"}
	})
	p := w.Player()
	hitOnce(t, w)
	if w.Talk() == nil || p.WeaponCount == 0 {
		t.Fatalf("the first hit: talk %v, count %d", w.Talk(), p.WeaponCount)
	}
	do(w, input.Leave{})
	p.WeaponCount = 5
	hitOnce(t, w)
	if o := p.Slots[SlotRightHand]; o == nil || o.Item != devpack.ItemSack || w.Counters()["var.level"] != 1 || w.Talk() == nil {
		t.Errorf("growing: hand %+v, counters %v, talk %v", o, w.Counters(), w.Talk())
	}
}

func TestGrownWeaponLevels(t *testing.T) {
	w, _ := besideTarget(t, 1000, func(p *content.Pack) {
		p.Items[devpack.ItemSword].Levels = &content.Levels{Every: 3, Max: 2, Counter: "var.level", Talk: devpack.ThingSage}
	})
	p := w.Player()
	w.setCounter("var.level", 1)
	p.WeaponCount = 3
	hitOnce(t, w)
	if w.Counters()["var.level"] != 2 || p.WeaponCount != 0 || w.Talk() == nil {
		t.Fatalf("a level: counters %v, count %d", w.Counters(), p.WeaponCount)
	}
	do(w, input.Leave{})
	p.WeaponCount = 3
	hitOnce(t, w)
	if w.Counters()["var.level"] != 2 || p.WeaponCount == 0 {
		t.Errorf("past the last level: counters %v, count %d", w.Counters(), p.WeaponCount)
	}
}

func TestPoisoningWeaponAndKillMessage(t *testing.T) {
	w, tgt := besideTarget(t, 1000, func(p *content.Pack) {
		p.Items[devpack.ItemSword].Poisons = true
		p.Things[devpack.ThingTarget].Name = "item.rock.name"
		p.Things[devpack.ThingTarget].KillPractice = 2
	})
	hitOnce(t, w)
	if !tgt.Poisoned {
		t.Error("the poisoning sword did not poison")
	}
	p := w.Player()
	before := p.Skills[SkillWeapon]
	tgt.HitPoints = 1
	ev := hitOnce(t, w)
	if !hasEvent(ev, EventMessage, content.MsgKilled) {
		t.Errorf("no kill message: %v", ev)
	}
	got := p.Skills[SkillWeapon]
	if got.Level == before.Level && got.Practice != before.Practice+2 {
		t.Errorf("kill practice: %+v, then %+v", before, got)
	}
}

func TestMirroringCreature(t *testing.T) {
	w, tgt := besideTarget(t, 50, func(p *content.Pack) {
		tm := p.Things[devpack.ThingTarget]
		tm.Behaviors = append(tm.Behaviors, content.BehaviorAnimated, content.BehaviorMirrors)
	})
	p := w.Player()
	hp := p.HitPoints
	hitOnce(t, w)
	if tgt.HitPoints != 50 || p.HitPoints >= hp {
		t.Errorf("the target has %d, the player %d of %d", tgt.HitPoints, p.HitPoints, hp)
	}
}

func TestChargesAbsorb(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2, func(p *content.Pack) {
		p.Items[devpack.ItemPotion].Power = powerCharges
	})
	p := w.Player()
	p.Status.Charges = true
	p.Slots[SlotLeftHand] = &Object{Item: devpack.ItemPotion, Value: 3}
	hp := p.HitPoints
	w.hurtPlayer(2, w.kind(content.KindFire), "test")
	if p.HitPoints != hp || p.Slots[SlotLeftHand].Value != 1 {
		t.Fatalf("2 fire: %d of %d hit points, %d charges", p.HitPoints, hp, p.Slots[SlotLeftHand].Value)
	}
	w.hurtPlayer(5, w.kind(content.KindFire), "test")
	// Out of charges, the object stays in the hand.
	if o := p.Slots[SlotLeftHand]; p.HitPoints != hp-4 || o.Item != devpack.ItemPotion || o.Value != 0 {
		t.Errorf("5 fire: %d of %d hit points, hand %+v", p.HitPoints, hp, o)
	}
	w.hurtPlayer(3, combat.Physical, "test")
	// The player wears no body armor, so armor takes nothing off.
	if p.HitPoints != hp-7 {
		t.Errorf("physical damage was absorbed: %d", p.HitPoints)
	}
}

func TestBreakingRunsItsActions(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2, func(p *content.Pack) {
		xp, kind := 100, devpack.ThingTarget
		it := p.Items[devpack.ItemPotion]
		it.BreaksInto = devpack.ItemRock
		it.OnBreak = []content.Action{{SetCounter: &content.SetCounter{Counter: "var.broke", Value: 1}},
			{RemoveAll: &kind}, {Experience: &xp}}
	})
	p := w.Player()
	p.Slots[SlotRightHand] = w.newObject(devpack.ItemPotion)
	do(w, input.Throw{Hand: 0})
	until(w, 500, func() bool { return w.Counters()["var.broke"] == 1 })
	if w.Current().Thing("target") != nil || p.Experience < 100 || w.Counters()["var.broke"] != 1 {
		t.Errorf("breaking: target %v, experience %d, counters %v", w.Current().Thing("target"), p.Experience, w.Counters())
	}
}

func TestAnimatedCreatureAndSpin(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2, func(p *content.Pack) {
		tm := p.Things[devpack.ThingTarget]
		tm.Behaviors = []string{content.BehaviorAnimated}
		tm.Animate = &content.Animate{Kind: content.AnimateCycle, Frames: []int{0, 1, 2, 1}}
	})
	tgt := w.Current().Thing("target")
	seen := map[int]bool{}
	until(w, 200, func() bool {
		if tgt.Frame != nil {
			seen[*tgt.Frame] = true
		}
		return len(seen) == 3
	})
	if len(seen) != 3 {
		t.Errorf("the animated target showed frames %v", seen)
	}

	// A creature a teleporter takes spins before it acts again.
	w = newWorld(t)
	do(w, input.Move{Thing: "guard", Cell: cell{1, 2}})
	g := w.Current().Thing("guard")
	if g.spin != spinTicks {
		t.Fatalf("the guard spins %d", g.spin)
	}
	facings := map[int]bool{}
	for range Steps(spinTicks) + 10 {
		facings[g.Facing] = true
		w.Update(nil)
	}
	if g.spin != 0 || len(facings) != 4 {
		t.Errorf("spin left %d, facings %v", g.spin, facings)
	}
}

func TestQuietCombat(t *testing.T) {
	w, _ := besideTarget(t, 1000, func(p *content.Pack) { p.Manifest.Counters.QuietCombat = "var.quiet" })
	w.setCounter("var.quiet", 1)
	for range 50 {
		w.Player().Delays[0] = 0
		if ev := do(w, input.Strike{Hand: 0}); hasEvent(ev, EventMessage, content.MsgHit) || hasEvent(ev, EventMessage, content.MsgMiss) {
			t.Fatalf("a combat message while quiet: %v", ev)
		}
	}
}

// TestMagnified magnifies floor objects while the pack's magnify counter is set or the player's
// timed sight lasts (docs/spec/items.md, Floor).
func TestMagnified(t *testing.T) {
	w, _ := besideTarget(t, 1000, func(p *content.Pack) { p.Manifest.Counters.Magnify = "var.magnify" })
	if w.Magnified() {
		t.Fatal("magnified at the start")
	}
	w.setCounter("var.magnify", 1)
	if !w.Magnified() {
		t.Error("not magnified with the counter set")
	}
	w.setCounter("var.magnify", 0)
	w.Player().Timed[TimedSight] = 5
	if !w.Magnified() {
		t.Error("not magnified with timed sight")
	}
}

// TestUseSlot drinks a potion where it lies, on a shelf and in the pack, with the right hand's
// delay; the potion's place takes its spent object, or empties (docs/spec/items.md, Using from a
// slot).
func TestUseSlot(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2, func(p *content.Pack) {
		p.Items[devpack.ItemPotion].Spent = devpack.ItemRock
	})
	p := w.Player()
	p.HitPoints = 1
	p.Slots[3] = &Object{Item: devpack.ItemPotion}
	w.Update([]input.Command{input.UseSlot{Slot: 3}})
	if p.HitPoints <= 1 || p.Slots[3] == nil || p.Slots[3].Item != devpack.ItemRock || p.Delays[0] == 0 {
		t.Fatalf("drinking from shelf 3: %d hit points, shelf %+v, delay %d", p.HitPoints, p.Slots[3], p.Delays[0])
	}
	// Not while the right hand is busy; a rock cannot be used.
	p.HitPoints = 1
	p.Slots[4] = &Object{Item: devpack.ItemPotion}
	if ev := w.Update([]input.Command{input.UseSlot{Slot: 4}}); !has(ev, EventRefused, "the hand is not ready") || p.Slots[4] == nil {
		t.Errorf("drinking with the hand busy: %v", ev)
	}
	p.Delays[0] = 0
	if ev := w.Update([]input.Command{input.UseSlot{Slot: 3}}); !has(ev, EventRefused, "") {
		t.Errorf("using a rock: %v", ev)
	}
	// From a bag in the left hand, with no spent object: the pack slot empties.
	w.pack.Items[devpack.ItemPotion].Spent = ""
	p.Delays[0] = 0
	p.Slots[SlotLeftHand] = &Object{Item: devpack.ItemSack, Contents: []*Object{{Item: devpack.ItemKey}, {Item: devpack.ItemPotion}}}
	w.Update([]input.Command{input.UseSlot{Slot: SlotPack + 1}})
	if bag := p.Slots[SlotLeftHand]; p.HitPoints <= 1 || len(bag.Contents) != 1 || bag.Contents[0].Item != devpack.ItemKey {
		t.Errorf("drinking from the pack: %d hit points, bag %+v", p.HitPoints, bag.Contents)
	}
}
