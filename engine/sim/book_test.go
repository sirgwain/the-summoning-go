package sim

import (
	"encoding/json"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

func stage(entries map[string]*Entry, id string) int {
	if e := entries[id]; e != nil {
		return e.Stage
	}
	return 0
}

func TestBookCreatures(t *testing.T) {
	// A wolf that strikes the player is fought.
	w := fightWorld(t, func(p *content.Pack) { p.Levels["level.walk"].Things[0].Thing = "thing.wolf" },
		"#######",
		"#@...W#",
		"#######")
	b := w.Book()
	w.Update(nil)
	if got := stage(b.Creatures, "thing.wolf"); got != StageSeen {
		t.Fatalf("the wolf in sight: stage %d", got)
	}
	if e := b.Creatures["thing.wolf"]; e.Level != "level.walk" {
		t.Errorf("first seen on %q", e.Level)
	}
	if _, ok := b.Creatures["thing.player"]; ok {
		t.Error("the player is in the book")
	}
	for range 300 {
		w.Update(nil)
		if stage(b.Creatures, "thing.wolf") >= StageFought {
			break
		}
	}
	if got := stage(b.Creatures, "thing.wolf"); got != StageFought {
		t.Fatalf("the wolf after striking: stage %d", got)
	}

	// A guard killed is counted, with what it drops, each item once.
	w = fightWorld(t, nil,
		"#####",
		"#@G.#",
		"#####")
	l, b := w.Current(), w.Book()
	g := l.Thing("guard")
	g.Items = []*Object{{Item: "item.key"}, {Item: "item.key"}}
	w.Player().Facing = 2
	besideGuard(w)
	for range 40 {
		if g.HitPoints == 0 {
			break
		}
		strike(w, 0, 0)
	}
	e := b.Creatures["thing.guard"]
	if e == nil || e.Stage != StageKilled || e.Kills != 1 || len(e.Drops) != 1 || e.Drops[0] != "item.key" {
		t.Fatalf("the guard killed: %+v", e)
	}
	if got := stage(b.Items, "item.key"); got != StageSeen {
		t.Errorf("the dropped key: stage %d", got)
	}
}

func TestBookItems(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		p.Items["item.potion"] = &content.Item{ID: "item.potion", Weight: 2, Size: 1, Power: 0x28,
			Actions: []content.ItemAction{{Kind: content.ActionUseUp, Delay: 3}}}
		p.Items["item.bag"] = &content.Item{ID: "item.bag", Weight: 1, Size: 2, Container: &content.Container{Capacity: 4, Largest: 2}}
	},
		"####",
		"#@.#",
		"####")
	p, b := w.Player(), w.Book()
	bag := w.newObject("item.bag")
	bag.Contents = []*Object{w.newObject("item.key")}
	p.Slots[SlotLeftHand] = bag
	w.Update(nil)
	if got := stage(b.Items, "item.key"); got != StageHandled {
		t.Errorf("a key in a bag in hand: stage %d", got)
	}
	p.Slots[SlotRightHand] = w.newObject("item.potion")
	w.Update(nil)
	if got := stage(b.Items, "item.potion"); got != StageHandled {
		t.Fatalf("a potion in hand: stage %d", got)
	}
	// Refused at full health, it is still tried: the refusal tells what it does.
	strike(w, 0, 0)
	if got := stage(b.Items, "item.potion"); got != StageUsed || p.Slots[SlotRightHand] == nil {
		t.Fatalf("a potion refused: stage %d, hand %v", got, p.Slots[SlotRightHand])
	}

	// The book is kept in saves.
	s, err := w.State()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back State
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	r, err := Restore(w.Pack(), &back)
	if err != nil {
		t.Fatal(err)
	}
	if got := stage(r.Book().Items, "item.potion"); got != StageUsed {
		t.Errorf("restored: potion stage %d", got)
	}

	// Boots worn in a save without a book are in use once the world runs.
	back.Book = nil
	r, err = Restore(w.Pack(), &back)
	if err != nil {
		t.Fatal(err)
	}
	r.Player().Slots[SlotFeet] = r.newObject("item.boots")
	r.Update(nil)
	if got := stage(r.Book().Items, "item.boots"); got != StageUsed {
		t.Errorf("boots worn in a restored world: stage %d", got)
	}
}

// TestBookDefenses learns a creature's defenses only by meeting them: a vulnerability to swords
// once a sword hits it, not bare hands; a resistance to fire once fire hurts it.
func TestBookDefenses(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		g := p.Things["thing.guard"]
		sword := 0
		g.Vulnerability = &sword
		g.HitPoints = content.Dice{Bonus: 500}
		g.Behaviors = append(g.Behaviors, content.BehaviorResistsFire)
	},
		"#####",
		"#@G.#",
		"#####")
	l, p, b := w.Current(), w.Player(), w.Book()
	g := l.Thing("guard")
	p.Facing = 2
	besideGuard(w)
	for range 5 {
		strike(w, 0, 0)
	}
	e := b.Creatures["thing.guard"]
	if e == nil || e.Stage != StageFought || len(e.Defenses) != 0 {
		t.Fatalf("struck with bare hands: %+v", e)
	}
	p.Slots[SlotRightHand] = w.newObject("item.sword")
	for range 10 {
		strike(w, 0, 0)
		if e.Knows(DefenseVulnerable) {
			break
		}
	}
	if !e.Knows(DefenseVulnerable) || e.Knows(content.BehaviorResistsFire) {
		t.Fatalf("struck with a sword: %+v", e)
	}
	w.hurtThing(l, g, 8, combat.Fire)
	if !e.Knows(content.BehaviorResistsFire) || e.Knows(content.BehaviorResistsPoison) {
		t.Errorf("burned: %+v", e)
	}
}
