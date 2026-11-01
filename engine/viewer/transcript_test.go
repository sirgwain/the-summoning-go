package viewer

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

func TestTranscriptLines(t *testing.T) {
	got := TranscriptLines("Ask of *magic, or my *name.\nNext")
	want := []string{"Ask of magic, or my name.", "       _____        ____", "Next"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q", got)
	}
	long := TranscriptLines(strings.Repeat("word ", 30))
	if len(long) != 2 || len(long[0]) >= TranscriptWidth {
		t.Errorf("wrapped %q", long)
	}
}

// TestTranscriptFile: with the transcript counter set, a conversation's lines and questions go
// to the transcript file, with @ the player's name.
func TestTranscriptFile(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	v.SaveDir = t.TempDir()
	v.Pack.Manifest.Counters.Transcript = "var.print"
	v.World.Update([]input.Command{input.SetCounter{Counter: "var.print", Value: 1}})
	v.World.Player().Name = "Mira"
	v.updateWorld([]input.Command{input.Talk{Thing: "sage"}})
	v.updateWorld([]input.Command{input.Continue{}})
	v.updateWorld([]input.Command{input.Ask{Word: "name"}})
	b, err := os.ReadFile(filepath.Join(v.SaveDir, TranscriptFile))
	if err != nil {
		t.Fatal(err)
	}
	if s := string(b); !strings.Contains(s, "Welcome to my study, Mira.") || !strings.Contains(s, "\nname\n") {
		t.Errorf("transcript:\n%s", s)
	}
}
