package inspect

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// What the book says of its entries (docs/spec/player.md, The book), and the names it and the
// other windows give templates, levels, spells and weapon classes.

// Hints for what the next stage would show, by label key.
const (
	HintFight  = "ui.hint.fight"
	HintDefeat = "ui.hint.defeat"
	HintPickUp = "ui.hint.pick_up"
	HintUse    = "ui.hint.use"
	// HintUseOrThrow and HintThrow are for items that poison when thrown.
	HintUseOrThrow = "ui.hint.use_or_throw"
	HintThrow      = "ui.hint.throw"
	HintWear       = "ui.hint.wear"
)

// ThingName returns thing template id's name: its own, else its conversation's, else its id.
func ThingName(p *content.Pack, id string) string {
	if t := p.Things[id]; t != nil {
		if t.Name != "" {
			return Text(p, t.Name)
		}
		if c := p.Conversations[t.Conversation]; c != nil && c.Name != "" {
			return c.Name
		}
	}
	return id
}

// LevelName returns level id's name, or its id without one.
func LevelName(p *content.Pack, id string) string {
	if l := p.Levels[id]; l != nil && l.Name != "" {
		return Text(p, l.Name)
	}
	return id
}

// SpellName returns spell id's name, or its id without one.
func SpellName(p *content.Pack, id string) string {
	if sp := p.Spells[id]; sp != nil && sp.Name != "" {
		return Text(p, sp.Name)
	}
	return id
}

// WeaponClassName returns weapon class n's name: the creation record's, or else the name of the
// weapon the class starts with.
func WeaponClassName(p *content.Pack, n int) string {
	if c := p.Creation; c != nil && n >= 0 && n < len(c.WeaponClasses) {
		tr := c.WeaponClasses[n]
		if tr.Name != "" {
			return Text(p, tr.Name)
		}
		for _, o := range tr.Inventory {
			if it := p.Items[o.Item]; it != nil && it.WeaponClass != nil && *it.WeaponClass == n {
				return ItemName(p, o.Item)
			}
		}
	}
	return UI(p, "ui.weapon_class", "n", n)
}

// KindName returns damage kind id's name: the pack's text of it, else its label, else its id.
func KindName(p *content.Pack, id string) string {
	if id == "" {
		if kinds := p.CombatRules().Kinds; len(kinds) > 0 {
			id = kinds[0].ID
		}
	}
	for _, k := range p.CombatRules().Kinds {
		if k.ID == id && k.Name != "" {
			return Text(p, k.Name)
		}
	}
	if _, ok := Labels["ui.kind."+id]; ok {
		return UI(p, "ui.kind."+id)
	}
	if s, ok := packText(p, "ui.kind."+id); ok {
		return s
	}
	return id
}

// PlayTime returns step as play time: hours, minutes and seconds.
func PlayTime(step int64) string {
	s := step / sim.StepsPerSecond
	return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
}

// Label returns the pack's inventory label key in words, or else engine label ui.
func Label(p *content.Pack, key, ui string) string {
	if inv := p.Inventory; inv != nil {
		if k, ok := inv.Labels[key]; ok {
			return Text(p, k)
		}
	}
	return UI(p, ui)
}

// Weight returns a weight in half kilograms in words.
func Weight(halves int) string { return Kilograms(halves) + " kg." }

// Kilograms returns a weight in half kilograms as kilograms, without the unit: "12.5".
func Kilograms(halves int) string {
	if halves%2 != 0 {
		return fmt.Sprintf("%d.5", halves/2)
	}
	return fmt.Sprintf("%d.0", halves/2)
}

// BookPicture returns creature template t's picture: the first frame its animate shows, or the
// first of its idle animation for its starting facing.
func BookPicture(p *content.Pack, t *content.Thing) content.Image {
	img := content.Image{Sprite: t.Sprite}
	if t.Animate != nil && len(t.Animate.Frames) > 0 {
		img.Frame = t.Animate.Frames[0]
	} else if a, ok := p.Animation(t, fmt.Sprintf("idle.%d", t.Facing), "idle"); ok && len(a.Frames) > 0 {
		img.Frame = a.Frames[0]
	}
	return img
}

// firstSeen says where and when entry e was first come across; an entry the player has not come
// across, which only the full book shows, has no level.
func firstSeen(p *content.Pack, e *sim.Entry) string {
	if e.Level == "" {
		return UI(p, "ui.book.not_seen")
	}
	return UI(p, "ui.book.first_seen", "level", LevelName(p, e.Level), "time", PlayTime(e.Step))
}

// CreatureLines returns what the book says of creature template id, as the entry e stands, and
// the hint of what the next stage would show, none for a creature that can be talked to.
func CreatureLines(p *content.Pack, id string, e *sim.Entry) (lines []string, hint string) {
	t := p.Things[id]
	if t == nil || e == nil {
		return nil, ""
	}
	talks := t.Has(content.BehaviorTalks)
	next := func(h string) string {
		if talks {
			return ""
		}
		return UI(p, h)
	}
	if t.Lore != "" {
		lines = append(lines, Text(p, t.Lore))
	}
	if talks {
		lines = append(lines, UI(p, "ui.book.talks"))
	}
	lines = append(lines, firstSeen(p, e))
	if e.Stage < sim.StageFought {
		return lines, next(HintFight)
	}
	if t.Attack != (content.Dice{}) {
		lines = append(lines, UI(p, "ui.book.strikes", "dice", t.Attack))
	}
	for _, b := range []struct{ behavior, label string }{
		{content.BehaviorPoisonous, "ui.book.poisonous"},
		{content.BehaviorHurtsOnContact, "ui.book.hurts_on_contact"},
		{content.BehaviorDeathBlow, "ui.book.death_blow"},
		{content.BehaviorMirrors, "ui.book.mirrors"},
	} {
		if t.Has(b.behavior) {
			lines = append(lines, UI(p, b.label))
		}
	}
	if t.Spell != "" {
		lines = append(lines, UI(p, "ui.book.casts", "spell", SpellName(p, t.Spell)))
	}
	if s := t.Shoots; s != nil {
		lines = append(lines, UI(p, "ui.book.shoots", "dice", s.Damage, "kind", KindName(p, s.Kind)))
	}
	for _, d := range e.Defenses {
		if k, ok := strings.CutPrefix(d, sim.DefenseResists); ok {
			lines = append(lines, UI(p, "ui.book.resists", "kind", KindName(p, k)))
			continue
		}
		if k, ok := strings.CutPrefix(d, sim.DefenseIgnores); ok {
			lines = append(lines, UI(p, "ui.book.ignores", "kind", KindName(p, k)))
			continue
		}
		if k, ok := strings.CutPrefix(d, sim.DefenseWeak); ok {
			lines = append(lines, UI(p, "ui.book.weak", "kind", KindName(p, k)))
			continue
		}
		switch d {
		case sim.DefenseVulnerable:
			if t.Vulnerability != nil {
				lines = append(lines, UI(p, "ui.book.vulnerable", "class", WeaponClassName(p, *t.Vulnerability)))
			}
		case content.BehaviorResistsFire:
			lines = append(lines, UI(p, "ui.book.resists_fire"))
		case content.BehaviorResistsLight:
			lines = append(lines, UI(p, "ui.book.resists_light"))
		case content.BehaviorResistsPoison:
			lines = append(lines, UI(p, "ui.book.resists_poison"))
		case content.BehaviorResistsTeleport:
			lines = append(lines, UI(p, "ui.book.resists_teleport"))
		case content.BehaviorPhysicalOnly:
			lines = append(lines, UI(p, "ui.book.physical_only"))
		}
	}
	if e.Stage < sim.StageKilled {
		return lines, next(HintDefeat)
	}
	if t.HitPoints != (content.Dice{}) {
		lines = append(lines, UI(p, "ui.book.hit_points", "dice", t.HitPoints))
	}
	if t.Experience != (content.Dice{}) {
		lines = append(lines, UI(p, "ui.book.worth", "dice", t.Experience))
	}
	lines = append(lines, UI(p, "ui.book.defeated", "n", e.Kills))
	if len(e.Drops) > 0 {
		names := make([]string, len(e.Drops))
		for i, d := range e.Drops {
			names[i] = ItemName(p, d)
		}
		lines = append(lines, UI(p, "ui.book.drops", "list", strings.Join(names, ", ")))
	}
	return lines, ""
}

// wornOn is the label saying where an item of wear w is worn, none for a hand.
var wornOn = map[string]string{
	content.WearHead: "ui.book.worn.head", content.WearBody: "ui.book.worn.body", content.WearFeet: "ui.book.worn.feet",
	content.WearArms: "ui.book.worn.arms", content.WearNeck: "ui.book.worn.neck", content.WearQuiver: "ui.book.worn.quiver",
}

// ItemLines returns what the book says of item id, as the entry e stands, with the recipes it
// takes part in, and the hint of what the next stage would show.
func ItemLines(p *content.Pack, id string, e *sim.Entry, recipes ...sim.Recipe) (lines []string, hint string) {
	it := p.Items[id]
	if it == nil || e == nil {
		return nil, ""
	}
	if it.Lore != "" {
		lines = append(lines, Text(p, it.Lore))
	}
	lines = append(lines, firstSeen(p, e))
	lines = append(lines, RecipeLines(p, id, recipes)...)
	if it.Teaches != "" {
		return append(lines, scrollLines(p, it.Teaches)...), ""
	}
	if e.Stage < sim.StageHandled {
		return lines, UI(p, HintPickUp)
	}
	lines = append(lines, Label(p, content.LabelWeight, "ui.weight")+Weight(it.Weight))
	lines = append(lines, strikeLines(p, it)...)
	if it.Armor > 0 {
		lines = append(lines, Label(p, content.LabelArmor, "ui.armor_bonus")+fmt.Sprint(it.Armor))
	}
	if s := wornOn[it.Wear]; s != "" {
		lines = append(lines, UI(p, s))
	}
	if slices.Contains(it.Tags, "two_handed") {
		lines = append(lines, Label(p, content.LabelTwoHanded, "ui.two_handed"))
	}
	if c := it.Container; c != nil {
		lines = append(lines, UI(p, "ui.book.holds", "n", c.Capacity, "m", c.Largest))
		if c.Weightless {
			lines = append(lines, UI(p, "ui.book.weightless"))
		}
	}
	if it.Charges > 1 {
		lines = append(lines, UI(p, "ui.book.charges", "n", it.Charges))
	}
	if e.Stage < sim.StageUsed {
		if h := useHint(it); h != "" {
			return lines, UI(p, h)
		}
		return lines, ""
	}
	if it.Effect != "" {
		lines = append(lines, Text(p, it.Effect))
	}
	if it.Casts != nil {
		lines = append(lines, UI(p, "ui.book.casts", "spell", SpellName(p, it.Casts.Spell)))
	}
	if it.PoisonsThrown {
		lines = append(lines, UI(p, "ui.book.thrown_poisons"))
	}
	return lines, ""
}

// RecipeLines say what the recipes made tell of item id: what it makes, what makes it, and what
// fills it (docs/spec/player.md, The book).
func RecipeLines(p *content.Pack, id string, recipes []sim.Recipe) []string {
	var lines []string
	for _, r := range recipes {
		spell, holder, into := SpellName(p, r.Spell), ItemName(p, r.Holder), ItemName(p, r.Into)
		from := ""
		if r.From != "" {
			from = ItemName(p, r.From)
		}
		args := []any{"spell", spell, "holder", article(holder), "into", into, "from", from}
		switch id {
		case r.From:
			lines = append(lines, UI(p, "ui.book.makes", args...))
		case r.Into:
			if r.From == "" {
				lines = append(lines, UI(p, "ui.book.made_any", args...))
			} else {
				lines = append(lines, UI(p, "ui.book.made", args...))
			}
		case r.Holder:
			if r.From == "" {
				lines = append(lines, UI(p, "ui.book.fills_any", args...))
			} else {
				lines = append(lines, UI(p, "ui.book.fills", args...))
			}
		}
	}
	return lines
}

// article puts "a" or "an" before name.
func article(name string) string {
	if name != "" && strings.ContainsRune("AEIOUaeiou", rune(name[0])) {
		return "an " + name
	}
	return "a " + name
}

// useHint returns the hint for an item not yet used: to wear one whose power starts worn and that
// has no use action, to use one with a power, spell or conversation, and none for anything else.
func useHint(it *content.Item) string {
	usable := it.Power != 0 || it.Casts != nil || it.Conversation != ""
	switch {
	case it.PoisonsThrown && usable:
		return HintUseOrThrow
	case it.PoisonsThrown:
		return HintThrow
	case !usable:
		return ""
	}
	used := slices.ContainsFunc(it.Actions, func(a content.ItemAction) bool {
		return a.Kind == content.ActionUse || a.Kind == content.ActionUseUp
	})
	if it.PowerWorn && !used {
		return HintWear
	}
	return HintUse
}

// scrollLines describe what a scroll teaching gesture teaches: the spell with that gesture, by
// name and school, its cost and what it does; or, for no spell's, the gesture's icons.
func scrollLines(p *content.Pack, gesture string) []string {
	for _, id := range sortedKeys(p.Spells) {
		sp := p.Spells[id]
		if sp.Gesture != gesture {
			continue
		}
		teaches := UI(p, "ui.book.teaches", "spell", SpellName(p, id))
		if m := p.Magic; m != nil && sp.School >= 0 && sp.School < len(m.Schools) {
			teaches = UI(p, "ui.book.teaches_school", "spell", SpellName(p, id), "school", Text(p, m.Schools[sp.School]))
		}
		lines := []string{teaches, UI(p, "ui.book.costs", "n", sp.Cost)}
		if sp.Effect != "" {
			lines = append(lines, Text(p, sp.Effect))
		}
		return lines
	}
	return []string{UI(p, "ui.book.gesture", "icons", strings.Join(strings.Split(gesture, ""), " "))}
}

// strikeLines describe item it's strike actions as the hotbar's boxes number them: a single one
// by its damage; with two, each with its damage, and the second whether it is slower and the
// skill it needs.
func strikeLines(p *content.Pack, it *content.Item) []string {
	var strikes []content.ItemAction
	for _, a := range it.Actions {
		if a.Attack() && a.Damage != (content.Dice{}) {
			strikes = append(strikes, a)
		}
	}
	if len(strikes) == 1 {
		return []string{UI(p, "ui.book.strikes", "dice", strikes[0].Damage)}
	}
	var out []string
	for i, a := range strikes {
		s := UI(p, "ui.book.strike_n", "n", i+1, "dice", a.Damage)
		if i > 0 && a.Delay > strikes[0].Delay {
			s += UI(p, "ui.book.slower")
		}
		if a.Skill > 0 {
			s += UI(p, "ui.book.needs_skill", "n", a.Skill)
		}
		out = append(out, s+".")
	}
	return out
}

// FullEntry returns the book's entry e as the full book shows it: at the last stage, with every
// defense of creature template t learned, keeping what e has come across.
func FullEntry(e *sim.Entry, t *content.Thing) *sim.Entry {
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
	for _, k := range t.Resists {
		defenses = append(defenses, sim.DefenseResists+k)
	}
	for _, k := range t.Ignores {
		defenses = append(defenses, sim.DefenseIgnores+k)
	}
	if t.Vulnerability == nil {
		for _, k := range t.WeakTo {
			defenses = append(defenses, sim.DefenseWeak+k)
		}
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
