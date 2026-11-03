package inspect

import (
	"fmt"
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// Place is where a creature template or an item is on a level, in words, and the thing or cell a
// tool selects to go there, if any.
type Place struct {
	Text  string
	Thing string
	Cell  *content.Cell
}

// Places are where each creature template and each item is on a level: a creature template's
// things; an item lying on the floor or in a container there, carried by a creature or the player,
// taken by a slot, made or shot by a procedure, or given or taken by the conversation of a
// creature on the level.
type Places struct {
	Creatures map[string][]Place
	Items     map[string][]Place
}

// LevelPlaces returns the places of the current level of world w, found in one pass over it.
func LevelPlaces(p *content.Pack, w *sim.World) Places {
	l := w.Current()
	ps := Places{Creatures: map[string][]Place{}, Items: map[string][]Place{}}
	item := func(id string, pl Place) { ps.Items[id] = append(ps.Items[id], pl) }
	var walk func(objs []*sim.Object, where Place)
	walk = func(objs []*sim.Object, where Place) {
		for _, o := range objs {
			if o == nil {
				continue
			}
			item(o.Item, where)
			if len(o.Contents) > 0 {
				in := where
				in.Text = "in " + ItemName(p, o.Item) + " " + where.Text
				walk(o.Contents, in)
			}
		}
	}
	cells := make([]content.Cell, 0, len(l.Objects))
	for c := range l.Objects {
		cells = append(cells, c)
	}
	slices.SortFunc(cells, func(a, b content.Cell) int {
		if a[1] != b[1] {
			return a[1] - b[1]
		}
		return a[0] - b[0]
	})
	for _, c := range cells {
		walk(l.Objects[c], Place{Text: fmt.Sprintf("on the floor at (%d, %d)", c[0], c[1]), Cell: &c})
	}
	for _, t := range l.Things {
		c := t.Cell()
		ps.Creatures[t.Template.ID] = append(ps.Creatures[t.Template.ID],
			Place{Text: fmt.Sprintf("%s at (%d, %d), %d hit points", t.ID, c[0], c[1], t.HitPoints), Thing: t.ID})
		walk(t.Items, Place{Text: "carried by " + t.ID, Thing: t.ID})
	}
	if pl := w.Player(); pl != nil {
		walk(append([]*sim.Object{pl.Hand}, pl.Slots[:]...), Place{Text: "carried by the player", Thing: sim.PlayerID})
	}
	for _, f := range l.Features {
		if f.Accepts != "" {
			c := f.Cell
			item(f.Accepts, Place{Text: fmt.Sprintf("taken by %s at (%d, %d)", f.ID, c[0], c[1]), Cell: &c})
		}
	}
	for _, proc := range sortedKeys(l.Def.Procedures) {
		for _, a := range l.Def.Procedures[proc] {
			if id := ActionItem(a); id != "" && a.SetAccepts == nil {
				item(id, Place{Text: proc + ": " + Action(p, a)})
			}
		}
	}
	for _, t := range l.Things {
		conv := p.Conversations[t.Template.Conversation]
		if conv == nil {
			continue
		}
		ConversationActions(nil, conv, func(i int, _ content.Clause, a content.Action, _ string) {
			switch {
			case a.Give != nil:
				item(a.Give.Item, Place{Text: fmt.Sprintf("given by %s (%s, clause %d)", t.ID, conv.ID, i), Thing: t.ID})
			case a.Take != nil:
				item(*a.Take, Place{Text: fmt.Sprintf("taken by %s (%s, clause %d)", t.ID, conv.ID, i), Thing: t.ID})
			}
		})
	}
	return ps
}
