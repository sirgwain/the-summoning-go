package sim

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// TestStateRoundTrip saves a world in the middle of play, as JSON, restores it, and runs both on
// with the same commands: they stay the same, random choices included.
func TestStateRoundTrip(t *testing.T) {
	w := studyWorld(t, devpack.StudyStart, 2)
	do(w, input.Learn{Gesture: "DJ"})
	do(w, input.Cast{School: 0})
	do(w, input.Talk{Thing: "sage"})
	do(w, input.Leave{})
	for range 30 {
		w.Update([]input.Command{input.Walk{Direction: 2}})
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
	for i := range 400 {
		cmds := []input.Command{input.Walk{Direction: (i / 50) % 8}}
		if i%60 == 0 {
			cmds = append(cmds, input.Strike{Hand: 0})
		}
		a, b := w.Update(cmds), r.Update(cmds)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("update %d: events differ:\n%v\n%v", i, a, b)
		}
	}
	sa, _ := w.State()
	sb, _ := r.State()
	ja, _ := json.Marshal(sa)
	jb, _ := json.Marshal(sb)
	if string(ja) != string(jb) {
		t.Error("the worlds' states differ after running on")
	}
	if !w.Records()[devpack.ThingSage].Talked || !r.Records()[devpack.ThingSage].Talked {
		t.Error("the records were not kept")
	}

	do(w, input.Talk{Thing: "sage"})
	if _, err := w.State(); err == nil {
		t.Error("saving during a conversation did not fail")
	}
}
