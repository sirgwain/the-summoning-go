package inspect

import (
	"reflect"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// talkWorld is a level with a door and a hermit whose conversation opens it, when asked about
// the key or shown a book.
func talkWorld(t *testing.T) *sim.World {
	t.Helper()
	p := content.NewPack(content.Manifest{Start: content.Start{Level: "level.00"}})
	p.Texts["text.en"] = &content.Text{Strings: map[string]string{
		"msg.hermit": "Hermit", "msg.book": "Book", "word.key": "key", "msg.hi": "Hello",
	}}
	p.Keywords = &content.Keywords{Keywords: []content.Keyword{{ID: "keyword.key", Word: "word.key"}}}
	p.Items["item.book"] = &content.Item{ID: "item.book", Name: "msg.book", Weight: 1}
	p.Things["thing.hermit"] = &content.Thing{ID: "thing.hermit", Kind: content.ThingCreature, Weight: 100,
		Name: "msg.hermit", Conversation: "conversation.hermit", Behaviors: []string{"talks"}}
	p.Conversations["conversation.hermit"] = &content.Conversation{ID: "conversation.hermit", Clauses: []content.Clause{
		{If: content.Condition{Counter: &content.CounterTest{Counter: "var.1", Equals: ptr(1)}}, Do: []content.Action{{Jump: ptr(3)}}},
		{If: content.Condition{Greeting: true}, Do: []content.Action{{Say: ptr("msg.hi")}, {Say: ptr("msg.hi")}, {Offer: ptr("keyword.key")}}},
		{If: content.Condition{Asked: []string{"keyword.key"}}, Do: []content.Action{
			{SetCounter: &content.SetCounter{Counter: "var.1", Value: 1}}, {Run: ptr("p0")}, {End: true}}},
		{If: content.Condition{Carries: []string{"item.book"}}, Do: []content.Action{{Take: ptr("item.book")}, {Run: ptr("p0")}}},
	}}
	g := content.Grid{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}}
	p.Levels["level.00"] = &content.Level{
		ID: "level.00", Size: [2]int{3, 3},
		FloorTypes: []content.FloorType{{}},
		Floor:      g, Wall: content.Grid{{-1, -1, -1}, {-1, -1, -1}, {-1, -1, -1}},
		Features:   []*content.Feature{{ID: "door", Kind: content.FeatureDoor, Cell: content.Cell{2, 0}, State: content.DoorClosed}},
		Procedures: map[string][]content.Action{"p0": {{On: ptr("door")}}},
		Things:     []*content.PlacedThing{{ID: "hermit", Thing: "thing.hermit", Cell: content.Cell{0, 2}}},
	}
	w, err := sim.New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestConversationLinks(t *testing.T) {
	w := talkWorld(t)
	links := Links(w.Pack(), w.Current())
	// Both clauses that run p0 make one link from the hermit to the door.
	if len(links) != 1 {
		t.Fatalf("links %+v", links)
	}
	lk := links[0]
	if lk.Kind != LinkOn || lk.From != "hermit" || lk.FromCell != (content.Cell{0, 2}) || lk.Feature != "door" ||
		lk.Conversation != "conversation.hermit" || lk.Procedure != "p0" {
		t.Errorf("link %+v", lk)
	}
	if got, want := lk.Text(), `talk, asked "key" / carries Book (p0): on`; got != want {
		t.Errorf("text %q, want %q", got, want)
	}
}

func TestConversationLines(t *testing.T) {
	w := talkWorld(t)
	got := Texts(ConversationLines(w.Pack(), "conversation.hermit", "Hermit", false, map[string]int{"var.1": 1}))
	want := []string{
		"Talks: conversation.hermit (Hermit)",
		"  0 var.1 = 1: jump 3",
		`  1 greeting: say ×2, offer "key"`,
		`  2 asked "key": var.1 := 1, run p0, end`,
		"  3 carries Book: take Book, run p0",
		"  counters: var.1 = 1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
}
