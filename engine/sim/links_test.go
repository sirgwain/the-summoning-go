package sim

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// TestLinksRecorded records the teleporter and the pit the player goes through, once each, and
// the levels in the order visited; a save keeps them (docs/spec/world.md, The engine's map).
func TestLinksRecorded(t *testing.T) {
	w := walkWorld(t,
		"########",
		"#@t..O.#",
		"########")
	walk(w, 2, 2)
	tele := Link{Kind: content.FeatureTeleporter, From: content.Destination{Level: "level.walk", Cell: cell{1, 2}},
		To: content.Destination{Level: "level.walk", Cell: cell{1, 4}}}
	if got := w.Links(); !slices.Equal(got, []Link{tele}) {
		t.Fatalf("after the teleporter: %+v", got)
	}
	walk(w, 2, 8)
	if w.Current().ID != "level.below" {
		t.Fatalf("on %s, not below", w.Current().ID)
	}
	pit := Link{Kind: content.FeaturePit, From: content.Destination{Level: "level.walk", Cell: cell{1, 5}},
		To: content.Destination{Level: "level.below", Cell: cell{1, 1}}}
	if got := w.Links(); !slices.Equal(got, []Link{tele, pit}) {
		t.Errorf("after the pit: %+v", got)
	}
	if got := w.Visited(); !slices.Equal(got, []string{"level.walk", "level.below"}) {
		t.Errorf("visited %v", got)
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
	if !slices.Equal(r.Links(), w.Links()) || !slices.Equal(r.Visited(), w.Visited()) {
		t.Errorf("restored links %+v, visited %v", r.Links(), r.Visited())
	}
	// A save without the order takes the levels with seen cells, by id.
	back.Visited = nil
	if r, err = Restore(w.Pack(), &back); err != nil {
		t.Fatal(err)
	}
	if got := r.Visited(); !slices.Equal(got, []string{"level.below", "level.walk"}) {
		t.Errorf("visited without the order: %v", got)
	}
}

// TestExitTeachesTheWayBack records the exit beside the arrival that leads back beside the exit
// taken; an exit beside the arrival that leads elsewhere teaches nothing.
func TestExitTeachesTheWayBack(t *testing.T) {
	for _, tc := range []struct {
		name string
		back content.Cell // where the exit beside the arrival leads on level.walk
		want int
	}{
		{"two-way", cell{1, 3}, 2},
		{"leads elsewhere", cell{1, 1}, 1},
	} {
		p := walkPack(
			"######",
			"#@.X.#",
			"######")
		b := p.Levels["level.below"]
		b.Features = append(b.Features, &content.Feature{ID: "up", Kind: content.FeatureExit, Cell: cell{1, 2},
			To: &content.Destination{Level: "level.walk", Cell: tc.back}})
		w, err := New(p, 1)
		if err != nil {
			t.Fatal(err)
		}
		for range 60 {
			if w.Current().ID == "level.below" {
				break
			}
			walk(w, 2, 1)
		}
		if w.Current().ID != "level.below" {
			t.Fatalf("%s: on %s", tc.name, w.Current().ID)
		}
		links := w.Links()
		if len(links) != tc.want {
			t.Fatalf("%s: links %+v", tc.name, links)
		}
		if tc.want == 2 {
			want := Link{Kind: content.FeatureExit, From: content.Destination{Level: "level.below", Cell: cell{1, 2}},
				To: content.Destination{Level: "level.walk", Cell: tc.back}}
			if links[1] != want {
				t.Errorf("the way back: %+v", links[1])
			}
		}
	}
}

// TestMouthsAndTalkersRecorded records what a mouth says when heard, and where a creature that
// talks was last seen, following it in sight and forgetting it once it is gone from that cell.
func TestMouthsAndTalkersRecorded(t *testing.T) {
	p := walkPack(
		"#########",
		"#@M..G#.#",
		"#########")
	p.Conversations["conv"] = &content.Conversation{ID: "conv"}
	g := p.Things["thing.guard"]
	g.Behaviors, g.Conversation = append(g.Behaviors, content.BehaviorTalks), "conv"
	w, err := New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	do(w, input.Talk{Thing: "mouth"})
	if got := w.Mouths(); len(got) != 1 || got[0].Cell != (cell{1, 2}) || !slices.Equal(got[0].Messages, []string{"msg.hello"}) {
		t.Errorf("mouths %+v", got)
	}
	if got := w.Talkers(); len(got) != 1 || got[0].Thing != "guard" || got[0].Cell != (cell{1, 5}) {
		t.Fatalf("talkers %+v", got)
	}
	s, err := w.State()
	if err != nil {
		t.Fatal(err)
	}
	r, err := Restore(w.Pack(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Mouths()) != 1 || len(r.Talkers()) != 1 {
		t.Errorf("restored mouths %+v, talkers %+v", r.Mouths(), r.Talkers())
	}
	do(w, input.Move{Thing: "guard", Cell: cell{1, 4}})
	if got := w.Talkers(); len(got) != 1 || got[0].Cell != (cell{1, 4}) {
		t.Errorf("moved in sight: %+v", got)
	}
	// Carried off behind the wall: its cell is in sight and it is not there.
	do(w, input.Move{Thing: "guard", Cell: cell{1, 7}})
	if got := w.Talkers(); len(got) != 0 {
		t.Errorf("gone from its seen cell, still recorded: %+v", got)
	}
}
