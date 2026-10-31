package sim

import (
	"testing"

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
	p.Items["item.boots"] = &content.Item{ID: "item.boots", Weight: 1, Size: 2, Wear: content.WearFeet, Armor: 1, Power: 0x0A}
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
