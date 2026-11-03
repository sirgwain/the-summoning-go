package viewer

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The full book (docs/design.md, Editor): the tools' book. It lists every creature and item of the
// pack, not only those come across, and shows each at its last stage with the whole of its
// record and where it is on the current level. It lists the current level's alone until the
// toggle shows them all; the current level's are marked either way.

// OpenFullBook opens the full book, with the tools.
func (v *Viewer) OpenFullBook() {
	if v.Play {
		return
	}
	v.BookOpen, v.BookFull, v.BookTyped, v.SpellsOpen = true, true, "", false
	v.bookView.scroll = 0
}

// fullBookEntries returns the full book's entries of tab: the pack's creature templates or items,
// each marked when it is on the current level, all of them with BookAll, else those on the level.
func (v *Viewer) fullBookEntries(tab int) []bookEntry {
	here := v.levelTemplates(tab)
	var out []bookEntry
	if tab == BookCreatures {
		for id, t := range v.Pack.Things {
			if t.Kind == content.ThingCreature {
				out = append(out, bookEntry{id: id, name: v.logName(id), inLevel: here[id]})
			}
		}
	} else {
		for id := range v.Pack.Items {
			out = append(out, bookEntry{id: id, name: inspect.ItemName(v.Pack, id), inLevel: here[id]})
		}
	}
	if !v.BookAll {
		out = slices.DeleteFunc(out, func(e bookEntry) bool { return !e.inLevel })
	}
	return out
}

// levelTemplates returns the templates of tab on the current level: the creature templates of
// its things, or the items its places name (levelPlaces).
func (v *Viewer) levelTemplates(tab int) map[string]bool {
	l := v.World.Current()
	here := map[string]bool{}
	if tab == BookCreatures {
		for _, t := range l.Things {
			here[t.Template.ID] = true
		}
		return here
	}
	for id := range v.Pack.Items {
		if len(v.itemPlaces(l, id)) > 0 {
			here[id] = true
		}
	}
	return here
}

// fullEntry returns the book's entry e as the full book shows it: at the last stage, with every
// defense of creature template t learned, keeping what e has come across.
func fullEntry(e *sim.Entry, t *content.Thing) *sim.Entry {
	full := sim.Entry{Stage: sim.StageKilled}
	if e != nil {
		full = *e
		full.Stage = sim.StageKilled
	}
	if t == nil {
		return &full
	}
	var defenses []string
	if t.Vulnerability != nil {
		defenses = append(defenses, sim.DefenseVulnerable)
	}
	for _, b := range []string{content.BehaviorResistsFire, content.BehaviorResistsLight, content.BehaviorResistsPoison,
		content.BehaviorResistsTeleport, content.BehaviorPhysicalOnly} {
		if t.Has(b) {
			defenses = append(defenses, b)
		}
	}
	full.Defenses = defenses
	return &full
}

// fullBookLines returns what the full book says of tab's entry id: the book's lines at the last
// stage; where it is on the current level; then its record, a line a field.
func (v *Viewer) fullBookLines(tab int, id string) []string {
	b := v.World.Book()
	l := v.World.Current()
	var lines, places []string
	var record any
	if tab == BookCreatures {
		t := v.Pack.Things[id]
		lines, _ = v.CreatureLines(id, fullEntry(b.Creatures[id], t))
		for _, th := range l.Things {
			if th.Template.ID == id {
				c := th.Cell()
				places = append(places, fmt.Sprintf("%s at (%d, %d), %d hit points", th.ID, c[0], c[1], th.HitPoints))
			}
		}
		if t != nil {
			record = t
		}
	} else {
		lines, _ = v.ItemLines(id, fullEntry(b.Items[id], nil))
		places = v.itemPlaces(l, id)
		if it := v.Pack.Items[id]; it != nil {
			record = it
		}
	}
	if len(places) > 0 {
		lines = append(lines, "", "On "+l.ID+":")
		for _, p := range places {
			lines = append(lines, "  "+p)
		}
	}
	if record != nil {
		lines = append(lines, "", "Record "+id+":")
		lines = append(lines, recordLines(record)...)
	}
	return lines
}

// recordLines returns record r's fields as its JSON has them, a line each, in the JSON's order.
func recordLines(r any) []string {
	b, err := json.Marshal(r)
	if err != nil {
		return []string{err.Error()}
	}
	var fields []struct {
		key string
		val json.RawMessage
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	if _, err := dec.Token(); err != nil {
		return nil
	}
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			break
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			break
		}
		fields = append(fields, struct {
			key string
			val json.RawMessage
		}{fmt.Sprint(k), raw})
	}
	var lines []string
	for _, f := range fields {
		if f.key == "id" {
			continue
		}
		lines = append(lines, "  "+f.key+": "+string(f.val))
	}
	return lines
}

// itemPlaces returns where item id is on level l, in words: lying on the floor or in a container
// there, carried by a creature or the player, taken by a slot, made or shot by a procedure, or
// given or taken by the conversation of a creature on the level.
func (v *Viewer) itemPlaces(l *sim.Level, id string) []string {
	var out []string
	var walk func(objs []*sim.Object, where string)
	walk = func(objs []*sim.Object, where string) {
		for _, o := range objs {
			if o.Item == id {
				out = append(out, where)
			}
			if len(o.Contents) > 0 {
				walk(o.Contents, "in "+inspect.ItemName(v.Pack, o.Item)+" "+where)
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
		walk(l.Objects[c], fmt.Sprintf("on the floor at (%d, %d)", c[0], c[1]))
	}
	for _, t := range l.Things {
		walk(t.Items, "carried by "+t.ID)
	}
	if p := v.World.Player(); p != nil {
		objs := []*sim.Object{p.Hand}
		objs = append(objs, p.Slots[:]...)
		walk(slices.DeleteFunc(objs, func(o *sim.Object) bool { return o == nil }), "carried by the player")
	}
	for _, f := range l.Features {
		if f.Accepts == id {
			out = append(out, fmt.Sprintf("taken by %s at (%d, %d)", f.ID, f.Cell[0], f.Cell[1]))
		}
	}
	for _, proc := range sortedIDs(l.Def.Procedures) {
		for _, a := range l.Def.Procedures[proc] {
			if inspect.ActionItem(a) == id && a.SetAccepts == nil {
				out = append(out, proc+": "+inspect.Action(v.Pack, a))
			}
		}
	}
	for _, t := range l.Things {
		conv := v.Pack.Conversations[t.Template.Conversation]
		if conv == nil {
			continue
		}
		for i, c := range conv.Clauses {
			for _, a := range c.Do {
				switch {
				case a.Give != nil && a.Give.Item == id:
					out = append(out, fmt.Sprintf("given by %s (%s, clause %d)", t.ID, conv.ID, i))
				case a.Take != nil && *a.Take == id:
					out = append(out, fmt.Sprintf("taken by %s (%s, clause %d)", t.ID, conv.ID, i))
				}
			}
		}
	}
	return out
}
