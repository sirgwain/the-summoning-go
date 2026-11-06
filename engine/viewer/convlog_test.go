package viewer

import (
	"image"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// paneText returns the log pane's rows as text, one per row.
func paneText(lv *LogView) []string {
	var out []string
	for _, t := range lv.Text {
		var words []string
		for _, w := range t.Words {
			words = append(words, w.Text)
		}
		out = append(out, strings.Join(words, " "))
	}
	return out
}

// TestConversationLogWindow opens the log after talking with the sage twice, finds a line by
// search and jumps to it, and closes the log with Escape and with the hotbar's box.
func TestConversationLogWindow(t *testing.T) {
	v := playIn(t, devpack.Study, content.Cell{1, 3})
	v.World.Update([]input.Command{input.Talk{}})
	v.World.Update([]input.Command{input.Continue{}})
	v.World.Update([]input.Command{input.Ask{Word: "magic"}})
	for v.World.Talk().Line != nil {
		v.World.Update([]input.Command{input.Continue{}})
	}
	v.World.Update([]input.Command{input.Leave{}})
	v.World.Update([]input.Command{input.Talk{Thing: "sage"}})
	v.World.Update([]input.Command{input.Leave{}})

	pressKey(v, "L")
	lv := v.Log()
	if lv == nil {
		t.Fatal("L did not open the log")
	}
	if len(lv.Rows) != 1 || lv.Selected != 0 || !lv.Rows[0].HasPortrait {
		t.Fatalf("people: %+v, selected %d", lv.Rows, lv.Selected)
	}
	// The newest conversation comes first, under its heading.
	text := paneText(lv)
	if len(text) < 4 || !strings.HasSuffix(text[0], "0:00:00") || !strings.Contains(text[1], "Back again, "+inspect.Labels["ui.stranger"]) {
		t.Fatalf("pane: %q", text)
	}
	all := strings.Join(text, "|")
	for _, want := range []string{inspect.Labels["ui.stranger"] + " asks: magic", "Watch my hands.", "Taught the gesture I B F"} {
		if !strings.Contains(all, want) {
			t.Errorf("the pane lacks %q: %q", want, text)
		}
	}
	keyword := false
	for _, tx := range lv.Text {
		for _, w := range tx.Words {
			keyword = keyword || w.Keyword && w.Text == "fireball."
		}
	}
	if !keyword {
		t.Error("the keyword fireball is not marked")
	}

	// Typing searches, ignoring case; Enter shows the match's line.
	typeText(v, "WATCH")
	lv = v.Log()
	if len(lv.Rows) != 1 || !strings.Contains(lv.Rows[0].Label, "Watch my hands") {
		t.Fatalf("matches: %+v", lv.Rows)
	}
	pressKey(v, KeyEnter)
	focused := ""
	for _, tx := range v.Log().Text {
		if tx.Focus {
			focused = tx.Words[len(tx.Words)-1].Text
		}
	}
	if focused == "" {
		t.Error("Enter did not show the match's line")
	}
	typeText(v, "zzz")
	if lv := v.Log(); len(lv.Rows) != 0 || lv.Empty == "" {
		t.Errorf("no match: %+v", lv)
	}

	// Escape clears the search, then closes; the world did not run while the log was open.
	step := v.World.Step()
	pressKey(v, KeyEscape)
	if !v.LogOpen || v.LogTyped != "" {
		t.Fatalf("the first Escape: open %v, typed %q", v.LogOpen, v.LogTyped)
	}
	pressKey(v, KeyEscape)
	if v.LogOpen || v.World.Step() != step {
		t.Fatalf("the second Escape: open %v, ran %d steps", v.LogOpen, v.World.Step()-step)
	}

	// The hotbar's Log box opens it too.
	h := v.Hotbar()
	tapAt(v, h.Screen(h.Log.Rect).Min.Add(image.Pt(2, 2)))
	if !v.LogOpen {
		t.Error("the hotbar's box did not open the log")
	}
}

// TestMessageLog keeps the message line's messages, newest first, in the log's Messages tab,
// one by one when shown together, and narrows them by the search.
func TestMessageLog(t *testing.T) {
	v := playIn(t, devpack.Study, content.Cell{1, 3})
	v.notice("Saved")
	v.show("Waiting 1 of 9")
	v.announce([]sim.Event{
		{Kind: sim.EventMessage, Text: "Creature hit!"},
		{Kind: sim.EventMessage, Text: "You are poisoned"},
	})
	pressKey(v, "L")
	if lv := v.Log(); lv == nil || lv.Tab != LogConversations {
		t.Fatalf("the log opens on its conversations: %+v", lv)
	}
	pressKey(v, KeyArrowRight)
	lv := v.Log()
	if got := strings.Join(paneText(lv), "|"); lv.Tab != LogMessages || got != "You are poisoned|Creature hit!|Saved" {
		t.Fatalf("messages: tab %d, %q", lv.Tab, got)
	}
	typeText(v, "HIT")
	if got := strings.Join(paneText(v.Log()), "|"); got != "Creature hit!" {
		t.Errorf("searched: %q", got)
	}
	typeText(v, "z")
	if lv := v.Log(); len(lv.Text) != 0 || lv.Empty != inspect.Labels["ui.log.no_match"] {
		t.Errorf("no match: %+v", lv)
	}
	tapAt(v, lv.Screen(lv.Tabs[LogConversations].Rect).Min.Add(image.Pt(2, 2)))
	if v.Log().Tab != LogConversations {
		t.Error("clicking the tab did not switch back")
	}
	for range maxMessages + 5 {
		v.record("again")
	}
	if len(v.messages) != maxMessages {
		t.Errorf("%d messages kept", len(v.messages))
	}
}
