// Package inspect describes the simulation's state in words, for tools that show it: what is on a
// cell, what a thing is, what a procedure does. It does not import Ebitengine.
package inspect

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
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

// Message returns the words an event shows, with an item's name in place of {item}, a skill's
// in place of {skill}, and a strike's numbers in place of {score} and {damage}.
func Message(p *content.Pack, e sim.Event) string {
	if e.Text == "" {
		return ""
	}
	skill := e.Item
	if n, ok := strings.CutPrefix(e.Item, "skill."); ok {
		skill = SkillName(n)
	}
	return strings.NewReplacer("{item}", ItemName(p, e.Item), "{skill}", skill, "{school}", Text(p, e.Item), "{name}", Text(p, e.Item),
		"{score}", strconv.Itoa(e.Score), "{damage}", strconv.Itoa(e.Damage)).Replace(Text(p, e.Text))
}

// SkillName names skill index n, as a string: weapon classes 0–3, then schools 0–3.
func SkillName(n string) string {
	i, err := strconv.Atoi(n)
	if err != nil || i < 0 || i >= sim.Skills {
		return n
	}
	if i < sim.SkillSchool {
		return fmt.Sprintf("weapon class %d", i)
	}
	return []string{"Wizardry", "Sorcery", "Enchantment", "Healing"}[i-sim.SkillSchool]
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

// Line is a line of a description, with the kind of link it describes, if any (a procedure's
// action, or where a feature sends things), and the item it names, if any.
type Line struct {
	Text string
	Kind LinkKind
	Item string
}

// Texts returns the lines' texts.
func Texts(lines []Line) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.Text
	}
	return out
}

// Cell describes cell c of level l: its floor and wall, its feature, its objects and the things
// standing in it.
func Cell(p *content.Pack, l *sim.Level, c content.Cell) []string { return Texts(CellLines(p, l, c)) }

// CellLines is Cell with each line's kind of link.
func CellLines(p *content.Pack, l *sim.Level, c content.Cell) []Line {
	if !l.InBounds(c) {
		return []Line{{Text: fmt.Sprintf("Cell (%d, %d): outside the level", c[0], c[1])}}
	}
	lines := []Line{{Text: fmt.Sprintf("Cell (%d, %d)", c[0], c[1])}}
	addItem := func(item, format string, args ...any) {
		lines = append(lines, Line{Text: "  " + fmt.Sprintf(format, args...), Item: item})
	}
	add := func(format string, args ...any) { addItem("", format, args...) }

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
		lines = append(lines, Line{Text: fmt.Sprintf("Objects (%d), top last:", len(objs))})
		for _, o := range objs {
			addItem(o.Item, "%s", object(p, o))
		}
	}
	for _, t := range l.ThingsAt(c) {
		lines = append(lines, Line{Text: fmt.Sprintf("Thing %s (%s, %s), %d hit points", t.ID, t.Template.ID, t.Template.Kind, t.HitPoints)})
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
func feature(p *content.Pack, l *sim.Level, f *sim.Feature) []Line {
	lines := []Line{{Text: fmt.Sprintf("Feature %s (%s)", f.ID, f.Kind)}}
	addLine := func(kind LinkKind, item, format string, args ...any) {
		lines = append(lines, Line{Text: "  " + fmt.Sprintf(format, args...), Kind: kind, Item: item})
	}
	addKind := func(kind LinkKind, format string, args ...any) { addLine(kind, "", format, args...) }
	addItem := func(item, format string, args ...any) { addLine("", item, format, args...) }
	add := func(format string, args ...any) { addLine("", "", format, args...) }
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
			addItem(f.SealedBy, "sealed: opens to %s", ItemName(p, f.SealedBy))
		}
		if f.OpenedBySpell {
			add("a spell opens it")
		}
	case content.FeatureLever:
		add("%s, %s", onOff(f.On, "on", "off"), armed())
	case content.FeaturePlate:
		add("%s, %s, weight %d of %d", onOff(f.Down, "down", "up"), armed(), f.Counter, f.Threshold)
	case content.FeatureSlot:
		addItem(f.Accepts, "takes %s, %s", ItemName(p, f.Accepts), armed())
	case content.FeatureTeleporter:
		addKind(LinkTravel, "%s; next to (%d, %d), then (%d, %d)", onOff(f.Armed, "armed", "disarmed"),
			f.Destinations[f.Next][0], f.Destinations[f.Next][1], f.Destinations[1-f.Next][0], f.Destinations[1-f.Next][1])
		if f.Turn != 0 || f.FlyOver {
			add("turns things in flight %d eighths; things in flight pass over: %v", f.Turn, f.FlyOver)
		}
	case content.FeatureExit, content.FeaturePit, content.FeatureTrapdoor:
		addKind(LinkTravel, "to %s (%d, %d)", f.To.Level, f.To.Cell[0], f.To.Cell[1])
		if f.Kind == content.FeatureTrapdoor {
			add("%s", onOff(f.Open, "open", "closed"))
		}
	case content.FeatureSocket:
		for i, it := range f.Places {
			addItem(it, "%s: %s", ItemName(p, it), onOff(slices.Contains(f.Set, i), "set", "empty"))
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
			addLine(ActionKind(a), ActionItem(a), "      %s", Action(p, a))
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

// ActionItem returns the item action a names: the one it makes, shoots or has a slot take;
// empty for none.
func ActionItem(a content.Action) string {
	switch {
	case a.SetAccepts != nil:
		return a.SetAccepts.Item
	case a.ShootItem != nil:
		return a.ShootItem.Item
	case a.Create != nil:
		return a.Create.Item
	}
	return ""
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
		return fmt.Sprintf("shoot spell %s from %s in direction %d, power %s", a.ShootSpell.Spell, cell(a.ShootSpell.Cell), a.ShootSpell.Direction, a.ShootSpell.Power)
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
func Thing(p *content.Pack, t *sim.Thing) []string { return Texts(ThingLines(p, t)) }

// ThingLines is Thing with the item each line names.
func ThingLines(p *content.Pack, t *sim.Thing) []Line {
	tm := t.Template
	c := t.Cell()
	lines := []Line{{Text: fmt.Sprintf("Thing %s (%s, %s)", t.ID, tm.ID, tm.Kind)}}
	addItem := func(item, format string, args ...any) {
		lines = append(lines, Line{Text: "  " + fmt.Sprintf(format, args...), Item: item})
	}
	add := func(format string, args ...any) { addItem("", format, args...) }
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
		addItem(o.Item, "carries %s", object(p, o))
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

// SlotName names inventory slot i.
func SlotName(i int) string {
	switch {
	case i < sim.Shelves:
		return fmt.Sprintf("shelf %d", i)
	case i >= sim.SlotPack:
		return fmt.Sprintf("pack %d", i-sim.SlotPack)
	}
	return [...]string{"head", "arms", "right hand", "left hand", "feet", "quiver", "body", "neck"}[i-sim.Shelves]
}

// Player describes the player: where it stands, its points, and what it holds and carries.
func Player(p *content.Pack, pl *sim.Player) []string {
	c := pl.Cell()
	lines := []string{
		fmt.Sprintf("Player at (%d, %d) in cell (%d, %d), facing %d", pl.U, pl.V, c[0], c[1], pl.Facing),
		fmt.Sprintf("  hit points %d/%d, magic points %d/%d", pl.HitPoints, pl.MaxHitPoints, pl.MagicPoints, pl.MaxMagicPoints),
	}
	if len(pl.Attributes) > 0 {
		a := pl.Attributes
		agility, _ := pl.Agility(p)
		lines = append(lines, fmt.Sprintf("  STR %d AGI %d (%d loaded) END %d ACC %d TAL %d POW %d",
			a[sim.AttrStrength], a[sim.AttrAgility], agility, a[sim.AttrEndurance], a[sim.AttrAccuracy], a[sim.AttrTalent], a[sim.AttrPower]))
	}
	lines = append(lines, fmt.Sprintf("  carrying %d, interval %+d ticks from the base", pl.Carried(p), pl.IntervalTicks(p)))
	lines = append(lines, fmt.Sprintf("  level %d, experience %d/%d, armor %d%s%s", pl.Level, pl.Experience, pl.NextLevel,
		pl.ArmorClass, onOff(pl.Poisoned, ", poisoned", ""), onOff(pl.Dead, ", DEAD", "")))
	var skills []string
	for i, s := range pl.Skills {
		if s.Level > 0 || s.Practice > 0 {
			skills = append(skills, fmt.Sprintf("%s %d (%d/%d)", SkillName(strconv.Itoa(i)), s.Level, s.Practice, s.Threshold))
		}
	}
	if len(skills) > 0 {
		lines = append(lines, "  skills: "+strings.Join(skills, ", "))
	}
	if st := statuses(pl); st != "" {
		lines = append(lines, "  "+st)
	}
	if pl.Delays != [2]int{} {
		lines = append(lines, fmt.Sprintf("  hands wait %d and %d", pl.Delays[0], pl.Delays[1]))
	}
	hand := "nothing"
	if pl.Hand != nil {
		hand = object(p, pl.Hand)
	}
	lines = append(lines, "  hand: "+hand)
	n := sim.SlotCount
	if bag := pl.Pack(p); bag != nil {
		n = sim.SlotPack + len(bag.Contents)
	}
	for i := range n {
		if o := pl.Slot(p, i); o != nil {
			lines = append(lines, fmt.Sprintf("  %s: %s", SlotName(i), object(p, o)))
		}
	}
	return lines
}

// statuses describes the player's statuses, timed effects, exertion and confusion.
func statuses(pl *sim.Player) string {
	var out []string
	data, _ := json.Marshal(pl.Status)
	var flags map[string]bool
	_ = json.Unmarshal(data, &flags)
	for _, k := range sortedKeys(flags) {
		out = append(out, k)
	}
	names := []string{"sight", "fire shield", "lightning shield", "battle rage", "haste", "invisibility", "weapon skills", "magic skills"}
	for i, n := range pl.Timed {
		if n > 0 {
			out = append(out, fmt.Sprintf("%s %d", names[i], n))
		}
	}
	if pl.Exertion > 0 || pl.Fatigue > 0 {
		out = append(out, fmt.Sprintf("exertion %d, fatigue %d", pl.Exertion, pl.Fatigue))
	}
	if pl.Confusion > 0 {
		out = append(out, fmt.Sprintf("confusion %d", pl.Confusion))
	}
	return strings.Join(out, ", ")
}

// behaviors describe the creature behaviors (docs/spec/creatures.md).
var behaviors = map[string]string{
	content.BehaviorHostile:         "hostile from the start",
	content.BehaviorNeverHostile:    "never turns hostile, even when attacked",
	content.BehaviorLeavesCorpse:    "leaves a corpse, which weighs on plates and can be pushed",
	content.BehaviorResistsTeleport: "not taken by teleporters, pits or open trapdoors",
	content.BehaviorInvulnerable:    "cannot be hurt, and never turns hostile",
	content.BehaviorPoisonous:       "its hits poison, 2 times in 8",
	content.BehaviorHurtsOnContact:  "does 1 damage when it walks into the player",
	content.BehaviorDeathBlow:       "its every attack kills the player outright, unless warded",
	content.BehaviorPhysicalOnly:    "hurt only by physical blows, and of weapons only by the Silver Sword",
	content.BehaviorResistsFire:     "fire does a quarter damage",
	content.BehaviorResistsLight:    "lightning does a quarter damage",
	content.BehaviorResistsPoison:   "poison does a quarter damage",
	content.BehaviorAnimated:        "an animated object: only animates in place, and cannot be struck",
	content.BehaviorStationary:      "stands still and casts",
	content.BehaviorTalks:           "can be talked to, until it turns hostile",
	content.BehaviorMirrors:         "passes the damage it takes to the player, and can be struck though it only animates",
}

// Behavior describes creature behavior b in words; empty for a behavior the engine does not know.
func Behavior(b string) string { return behaviors[b] }
