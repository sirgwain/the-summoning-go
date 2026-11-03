package viewer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
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

// paneItem is a line of the full book's pane: its text, the pictures shown below it, and what a
// click on it selects, a thing or a cell of the current level, if anything.
type paneItem struct {
	text     string
	pictures []content.Image
	thing    string
	cell     *content.Cell
}

// place is where an entry is on the current level: in words, and the thing or cell it is at.
type place struct {
	text  string
	thing string
	cell  *content.Cell
}

// fullBookLines returns what the full book says of tab's entry id: the book's lines at the last
// stage; where it is on the current level; then its record, a line a field, an image field as its
// picture and an item's actions in words.
func (v *Viewer) fullBookLines(tab int, id string) []paneItem {
	b := v.World.Book()
	l := v.World.Current()
	var lines []string
	var places []place
	var record any
	if tab == BookCreatures {
		t := v.Pack.Things[id]
		lines, _ = v.CreatureLines(id, fullEntry(b.Creatures[id], t))
		for _, th := range l.Things {
			if th.Template.ID == id {
				c := th.Cell()
				places = append(places, place{text: fmt.Sprintf("%s at (%d, %d), %d hit points", th.ID, c[0], c[1], th.HitPoints), thing: th.ID})
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
	var items []paneItem
	for _, s := range lines {
		items = append(items, paneItem{text: s})
	}
	if len(places) > 0 {
		items = append(items, paneItem{}, paneItem{text: "On " + l.ID + " (click to go there):"})
		for _, p := range places {
			items = append(items, paneItem{text: "  " + p.text, thing: p.thing, cell: p.cell})
		}
	}
	if record != nil {
		items = append(items, paneItem{}, paneItem{text: "Record " + id + ":"})
		items = append(items, v.recordItems(record)...)
	}
	return items
}

// recordItems returns record r's fields as its JSON has them, a line each in the JSON's order:
// an image, or an object holding one, as its picture, the actions of an item in words, and a
// creature's animations as their frames.
func (v *Viewer) recordItems(r any) []paneItem {
	thing, _ := r.(*content.Thing)
	b, err := json.Marshal(r)
	if err != nil {
		return []paneItem{{text: err.Error()}}
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err := dec.Token(); err != nil {
		return nil
	}
	var out []paneItem
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			break
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			break
		}
		key := fmt.Sprint(k)
		switch {
		case key == "id":
		case key == "animations" && thing != nil:
			// The template's and its sprite's together, below.
		case key == "animate" && thing != nil && thing.Animate != nil:
			a := thing.Animate
			text := "  animate: " + a.Kind
			if a.Random {
				text += ", random"
			}
			out = append(out, paneItem{text: text, pictures: frames(thing.Sprite, a.Frames)})
		case key == "actions":
			var acts []content.ItemAction
			if json.Unmarshal(raw, &acts) == nil {
				out = append(out, paneItem{text: "  actions:"})
				for i, a := range acts {
					it := paneItem{text: fmt.Sprintf("    %d. %s", i+1, describeAction(a))}
					if a.Icon != nil {
						it.pictures = []content.Image{*a.Icon}
					}
					out = append(out, it)
				}
				continue
			}
			fallthrough
		default:
			if img, rest, ok := v.imageField(raw); ok {
				text := "  " + key + ":"
				if rest != "" {
					text += " " + rest
				}
				out = append(out, paneItem{text: text, pictures: []content.Image{img}})
				continue
			}
			out = append(out, paneItem{text: "  " + key + ": " + string(raw)})
		}
	}
	if thing != nil {
		out = append(out, v.animationItems(thing)...)
	}
	return out
}

// animationDir is the direction whose animations the full book shows of those that have one for
// each direction: 2, down and right on the screen (docs/spec/README.md, Directions).
const animationDir = 2

// animationItems returns creature template t's animations, its own and its sprite's, a line each
// with its frames as pictures: of the animations named for directions ("walk.0" to "walk.6"),
// only animationDir's, or the first direction's when it has none for animationDir.
func (v *Viewer) animationItems(t *content.Thing) []paneItem {
	all := map[string]content.Animation{}
	if sp := v.Pack.Sprites[t.Sprite]; sp != nil {
		maps.Copy(all, sp.Animations)
	}
	maps.Copy(all, t.Animations)
	if len(all) == 0 {
		return nil
	}
	// The animations by base name, and for each its directions.
	dirs := map[string][]int{}
	var bases []string
	for name := range all {
		base, dir := name, -1
		if i := strings.LastIndexByte(name, '.'); i >= 0 {
			if d, err := strconv.Atoi(name[i+1:]); err == nil {
				base, dir = name[:i], d
			}
		}
		if _, ok := dirs[base]; !ok {
			bases = append(bases, base)
		}
		dirs[base] = append(dirs[base], dir)
	}
	slices.Sort(bases)
	out := []paneItem{{text: "  animations:"}}
	for _, base := range bases {
		ds := dirs[base]
		slices.Sort(ds)
		name, label := base, base
		if ds[0] >= 0 {
			d := ds[0]
			if slices.Contains(ds, animationDir) {
				d = animationDir
			}
			name = fmt.Sprintf("%s.%d", base, d)
			label = fmt.Sprintf("%s (direction %d", base, d)
			if d == animationDir {
				label += ", down-right"
			}
			label += fmt.Sprintf("; %d directions)", len(ds))
		}
		a := all[name]
		text := "    " + label
		if a.MS != (content.Duration{}) {
			if a.MS.Min == a.MS.Max {
				text += fmt.Sprintf(", %d ms a frame", a.MS.Min)
			} else {
				text += fmt.Sprintf(", %d-%d ms a frame", a.MS.Min, a.MS.Max)
			}
		}
		if a.Mode != "" {
			text += ", " + a.Mode
		}
		out = append(out, paneItem{text: text, pictures: frames(t.Sprite, a.Frames)})
	}
	return out
}

// frames returns frames of sprite as images.
func frames(sprite string, fs []int) []content.Image {
	out := make([]content.Image, len(fs))
	for i, f := range fs {
		out[i] = content.Image{Sprite: sprite, Frame: f}
	}
	return out
}

// imageField returns the image a field's value is, or holds under "image", with the rest of the
// value in words (a doll's place on the figure); false when it is no image of the pack's sprites.
func (v *Viewer) imageField(raw json.RawMessage) (content.Image, string, bool) {
	var img content.Image
	if json.Unmarshal(raw, &img) == nil && v.Pack.Sprites[img.Sprite] != nil {
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) == nil && len(fields) <= 2 {
			return img, "", true
		}
	}
	var d content.DollImage
	if json.Unmarshal(raw, &d) == nil && v.Pack.Sprites[d.Image.Sprite] != nil {
		return d.Image, fmt.Sprintf("at (%d, %d)", d.At[0], d.At[1]), true
	}
	return content.Image{}, "", false
}

// describeAction says what an item's action does in hand: its kind, damage, wait, the skill it
// needs, and its swing and sound.
func describeAction(a content.ItemAction) string {
	var s string
	switch a.Kind {
	case content.ActionAttack:
		s = "strike"
	case content.ActionAttackBoth:
		s = "strike with both hands"
	case content.ActionFire:
		s = "fire"
	case content.ActionUse:
		s = "use"
	case content.ActionUseUp:
		s = "use up"
	default:
		s = a.Kind
	}
	if a.Damage != (content.Dice{}) {
		s += " for " + a.Damage.String()
	}
	s += fmt.Sprintf(", then wait %d updates", a.Delay)
	if a.Skill > 0 {
		s += fmt.Sprintf(", needs skill %d", a.Skill)
	}
	if a.Swings {
		s += ", swings"
	}
	if a.Sound != "" {
		s += ", sound " + a.Sound
	}
	return s
}

// itemPlaces returns where item id is on level l: lying on the floor or in a container there,
// carried by a creature or the player, taken by a slot, made or shot by a procedure, or given or
// taken by the conversation of a creature on the level.
func (v *Viewer) itemPlaces(l *sim.Level, id string) []place {
	var out []place
	var walk func(objs []*sim.Object, where place)
	walk = func(objs []*sim.Object, where place) {
		for _, o := range objs {
			if o.Item == id {
				out = append(out, where)
			}
			if len(o.Contents) > 0 {
				in := where
				in.text = "in " + inspect.ItemName(v.Pack, o.Item) + " " + where.text
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
		walk(l.Objects[c], place{text: fmt.Sprintf("on the floor at (%d, %d)", c[0], c[1]), cell: &c})
	}
	for _, t := range l.Things {
		walk(t.Items, place{text: "carried by " + t.ID, thing: t.ID})
	}
	if p := v.World.Player(); p != nil {
		objs := []*sim.Object{p.Hand}
		objs = append(objs, p.Slots[:]...)
		walk(slices.DeleteFunc(objs, func(o *sim.Object) bool { return o == nil }), place{text: "carried by the player", thing: sim.PlayerID})
	}
	for _, f := range l.Features {
		if f.Accepts == id {
			c := f.Cell
			out = append(out, place{text: fmt.Sprintf("taken by %s at (%d, %d)", f.ID, c[0], c[1]), cell: &c})
		}
	}
	for _, proc := range sortedIDs(l.Def.Procedures) {
		for _, a := range l.Def.Procedures[proc] {
			if inspect.ActionItem(a) == id && a.SetAccepts == nil {
				out = append(out, place{text: proc + ": " + inspect.Action(v.Pack, a)})
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
					out = append(out, place{text: fmt.Sprintf("given by %s (%s, clause %d)", t.ID, conv.ID, i), thing: t.ID})
				case a.Take != nil && *a.Take == id:
					out = append(out, place{text: fmt.Sprintf("taken by %s (%s, clause %d)", t.ID, conv.ID, i), thing: t.ID})
				}
			}
		}
	}
	return out
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
