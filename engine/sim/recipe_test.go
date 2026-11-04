package sim

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

func TestTransmuteLearnsRecipe(t *testing.T) {
	w := fightWorld(t, func(p *content.Pack) {
		for _, id := range []string{"item.bottle", "item.gem", "item.potion", "item.jera"} {
			p.Items[id] = &content.Item{ID: id, Weight: 1, Size: 1}
		}
		p.Spells["spell.liquify"] = &content.Spell{ID: "spell.liquify"}
	},
		"#####",
		"#@..#",
		"#####")
	p, l := w.Player(), w.Current()
	sp := w.Pack().Spells["spell.liquify"]
	tr := &content.Transmute{Holder: "item.bottle", Table: map[string]string{"item.gem": "item.potion"}, Otherwise: "item.jera"}
	w.Update(nil)

	p.Slots[SlotRightHand] = w.newObject("item.bottle")
	p.Slots[SlotRightHand+1] = w.newObject("item.gem")
	w.transmute(l, sp, tr, 5)
	p.Slots[SlotRightHand] = w.newObject("item.bottle")
	p.Slots[SlotRightHand+1] = nil
	w.transmute(l, sp, tr, 5)
	// Made again, it is not recorded twice.
	p.Slots[SlotRightHand] = w.newObject("item.bottle")
	p.Slots[SlotRightHand+1] = w.newObject("item.gem")
	w.transmute(l, sp, tr, 5)

	want := []Recipe{
		{Spell: "spell.liquify", Holder: "item.bottle", From: "item.gem", Into: "item.potion"},
		{Spell: "spell.liquify", Holder: "item.bottle", Into: "item.jera"},
	}
	if b := w.Book(); !slices.Equal(b.Recipes, want) {
		t.Errorf("recipes %+v, want %+v", b.Recipes, want)
	}
	for _, id := range []string{"item.bottle", "item.gem", "item.potion", "item.jera"} {
		if e := w.Book().Items[id]; e == nil || e.Stage < StageSeen {
			t.Errorf("%s is not in the book", id)
		}
	}

	// A save keeps them.
	st, err := w.State()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(st)
	var back State
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	r, err := Restore(w.Pack(), &back)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Book().Recipes, want) {
		t.Errorf("restored recipes %+v", r.Book().Recipes)
	}
}
