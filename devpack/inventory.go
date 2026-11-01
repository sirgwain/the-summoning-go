package devpack

import (
	"fmt"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The dev pack's inventory: a plain figure, how its boots, mail and sword look on it, the action
// icons, and the skills' ratings and the levels' titles.

const (
	spriteDoll        = "sprite.doll"
	spriteDollParts   = "sprite.doll.parts"
	spriteActionIcons = "sprite.action_icons"
	spriteBareHands   = "sprite.bare_hands"
)

// Doll parts, frames of the parts sprite, each drawn at its place on the 96×128 figure.
const (
	partBoots = iota
	partMail
	partSword
	partQuiver
)

var partAt = map[int]content.Offset{partBoots: {28, 104}, partMail: {26, 36}, partSword: {6, 40}, partQuiver: {2, 2}}

// Action icons, frames of the action icons sprite.
const (
	iconThrow = iota
	iconStrike
	iconSlash
	iconThrust
	iconDrink
)

func inventoryArt(p *content.Pack) {
	bank(p, spriteDoll, []*canvas{doll()}, nil)
	bank(p, spriteDollParts, dollParts(), nil)
	bank(p, spriteActionIcons, actionIcons(), nil)
	bank(p, spriteBareHands, []*canvas{bareHand(true), bareHand(false)}, nil)
}

// doll draws the figure: a gray body standing, arms down.
func doll() *canvas {
	c := newCanvas(96, 128, clear)
	c.fill(ellipse(48, 18, 11, 13), solid(gray+9))
	c.rect(42, 30, 54, 36, gray+8)
	c.fill(ellipse(48, 56, 20, 24), solid(gray+8))
	c.rect(22, 38, 30, 84, gray+7)
	c.rect(66, 38, 74, 84, gray+7)
	c.rect(36, 76, 46, 120, gray+7)
	c.rect(50, 76, 60, 120, gray+7)
	return c
}

// dollParts draws the worn looks, each in a 40×40 frame from its top-left.
func dollParts() []*canvas {
	boots := newCanvas(40, 40, clear)
	boots.rect(6, 4, 18, 18, wood+2)
	boots.rect(22, 4, 34, 18, wood+2)
	mail := newCanvas(40, 40, clear)
	mail.fill(ellipse(22, 20, 19, 19), func(x, y int) byte { return byte(gray + 10 + (x+y)%3) })
	sword := newCanvas(40, 40, clear)
	sword.line(8, 38, 8, 2, gray+14)
	sword.rect(3, 30, 13, 32, wood+4)
	quiver := newCanvas(40, 40, clear)
	quiver.rect(6, 4, 14, 30, wood+3)
	return []*canvas{boots, mail, sword, quiver}
}

// actionIcons draws the action icons, 16 pixels square: an arrow out for throwing, a fist, a
// slash, a thrust and a drop.
func actionIcons() []*canvas {
	var out []*canvas
	for i := range 5 {
		c := newCanvas(16, 16, clear)
		switch i {
		case iconThrow:
			c.line(2, 13, 13, 2, gold+6)
			c.line(9, 2, 13, 2, gold+6)
			c.line(13, 2, 13, 6, gold+6)
		case iconStrike:
			c.fill(ellipse(8, 9, 5, 4), solid(tan+5))
		case iconSlash:
			c.line(2, 2, 13, 13, gray+15)
			c.line(3, 2, 14, 13, gray+12)
		case iconThrust:
			c.line(1, 8, 14, 8, gray+15)
			c.line(11, 5, 14, 8, gray+15)
			c.line(11, 11, 14, 8, gray+15)
		case iconDrink:
			c.fill(ellipse(8, 10, 4, 5), solid(red+5))
		}
		out = append(out, c)
	}
	return out
}

// bareHand draws an empty hand, its thumb toward the figure's middle.
func bareHand(right bool) *canvas {
	c := newCanvas(16, 16, clear)
	c.fill(ellipse(8, 9, 5, 5), solid(tan+4))
	x := 3
	if right {
		x = 12
	}
	c.rect(x, 5, x+1, 9, tan+5)
	return c
}

// inventoryRecords adds the inventory record, the items' worn looks and their action icons.
func inventoryRecords(p *content.Pack) {
	s := p.Texts[textID].Strings
	key := func(k, text string) string { s[k] = text; return k }
	in := &content.Inventory{ID: "inventory",
		Doll:       &content.Image{Sprite: spriteDoll},
		QuiverSlot: &content.DollImage{Image: content.Image{Sprite: spriteDollParts, Frame: partQuiver}, At: partAt[partQuiver]},
		BareHands:  []content.Image{{Sprite: spriteBareHands}, {Sprite: spriteBareHands, Frame: 1}},
		StrikeIcon: &content.Image{Sprite: spriteActionIcons, Frame: iconStrike},
		ThrowIcon:  &content.Image{Sprite: spriteActionIcons, Frame: iconThrow},
		Labels: map[string]string{
			content.LabelWeight: key("label.weight", "Weight: "), content.LabelDamage: key("label.damage", "Can do "),
			content.LabelDamageAfter: key("label.damage_after", " damage"), content.LabelArmor: key("label.armor", "Armor "),
			content.LabelTwoHanded: key("label.two_handed", "Needs both hands"),
		},
	}
	for i, r := range []string{"Beginner", "Novice", "Adept", "Expert", "Master"} {
		in.WeaponRatings = append(in.WeaponRatings, key(fmt.Sprintf("rating.weapon.%d", i+1), r))
		in.SchoolRatings = append(in.SchoolRatings, key(fmt.Sprintf("rating.school.%d", i+1), r+" mage"))
	}
	for i, t := range []string{"Wanderer", "Scout", "Ranger", "Hero"} {
		in.LevelTitles = append(in.LevelTitles, key(fmt.Sprintf("title.%d", i), t))
	}
	p.Inventory = in

	dollLook := func(part int) *content.DollImage {
		return &content.DollImage{Image: content.Image{Sprite: spriteDollParts, Frame: part}, At: partAt[part]}
	}
	p.Items[ItemBoots].Doll = dollLook(partBoots)
	p.Items[ItemMail].Doll = dollLook(partMail)
	p.Items[ItemSword].Doll = dollLook(partSword)
	icon := func(f int) *content.Image { return &content.Image{Sprite: spriteActionIcons, Frame: f} }
	p.Items[ItemSword].Actions[0].Icon = icon(iconSlash)
	p.Items[ItemSword].Actions[1].Icon = icon(iconThrust)
	p.Items[ItemPotion].Actions[0].Icon = icon(iconDrink)
}
