package viewer

import (
	"image"
	"slices"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// bookText returns the book pane's lines as text, one per line.
func bookText(bv *BookView) []string {
	var out []string
	for _, l := range bv.Lines {
		var words []string
		for _, w := range l.Words {
			words = append(words, w.Text)
		}
		out = append(out, strings.Join(words, " "))
	}
	return out
}

// TestBookWindow plays in the cellar, where the brute stands in sight, and opens the book on it
// with J; then switches to the items, which hold the boots the player wears, and closes the book
// and opens it again from the hotbar's box.
func TestBookWindow(t *testing.T) {
	v := playIn(t, devpack.Cellar, content.Cell{3, 3})
	v.World.Update(nil)
	pressKey(v, "J")
	bv := v.Book()
	if bv == nil {
		t.Fatal("J did not open the book")
	}
	if bv.Tab != BookCreatures || bv.Count != 1 || len(bv.Rows) != 1 || bv.Rows[0].Label != "Brute" || bv.Selected != 0 {
		t.Fatalf("creatures: tab %d, %d found, rows %+v", bv.Tab, bv.Count, bv.Rows)
	}
	if !bv.HasPicture || bv.Picture.Sprite != "sprite.brute" {
		t.Errorf("the brute's picture: %+v", bv.Picture)
	}
	text := strings.Join(bookText(bv), "|")
	for _, want := range []string{"Brute|", "A hulking cellar dweller", "First seen: ", hintFight} {
		if !strings.Contains(text, want) {
			t.Errorf("the pane lacks %q: %q", want, text)
		}
	}
	if strings.Contains(text, "Strikes for") {
		t.Errorf("the brute's blow shows before a fight: %q", text)
	}
	pressKey(v, "W")
	if v.World.Player().Cell() != (content.Cell{3, 3}) {
		t.Error("the world ran while the book was open")
	}

	pressKey(v, KeyArrowRight)
	bv = v.Book()
	labels := []string{}
	for _, r := range bv.Rows {
		labels = append(labels, r.Label)
	}
	if bv.Tab != BookItems || !slices.Contains(labels, inspect.ItemName(v.Pack, devpack.ItemBoots)) {
		t.Fatalf("items: tab %d, rows %q", bv.Tab, labels)
	}
	pressKey(v, KeyEscape)
	if v.Book() != nil {
		t.Fatal("Escape did not close the book")
	}

	h := v.Hotbar()
	tapAt(v, h.Rect.Min.Add(h.Book.Rect.Min.Mul(h.Scale)).Add(image.Pt(2, 2)))
	if bv := v.Book(); bv == nil || bv.Tab != BookItems {
		t.Fatalf("the hotbar's Book box: %+v", bv)
	}
}

// TestBookStages checks what each stage adds to a creature's and an item's lines.
func TestBookStages(t *testing.T) {
	v := playIn(t, devpack.Cellar, content.Cell{3, 3})
	at := &sim.Entry{Stage: sim.StageFought, Level: devpack.Cellar}
	lines, hint := v.CreatureLines(devpack.ThingBrute, at)
	if hint != hintDefeat || !slices.Contains(lines, "Strikes for 1d6.") || slices.Contains(lines, "Hit points: 2d10+4") {
		t.Errorf("fought: %q, %q", lines, hint)
	}
	at = &sim.Entry{Stage: sim.StageKilled, Level: devpack.Cellar, Kills: 2, Drops: []string{devpack.ItemKey}}
	lines, hint = v.CreatureLines(devpack.ThingBrute, at)
	all := strings.Join(lines, "|")
	for _, want := range []string{"Hit points: 2d10+4", "Worth 2d10+30 experience.", "Defeated: 2", "Drops: "} {
		if !strings.Contains(all, want) {
			t.Errorf("killed: lacks %q: %q", want, lines)
		}
	}
	if hint != "" {
		t.Errorf("killed: hint %q", hint)
	}

	// A creature that can be talked to has no hint to fight it.
	if lines, hint := v.CreatureLines(devpack.ThingSage, &sim.Entry{Stage: sim.StageSeen}); hint != "" || !slices.Contains(lines, "Can be talked to.") {
		t.Errorf("the sage: %q, hint %q", lines, hint)
	}

	// A defense shows only once learned.
	sword := 0
	v.Pack.Things[devpack.ThingBrute].Vulnerability = &sword
	if lines, _ := v.CreatureLines(devpack.ThingBrute, at); slices.ContainsFunc(lines, func(s string) bool { return strings.HasPrefix(s, "Vulnerable") }) {
		t.Errorf("a vulnerability not learned shows: %q", lines)
	}
	at.Defenses = []string{sim.DefenseVulnerable}
	if lines, _ := v.CreatureLines(devpack.ThingBrute, at); !slices.Contains(lines, "Vulnerable to: "+v.weaponClassName(0)) {
		t.Errorf("a vulnerability learned: %q", lines)
	}

	lines, hint = v.ItemLines(devpack.ItemPotion, &sim.Entry{Stage: sim.StageSeen})
	if hint == "" || strings.Contains(strings.Join(lines, "|"), "Weight") {
		t.Errorf("a potion seen: %q, %q", lines, hint)
	}
	lines, hint = v.ItemLines(devpack.ItemPotion, &sim.Entry{Stage: sim.StageHandled})
	if hint != hintUse || slices.Contains(lines, "Heals 5d10 hit points.") {
		t.Errorf("a potion handled: %q, %q", lines, hint)
	}
	lines, hint = v.ItemLines(devpack.ItemPotion, &sim.Entry{Stage: sim.StageUsed})
	if hint != "" || !slices.Contains(lines, "Heals 5d10 hit points.") {
		t.Errorf("a potion used: %q, %q", lines, hint)
	}
	// A scroll says what it teaches from first sight, with no hint.
	lines, hint = v.ItemLines(devpack.ItemScroll, &sim.Entry{Stage: sim.StageSeen})
	if hint != "" || !slices.Contains(lines, "Teaches Fireball (Wizardry).") || !slices.Contains(lines, "Costs 15 magic points.") ||
		!slices.Contains(lines, "Hurls a ball of fire.") || slices.ContainsFunc(lines, func(s string) bool { return strings.HasPrefix(s, "Weight") }) {
		t.Errorf("a scroll seen: %q, %q", lines, hint)
	}
	v.Pack.Items[devpack.ItemScroll].Teaches = "AB"
	if lines, _ := v.ItemLines(devpack.ItemScroll, &sim.Entry{Stage: sim.StageSeen}); !slices.Contains(lines, "Teaches the gesture A B.") {
		t.Errorf("a scroll of no spell: %q", lines)
	}

	// A weapon with two strikes names each; a single charge is not worth a line.
	v.Pack.Items["item.blade"] = &content.Item{ID: "item.blade", Name: "item.blade", Weight: 6, Charges: 1, Actions: []content.ItemAction{
		{Kind: content.ActionAttack, Damage: content.Dice{Count: 1, Sides: 6}, Delay: 12},
		{Kind: content.ActionAttack, Damage: content.Dice{Count: 2, Sides: 6}, Delay: 18, Skill: 2},
	}}
	lines, _ = v.ItemLines("item.blade", &sim.Entry{Stage: sim.StageHandled})
	if !slices.Contains(lines, "Strike 1: 1d6 damage.") || !slices.Contains(lines, "Strike 2: 2d6 damage, slower, needs skill 2.") ||
		slices.ContainsFunc(lines, func(s string) bool { return strings.HasPrefix(s, "Charges") }) {
		t.Errorf("a blade with two strikes: %q", lines)
	}
	lines, hint = v.ItemLines(devpack.ItemBoots, &sim.Entry{Stage: sim.StageHandled})
	if hint != hintWear || slices.Contains(lines, "Makes you faster (speed +2).") {
		t.Errorf("boots handled: %q, %q", lines, hint)
	}
	lines, hint = v.ItemLines(devpack.ItemBoots, &sim.Entry{Stage: sim.StageUsed})
	if hint != "" || !slices.Contains(lines, "Makes you faster (speed +2).") {
		t.Errorf("boots worn: %q, %q", lines, hint)
	}
	lines, hint = v.ItemLines(devpack.ItemMail, &sim.Entry{Stage: sim.StageHandled})
	if hint != "" || !slices.Contains(lines, "Worn on the body.") {
		t.Errorf("mail handled: %q, %q", lines, hint)
	}
}

// TestBookFilter types into the book's filter field: both tabs narrow to the names that hold it,
// ignoring case, Backspace takes a letter off, and Escape clears it before it closes the book.
func TestBookFilter(t *testing.T) {
	v := playIn(t, devpack.Cellar, content.Cell{3, 3})
	v.World.Update(nil)
	pressKey(v, "J")
	pressKey(v, KeyArrowRight)
	all := v.Book().Count
	if all < 2 {
		t.Fatalf("items found: %d", all)
	}
	typeText(v, "MAIL")
	bv := v.Book()
	mail := inspect.ItemName(v.Pack, devpack.ItemMail)
	if bv.Typed != "MAIL" || bv.Count != 1 || len(bv.Rows) != 1 || bv.Rows[0].Label != mail {
		t.Fatalf("filtered: typed %q, %d, rows %+v", bv.Typed, bv.Count, bv.Rows)
	}
	if text := bookText(bv); len(text) == 0 || text[0] != mail {
		t.Errorf("the pane does not show the match: %q", text)
	}
	if !strings.HasPrefix(bv.Tabs[BookCreatures].Label, "Creatures (0)") {
		t.Errorf("the creatures tab: %q", bv.Tabs[BookCreatures].Label)
	}
	typeText(v, "zz")
	if bv := v.Book(); bv.Count != 0 || bv.Empty != "Nothing matches." {
		t.Errorf("no match: %d, %q", bv.Count, bv.Empty)
	}
	pressKey(v, KeyBackspace)
	pressKey(v, KeyBackspace)
	if bv := v.Book(); bv.Typed != "MAIL" || bv.Count != 1 {
		t.Errorf("after Backspace: %q, %d", bv.Typed, bv.Count)
	}
	pressKey(v, KeyEscape)
	if bv := v.Book(); bv == nil || bv.Typed != "" || bv.Count != all {
		t.Fatalf("Escape did not clear the filter: %+v", bv)
	}
	pressKey(v, KeyEscape)
	if v.Book() != nil {
		t.Error("a second Escape did not close the book")
	}
}

// TestFullBook opens the tools' full book in the cellar: it lists the cellar's creatures alone,
// at their last stage with their records and where they stand, until Tab lists the whole pack's,
// those not in the cellar unmarked; and it finds the key the brute carries.
func TestFullBook(t *testing.T) {
	v, err := New(devpack.Build(), 1, devpack.Cellar, 2, false, image.Pt(1280, 800))
	if err != nil {
		t.Fatal(err)
	}
	pressKey(v, "J")
	bv := v.Book()
	if bv == nil || !bv.Full || bv.All {
		t.Fatalf("J did not open the full book: %+v", bv)
	}
	here := map[string]bool{}
	for _, th := range v.World.Current().Things {
		if th.Template.Kind == content.ThingCreature {
			here[th.Template.ID] = true
		}
	}
	if bv.Count != len(here) || !slices.ContainsFunc(bv.Rows, func(r BookRow) bool { return r.Label == "Brute" }) {
		t.Fatalf("the cellar's creatures: %d of %d, rows %+v", bv.Count, len(here), bv.Rows)
	}
	for _, r := range bv.Rows {
		if !r.InLevel {
			t.Errorf("row %q is listed for the cellar but not marked", r.Label)
		}
	}
	typeText(v, "brute")
	text := strings.Join(bookText(v.Book()), "|")
	// The brute stands in sight at the start, so it is seen; the full book shows past fighting it.
	for _, want := range []string{"First seen: ", "Strikes for", "Hit points:", "On " + devpack.Cellar + ":",
		"brute at (", "Record " + devpack.ThingBrute + ":", `sprite: "sprite.brute"`} {
		if !strings.Contains(text, want) {
			t.Errorf("the brute's pane lacks %q: %q", want, text)
		}
	}
	if strings.Contains(text, hintFight) {
		t.Errorf("the full book hints: %q", text)
	}
	pressKey(v, KeyEscape)

	pressKey(v, KeyTab)
	bv = v.Book()
	creatures := 0
	for _, th := range v.Pack.Things {
		if th.Kind == content.ThingCreature {
			creatures++
		}
	}
	if !bv.All || bv.Count != creatures || bv.Toggle.Label != "All" {
		t.Fatalf("Tab: all %v, %d of %d creatures", bv.All, bv.Count, creatures)
	}
	if creatures > len(here) && !slices.ContainsFunc(bv.Rows, func(r BookRow) bool { return !r.InLevel }) {
		t.Errorf("no creature is unmarked among all: %+v", bv.Rows)
	}
	// The ghost haunts another level: the player has not come across it.
	typeText(v, v.logName(devpack.ThingGhost))
	if text := strings.Join(bookText(v.Book()), "|"); !strings.Contains(text, "Not come across yet.") || strings.Contains(text, "On "+devpack.Cellar) {
		t.Errorf("the ghost's pane: %q", text)
	}
	pressKey(v, KeyEscape)

	pressKey(v, KeyArrowRight)
	typeText(v, inspect.ItemName(v.Pack, devpack.ItemKey))
	if text := strings.Join(bookText(v.Book()), "|"); !strings.Contains(text, "carried by brute") {
		t.Errorf("the key's pane lacks the brute: %q", text)
	}
	pressKey(v, KeyEscape)
	pressKey(v, KeyEscape)
	if v.BookOpen || v.BookFull {
		t.Error("Escape did not close the full book")
	}
}
