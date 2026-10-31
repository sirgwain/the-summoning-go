package inspect

import (
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

func ptr[T any](v T) *T { return &v }

func testWorld(t *testing.T) *sim.World {
	t.Helper()
	p := content.NewPack(content.Manifest{Start: content.Start{Level: "level.00"}})
	p.Texts["text.en"] = &content.Text{Strings: map[string]string{
		"msg.key": "Key", "msg.sack": "Sack", "msg.wrong": "The {item} doesn't fit", "msg.hi": "Hello",
	}}
	p.Items["item.key"] = &content.Item{ID: "item.key", Name: "msg.key", Weight: 1}
	p.Items["item.sack"] = &content.Item{ID: "item.sack", Name: "msg.sack", Weight: 2, Container: &content.Container{Capacity: 2}}
	p.Things["thing.guard"] = &content.Thing{ID: "thing.guard", Kind: content.ThingCreature, Weight: 100,
		HitPoints: content.Dice{Bonus: 7}, Behaviors: []string{content.BehaviorHostile}, Stats: map[string]int{"accuracy": 9}}
	g := content.Grid{{0, 0}, {0, 0}}
	p.Levels["level.00"] = &content.Level{
		ID: "level.00", Size: [2]int{2, 2},
		FloorTypes: []content.FloorType{{Variant: 1, Enter: &content.Effect{Damage: content.Dice{Count: 2, Sides: 2}, Type: "fire"}}},
		Floor:      g, Wall: content.Grid{{-1, -1}, {-1, -1}},
		Features: []*content.Feature{
			{ID: "plate", Kind: content.FeaturePlate, Cell: content.Cell{0, 0}, Armed: true, Threshold: 50, OnPress: "p0"},
			{ID: "mouth", Kind: content.FeatureMouth, Cell: content.Cell{1, 1}, Message: "msg.hi"},
			{ID: "door", Kind: content.FeatureDoor, Cell: content.Cell{0, 1}, State: content.DoorClosed},
		},
		Procedures: map[string][]content.Action{"p0": {{On: ptr("door")}, {Create: &content.Create{Cell: content.Cell{1, 0}, Item: "item.key"}}}},
		Things:     []*content.PlacedThing{{ID: "guard", Thing: "thing.guard", Cell: content.Cell{1, 0}, Items: []content.Object{{Item: "item.key"}}}},
		Objects: []*content.FloorObject{{Cell: content.Cell{0, 0}, Object: content.Object{
			Item: "item.sack", Contents: []content.Object{{Item: "item.key"}}}}},
	}
	w, err := sim.New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func contains(t *testing.T, lines []string, want ...string) {
	t.Helper()
	all := strings.Join(lines, "\n")
	for _, w := range want {
		if !strings.Contains(all, w) {
			t.Errorf("no %q in:\n%s", w, all)
		}
	}
}

func TestCell(t *testing.T) {
	w := testWorld(t)
	p, l := w.Pack(), w.Current()
	contains(t, Cell(p, l, content.Cell{0, 0}),
		"Cell (0, 0)", "variant 1; entering: 2d2 fire damage",
		"Feature plate (plate)", "up, armed, weight 3 of 50",
		"[1] on_press runs p0:", "switch on door", "make Key at (1, 0)",
		"Sack holding Key (weight 1) (weight 3)")
	contains(t, Cell(p, l, content.Cell{0, 1}), "closed, stage 0 of 4", "blocked")
	contains(t, Cell(p, l, content.Cell{1, 1}), `says "Hello"`)
	contains(t, Cell(p, l, content.Cell{1, 0}), "Thing guard (thing.guard, creature), 7 hit points")
	contains(t, Cell(p, l, content.Cell{5, 5}), "outside the level")
	if h := Hooks(l.FeatureAt(content.Cell{0, 0})); len(h) != 1 || h[0] != "p0" {
		t.Errorf("hooks %v", h)
	}
}

func TestThing(t *testing.T) {
	w := testWorld(t)
	contains(t, Thing(w.Pack(), w.Current().Thing("guard")),
		"at (48, 16) in cell (1, 0)", "hit points 7", "behaviors: hostile", "stats: accuracy 9", "carries Key")
}

func TestMessage(t *testing.T) {
	w := testWorld(t)
	e := sim.Event{Kind: sim.EventMessage, Text: "msg.wrong", Item: "item.key"}
	if got := Message(w.Pack(), e); got != "The Key doesn't fit" {
		t.Errorf("%q", got)
	}
	ev := w.Update([]input.Command{input.Use{Cell: content.Cell{1, 1}}})
	if got := Message(w.Pack(), ev[0]); got != "Hello" {
		t.Errorf("mouth: %q", got)
	}
}
