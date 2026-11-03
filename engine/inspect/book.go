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

// Hints for what the next stage would show.
const (
	HintFight  = "Fight one to learn more."
	HintDefeat = "Defeat one to learn more."
	HintPickUp = "Pick one up to learn more."
	HintUse    = "Use one to learn more."
	HintWear   = "Wear one to learn more."
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
	return fmt.Sprintf("weapon class %d", n)
}

// PlayTime returns step as play time: hours, minutes and seconds.
func PlayTime(step int64) string {
	s := step / sim.StepsPerSecond
	return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
}

// Label returns the pack's inventory label key in words, or fallback when the pack has none.
func Label(p *content.Pack, key, fallback string) string {
	if inv := p.Inventory; inv != nil {
		if k, ok := inv.Labels[key]; ok {
			return Text(p, k)
		}
	}
	return fallback
}

// Weight returns a weight in half kilograms in words.
func Weight(halves int) string {
	if halves%2 != 0 {
		return fmt.Sprintf("%d.5 kg.", halves/2)
	}
	return fmt.Sprintf("%d.0 kg.", halves/2)
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
		return "Not come across yet."
	}
	return "First seen: " + LevelName(p, e.Level) + ", " + PlayTime(e.Step)
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
		return h
	}
	if t.Lore != "" {
		lines = append(lines, Text(p, t.Lore))
	}
	if talks {
		lines = append(lines, "Can be talked to.")
	}
	lines = append(lines, firstSeen(p, e))
	if e.Stage < sim.StageFought {
		return lines, next(HintFight)
	}
	if t.Attack != (content.Dice{}) {
		lines = append(lines, "Strikes for "+t.Attack.String()+".")
	}
	for _, b := range []struct{ behavior, text string }{
		{content.BehaviorPoisonous, "Its blows poison."},
		{content.BehaviorHurtsOnContact, "Touching it hurts."},
		{content.BehaviorDeathBlow, "One blow from it can kill."},
		{content.BehaviorMirrors, "Hurting it hurts you as much."},
	} {
		if t.Has(b.behavior) {
			lines = append(lines, b.text)
		}
	}
	if t.Spell != "" {
		lines = append(lines, "Casts "+SpellName(p, t.Spell)+".")
	}
	for _, d := range e.Defenses {
		switch d {
		case sim.DefenseVulnerable:
			if t.Vulnerability != nil {
				lines = append(lines, "Vulnerable to: "+WeaponClassName(p, *t.Vulnerability))
			}
		case content.BehaviorResistsFire:
			lines = append(lines, "Resists fire.")
		case content.BehaviorResistsLight:
			lines = append(lines, "Resists lightning.")
		case content.BehaviorResistsPoison:
			lines = append(lines, "Resists poison.")
		case content.BehaviorResistsTeleport:
			lines = append(lines, "Resists teleporting.")
		case content.BehaviorPhysicalOnly:
			lines = append(lines, "Magic does not hurt it, and of weapons only silver does.")
		}
	}
	if e.Stage < sim.StageKilled {
		return lines, next(HintDefeat)
	}
	if t.HitPoints != (content.Dice{}) {
		lines = append(lines, "Hit points: "+t.HitPoints.String())
	}
	if t.Experience != (content.Dice{}) {
		lines = append(lines, "Worth "+t.Experience.String()+" experience.")
	}
	lines = append(lines, fmt.Sprintf("Defeated: %d", e.Kills))
	if len(e.Drops) > 0 {
		names := make([]string, len(e.Drops))
		for i, d := range e.Drops {
			names[i] = ItemName(p, d)
		}
		lines = append(lines, "Drops: "+strings.Join(names, ", "))
	}
	return lines, ""
}

// wornOn says where an item of wear w is worn, "" for a hand.
var wornOn = map[string]string{
	content.WearHead: "Worn on the head.", content.WearBody: "Worn on the body.", content.WearFeet: "Worn on the feet.",
	content.WearArms: "Worn on the arms.", content.WearNeck: "Worn about the neck.", content.WearQuiver: "Worn as a quiver.",
}

// ItemLines returns what the book says of item id, as the entry e stands, and the hint of what
// the next stage would show.
func ItemLines(p *content.Pack, id string, e *sim.Entry) (lines []string, hint string) {
	it := p.Items[id]
	if it == nil || e == nil {
		return nil, ""
	}
	if it.Lore != "" {
		lines = append(lines, Text(p, it.Lore))
	}
	lines = append(lines, firstSeen(p, e))
	if it.Teaches != "" {
		return append(lines, scrollLines(p, it.Teaches)...), ""
	}
	if e.Stage < sim.StageHandled {
		return lines, HintPickUp
	}
	lines = append(lines, Label(p, content.LabelWeight, "Weight: ")+Weight(it.Weight))
	lines = append(lines, strikeLines(it)...)
	if it.Armor > 0 {
		lines = append(lines, Label(p, content.LabelArmor, "AC bonus of ")+fmt.Sprint(it.Armor))
	}
	if s := wornOn[it.Wear]; s != "" {
		lines = append(lines, s)
	}
	if slices.Contains(it.Tags, "two_handed") {
		lines = append(lines, Label(p, content.LabelTwoHanded, "Requires two hands"))
	}
	if c := it.Container; c != nil {
		lines = append(lines, fmt.Sprintf("Holds %d objects, of size up to %d.", c.Capacity, c.Largest))
		if c.Weightless {
			lines = append(lines, "What it holds weighs nothing.")
		}
	}
	if it.Charges > 1 {
		lines = append(lines, fmt.Sprintf("Charges: %d", it.Charges))
	}
	if e.Stage < sim.StageUsed {
		return lines, useHint(it)
	}
	if it.Effect != "" {
		lines = append(lines, Text(p, it.Effect))
	}
	if it.Casts != nil {
		lines = append(lines, "Casts "+SpellName(p, it.Casts.Spell)+".")
	}
	return lines, ""
}

// useHint returns the hint for an item not yet used: to wear one whose power starts worn and that
// has no use action, to use one with a power, spell or conversation, and none for anything else.
func useHint(it *content.Item) string {
	if it.Power == 0 && it.Casts == nil && it.Conversation == "" {
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
		teaches := "Teaches " + SpellName(p, id)
		if m := p.Magic; m != nil && sp.School >= 0 && sp.School < len(m.Schools) {
			teaches += " (" + Text(p, m.Schools[sp.School]) + ")"
		}
		lines := []string{teaches + ".", fmt.Sprintf("Costs %d magic points.", sp.Cost)}
		if sp.Effect != "" {
			lines = append(lines, Text(p, sp.Effect))
		}
		return lines
	}
	return []string{"Teaches the gesture " + strings.Join(strings.Split(gesture, ""), " ") + "."}
}

// strikeLines describe item it's strike actions as the hotbar's boxes number them: a single one
// by its damage; with two, each with its damage, and the second whether it is slower and the
// skill it needs.
func strikeLines(it *content.Item) []string {
	var strikes []content.ItemAction
	for _, a := range it.Actions {
		if a.Attack() && a.Damage != (content.Dice{}) {
			strikes = append(strikes, a)
		}
	}
	if len(strikes) == 1 {
		return []string{"Strikes for " + strikes[0].Damage.String() + "."}
	}
	var out []string
	for i, a := range strikes {
		s := fmt.Sprintf("Strike %d: %s damage", i+1, a.Damage)
		if i > 0 && a.Delay > strikes[0].Delay {
			s += ", slower"
		}
		if a.Skill > 0 {
			s += fmt.Sprintf(", needs skill %d", a.Skill)
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
	for _, b := range []string{content.BehaviorResistsFire, content.BehaviorResistsLight, content.BehaviorResistsPoison,
		content.BehaviorResistsTeleport, content.BehaviorPhysicalOnly} {
		if t.Has(b) {
			defenses = append(defenses, b)
		}
	}
	full.Defenses = defenses
	return &full
}
