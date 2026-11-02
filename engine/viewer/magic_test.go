package viewer

import (
	"image"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// TestSpellbookList: the spell panel lists the spellbook by school under headings with each
// school's memorizations and limit, scrolls with the wheel and Page Down, and a row's Memorize
// button memorizes its spell; a gesture entered on the panel adds its spell to the list.
func TestSpellbookList(t *testing.T) {
	v := playIn(t, devpack.Study, devpack.StudyStart)
	p := v.World.Player()
	v.OpenSpells()
	if sp := v.SpellPanel(); sp.Rows != 0 || len(sp.Book) != 0 {
		t.Fatalf("a new spellbook lists %d rows", sp.Rows)
	}
	// Entering Flame Arrow's gesture learns it into the list.
	v.World.Update([]input.Command{input.Learn{Gesture: v.Pack.Spells[devpack.SpellFlameArrow].Gesture}})
	if sp := v.SpellPanel(); sp.Rows != 2 || len(sp.Headers) != 1 || len(sp.Book) != 1 || sp.Book[0].Spell != devpack.SpellFlameArrow {
		t.Fatalf("after learning flame arrow: %+v %+v", sp.Headers, sp.Book)
	}

	all := []string{devpack.SpellFlameArrow, devpack.SpellHeal, devpack.SpellFireball, devpack.SpellKano,
		devpack.SpellSwiftness, devpack.SpellMagicWall, devpack.SpellGateway}
	p.Magic.Spellbook = all
	schools := map[int]bool{}
	for _, id := range all {
		schools[v.Pack.Spells[id].School] = true
	}
	sp := v.SpellPanel()
	if want := len(all) + len(schools); sp.Rows != want {
		t.Fatalf("rows %d, want %d", sp.Rows, want)
	}
	if len(sp.Headers)+len(sp.Book) != min(sp.Rows, bookRows) || sp.Bar.Empty() != (sp.Rows <= bookRows) {
		t.Fatalf("showing %d headers and %d spells of %d rows, bar %v", len(sp.Headers), len(sp.Book), sp.Rows, sp.Bar)
	}
	// The rows go school by school, each spell under its own school's heading.
	school := -1
	for _, r := range v.bookList() {
		if r.spell == "" {
			if r.school <= school {
				t.Fatalf("school %d heads after school %d", r.school, school)
			}
			school = r.school
		} else if s := v.Pack.Spells[r.spell].School; s != school {
			t.Fatalf("%s of school %d under school %d", r.spell, s, school)
		}
	}
	if h := sp.Headers[0]; h.School != v.bookList()[0].school {
		t.Errorf("first heading %+v", h)
	}

	if sp.Rows <= bookRows {
		t.Fatalf("%d rows do not scroll", sp.Rows)
	}
	{
		// The wheel turned toward the player scrolls down, a row a unit, no further than the end.
		v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: mid(sp.Screen(sp.List)), Wheel: -1})
		if got := v.SpellPanel().Scroll; got != 1 {
			t.Errorf("the wheel scrolled to %d", got)
		}
		pressKey(v, KeyPageDown)
		pressKey(v, KeyPageDown)
		if got := v.SpellPanel().Scroll; got != sp.Rows-bookRows {
			t.Errorf("Page Down scrolled to %d, want the end %d", got, sp.Rows-bookRows)
		}
		pressKey(v, KeyPageUp)
		pressKey(v, KeyPageUp)
		if got := v.SpellPanel().Scroll; got != 0 {
			t.Errorf("Page Up scrolled to %d", got)
		}
	}

	// Memorize memorizes the row's spell once more; clicking the row's name does nothing.
	sp = v.SpellPanel()
	e := sp.Book[0]
	before := p.Magic.Memorized[e.Spell]
	tapAt(v, sp.Screen(e.Rect).Min.Add(image.Pt(sp.Scale*(rowHeight+4), sp.Scale)))
	if p.Magic.Memorized[e.Spell] != before {
		t.Fatalf("clicking %s's name memorized it", e.Spell)
	}
	tapAt(v, mid(sp.Screen(e.Button)))
	if p.Magic.Memorized[e.Spell] != before+1 || p.Magic.Selected[v.Pack.Spells[e.Spell].School] != e.Spell {
		t.Errorf("Memorize %s: memorized %d, selected %v", e.Spell, p.Magic.Memorized[e.Spell], p.Magic.Selected)
	}
	if !v.SpellsOpen {
		t.Error("memorizing closed the panel")
	}
}

// mid is the middle of r.
func mid(r image.Rectangle) image.Point { return r.Min.Add(r.Max).Div(2) }
