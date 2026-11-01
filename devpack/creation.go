package devpack

import (
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// creation adds the character creation record: the player's portrait, the three attribute
// methods, a sword fighter and a fire school, and one premade character.
func creation(p *content.Pack) {
	sword := []content.InventoryObject{{Slot: 22, Object: content.Object{Item: ItemSword}}}
	p.Creation = &content.Creation{ID: "creation",
		Portraits: []string{PortraitPlayer},
		Methods: []content.AttributeMethod{
			{Name: "Rolled", Roll: []content.Dice{{Count: 3, Sides: 4}, {Count: 2, Sides: 5}}, Max: 20},
			{Name: "Spent", Points: []content.Dice{{Count: 28, Sides: 5}}, Max: 20},
			{Name: "Rolled and spent", Roll: []content.Dice{{Count: 2, Sides: 5}}, Points: []content.Dice{{Count: 14, Sides: 5}}, Max: 20},
		},
		WeaponClasses: []content.Training{{Name: "msg.class.sword", Inventory: sword}},
		Schools: []content.Training{{Name: "msg.school.fire", Icons: "BDJ",
			Inventory: []content.InventoryObject{{Slot: 2, Object: content.Object{Item: ItemScroll}}}}},
		Inventory: []content.InventoryObject{{Slot: 0, Object: content.Object{Item: ItemPotion}},
			{Slot: 24, Object: content.Object{Item: ItemBoots}}},
		BodyArmor: []string{ItemMail},
		Premade: []content.Premade{{Name: "Arden", Portrait: PortraitPlayer, Attributes: map[string]int{
			"strength": 18, "agility": 20, "endurance": 18, "accuracy": 16, "talent": 16, "power": 16}}},
	}
}

// ThingMenu holds the disk menu's conversation, opened with the menu key.
const ThingMenu = "thing.menu"

// menu adds the disk menu: a conversation offering each of the game's menus by keyword.
func menu(p *content.Pack) {
	s := p.Texts[textID].Strings
	str := func(v string) *string { return &v }
	s["talk.menu"] = "The disk menu. *Save, *restore, *delete or *rename a game, *leave for the main menu, or *quit."
	conv := &content.Conversation{ID: "conversation.menu", Name: "Menu", Clauses: []content.Clause{
		{If: content.Condition{Greeting: true}, Do: []content.Action{{Say: str("talk.menu")}}},
	}}
	for _, m := range []string{content.MenuSave, content.MenuRestore, content.MenuDelete, content.MenuRename, content.MenuLeave, content.MenuQuit} {
		id := keywordID(m)
		s[id+".word"], s[id+".answer"] = m, strings.ToUpper(m[:1])+m[1:]+"."
		p.Keywords.Keywords = append(p.Keywords.Keywords, content.Keyword{ID: id, Word: id + ".word", Answer: id + ".answer"})
		conv.Clauses[0].Do = append(conv.Clauses[0].Do, content.Action{Offer: str(id)})
		conv.Clauses = append(conv.Clauses, content.Clause{If: content.Condition{Asked: []string{id}},
			Do: []content.Action{{Menu: str(m)}, {End: true}}})
	}
	p.Conversations[conv.ID] = conv
	t := *p.Things[ThingSage]
	t.ID, t.Conversation = ThingMenu, conv.ID
	p.Things[t.ID] = &t
	p.Manifest.Menu = t.ID
}
