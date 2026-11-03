package viewer

import (
	"image"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
)

// The full book (docs/design.md, Editor): the tools' book. It lists every creature and item of the
// pack, not only those come across, and shows each at its last stage with the whole of its
// record and where it is on the current level. It lists the current level's alone until the
// toggle shows them all; the current level's are marked either way.

// fullBook is the tools' full book.
type fullBook struct{ v *Viewer }

func (f fullBook) entries(tab int) []bookEntry {
	v := f.v
	ps := v.bookPlaces()
	var out []bookEntry
	add := func(id, name string, here bool) {
		if here || v.book.all {
			out = append(out, bookEntry{id: id, name: name, inLevel: here})
		}
	}
	if tab == BookCreatures {
		for id, t := range v.Pack.Things {
			if t.Kind == content.ThingCreature {
				add(id, inspect.ThingName(v.Pack, id), len(ps.Creatures[id]) > 0)
			}
		}
	} else {
		for id := range v.Pack.Items {
			add(id, inspect.ItemName(v.Pack, id), len(ps.Items[id]) > 0)
		}
	}
	return out
}

func (f fullBook) pane(tab int, id string) ([]inspect.PaneItem, string) {
	v := f.v
	ps := v.bookPlaces()
	c := &v.book.cache
	if c.pane == nil || c.tab != tab || c.id != id {
		if tab == BookCreatures {
			c.pane = inspect.FullCreature(v.Pack, v.World, ps, id)
		} else {
			c.pane = inspect.FullItem(v.Pack, v.World, ps, id)
		}
		c.tab, c.id = tab, id
	}
	return c.pane, ""
}

func (fullBook) empty(int) string { return "None on this level. Tab shows all." }

// bookPlaces returns the current level's places, worked out again only when the level or the
// step has changed since, which drops the pane kept too.
func (v *Viewer) bookPlaces() inspect.Places {
	c := &v.book.cache
	l := v.World.Current()
	if !c.ok || c.level != l.ID || c.step != v.World.Step() {
		*c = bookCache{level: l.ID, step: v.World.Step(), ok: true, places: inspect.LevelPlaces(v.Pack, v.World)}
	}
	return c.places
}

// OpenFullBook opens the full book, with the tools.
func (v *Viewer) OpenFullBook() {
	if v.Play {
		return
	}
	v.book.open, v.book.full, v.book.typed, v.SpellsOpen = true, true, "", false
	v.book.scroll, v.book.cache = 0, bookCache{}
}

// fullBookInput takes the full book's own input, reporting whether it took in: Tab or the toggle
// switches between the current level's entries and all, the wheel over the pane scrolls it, and
// a click on a line of where the entry is goes there.
func (v *Viewer) fullBookInput(in Input, bv *BookView, cursor image.Point) bool {
	switch {
	case in.pressed(KeyTab) || in.Left.Released && cursor.In(bv.Toggle.Rect):
		v.book.all = !v.book.all
		v.book.first, v.book.scroll = 0, 0
		return true
	case in.Wheel != 0 && cursor.In(bv.Pane):
		v.book.scroll = max(0, v.book.scroll-int(in.Wheel)*3)
		return true
	case in.Left.Released:
		for _, l := range bv.Lines {
			if (l.Thing != "" || l.Cell != nil) && cursor.In(l.Rect()) {
				v.bookGo(l.Thing, l.Cell)
				return true
			}
		}
	}
	return false
}

// bookGo closes the book and selects what a pane line names, a thing or a cell, centering the
// view on it.
func (v *Viewer) bookGo(thing string, cell *content.Cell) {
	v.CloseBook()
	if thing != "" && v.SelectThing(thing) == nil {
		v.CenterOn(*v.Selected)
		return
	}
	if cell != nil {
		v.Select(*cell)
		v.CenterOn(*cell)
	}
}
