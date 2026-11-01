package viewer

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

func TestFlipbookRun(t *testing.T) {
	f := &content.Flipbook{ID: "flipbook.t", Steps: []content.FlipbookStep{
		{Frame: 0, MS: 100, Sounds: []string{"sfx.1"}},
		{Frame: 2, MS: 50, Cancel: true},
		{Frame: 1, MS: 100, Music: "music.3"},
	}}
	r := NewFlipbookRun(f)
	var entered []int
	enter := func(s *content.FlipbookStep) { entered = append(entered, s.Frame) }
	r.Update(0, enter)
	if r.Frame() != 0 || len(entered) != 1 || r.Cancel() {
		t.Fatalf("start: frame %d, entered %v, cancelled", r.Frame(), entered)
	}
	r.Update(120, enter)
	if r.Frame() != 2 || r.MS != 20 || !r.Cancellable() {
		t.Fatalf("after 120 ms: frame %d at %d ms, cancellable %v", r.Frame(), r.MS, r.Cancellable())
	}
	r.Update(500, enter)
	if !r.Done || r.Frame() != 1 || len(entered) != 3 {
		t.Fatalf("after 620 ms: done %v, frame %d, entered %v", r.Done, r.Frame(), entered)
	}

	r = NewFlipbookRun(f)
	r.Update(110, enter)
	if !r.Cancel() || !r.Done {
		t.Fatal("cancelling during a cancellable step did not end the run")
	}
}
