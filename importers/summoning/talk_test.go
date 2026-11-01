package summoning

import (
	"reflect"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

func TestReadKeywords(t *testing.T) {
	// Two keywords and the sentinel, at offsets after the table.
	f := fixture{}.u16(3, 8, 12, 17).str("aid\x00").str("Gate\x00").str("zzzz\x00")
	words, err := ReadKeywords(f)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(words, []string{"aid", "Gate"}) {
		t.Errorf("keywords %q", words)
	}
	if _, err := ReadKeywords(fixture{}.u16(1, 99)); err == nil {
		t.Error("an offset outside the file did not fail")
	}
}

// scriptFixture is an INTERACT file of one record named "Sage", text base 10, with words.
func scriptFixture(words ...int16) []byte {
	f := fixture{}.u16(1, 10).u32(8).str("Sage").zeros(16).u16(uint16(2 * len(words)))
	for _, w := range words {
		f = f.u16(uint16(w))
	}
	return f
}

func TestReadInteract(t *testing.T) {
	in, err := ReadInteract(scriptFixture(-2, 0x11, -4, -3, 0x15, -4, -1))
	if err != nil {
		t.Fatal(err)
	}
	if in.TextBase != 10 || len(in.Scripts) != 1 || in.Scripts[0].Name != "Sage" ||
		!reflect.DeepEqual(in.Scripts[0].Words, []int16{-2, 0x11, -4, -3, 0x15, -4, -1}) {
		t.Errorf("interact %+v", in)
	}
}

// talkImporter is an importer with three items, two keywords and texts 10..13.
func talkImporter() *importer {
	in := &Init{}
	in.SpellGestures[0][1] = "ABX"
	im := &importer{
		pack: content.NewPack(content.Manifest{}), report: &Report{}, init: in, talkers: map[string]bool{},
		strings:    map[string]string{"msg.10": "Hello", "msg.11": "Bye", "msg.12": "Gift", "msg.13": "Mine"},
		itemIDs:    []string{"item.a", "item.b", "item.c"},
		keywordIDs: []string{"keyword.aid", "keyword.gate"},
		interact:   &Interact{TextBase: 10},
	}
	im.pack.Portraits[portraitID(5)] = &content.Portrait{ID: portraitID(5)}
	return im
}

func TestConversation(t *testing.T) {
	im := talkImporter()
	words := []int16{
		-2, 4, -4, -3, 0x12, 0, 0x13, 1, -4, // first meeting: say, offer gate
		-2, 6, 0, 1, -4, -3, 0x16, 20, 3, 0x17, 20, 0x28, 3, -4, // asked aid or gate: set, add, jump
		-2, 7, -4, -3, -4, // never
		-2, 0x0A, 20, 4, -4, -3, 0x29, 2, 0, 1, 0xFF, 0xFF, 0xFF, 0x27, 5, 3, -4, // var 20 = 4: give a container, say beside 5
		-2, 0x0C, 7, -4, -3, 0x2B, 1, 0x2B, 203, 0x1D, 6, 0x15, -4, // NPC 7 dead: teach, effect, end
		-1,
	}
	c, err := im.conversation(3, Script{Name: "Sage", Words: words})
	if err != nil {
		t.Fatal(err)
	}
	str := func(s string) *string { return &s }
	num := func(n int) *int { return &n }
	want := &content.Conversation{ID: "conversation.3", Name: "Sage", Clauses: []content.Clause{
		{If: content.Condition{FirstMeeting: true}, Do: []content.Action{{Say: str("msg.10")}, {Offer: str("keyword.gate")}}},
		{If: content.Condition{Asked: []string{"keyword.aid", "keyword.gate"}}, Do: []content.Action{
			{SetCounter: &content.SetCounter{Counter: "var.20", Value: 3}}, {AddCounter: str("var.20")}, {Jump: num(3)}}},
		{If: content.Condition{Never: true}},
		{If: content.Condition{Counter: &content.CounterTest{Counter: "var.20", Equals: num(4)}}, Do: []content.Action{
			{Give: &content.Object{Item: "item.c", Contents: []content.Object{{Item: "item.a"}, {Item: "item.b"}}}},
			{SayAs: &content.SayAs{Portrait: "portrait.5", Text: "msg.13"}}}},
		{If: content.Condition{Dead: "thing.npc.7"}, Do: []content.Action{
			{Teach: str("AB")}, {Teach: str("D")}, {Effect: str("effect.6")}, {End: true}}},
	}}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("got  %+v\nwant %+v", c, want)
	}
	if !im.talkers["thing.npc.7"] {
		t.Error("the dead NPC's template was not asked for")
	}
	for _, bad := range [][]int16{
		{-2, 0x11, -4, -3, 0x99, -4, -1}, // an unknown action
		{-2, 0x11, -4, -3, 0x12},         // ends early
		{-2, 6, 9, -4, -3, -4, -1},       // no keyword 9
	} {
		if _, err := im.conversation(0, Script{Words: bad}); err == nil {
			t.Errorf("%v did not fail", bad)
		}
	}
}

func TestKeywordAnswers(t *testing.T) {
	im := talkImporter()
	// TEXT holds 14 records with END: keyword k answers with record 14 − 2 − k.
	im.keywords([]string{"aid", "Gate"}, 14)
	k := im.pack.Keywords.Keywords
	if len(k) != 2 || k[0].ID != "keyword.aid" || k[0].Answer != "msg.12" || k[1].Answer != "msg.11" ||
		im.strings[k[1].Word] != "Gate" {
		t.Errorf("keywords %+v", k)
	}
}
