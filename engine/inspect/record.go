package inspect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The full book's pane (docs/design.md, Editor): an entry's book lines at the last stage, where
// it is on the current level, and its whole record.

// PaneItem is a line of the full book's pane: its text, the pictures shown below it, and the
// thing or cell of the current level a click on it selects, if any.
type PaneItem struct {
	Text     string
	Pictures []content.Image
	Thing    string
	Cell     *content.Cell
}

// FullCreature returns what the full book says of creature template id on the current level of
// world w, whose places are ps.
func FullCreature(p *content.Pack, w *sim.World, ps Places, id string) []PaneItem {
	t := p.Things[id]
	lines, _ := CreatureLines(p, id, FullEntry(w.Book().Creatures[id], t))
	var record any
	if t != nil {
		record = t
	}
	return fullPane(p, w, lines, ps.Creatures[id], id, record)
}

// FullItem returns what the full book says of item id on the current level of world w, whose
// places are ps.
func FullItem(p *content.Pack, w *sim.World, ps Places, id string) []PaneItem {
	lines, _ := ItemLines(p, id, FullEntry(w.Book().Items[id], nil))
	var record any
	if it := p.Items[id]; it != nil {
		record = it
	}
	return fullPane(p, w, lines, ps.Items[id], id, record)
}

// fullPane lays out the full book's pane: the book's lines, the places, then the record.
func fullPane(p *content.Pack, w *sim.World, lines []string, places []Place, id string, record any) []PaneItem {
	var items []PaneItem
	for _, s := range lines {
		items = append(items, PaneItem{Text: s})
	}
	if len(places) > 0 {
		items = append(items, PaneItem{}, PaneItem{Text: "On " + w.Current().ID + " (click to go there):"})
		for _, pl := range places {
			items = append(items, PaneItem{Text: "  " + pl.Text, Thing: pl.Thing, Cell: pl.Cell})
		}
	}
	if record != nil {
		items = append(items, PaneItem{}, PaneItem{Text: "Record " + id + ":"})
		items = append(items, Record(p, record)...)
	}
	return items
}

// field renders a record's field: its key, its value as JSON, and the record.
type field func(p *content.Pack, key string, raw json.RawMessage, r any) []PaneItem

// fields are the fields rendered their own way, by their key, for a creature template and an
// item; nil leaves the field out. Every other field shows as plainField does.
var fields = map[string]field{
	"id": nil,
	"behaviors": func(p *content.Pack, key string, raw json.RawMessage, r any) []PaneItem {
		t, ok := r.(*content.Thing)
		if !ok {
			return plainField(p, key, raw, r)
		}
		out := []PaneItem{{Text: "  behaviors:"}}
		for _, b := range t.Behaviors {
			text := "    " + b
			if d := Behavior(b); d != "" {
				text += ": " + d
			}
			out = append(out, PaneItem{Text: text})
		}
		return out
	},
	"animations": func(p *content.Pack, key string, raw json.RawMessage, r any) []PaneItem {
		if t, ok := r.(*content.Thing); ok {
			return Animations(p, t)
		}
		return plainField(p, key, raw, r)
	},
	"animate": func(p *content.Pack, key string, raw json.RawMessage, r any) []PaneItem {
		t, ok := r.(*content.Thing)
		if !ok || t.Animate == nil {
			return plainField(p, key, raw, r)
		}
		text := "  animate: " + t.Animate.Kind
		if t.Animate.Random {
			text += ", random"
		}
		return []PaneItem{{Text: text, Pictures: Frames(t.Sprite, t.Animate.Frames)}}
	},
	"actions": func(p *content.Pack, key string, raw json.RawMessage, r any) []PaneItem {
		var acts []content.ItemAction
		if json.Unmarshal(raw, &acts) != nil {
			return plainField(p, key, raw, r)
		}
		out := []PaneItem{{Text: "  actions:"}}
		for i, a := range acts {
			it := PaneItem{Text: fmt.Sprintf("    %d. %s", i+1, ActionText(a))}
			if a.Icon != nil {
				it.Pictures = []content.Image{*a.Icon}
			}
			out = append(out, it)
		}
		return out
	},
	"power": func(p *content.Pack, key string, raw json.RawMessage, r any) []PaneItem {
		it, ok := r.(*content.Item)
		if !ok {
			return plainField(p, key, raw, r)
		}
		return []PaneItem{{Text: "  power: " + string(raw) + " (" + PowerText(p, it) + ")"}}
	},
}

// Record returns record r's fields as its JSON has them, in the JSON's order, a line each or as
// fields renders them; a creature template's animations include its sprite's.
func Record(p *content.Pack, r any) []PaneItem {
	b, err := json.Marshal(r)
	if err != nil {
		return []PaneItem{{Text: err.Error()}}
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err := dec.Token(); err != nil {
		return nil
	}
	var out []PaneItem
	animations := false
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
		animations = animations || key == "animations"
		f, special := fields[key]
		switch {
		case !special:
			out = append(out, plainField(p, key, raw, r)...)
		case f != nil:
			out = append(out, f(p, key, raw, r)...)
		}
	}
	// A creature template without animations of its own shows its sprite's.
	if t, ok := r.(*content.Thing); ok && !animations {
		out = append(out, Animations(p, t)...)
	}
	return out
}

// plainField renders a field as its key and value: an image, or an object holding one, as its
// picture; a text key with its words; anything else as its JSON.
func plainField(p *content.Pack, key string, raw json.RawMessage, _ any) []PaneItem {
	if img, rest, ok := imageField(p, raw); ok {
		text := "  " + key + ":"
		if rest != "" {
			text += " " + rest
		}
		return []PaneItem{{Text: text, Pictures: []content.Image{img}}}
	}
	text := "  " + key + ": " + string(raw)
	if s := textOf(p, raw); s != "" {
		text += " " + strconv.Quote(s)
	}
	return []PaneItem{{Text: text}}
}

// textOf returns the words of the text a field's value names, empty for a value that names none.
func textOf(p *content.Pack, raw json.RawMessage) string {
	var key string
	if json.Unmarshal(raw, &key) != nil || key == "" {
		return ""
	}
	if s := Text(p, key); s != key {
		return s
	}
	return ""
}

// imageField returns the image a field's value is, or holds under "image", with the rest of the
// value in words (a doll's place on the figure); false when it is no image of the pack's sprites.
func imageField(p *content.Pack, raw json.RawMessage) (content.Image, string, bool) {
	var img content.Image
	if json.Unmarshal(raw, &img) == nil && p.Sprites[img.Sprite] != nil {
		var fs map[string]json.RawMessage
		if json.Unmarshal(raw, &fs) == nil && len(fs) <= 2 {
			return img, "", true
		}
	}
	var d content.DollImage
	if json.Unmarshal(raw, &d) == nil && p.Sprites[d.Image.Sprite] != nil {
		return d.Image, fmt.Sprintf("at (%d, %d)", d.At[0], d.At[1]), true
	}
	return content.Image{}, "", false
}

// PowerText says what item it's power does: its effect text, else the spell it casts or the
// conversation it starts.
func PowerText(p *content.Pack, it *content.Item) string {
	switch {
	case it.Effect != "":
		return Text(p, it.Effect)
	case it.Casts != nil:
		return "casts " + SpellName(p, it.Casts.Spell)
	case it.Conversation != "":
		return "starts the conversation of " + ThingName(p, it.Conversation)
	}
	return "no effect of its own"
}

// ActionText says what an item's action does in hand: its kind, damage, wait, the skill it needs,
// and its swing and sound.
func ActionText(a content.ItemAction) string {
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

// AnimationDir is the direction whose animations the full book shows of those that have one for
// each direction: 2, down and right on the screen (docs/spec/README.md, Directions).
const AnimationDir = 2

// Animations returns creature template t's animations, its own and its sprite's, a line each with
// its frames as pictures: of the animations named for directions ("walk.0" to "walk.6"), only
// AnimationDir's, or the first direction's when it has none for AnimationDir.
func Animations(p *content.Pack, t *content.Thing) []PaneItem {
	all := map[string]content.Animation{}
	if sp := p.Sprites[t.Sprite]; sp != nil {
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
	out := []PaneItem{{Text: "  animations:"}}
	for _, base := range bases {
		ds := dirs[base]
		slices.Sort(ds)
		name, label := base, base
		if ds[0] >= 0 {
			d := ds[0]
			if slices.Contains(ds, AnimationDir) {
				d = AnimationDir
			}
			name = fmt.Sprintf("%s.%d", base, d)
			label = fmt.Sprintf("%s (direction %d", base, d)
			if d == AnimationDir {
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
		out = append(out, PaneItem{Text: text, Pictures: Frames(t.Sprite, a.Frames)})
	}
	return out
}

// Frames returns frames of sprite as images.
func Frames(sprite string, fs []int) []content.Image {
	out := make([]content.Image, len(fs))
	for i, f := range fs {
		out[i] = content.Image{Sprite: sprite, Frame: f}
	}
	return out
}
