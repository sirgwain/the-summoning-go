package sim

import (
	"slices"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// fightPack is a walk pack whose player always hits (agility 60) and whose creatures are:
// "guard", a standing creature with 20 hit points that leaves a corpse and carries a key; and
// "wolf" (W in the map), a hostile walker that always hits for 3.
func fightPack(rows ...string) *content.Pack {
	for i, r := range rows {
		b := []byte(r)
		for j := range b {
			if b[j] == 'W' {
				b[j] = 'G'
			}
		}
		rows[i] = string(b)
	}
	p := walkPack(rows...)
	p.Things["thing.player"].Stats = map[string]int{AttrStrength: 10, AttrAgility: 60, AttrEndurance: 10, AttrAccuracy: 12, "weapon.0": 1}
	g := p.Things["thing.guard"]
	g.Behaviors = []string{content.BehaviorLeavesCorpse, content.BehaviorStationary}
	g.Experience = content.Dice{Bonus: 600}
	sword := 0
	p.Items["item.sword"] = &content.Item{ID: "item.sword", Weight: 10, Size: 3, Durability: 6000, Wear: content.WearHand,
		WeaponClass: &sword, Actions: []content.ItemAction{{Kind: content.ActionAttack, Damage: content.Dice{Bonus: 4}, Delay: 5}}}
	p.Items["item.boots"] = &content.Item{ID: "item.boots", Weight: 1, Size: 2, Wear: content.WearFeet, Armor: 1, Power: 0x0A, PowerWorn: true}
	p.Items["item.mail"] = &content.Item{ID: "item.mail", Weight: 20, Size: 3, Wear: content.WearBody, Armor: 4, Durability: 7500}
	p.Things["thing.wolf"] = &content.Thing{ID: "thing.wolf", Kind: content.ThingCreature, Speed: 8, Weight: 100,
		HitPoints: content.Dice{Bonus: 50}, Attack: content.Dice{Bonus: 3}, Stats: map[string]int{"accuracy": 100},
		Behaviors: []string{content.BehaviorHostile}}
	return p
}

func fightWorld(t *testing.T, edit func(p *content.Pack), rows ...string) *World {
	t.Helper()
	p := fightPack(rows...)
	if edit != nil {
		edit(p)
	}
	w, err := New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// besideGuard stands the player just short of the guard along v, a step from touching it.
func besideGuard(w *World) {
	w.Player().V = w.Current().Thing("guard").V - 16
}

// strike strikes and lets the hand's delay run out.
func strike(w *World, hand, action int) []Event {
	ev := w.Update([]input.Command{input.Strike{Hand: hand, Action: action}})
	for range 20 {
		w.Update(nil)
	}
	return ev
}

func TestStrikeKillsCreature(t *testing.T) {
	w := fightWorld(t, nil,
		"#####",
		"#@G.#",
		"#####")
	l, p := w.Current(), w.Player()
	g := l.Thing("guard")
	g.Items = []*Object{{Item: "item.key"}}
	p.Facing = 2
	besideGuard(w)
	ev := strike(w, 0, 0)
	if !has(ev, EventMessage, content.MsgHit) || g.HitPoints >= 20 {
		t.Fatalf("bare hands did not hit: %v, %d hit points", ev, g.HitPoints)
	}
	for range 30 {
		if g.HitPoints == 0 {
			break
		}
		strike(w, 0, 0)
	}
	if g.HitPoints != 0 || l.Thing("guard") == nil {
		t.Fatalf("the guard is not a corpse: %d hit points", g.HitPoints)
	}
	if len(l.Objects[g.Cell()]) != 1 || len(g.Items) != 0 {
		t.Error("the guard did not drop its key")
	}
	if p.Experience < 600 || p.Level != 1 || p.MaxHitPoints <= 30 {
		t.Errorf("experience %d, level %d, %d hit points", p.Experience, p.Level, p.MaxHitPoints)
	}
	if ev := strike(w, 0, 0); !has(ev, EventMessage, content.MsgNotACreature) {
		t.Errorf("striking a corpse: %v", ev)
	}
}

func TestStrikeMissesAndWaits(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) { p.Things["thing.guard"].Stats = map[string]int{"accuracy": 200} },
		"#####",
		"#@G.#",
		"#####")
	w.Player().Facing = 2
	besideGuard(w)
	ev := w.Update([]input.Command{input.Strike{}, input.Strike{}})
	if !has(ev, EventMessage, content.MsgMiss) || !has(ev, EventRefused, "the hand is not ready") {
		t.Errorf("a miss, then a hand not ready: %v", ev)
	}
}

func TestWeaponPracticeAndWear(t *testing.T) {
	w := fightWorld(t, nil,
		"#####",
		"#@G.#",
		"#####")
	p := w.Player()
	p.Facing = 2
	besideGuard(w)
	p.Hand = w.newObject("item.sword")
	w.Update([]input.Command{input.PutOn{}})
	sword := p.Slots[SlotRightHand]
	if sword == nil {
		t.Fatal("the sword did not go in the right hand")
	}
	strike(w, 0, 0)
	if sword.Value >= 6000 || p.Skills[SkillWeapon].Practice != 1 {
		t.Errorf("sword value %d, practice %d", sword.Value, p.Skills[SkillWeapon].Practice)
	}
	sword.Value = 1
	ev := strike(w, 0, 0)
	if !has(ev, EventMessage, content.MsgWeaponBreaks) || p.Slots[SlotRightHand] != nil {
		t.Errorf("the worn-out sword did not break: %v", ev)
	}
	if !wornOut(ev, "item.sword", SlotRightHand) {
		t.Errorf("no worn_out event for the right hand: %v", ev)
	}
}

// wornOut reports whether events say item broke or ran out in slot.
func wornOut(events []Event, item string, slot int) bool {
	for _, e := range events {
		if e.Kind == EventObject && e.Detail == "worn_out" && e.Item == item && e.Slot == slot {
			return true
		}
	}
	return false
}

func TestWearing(t *testing.T) {
	w := fightWorld(t, nil,
		"####",
		"#@.#",
		"####")
	p := w.Player()
	p.Hand = w.newObject("item.boots")
	if ev := w.Update([]input.Command{input.Swap{Slot: SlotHead}}); !has(ev, EventRefused, "that is not worn there") {
		t.Fatalf("boots went on the head: %v", ev)
	}
	w.Update([]input.Command{input.PutOn{}})
	if p.Slots[SlotFeet] == nil || p.ArmorClass != 1 || p.IntervalBonus != -2 {
		t.Fatalf("boots on: armor %d, interval bonus %d", p.ArmorClass, p.IntervalBonus)
	}
	p.Hand = w.newObject("item.mail")
	w.Update([]input.Command{input.PutOn{}})
	if p.ArmorClass != 5 {
		t.Errorf("mail and boots: armor %d", p.ArmorClass)
	}
	w.Update([]input.Command{input.Swap{Slot: SlotFeet}})
	if p.IntervalBonus != 0 || p.ArmorClass != 4 || p.Hand == nil {
		t.Errorf("boots off: armor %d, interval bonus %d", p.ArmorClass, p.IntervalBonus)
	}
	// A blow wears the body armor by up to the armor class and the armor takes it off.
	hp := p.HitPoints
	w.hurtPlayer(10, 0, "a test")
	if p.HitPoints != hp-6 || p.Slots[SlotBody].Value != 7500-4 {
		t.Errorf("a blow of 10 through mail: %d hit points taken, mail at %d", hp-p.HitPoints, p.Slots[SlotBody].Value)
	}
}

func TestCreatureChasesAndStrikes(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		l := p.Levels["level.walk"]
		l.Things[0].Thing = "thing.wolf"
	},
		"#######",
		"#@...W#",
		"#######")
	p := w.Player()
	for range 600 {
		w.Update(nil)
		if p.Dead {
			break
		}
	}
	if !p.Dead || p.HitPoints != 0 {
		t.Errorf("the wolf did not kill the player: %d hit points", p.HitPoints)
	}
	if ev := w.Update([]input.Command{input.Walk{Direction: 2}, input.Strike{}}); !has(ev, EventRefused, "the player is dead") {
		t.Errorf("a dead player acts: %v", ev)
	}
}

func TestWanderingCreatureStaysOnFloor(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		p.Things["thing.wolf"].Behaviors = nil
		p.Levels["level.walk"].Things[0].Thing = "thing.wolf"
	},
		"########",
		"#@.....#",
		"#..O..W#",
		"########")
	l := w.Current()
	moved := false
	start := l.Thing("guard").Cell()
	for range 3000 {
		w.Update(nil)
		g := l.Thing("guard")
		if g == nil {
			t.Fatal("the wolf fell in the pit")
		}
		if c := g.Cell(); l.Blocked(c) {
			t.Fatalf("the wolf walked into %v", c)
		} else if c != start {
			moved = true
		}
	}
	if !moved {
		t.Error("the wolf never wandered")
	}
	if w.Player().HitPoints != 30 {
		t.Error("a wolf that is not hostile hurt the player")
	}
}

func TestThrowHitsAndLands(t *testing.T) {
	w := fightWorld(t, nil,
		"#########",
		"#@..G...#",
		"#########")
	p, l := w.Player(), w.Current()
	g := l.Thing("guard")
	p.Facing = 2
	p.Slots[SlotRightHand] = w.newObject("item.rock")
	w.Update([]input.Command{input.Throw{Hand: 0}})
	if p.Slots[SlotRightHand] != nil || len(l.Flights) != 1 {
		t.Fatalf("the rock did not fly: hand %v, %d flights", p.Slots[SlotRightHand], len(l.Flights))
	}
	for range 200 {
		w.Update(nil)
	}
	if len(l.Flights) != 0 || g.HitPoints != 20-(60/10+1) || !g.Hostile {
		t.Errorf("the rock: %d flights, guard at %d hit points, hostile %v", len(l.Flights), g.HitPoints, g.Hostile)
	}
	landed := 0
	for _, objs := range l.Objects {
		landed += len(objs)
	}
	if landed != 1 {
		t.Errorf("%d objects landed", landed)
	}
}

func TestArrowTrapAndBow(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		p.Items["item.arrow"] = &content.Item{ID: "item.arrow", Weight: 1, Size: 1, Tags: []string{"arrows"},
			Missile: &content.Missile{Damage: content.Dice{Bonus: 5}}}
		p.Items["item.quiver"] = &content.Item{ID: "item.quiver", Weight: 1, Size: 2, Wear: content.WearQuiver,
			Container: &content.Container{Capacity: 4, Largest: 1}}
		bow := 3
		p.Items["item.bow"] = &content.Item{ID: "item.bow", Weight: 3, Size: 3, Wear: content.WearHand, WeaponClass: &bow,
			Actions: []content.ItemAction{{Kind: content.ActionFire, Delay: 3, NeedsRoom: true}}}
		p.Levels["level.walk"].Procedures["trap"] = []content.Action{{ShootItem: &content.ShootItem{Cell: cell{1, 6}, Item: "item.arrow", Direction: 6}}}
	},
		"########",
		"#@.....#",
		"########")
	p, l := w.Player(), w.Current()
	w.Update([]input.Command{input.Run{Procedure: "trap"}})
	for range 200 {
		w.Update(nil)
	}
	if p.HitPoints != 30-5 || len(l.Objects) != 0 {
		t.Fatalf("the trap's arrow: %d hit points, %d cells with objects (a trap's arrow vanishes)", p.HitPoints, len(l.Objects))
	}
	p.Facing = 2
	p.Slots[SlotRightHand] = w.newObject("item.bow")
	p.Hand = w.newObject("item.quiver")
	w.Update([]input.Command{input.PutOn{}})
	if ev := strike(w, 0, 0); !has(ev, EventMessage, content.MsgNoArrows) {
		t.Errorf("firing from an empty quiver: %v", ev)
	}
	l.Objects[p.Cell()] = []*Object{w.newObject("item.arrow")}
	w.Update([]input.Command{input.PickUp{}})
	if q := p.Slots[SlotQuiver]; len(q.Contents) != 1 || p.Hand != nil {
		t.Fatal("the arrow picked up did not go into the quiver")
	}
	strike(w, 0, 0)
	if len(p.Slots[SlotQuiver].Contents) != 0 || p.Skills[SkillWeapon+3].Practice != 1 {
		t.Errorf("firing: quiver %v, bow practice %d", p.Slots[SlotQuiver].Contents, p.Skills[SkillWeapon+3].Practice)
	}
}

func TestThrownSphereUnsealsDoor(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		for _, f := range p.Levels["level.walk"].Features {
			if f.ID == "door" {
				f.SealedBy = "item.key"
			}
		}
	},
		"#######",
		"#@...D#",
		"#######")
	p, l := w.Player(), w.Current()
	p.Facing = 2
	p.Slots[SlotLeftHand] = w.newObject("item.key")
	w.Update([]input.Command{input.Throw{Hand: 1}})
	for range 200 {
		w.Update(nil)
	}
	if f := l.Feature("door"); f.SealedBy != "" || f.State != content.DoorOpen || len(l.Objects) != 0 {
		t.Errorf("door %+v, objects %v", f.State, l.Objects)
	}
}

func TestUsingObjects(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		p.Items["item.bottle"] = &content.Item{ID: "item.bottle", Weight: 1, Size: 1}
		p.Items["item.potion"] = &content.Item{ID: "item.potion", Weight: 2, Size: 1, Power: 0x28, Spent: "item.bottle",
			Actions: []content.ItemAction{{Kind: content.ActionUseUp, Delay: 3}}}
		p.Items["item.wand"] = &content.Item{ID: "item.wand", Weight: 1, Size: 1, Power: 0x0F, Charges: 2,
			Actions: []content.ItemAction{{Kind: content.ActionUse, Delay: 3}}}
		p.Items["item.rune"] = &content.Item{ID: "item.rune", Weight: 1, Size: 1, Power: 0x28,
			Actions: []content.ItemAction{{Kind: content.ActionUseUp, Delay: 3}}}
	},
		"####",
		"#@.#",
		"####")
	p := w.Player()
	p.Slots[SlotRightHand] = w.newObject("item.potion")
	if ev := strike(w, 0, 0); !has(ev, EventMessage, content.MsgFullHealth) || p.Slots[SlotRightHand].Item != "item.potion" {
		t.Fatalf("a healing potion at full health: %v", ev)
	}
	p.HitPoints = 5
	strike(w, 0, 0)
	if p.HitPoints <= 5 || p.Slots[SlotRightHand].Item != "item.bottle" {
		t.Errorf("drinking: %d hit points, hand %v", p.HitPoints, p.Slots[SlotRightHand])
	}
	p.Slots[SlotLeftHand] = w.newObject("item.wand")
	strike(w, 1, 0)
	if !p.Status.Levitating || p.Slots[SlotLeftHand].Value != 1 {
		t.Errorf("the wand: levitating %v, charges %d", p.Status.Levitating, p.Slots[SlotLeftHand].Value)
	}
	strike(w, 1, 0)
	if o := p.Slots[SlotLeftHand]; o == nil || o.Item != "item.wand" || o.Value != 0 {
		t.Fatalf("the wand out of charges: %+v", o)
	}
	if ev := strike(w, 1, 0); !has(ev, EventRefused, "") || p.Slots[SlotLeftHand].Value != 0 {
		t.Errorf("a wand with no charges was used: %v", ev)
	}
	p.HitPoints = 5
	p.Slots[SlotRightHand] = w.newObject("item.rune")
	strike(w, 0, 0)
	if p.HitPoints <= 5 || p.Slots[SlotRightHand] != nil {
		t.Errorf("a used-up object with no spent object: %d hit points, hand %v", p.HitPoints, p.Slots[SlotRightHand])
	}
}

func TestStatusesAndStartingInventory(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		p.Things["thing.player"].Inventory = []content.InventoryObject{
			{Slot: SlotFeet, Object: content.Object{Item: "item.boots"}}, {Slot: 0, Object: content.Object{Item: "item.key"}},
		}
	},
		"#####",
		"#@O.#",
		"#####")
	p := w.Player()
	if p.Slots[SlotFeet] == nil || p.IntervalBonus != -2 || p.ArmorClass != 1 || p.Slots[0] == nil {
		t.Fatalf("starting inventory: feet %v, bonus %d, armor %d", p.Slots[SlotFeet], p.IntervalBonus, p.ArmorClass)
	}
	// Levitating, the player walks over the pit.
	p.Status.Levitating = true
	walk(w, 2, 6)
	if w.Current().ID != "level.walk" || p.Cell() != (cell{1, 3}) {
		t.Errorf("levitating over the pit: %s %v", w.Current().ID, p.Cell())
	}
	p.addTimed(TimedHaste, 1)
	if got := p.IntervalTicks(w.Pack()); got != -2-1-1 {
		t.Errorf("hasted interval: %+d ticks", got)
	}
	for range Steps(statusTicks) + 2 {
		w.Update(nil)
	}
	if p.Timed[TimedHaste] != 0 {
		t.Error("haste did not run out")
	}
	// Exertion past END + 3 costs a point of endurance.
	end := p.Attributes[AttrEndurance]
	w.exert(end + 4)
	if p.Attributes[AttrEndurance] != end-1 || p.Fatigue != 1 {
		t.Errorf("exertion: endurance %d, fatigue %d", p.Attributes[AttrEndurance], p.Fatigue)
	}
}

func TestSkillRiseFanfares(t *testing.T) {
	w := fightWorld(t, nil, "###", "#@#", "###")
	p := w.Player()
	cues := func() []string {
		var out []string
		for _, e := range w.events {
			if e.Kind == EventSound {
				out = append(out, e.Detail)
			}
		}
		w.events = nil
		return out
	}
	w.events = nil
	w.practice(SkillSchool+1, p.Skills[SkillSchool+1].Threshold)
	if got := cues(); !slices.Equal(got, []string{content.CueMagicSkillUp}) {
		t.Errorf("a school rising: %v", got)
	}
	p.Experience = p.NextLevel - 1
	w.practice(SkillWeapon, p.Skills[SkillWeapon].Threshold)
	if got := cues(); !slices.Equal(got, []string{content.CueWeaponSkillUp, content.CueLevelUp}) {
		t.Errorf("a weapon skill rising to a level: %v", got)
	}
}

// playerUpdate runs the world through the player's next update, asking for cmds first.
func playerUpdate(w *World, cmds ...input.Command) {
	p := w.Player()
	last := p.last
	w.Update(cmds)
	for p.last == last {
		w.Update(nil)
	}
}

func TestSwingPoseHolds(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) { p.Things["thing.player"].IntervalMS = 275 },
		"######",
		"#@...#",
		"######")
	p := w.Player()
	p.Facing = 2
	playerUpdate(w)
	w.Update([]input.Command{input.Strike{Hand: 1}})
	if p.Pose != "attack.left" {
		t.Fatalf("the left hand's swing shows pose %q", p.Pose)
	}
	u, v := p.U, p.V
	for i := range poseUpdates {
		playerUpdate(w, input.Walk{Direction: 2})
		if p.Pose != "attack.left" || p.U != u || p.V != v {
			t.Fatalf("update %d of the pose: pose %q, moved %d,%d", i+1, p.Pose, p.U-u, p.V-v)
		}
	}
	playerUpdate(w)
	if p.Pose != "" {
		t.Errorf("the pose still shows after %d updates", poseUpdates+1)
	}
	playerUpdate(w, input.Walk{Direction: 2})
	if p.U == u && p.V == v {
		t.Error("the player does not walk after the pose")
	}
}

func TestUseSwings(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		class := 0
		p.Items["item.staff"] = &content.Item{ID: "item.staff", Weight: 8, Size: 3, Wear: content.WearHand, WeaponClass: &class,
			Tags: []string{"weapon", "two_handed"}, Power: 0x28, Charges: 5,
			Actions: []content.ItemAction{{Kind: content.ActionUse, Delay: 9, Swings: true}}}
	},
		"#####",
		"#@G.#",
		"#####")
	g := w.Current().Thing("guard")
	p := w.Player()
	p.Facing = 2
	besideGuard(w)
	p.Slots[SlotLeftHand] = w.newObject("item.staff")
	hp := g.HitPoints
	ev := w.Update([]input.Command{input.Strike{Hand: 1}})
	if p.Pose != "attack.both" {
		t.Errorf("using the staff shows pose %q", p.Pose)
	}
	if g.HitPoints != hp || has(ev, EventMessage, content.MsgHit) || has(ev, EventMessage, content.MsgMiss) {
		t.Errorf("using the staff struck a blow: %v", ev)
	}
	if p.Delays[0] != p.Delays[1] || p.Delays[1] < 8 {
		t.Errorf("a two-handed use delays the hands %v", p.Delays)
	}
}

func TestHandWait(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) { p.Things["thing.player"].IntervalMS = 275 },
		"#####",
		"#@..#",
		"#####")
	p := w.Player()
	playerUpdate(w)
	w.Update(nil)
	if got := w.HandWait(0); got != 0 {
		t.Fatalf("a ready hand waits %v", got)
	}
	w.Update([]input.Command{input.Strike{}})
	if got := w.HandWait(0); got < 0.9 {
		t.Fatalf("just after striking the hand waits %v, want nearly 1", got)
	}
	if got := w.HandWait(1); got != 0 {
		t.Errorf("the other hand waits %v", got)
	}
	last := 1.0
	for p.Delays[0] > 0 {
		w.Update(nil)
		got := w.HandWait(0)
		if got > last || got < 0 {
			t.Fatalf("the wait went from %v to %v", last, got)
		}
		last = got
	}
	if last != 0 {
		t.Errorf("a ready hand waits %v", last)
	}
}

func TestGestureWait(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2)
	do(w, input.Learn{Gesture: "DJ"})
	if got := w.GestureWait(); got != 0 {
		t.Fatalf("with no gesture playing the wait is %v", got)
	}
	do(w, input.Cast{School: 0})
	if got := w.GestureWait(); got < 0.9 {
		t.Fatalf("as the gesture starts the wait is %v, want nearly 1", got)
	}
	last := 1.0
	until(w, 2000, func() bool {
		got := w.GestureWait()
		if got > last {
			t.Fatalf("the wait went from %v to %v", last, got)
		}
		last = got
		return w.Player().Magic.Gesture == nil
	})
	if got := w.GestureWait(); got != 0 {
		t.Errorf("after the cast the wait is %v", got)
	}
}

// TestSharedCharges uses objects whose item shares one count of charges: using one drains the
// other, making a new one refills it, the count is saved, and a weapon with no charges of its own
// cannot be used.
func TestSharedCharges(t *testing.T) {
	class := 0
	w := fightWorld(t, func(p *content.Pack) {
		p.Items["item.sword"] = &content.Item{ID: "item.sword", Weight: 4, Size: 1, Power: 0x0F, Durability: 1000,
			Charges: 2, SharedCharges: true, WeaponClass: &class, Tags: []string{"weapon"},
			Actions: []content.ItemAction{{Kind: content.ActionUse, Delay: 3}}}
		p.Items["item.staff"] = &content.Item{ID: "item.staff", Weight: 4, Size: 1, Power: 0x0F, Durability: 1000,
			WeaponClass: &class, Tags: []string{"weapon"}, Actions: []content.ItemAction{{Kind: content.ActionUse, Delay: 3}}}
	},
		"####",
		"#@.#",
		"####")
	p := w.Player()
	p.Slots[SlotRightHand], p.Slots[SlotLeftHand] = w.newObject("item.sword"), w.newObject("item.sword")
	strike(w, 0, 0)
	strike(w, 1, 0)
	if w.Quantity(p.Slots[SlotRightHand]) != 0 || p.Slots[SlotRightHand].Value != 1000 {
		t.Fatalf("two uses left %d charges, durability %d", w.Quantity(p.Slots[SlotRightHand]), p.Slots[SlotRightHand].Value)
	}
	if ev := strike(w, 0, 0); !has(ev, EventRefused, "") {
		t.Errorf("a sword with no shared charges was used: %v", ev)
	}
	st, err := w.State()
	if err != nil {
		t.Fatal(err)
	}
	w.newObject("item.sword")
	if w.Quantity(p.Slots[SlotLeftHand]) != 2 {
		t.Errorf("a new sword did not refill the charges: %d", w.Quantity(p.Slots[SlotLeftHand]))
	}
	if st.Shared["item.sword"] != 0 {
		t.Errorf("saved shared %v", st.Shared)
	}
	p.Slots[SlotRightHand] = w.newObject("item.staff")
	if ev := strike(w, 0, 0); !has(ev, EventRefused, "") || w.Quantity(p.Slots[SlotRightHand]) != 0 {
		t.Errorf("a weapon without charges was used: %v", ev)
	}
}

// TestTimedObjects runs down worn objects whose power lasts while worn: one runs out after its
// value's status ticks, coming off and vanishing; a value of 0 lasts 65535 ticks; objects of a
// shared count each take one from it; an object whose power does not last is not timed.
func TestTimedObjects(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		p.Items["item.bracers"] = &content.Item{ID: "item.bracers", Weight: 1, Size: 1, Wear: content.WearArms,
			Power: 0x17, PowerWorn: true, Charges: 2}
		p.Items["item.boots"] = &content.Item{ID: "item.boots", Weight: 1, Size: 1, Wear: content.WearFeet,
			Power: powerSpeed2Lasting, PowerWorn: true}
		p.Items["item.buckler"] = &content.Item{ID: "item.buckler", Weight: 1, Size: 1, Power: powerSpeed2Lasting,
			PowerWorn: true, Durability: 1000, Charges: 5, SharedCharges: true}
		p.Items["item.ring"] = &content.Item{ID: "item.ring", Weight: 1, Size: 1, Wear: content.WearNeck,
			Power: powerSpeed2, PowerWorn: true, Charges: 1}
	},
		"####",
		"#@.#",
		"####")
	p := w.Player()
	p.Slots[SlotFeet] = nil
	wear := func(slot int, item string) *Object {
		o := w.newObject(item)
		p.Slots[slot] = o
		w.putOn(o)
		return o
	}
	wear(SlotArms, "item.bracers")
	boots := wear(SlotFeet, "item.boots")
	wear(SlotRightHand, "item.buckler")
	buckler := wear(SlotLeftHand, "item.buckler")
	ring := wear(SlotNeck, "item.ring")
	if !p.Status.Regenerating {
		t.Fatal("the bracers do not regenerate")
	}
	w.statusTickPlayer()
	if p.Slots[SlotArms] == nil || w.Quantity(buckler) != 3 || buckler.Value != 1000 || ring.Value != 1 {
		t.Fatalf("one tick: bracers %v, shared %d, buckler %d, ring %d", p.Slots[SlotArms], w.Quantity(buckler), buckler.Value, ring.Value)
	}
	w.events = nil
	w.statusTickPlayer()
	if p.Slots[SlotArms] != nil || p.Status.Regenerating || !has(w.events, EventMessage, content.MsgVanishes) {
		t.Errorf("two ticks: bracers %v, regenerating %v, events %v", p.Slots[SlotArms], p.Status.Regenerating, w.events)
	}
	if !wornOut(w.events, "item.bracers", SlotArms) {
		t.Errorf("no worn_out event for the arms: %v", w.events)
	}
	if boots.Value != 65534 {
		t.Errorf("boots of value 0 have %d after two ticks", boots.Value)
	}
	w.statusTickPlayer()
	if p.Slots[SlotRightHand] != nil || p.Slots[SlotLeftHand] == nil {
		t.Errorf("the shared count ran out: right %v, left %v", p.Slots[SlotRightHand], p.Slots[SlotLeftHand])
	}
}

// TestTwoHandedBlocksItsHand: a hand holding a two-handed sword while the other hand holds
// anything is blocked: it neither strikes nor throws, and says nothing; the other hand still
// acts, and emptying it frees the sword's hand.
func TestTwoHandedBlocksItsHand(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) { p.Items["item.sword"].Tags = []string{"weapon", "two_handed"} },
		"#####",
		"#@G.#",
		"#####")
	p := w.Player()
	p.Facing = 2
	besideGuard(w)
	sword := w.newObject("item.sword")
	p.Slots[SlotRightHand] = sword
	p.Slots[SlotRightHand+1] = w.newObject("item.boots")
	if !w.HandBlocked(0) || w.HandBlocked(1) {
		t.Fatalf("blocked: right %v, left %v", w.HandBlocked(0), w.HandBlocked(1))
	}
	ev := strike(w, 0, 0)
	if !has(ev, EventRefused, "the hand is blocked") || p.Delays[0] != 0 {
		t.Fatalf("struck with a blocked hand: %v, delays %v", ev, p.Delays)
	}
	for _, e := range ev {
		if e.Kind == EventMessage {
			t.Errorf("a blocked hand said %v", e)
		}
	}
	if ev := w.Update([]input.Command{input.Throw{Hand: 0}}); !has(ev, EventRefused, "the hand is blocked") || p.Slots[SlotRightHand] != sword {
		t.Errorf("threw from a blocked hand: %v", ev)
	}
	if ev := strike(w, 1, 0); has(ev, EventRefused, "the hand is blocked") {
		t.Errorf("the other hand is blocked too: %v", ev)
	}
	p.Slots[SlotRightHand+1] = nil
	if ev := strike(w, 0, 0); has(ev, EventRefused, "the hand is blocked") || w.HandBlocked(0) {
		t.Errorf("still blocked with the other hand empty: %v", ev)
	}
}

// TestFlightGlides draws a thing in flight gliding from where it was to where it is, reaching
// it by its next update.
func TestFlightGlides(t *testing.T) {
	w := fightWorld(t, nil,
		"############",
		"#@.........#",
		"############")
	p, l := w.Player(), w.Current()
	p.Facing = 2
	p.Slots[SlotRightHand] = w.newObject("item.rock")
	w.Update([]input.Command{input.Throw{Hand: 0}})
	f := l.Flights[0]
	if g := f.Glide(w.Step()); g.Span != 0 {
		t.Errorf("a flight just thrown glides %+v, want drawn where it is", g)
	}
	glided := false
	for range 30 {
		w.Update(nil)
		if len(l.Flights) == 0 {
			break
		}
		g := f.Glide(w.Step())
		if g.Span == 0 {
			continue
		}
		glided = true
		if g.FromU != f.U || f.V-g.FromV != flightSpeed || g.Left <= 0 || g.Left > g.Span {
			t.Fatalf("step %d: glide %+v to (%d, %d), want 8 along v with some of its span left", w.Step(), g, f.U, f.V)
		}
		if g := f.Glide(f.next); g.Span != 0 {
			t.Fatalf("at its next update it still glides: %+v", g)
		}
	}
	if !glided {
		t.Error("the flight never glided")
	}
}

// TestStrikePastCorpse strikes a living creature standing behind a corpse, rather than meeting
// the corpse and finding nothing to hit.
func TestStrikePastCorpse(t *testing.T) {
	w := fightWorld(t, nil,
		"#######",
		"#@.G..#",
		"#######")
	p, l := w.Player(), w.Current()
	g := l.Thing("guard")
	p.Facing = 2
	besideGuard(w)
	// A corpse lying between them, first among the level's things.
	corpse := *g
	corpse.ID, corpse.HitPoints, corpse.Items, corpse.V = "corpse", 0, nil, g.V-6
	l.Things = append([]*Thing{&corpse}, l.Things...)
	ev := strike(w, 0, 0)
	if has(ev, EventMessage, content.MsgNotACreature) || g.HitPoints >= 20 {
		t.Errorf("the swing met the corpse: %v, the guard at %d hit points", ev, g.HitPoints)
	}
}

// windUpWorld is a wolf beside the player whose blow winds up for 500 ms, run until it starts.
func windUpWorld(t *testing.T) (*World, *Thing) {
	t.Helper()
	w := fightWorld(t, func(p *content.Pack) {
		p.Levels["level.walk"].Things[0].Thing = "thing.wolf"
		p.Things["thing.wolf"].WindUpMS = 500
	},
		"#######",
		"#@.W..#",
		"#######")
	wolf := w.Current().Things[0]
	for range 600 {
		w.Update(nil)
		if wolf.WindUpUntil > 0 {
			break
		}
	}
	if wolf.WindUpUntil != wolf.WindUpFrom+stepsMS(500) {
		t.Fatalf("the wolf did not wind up: from %d until %d", wolf.WindUpFrom, wolf.WindUpUntil)
	}
	if p := w.Player(); p.HitPoints != p.MaxHitPoints {
		t.Fatalf("the wolf struck before its wind-up: %d hit points", p.HitPoints)
	}
	return w, wolf
}

func TestWindUpLands(t *testing.T) {
	w, wolf := windUpWorld(t)
	p := w.Player()
	u, v := wolf.U, wolf.V
	var struck bool
	until := wolf.WindUpUntil
	for wolf.WindUpUntil > 0 && w.Step() <= until {
		struck = has(w.Update(nil), EventThing, "strikes") || struck
		if wolf.U != u || wolf.V != v {
			t.Fatal("the wolf moved while winding up")
		}
	}
	if w.Step() < until {
		t.Errorf("the blow landed at step %d, before %d", w.Step(), until)
	}
	if !struck || p.HitPoints != p.MaxHitPoints-3 {
		t.Errorf("the wound-up blow did not land: struck %v, %d hit points", struck, p.HitPoints)
	}
	if wolf.WindUpUntil != 0 || wolf.Pose != PoseAttack {
		t.Errorf("after the blow: until %d, pose %q", wolf.WindUpUntil, wolf.Pose)
	}
}

func TestWindUpDodged(t *testing.T) {
	w, wolf := windUpWorld(t)
	p := w.Player()
	p.V += 64
	var missed bool
	until := wolf.WindUpUntil
	for wolf.WindUpUntil > 0 && w.Step() <= until {
		missed = has(w.Update(nil), EventThing, "strikes nothing") || missed
	}
	if !missed || p.HitPoints != p.MaxHitPoints {
		t.Errorf("the dodged blow: missed %v, %d hit points", missed, p.HitPoints)
	}
	if wolf.walkDir != restDir {
		t.Errorf("the wolf does not rest after its blow: walk %d", wolf.walkDir)
	}
}

func TestStrikeThawsFrozenCreature(t *testing.T) {
	w := fightWorld(t, nil,
		"#####",
		"#@G.#",
		"#####")
	l, p := w.Current(), w.Player()
	g := l.Thing("guard")
	before := g.next
	w.freeze(l, Steps(1000))
	if g.next != before+Steps(1000) {
		t.Fatalf("freeze delayed the guard to %d, want %d", g.next, before+Steps(1000))
	}
	s, err := w.State()
	if err != nil {
		t.Fatal(err)
	}
	if w, err = Restore(w.pack, s); err != nil {
		t.Fatal(err)
	}
	l, p = w.Current(), w.Player()
	g = l.Thing("guard")
	if g.frozen != Steps(1000) {
		t.Fatalf("restored freeze delay %d, want %d", g.frozen, Steps(1000))
	}
	p.Facing = 2
	besideGuard(w)
	if ev := strike(w, 0, 0); !has(ev, EventMessage, content.MsgHit) {
		t.Fatalf("the strike missed: %v", ev)
	}
	if g.frozen != 0 || g.next > w.step+stepsMS(g.Template.IntervalMS) {
		t.Errorf("the struck guard is still frozen: next %d at step %d", g.next, w.step)
	}
}
