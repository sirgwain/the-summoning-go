package devpack

import "github.com/sirgwain/the-summoning-go/engine/content"

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
