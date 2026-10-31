// Package inspect describes the simulation's state in words, for tools that show it: what is on a
// cell, what a thing is, what a procedure does. It does not import Ebitengine.
package inspect

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// Text returns the string of text key in the pack's first language, or the key itself.
func Text(p *content.Pack, key string) string {
	for _, id := range sortedKeys(p.Texts) {
		if s, ok := p.Texts[id].Strings[key]; ok {
			return s
		}
	}
	return key
}

// ItemName returns an item's name.
func ItemName(p *content.Pack, item string) string {
	if it, ok := p.Items[item]; ok {
		return Text(p, it.Name)
	}
	return item
}

// Message returns the words an event shows, with an item's name in place of {item}.
func Message(p *content.Pack, e sim.Event) string {
	if e.Text == "" {
		return ""
	}
	return strings.ReplaceAll(Text(p, e.Text), "{item}", ItemName(p, e.Item))
}

// Hooks returns the procedures feature f's hooks run, in the order Cell lists them, skipping
// empty hooks.
func Hooks(f *sim.Feature) []string {
	var procs []string
	for _, h := range hookNames {
		if p := f.Hook(h); p != nil && *p != "" {
			procs = append(procs, *p)
		}
	}
	return procs
}

var hookNames = []string{content.HookTurnOn, content.HookTurnOff, content.HookPress, content.HookRelease, content.HookInsert, content.HookComplete}

// Cell describes cell c of level l: its floor and wall, its feature, its objects and the things
// standing in it.
func Cell(p *content.Pack, l *sim.Level, c content.Cell) []string {
	if !l.InBounds(c) {
		return []string{fmt.Sprintf("Cell (%d, %d): outside the level", c[0], c[1])}
	}
	lines := []string{fmt.Sprintf("Cell (%d, %d)", c[0], c[1])}
	add := func(format string, args ...any) { lines = append(lines, "  "+fmt.Sprintf(format, args...)) }

	if ft := l.FloorType(c); ft != nil {
		add("floor: type %d, variant %d%s", l.Floor[c[0]][c[1]], ft.Variant, floorEffects(ft))
	} else {
		add("floor: none")
	}
	if s := l.Wall[c[0]][c[1]]; s != content.NoFrame {
		add("wall: shape %d", s)
	}
	if l.Blocked(c) {
		add("blocked")
	}

	if f := l.FeatureAt(c); f != nil {
		lines = append(lines, feature(p, l, f)...)
	}
	if objs := l.Objects[c]; len(objs) > 0 {
		lines = append(lines, fmt.Sprintf("Objects (%d), top last:", len(objs)))
		for _, o := range objs {
			add("%s", object(p, o))
		}
	}
	for _, t := range l.ThingsAt(c) {
		lines = append(lines, fmt.Sprintf("Thing %s (%s, %s), %d hit points", t.ID, t.Template.ID, t.Template.Kind, t.HitPoints))
	}
	return lines
}

func floorEffects(ft *content.FloorType) string {
	var parts []string
	if e := ft.Enter; e != nil {
		parts = append(parts, "entering: "+effect(e))
	}
	if e := ft.Tick; e != nil {
		parts = append(parts, "each status tick: "+effect(e))
	}
	if ft.Depth > 0 {
		parts = append(parts, fmt.Sprintf("depth %d", ft.Depth))
	}
	if ft.Sinks {
		parts = append(parts, "objects sink")
	}
	if ft.Avoided {
		parts = append(parts, "creatures avoid it")
	}
	if len(parts) == 0 {
		return ""
	}
	return "; " + strings.Join(parts, "; ")
}

func effect(e *content.Effect) string {
	var parts []string
	if e.Damage != (content.Dice{}) {
		s := e.Damage.String() + " damage"
		if e.Type != "" {
			s = e.Damage.String() + " " + e.Type + " damage"
		}
		parts = append(parts, s)
	}
	if e.DrainMagic > 0 {
		parts = append(parts, fmt.Sprintf("%d magic drained", e.DrainMagic))
	}
	return strings.Join(parts, ", ")
}

func object(p *content.Pack, o *sim.Object) string {
	s := ItemName(p, o.Item)
	if len(o.Contents) > 0 {
		var in []string
		for _, c := range o.Contents {
			in = append(in, object(p, c))
		}
		s += " holding " + strings.Join(in, ", ")
	}
	return fmt.Sprintf("%s (weight %d)", s, o.Weight(p))
}

// feature describes a feature and the procedures its hooks run.
func feature(p *content.Pack, l *sim.Level, f *sim.Feature) []string {
	lines := []string{fmt.Sprintf("Feature %s (%s)", f.ID, f.Kind)}
	add := func(format string, args ...any) { lines = append(lines, "  "+fmt.Sprintf(format, args...)) }
	armed := func() string {
		if f.Armed {
			return "armed"
		}
		return "disarmed"
	}
	switch f.Kind {
	case content.FeatureDoor:
		add("%s, stage %d of 4", f.State, f.Stage)
		if f.SealedBy != "" {
			add("sealed: opens to %s", ItemName(p, f.SealedBy))
		}
		if f.OpenedBySpell {
			add("a spell opens it")
		}
	case content.FeatureLever:
		add("%s, %s", onOff(f.On, "on", "off"), armed())
	case content.FeaturePlate:
		add("%s, %s, weight %d of %d", onOff(f.Down, "down", "up"), armed(), f.Counter, f.Threshold)
	case content.FeatureSlot:
		add("takes %s, %s", ItemName(p, f.Accepts), armed())
	case content.FeatureTeleporter:
		add("%s; next to (%d, %d), then (%d, %d)", onOff(f.Armed, "armed", "disarmed"),
			f.Destinations[f.Next][0], f.Destinations[f.Next][1], f.Destinations[1-f.Next][0], f.Destinations[1-f.Next][1])
		if f.Turn != 0 || f.FlyOver {
			add("turns things in flight %d eighths; things in flight pass over: %v", f.Turn, f.FlyOver)
		}
	case content.FeatureExit, content.FeaturePit, content.FeatureTrapdoor:
		add("to %s (%d, %d)", f.To.Level, f.To.Cell[0], f.To.Cell[1])
		if f.Kind == content.FeatureTrapdoor {
			add("%s", onOff(f.Open, "open", "closed"))
		}
	case content.FeatureSocket:
		for i, it := range f.Places {
			add("%s: %s", ItemName(p, it), onOff(slices.Contains(f.Set, i), "set", "empty"))
		}
	case content.FeatureMouth:
		add("says %q", Text(p, f.Message))
	case content.FeatureMark:
		add("rune %s", f.Rune)
	}
	n := 0
	for _, h := range hookNames {
		hp := f.Hook(h)
		if hp == nil || *hp == "" {
			continue
		}
		n++
		add("[%d] %s runs %s:", n, h, *hp)
		for _, a := range l.Def.Procedures[*hp] {
			add("      %s", Action(p, a))
		}
	}
	return lines
}

func onOff(b bool, yes, no string) string {
	if b {
		return yes
	}
	return no
}

// Action describes an action in words.
func Action(p *content.Pack, a content.Action) string {
	cell := func(c content.Cell) string { return fmt.Sprintf("(%d, %d)", c[0], c[1]) }
	switch {
	case a.On != nil:
		return "switch on " + *a.On
	case a.Off != nil:
		return "switch off " + *a.Off
	case a.Toggle != nil:
		return "toggle " + *a.Toggle
	case a.SetHook != nil:
		to := "nothing"
		if a.SetHook.Procedure != nil {
			to = *a.SetHook.Procedure
		}
		return fmt.Sprintf("point %s's %s at %s", a.SetHook.Feature, a.SetHook.Hook, to)
	case a.SetThreshold != nil:
		return fmt.Sprintf("set %s's threshold to %d", a.SetThreshold.Feature, a.SetThreshold.Value)
	case a.SetAccepts != nil:
		return fmt.Sprintf("make %s take %s", a.SetAccepts.Feature, ItemName(p, a.SetAccepts.Item))
	case a.SetMessage != nil:
		return fmt.Sprintf("make %s say %q", a.SetMessage.Feature, Text(p, a.SetMessage.Message))
	case a.Roll != nil:
		return fmt.Sprintf("roll %s in direction %d%s", a.Roll.Thing, a.Roll.Direction, onOff(a.Roll.Shuttle, ", shuttling", ""))
	case a.Stop != nil:
		return "stop " + *a.Stop
	case a.ShootItem != nil:
		return fmt.Sprintf("shoot %s from %s in direction %d", ItemName(p, a.ShootItem.Item), cell(a.ShootItem.Cell), a.ShootItem.Direction)
	case a.ShootSpell != nil:
		return fmt.Sprintf("shoot spell %d from %s in direction %d, power %s", a.ShootSpell.Spell, cell(a.ShootSpell.Cell), a.ShootSpell.Direction, a.ShootSpell.Power)
	case a.Create != nil:
		return fmt.Sprintf("make %s at %s", ItemName(p, a.Create.Item), cell(a.Create.Cell))
	case a.SetFloor != nil:
		return fmt.Sprintf("make %s floor of type %d", cell(a.SetFloor.Cell), a.SetFloor.Type)
	case a.SetWall != nil:
		return fmt.Sprintf("make %s a wall of shape %d", cell(a.SetWall.Cell), a.SetWall.Shape)
	}
	return "nothing"
}

// Thing describes thing t.
func Thing(p *content.Pack, t *sim.Thing) []string {
	tm := t.Template
	c := t.Cell()
	lines := []string{fmt.Sprintf("Thing %s (%s, %s)", t.ID, tm.ID, tm.Kind)}
	add := func(format string, args ...any) { lines = append(lines, "  "+fmt.Sprintf(format, args...)) }
	add("at (%d, %d) in cell (%d, %d), facing %d", t.U, t.V, c[0], c[1], t.Facing)
	if tm.HitPoints != (content.Dice{}) || t.HitPoints != 0 {
		add("hit points %d (rolled %s)", t.HitPoints, tm.HitPoints)
	}
	add("speed %d, update every %d ms, weight %d", tm.Speed, tm.IntervalMS, tm.Weight)
	if len(tm.Behaviors) > 0 {
		add("behaviors: %s", strings.Join(tm.Behaviors, ", "))
	}
	if len(tm.Stats) > 0 {
		var stats []string
		for _, k := range sortedKeys(tm.Stats) {
			stats = append(stats, fmt.Sprintf("%s %d", k, tm.Stats[k]))
		}
		add("stats: %s", strings.Join(stats, ", "))
	}
	for _, o := range t.Items {
		add("carries %s", object(p, o))
	}
	if r := t.Rolling; r != nil {
		add("rolling in direction %d%s", r.Direction, onOff(r.Shuttle, ", shuttling", ""))
	}
	return lines
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
