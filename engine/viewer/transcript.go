package viewer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The conversation transcript (docs/spec/dialogue.md): what conversations say while the pack's
// transcript counter is set, written to a text file in the save directory in place of the
// game's printer.

// TranscriptFile is the transcript's file name; TranscriptWidth its columns.
const (
	TranscriptFile  = "transcript.txt"
	TranscriptWidth = 80
)

// TranscriptLines lays text out for the transcript: words wrapped at TranscriptWidth columns,
// each line holding a keyword (a word marked *) followed by a line underlining its letters.
func TranscriptLines(text string) []string {
	var out []string
	for _, para := range strings.Split(text, "\n") {
		line, under := "", ""
		keyword := false
		flush := func() {
			out = append(out, strings.TrimRight(line, " "))
			if keyword {
				out = append(out, strings.TrimRight(under, " "))
			}
			line, under, keyword = "", "", false
		}
		for _, f := range strings.Fields(para) {
			kw := strings.HasPrefix(f, "*")
			f = strings.TrimPrefix(f, "*")
			if line != "" && len(line)+len(f) >= TranscriptWidth {
				flush()
			}
			mark := strings.Repeat(" ", len(f))
			if kw {
				keyword = true
				start := strings.IndexFunc(f, isWordRune)
				end := strings.LastIndexFunc(f, isWordRune)
				if start >= 0 {
					mark = strings.Repeat(" ", start) + strings.Repeat("_", end-start+1) + strings.Repeat(" ", len(f)-end-1)
				}
			}
			line += f + " "
			under += mark + " "
		}
		flush()
	}
	return out
}

func isWordRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

// transcribe appends the transcript's events to the transcript file.
func (v *Viewer) transcribe(events []sim.Event) {
	if v.SaveDir == "" {
		return
	}
	var b strings.Builder
	for _, e := range events {
		if e.Kind != sim.EventTranscript {
			continue
		}
		text := e.Text
		if e.Detail != sim.TranscriptQuestion {
			text = v.lineText(e.Text)
		}
		for _, l := range TranscriptLines(text) {
			b.WriteString(l + "\n")
		}
		b.WriteString("\n")
	}
	if b.Len() == 0 {
		return
	}
	if err := os.MkdirAll(v.SaveDir, 0o755); err != nil {
		v.notice("No transcript: " + err.Error())
		return
	}
	f, err := os.OpenFile(filepath.Join(v.SaveDir, TranscriptFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		v.notice("No transcript: " + err.Error())
		return
	}
	defer f.Close()
	if _, err := f.WriteString(b.String()); err != nil {
		v.notice("No transcript: " + err.Error())
	}
}
