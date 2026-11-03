package viewer

import (
	"os"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// No save is taken while a transition plays, and a change of level that comes with one is
// autosaved once it ends.
func TestNoSaveWhileHolding(t *testing.T) {
	v := transitionViewer(t)
	v.SaveDir = t.TempDir()
	v.took([]sim.Event{
		{Kind: sim.EventLevelChanged, Level: v.Shown},
		{Kind: sim.EventTransition, Level: v.Shown, Detail: content.TransitionTeleporter},
	})
	if !v.Holding() {
		t.Fatal("no transition holds the world")
	}
	v.SaveGame(QuickSave)
	if _, err := os.Stat(v.savePath(QuickSave)); err == nil {
		t.Error("saved while a transition plays")
	}
	if _, err := os.Stat(v.savePath(AutoSave)); err == nil {
		t.Error("autosaved while a transition plays")
	}
	for i := 0; v.Holding(); i++ {
		v.Update(Input{})
		if i > 100 {
			t.Fatal("the transition never ended")
		}
	}
	v.Update(Input{})
	if _, err := os.Stat(v.savePath(AutoSave)); err != nil {
		t.Errorf("not autosaved after the transition: %v", err)
	}
}
