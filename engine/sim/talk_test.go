package sim

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// TestConversationLog checks that conversations go into the log as they are held, leaving out the
// menu's, and that the log is kept in saves (docs/spec/dialogue.md, Engine model).
func TestConversationLog(t *testing.T) {
	w := studyWorld(t, cell{1, 3}, 2)
	if len(w.Log()) != 0 {
		t.Fatalf("a new world's log: %+v", w.Log())
	}
	step := w.Step()
	do(w, input.Talk{})
	if n := len(w.Log()); n != 1 {
		t.Fatalf("the greeting did not go into the log: %d conversations", n)
	}
	do(w, input.Continue{})
	do(w, input.Ask{Word: "Magic!"})
	for w.Talk().Line != nil {
		do(w, input.Continue{})
	}
	do(w, input.Leave{})

	// The menu's conversation is not logged.
	do(w, input.Menu{})
	if w.Talk() == nil {
		t.Fatal("the menu did not open")
	}
	do(w, input.Leave{})
	if n := len(w.Log()); n != 1 {
		t.Fatalf("%d conversations logged, want 1", n)
	}

	got := w.Log()[0]
	want := &Logged{Template: devpack.ThingSage, Portrait: devpack.PortraitSage, Level: w.Current().ID, Step: step, Entries: []LogEntry{
		{Kind: LogLine, Text: "talk.sage.hello", Portrait: devpack.PortraitSage},
		{Kind: LogQuestion, Text: "Magic!"},
		{Kind: LogLine, Text: "keyword.magic.answer"},
		{Kind: LogLine, Text: "talk.sage.magic", Portrait: devpack.PortraitSage},
		{Kind: LogTeach, Icons: "IBF"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("logged\n %+v\nwant\n %+v", got, want)
	}

	// A second conversation follows the first.
	do(w, input.Talk{Thing: "sage"})
	do(w, input.Leave{})
	if n := len(w.Log()); n != 2 || w.Log()[1].Entries[0].Text != "talk.sage.again" {
		t.Fatalf("the second conversation: %+v", w.Log())
	}

	// Talking again, with nothing new said, adds nothing; nor does asking a question again.
	do(w, input.Talk{Thing: "sage"})
	do(w, input.Continue{Ask: "magic"})
	for w.Talk() != nil && w.Talk().Line != nil {
		do(w, input.Continue{})
	}
	do(w, input.Leave{})
	if n := len(w.Log()); n != 2 {
		t.Fatalf("talking again with nothing new: %d conversations, %+v", n, w.Log()[n-1])
	}

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
	if !reflect.DeepEqual(r.Log(), w.Log()) {
		t.Errorf("restored log\n %+v\nwant\n %+v", r.Log(), w.Log())
	}

	// A saved log holding a conversation twice is restored with the copy taken out.
	dup := *back.Log[0]
	back.Log = append(back.Log, &dup)
	r, err = Restore(w.Pack(), &back)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Log(), w.Log()) {
		t.Errorf("restored a doubled log\n %+v\nwant\n %+v", r.Log(), w.Log())
	}
}
