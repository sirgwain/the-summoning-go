package inspect

import (
	"slices"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

func TestRecipeLines(t *testing.T) {
	p := &content.Pack{
		Texts: map[string]*content.Text{"text.en": {Strings: map[string]string{
			"n.bottle": "Empty Bottle", "n.gem": "Amethyst", "n.teiwaz": "Teiwaz", "n.jera": "Jera", "n.liquify": "Liquify",
		}}},
		Items: map[string]*content.Item{
			"item.bottle": {ID: "item.bottle", Name: "n.bottle"}, "item.gem": {ID: "item.gem", Name: "n.gem"},
			"item.teiwaz": {ID: "item.teiwaz", Name: "n.teiwaz"}, "item.jera": {ID: "item.jera", Name: "n.jera"},
		},
		Spells: map[string]*content.Spell{"spell.liquify": {ID: "spell.liquify", Name: "n.liquify"}},
	}
	recipes := []sim.Recipe{
		{Spell: "spell.liquify", Holder: "item.bottle", From: "item.gem", Into: "item.teiwaz"},
		{Spell: "spell.liquify", Holder: "item.bottle", Into: "item.jera"},
	}
	for id, want := range map[string][]string{
		"item.gem":    {"Liquify, with an Empty Bottle, makes it into Teiwaz."},
		"item.teiwaz": {"Made by Liquify from Amethyst in an Empty Bottle."},
		"item.jera":   {"Made by Liquify in an Empty Bottle from anything else."},
		"item.bottle": {"Liquify fills it from Amethyst with Teiwaz.", "Liquify fills it with Jera from anything else."},
	} {
		if got := RecipeLines(p, id, recipes); !slices.Equal(got, want) {
			t.Errorf("%s: %q, want %q", id, got, want)
		}
	}
	lines, _ := ItemLines(p, "item.gem", &sim.Entry{Stage: sim.StageSeen, Level: "level.00"}, recipes...)
	if !slices.Contains(lines, "Liquify, with an Empty Bottle, makes it into Teiwaz.") {
		t.Errorf("the gem's entry: %q", lines)
	}
}
