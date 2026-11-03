package summoning

import (
	"fmt"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The book's effect texts (player.md, The book): what each power does, in the importer's own
// words, for the items whose power is not a spell or a conversation.

// attributeWords are the attributes 0–5 as the effect texts name them.
var attributeWords = [6]string{"strength", "agility", "endurance", "accuracy", "talent", "power"}

// powerTexts gives every item whose power the table describes its effect text.
func (im *importer) powerTexts() {
	o := im.objects
	name := func(t int) string {
		if t < 0 || t >= len(o.Templates) {
			return ""
		}
		return im.strings[msgID(o.NameBase+o.Templates[t].NameIndex())]
	}
	var classes, schools [4]string
	for i, t := range weaponTemplates {
		classes[i] = name(t)
	}
	for i := range schools {
		schools[i] = im.strings[msgID(msgFirstSchool+i)]
	}
	for i := range o.Templates {
		it := im.pack.Items[im.itemIDs[i]]
		if it == nil || it.Casts != nil || it.Conversation != "" {
			continue
		}
		p, worn := o.Templates[i].Power()
		text := powerEffect(p, classes, schools)
		if text == "" {
			continue
		}
		key := fmt.Sprintf("effect.%02x", p)
		if worn && content.PowerLasts(p) {
			key += ".worn"
			text = strings.TrimSuffix(text, ".") + " while worn."
		}
		im.strings[key] = text
		it.Effect = key
	}
}

// powerEffect says what power p does, "" for none to tell: classes and schools are the weapon
// classes' and schools' names.
func powerEffect(p int, classes, schools [4]string) string {
	skill := func(name string, n int) string { return fmt.Sprintf("Raises %s skill by %d.", name, n) }
	weapon := func(name string, n int) string {
		if name == "" {
			return fmt.Sprintf("Raises a weapon skill by %d.", n)
		}
		return fmt.Sprintf("Raises skill with weapons like the %s by %d.", name, n)
	}
	switch {
	case p >= 0x05 && p <= 0x08:
		return skill(schools[p-0x05], 1)
	case p >= 0x2A && p <= 0x2F:
		return fmt.Sprintf("Raises %s by 2 for a while.", attributeWords[p-0x2A])
	case p >= 0x30 && p <= 0x41:
		n := p - 0x30
		return fmt.Sprintf("Raises %s by %d.", attributeWords[n/3], n%3+1)
	case p >= 0x48 && p <= 0x4B:
		return weapon(classes[p-0x48], 1)
	}
	switch p {
	case 0x01:
		return "Restores 2d6 magic points."
	case 0x02:
		return "Restores 4d6 magic points."
	case 0x03:
		return "Restores all magic points."
	case 0x45:
		return "Heals every wound."
	case 0x04:
		return "Confuses and hurts whoever drinks it (2d8)."
	case 0x09:
		return "Makes you faster (speed +1)."
	case 0x0A, 0x0B:
		return "Makes you faster (speed +2)."
	case 0x0C:
		return "Turns aside a quarter of all harm."
	case 0x0D:
		return "Turns aside two fifths of all harm."
	case 0x0E:
		return "Guards against creatures' blows."
	case 0x0F:
		return "Lifts you above pits, plates and trapdoors."
	case 0x10:
		return weapon(classes[3], 3)
	case 0x11, 0x1A, 0x26:
		return "Takes you to a rune mark on the level."
	case 0x12:
		return "Smites every creature near you."
	case 0x14:
		return "Changes you at random: your sex, or one of your attributes."
	case 0x15:
		return "Cures poison and confusion."
	case 0x17:
		return "Heals you slowly."
	case 0x1B:
		return "Turns into another object."
	case 0x1C:
		return "Shields you from fire for a while."
	case 0x1D:
		return "Heals 6d6 hit points."
	case 0x1F:
		return weapon(classes[1], 2)
	case 0x20:
		return "Raises a school's skill, at random, by 1."
	case 0x21:
		return "Holds every other creature still for a while."
	case 0x22:
		return "Shows the map of the level."
	case 0x23:
		return "Lets the objects in your hands soak up magical harm."
	case 0x24:
		return "Cures poison."
	case 0x25:
		return "Raises a weapon skill, at random, by 1."
	case 0x27:
		return "Shields you from lightning for a while."
	case 0x28:
		return "Heals 5d10 hit points."
	case 0x43:
		return "Makes your blows truer."
	case 0x44:
		return "Eases fatigue."
	case 0x46:
		return "Heals 20d4 hit points."
	case 0x47:
		return "Makes you invisible."
	case 0x4C:
		return "Restores magic points slowly."
	case 0x4D:
		return "Brings you back from death, once."
	case 0x4F:
		return "Makes you proof against fire, and faster (speed +2)."
	case 0x50:
		return "Confuses whoever drinks it."
	case 0x5B:
		return "Wards off death blows."
	}
	return ""
}

// spellEffects say what each spell does, by number, for the book (spells.md, Effects).
var spellEffects = map[int]string{
	0:  "Shoots a bolt of fire.",
	1:  "Hurls fireballs four ways at once.",
	2:  "Hurls a fireball that bounces off walls.",
	3:  "Shoots a cloud of poison.",
	4:  "Hurls a ball of fire.",
	5:  "Shoots a bolt of lightning.",
	6:  "Makes you stronger and tougher, but clumsier, for a while.",
	9:  "Fires an arrow.",
	10: "Lets you see objects on the floor more clearly for a while.",
	11: "Sets a boulder rolling, or opens a door that magic can open.",
	12: "Raises a wall ahead of you for a while.",
	13: "Makes you faster for a while.",
	14: "Sends what it hits somewhere else on the level.",
	15: "Takes you to a rune mark on the level.",
	16: "Makes you invisible for a while.",
	17: "Ends the spells flying near you.",
	18: "Opens a portal to another place.",
	19: "Joins the two halves of a staff held in your hands.",
	20: "Holds every other creature still for a while.",
	21: "Shields you from lightning for a while.",
	22: "Shields you from fire for a while.",
	23: "Raises every weapon skill for a while.",
	24: "Raises every school's skill for a while.",
	25: "Gives you the look of a creature near you.",
	26: "Destroys one kind of creature, and does nothing to others.",
	30: "With an empty bottle in one hand, makes a potion of what the other hand holds.",
	31: "Cures poison.",
	32: "Eases fatigue.",
	33: "Heals every wound.",
}
