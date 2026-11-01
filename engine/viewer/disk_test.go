package viewer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// askMenu opens the disk menu with O and asks word, going on past the lines until a menu opens.
func askMenu(t *testing.T, v *Viewer, word string) {
	t.Helper()
	pressKey(v, "O")
	for i := 0; i < 20 && v.World.Talk() == nil; i++ {
		v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}})
	}
	if v.World.Talk() == nil {
		t.Fatal("O did not open the disk menu")
	}
	pressKey(v, KeyEnter)
	v.updateWorld([]input.Command{input.Ask{Word: word}})
	for range 5 {
		if v.Front != nil {
			return
		}
		if l := v.World.Talk().Line; l != nil && l.Kind == sim.LineMenu {
			v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}})
			continue
		}
		pressKey(v, KeyEnter)
	}
	if v.Front == nil {
		t.Fatalf("asking %s opened no menu: %+v", word, v.World.Talk().Line)
	}
}

// TestDiskMenu saves a game by name, renames it, restores it and deletes it through the disk
// menu's conversation, and leaves for the main menu.
func TestDiskMenu(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	v.SaveDir = t.TempDir()
	at := v.World.Player().Cell()

	askMenu(t, v, "save")
	if v.Front.Screen != ScreenSave {
		t.Fatalf("screen %s", v.Front.Screen)
	}
	typeText(v, "First/game")
	pressKey(v, KeyEnter)
	if v.Front != nil || v.Notice != "Saved" {
		t.Fatalf("saving: front %+v, notice %q", v.Front, v.Notice)
	}
	if _, err := os.Stat(filepath.Join(v.SaveDir, "Firstgame.sav")); err != nil {
		t.Fatal(err)
	}
	if v.World.Talk() != nil {
		t.Error("the conversation did not go on to its end after the menu")
	}

	askMenu(t, v, "rename")
	clickButton(t, v, "save.0")
	for range len("Firstgame") {
		pressKey(v, KeyBackspace)
	}
	typeText(v, "Kept")
	pressKey(v, KeyEnter)
	if got := v.SaveNames(); len(got) != 1 || got[0] != "Kept" || v.Notice != "Renamed" {
		t.Fatalf("renamed: %v, %q", got, v.Notice)
	}

	v.World.Update([]input.Command{input.PlacePlayer{Cell: devpack.StudyScroll}})
	askMenu(t, v, "restore")
	clickButton(t, v, "save.0")
	if v.Front != nil || v.World.Talk() != nil || v.World.Player().Cell() != at {
		t.Fatalf("restored: front %v, talk %v, at %v", v.Front, v.World.Talk(), v.World.Player().Cell())
	}

	askMenu(t, v, "delete")
	clickButton(t, v, "save.0")
	if got := v.SaveNames(); len(got) != 0 {
		t.Fatalf("after deleting: %v", got)
	}

	askMenu(t, v, "leave")
	clickButton(t, v, "back")
	if v.Front != nil {
		t.Fatal("No did not return to play")
	}
	askMenu(t, v, "leave")
	clickButton(t, v, "yes")
	if v.Front == nil || v.Front.Screen != ScreenMenu {
		t.Fatalf("leaving: %+v", v.Front)
	}
}
