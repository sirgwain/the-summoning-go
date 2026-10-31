// Package devpack builds the dev pack: a small content pack made entirely by code, with plain
// drawn art and levels that lay out each of the engine's mechanisms. Tests run on it, and the
// viewer can open it (summoning devpack), so nothing needs the original game's data
// (docs/design.md, Testing).
//
// Its art is drawn to read clearly, not to look like any game: gray stone blocks for walls,
// wooden blocks for doors, and plain shapes for features, items and figures.
package devpack

import (
	"fmt"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

const paletteID = "palette.dev"

// Build returns the dev pack. It is the same on every call.
func Build() *content.Pack {
	p := content.NewPack(content.Manifest{
		ID: "dev", Name: "Dev pack", Version: "0.1.0", Format: content.Format, Depends: []string{},
		Palette: paletteID, Start: content.Start{Level: Hall, Cell: HallStart}, Player: ThingPlayer,
	})
	p.Palettes[paletteID] = palette()
	art(p)
	text(p)
	records(p)
	hall(p)
	cellar(p)
	return p
}

// Sprite, set and look ids.
const (
	spriteFloors   = "sprite.floors"
	spriteWalls    = "sprite.walls"
	spriteDoors    = "sprite.doors"
	spriteFeatures = "sprite.features"
	spriteIcons    = "sprite.items.icons"
	spriteFloor    = "sprite.items.floor"
	spriteBoulder  = "sprite.boulder"
	spriteCrate    = "sprite.crate"

	floorSet = "floors.dev"
	wallSet  = "walls.dev"

	lookDoor       = "look.door"
	lookLever      = "look.lever"
	lookPlate      = "look.plate"
	lookSlot       = "look.slot"
	lookTeleporter = "look.teleporter"
	lookPit        = "look.pit"
	lookTrapdoor   = "look.trapdoor"
	lookSocket     = "look.socket"
	lookMouth      = "look.mouth"
	lookMark       = "look.mark"
	lookExitFloor  = "look.exit.floor"
	lookExitLeft   = "look.exit.left"
	lookExitRight  = "look.exit.right"
)

// figures are the figure sprites, by name, with their body and skin colors.
var figures = []struct {
	name       string
	body, skin byte
}{
	{"guard", blue + 1, tan + 5},
	{"ghost", purple + 3, gray + 13},
	{"player", green + 1, tan + 5},
}

func art(p *content.Pack) {
	var floors []*canvas
	for v := range 4 {
		floors = append(floors, floorTile(v))
	}
	bank(p, spriteFloors, floors, nil)
	p.Floors[floorSet] = &content.FloorSet{ID: floorSet, Sprite: spriteFloors, Variants: len(floors), Shape: content.FloorDiamond}

	bank(p, spriteWalls, walls(), nil)
	p.Walls[wallSet] = &content.WallSet{ID: wallSet, Sprite: spriteWalls, Offset: blockOffset, Shapes: []*content.WallShape{
		shapeBlock:     {Frame: shapeBlock},
		shapeFaceLeft:  {Frame: shapeFaceLeft, Face: true},
		shapeDiagonal:  {Frame: shapeDiagonal, Diagonal: content.CornerBottom},
		shapeFaceRight: {Frame: shapeFaceRight, Face: true},
	}}
	bank(p, spriteDoors, doors(), nil)
	bank(p, spriteFeatures, features(), nil)
	bank(p, spriteIcons, items(16), nil)
	bank(p, spriteFloor, items(14), nil)
	for _, f := range figures {
		b := bank(p, "sprite."+f.name, figure(f.body, f.skin), &figureAnchor)
		b.Animations = map[string]content.Animation{"idle": {Frames: []int{2}}}
		for d := range 8 {
			b.Animations[fmt.Sprintf("idle.%d", d)] = content.Animation{Frames: []int{d}}
			b.Animations[fmt.Sprintf("walk.%d", d)] = content.Animation{Frames: []int{8 + d, d, 16 + d, d}, MS: content.Fixed(275)}
		}
	}
	b := bank(p, spriteBoulder, boulders(), &content.Offset{16, 29})
	b.Animations = map[string]content.Animation{}
	for d := 0; d < 8; d += 2 {
		frames := []int{0, 1, 2, 3}
		if d >= 4 {
			frames = []int{3, 2, 1, 0}
		}
		b.Animations[fmt.Sprintf("roll.%d", d)] = content.Animation{Frames: frames, MS: content.Fixed(110)}
	}
	bank(p, spriteCrate, []*canvas{crate()}, &content.Offset{20, 30})

	part := func(sprite string, frame int, off content.Offset) []content.LookPart {
		return []content.LookPart{{Sprite: sprite, Frame: frame, Offset: off}}
	}
	feature := func(frame int) []content.LookPart { return part(spriteFeatures, frame, featureOffset) }
	look := func(id, layer string, states map[string][]content.LookPart) {
		p.Looks[id] = &content.Look{ID: id, Layer: layer, States: states}
	}
	door := map[string][]content.LookPart{
		"closed": part(spriteDoors, 0, blockOffset), "open": {}, "sealed": part(spriteDoors, 4, blockOffset),
	}
	for s := 1; s <= 3; s++ {
		door[fmt.Sprint(s)] = part(spriteDoors, s, blockOffset)
	}
	look(lookDoor, content.LayerStand, door)
	look(lookLever, content.LayerStand, map[string][]content.LookPart{"off": feature(frameLeverOff), "on": feature(frameLeverOn)})
	look(lookPlate, content.LayerFloor, map[string][]content.LookPart{"up": feature(framePlateUp), "down": feature(framePlateDown)})
	look(lookSlot, content.LayerFloor, map[string][]content.LookPart{"slot": feature(frameSlot)})
	look(lookTeleporter, content.LayerFloor, map[string][]content.LookPart{"armed": feature(frameTeleArmed), "disarmed": feature(frameTeleDisarmed)})
	look(lookPit, content.LayerFloor, map[string][]content.LookPart{"pit": feature(framePit)})
	look(lookTrapdoor, content.LayerFloor, map[string][]content.LookPart{"open": feature(frameTrapOpen), "closed": feature(frameTrapClosed)})
	socket := map[string][]content.LookPart{"empty": feature(frameSocketEmpty), "full": feature(frameSocketFull)}
	for i := range socketPlaces {
		socket[content.SocketSetState(i)] = feature(frameSocketSet + i)
	}
	look(lookSocket, content.LayerFloor, socket)
	look(lookMouth, content.LayerFloor, map[string][]content.LookPart{"mouth": feature(frameMouth)})
	look(lookMark, content.LayerFloor, map[string][]content.LookPart{"mark": feature(frameMark)})
	look(lookExitFloor, content.LayerFloor, map[string][]content.LookPart{"exit": feature(frameExitFloor)})
	look(lookExitLeft, content.LayerWall, map[string][]content.LookPart{"exit": part(spriteWalls, frameExitLeft, blockOffset)})
	look(lookExitRight, content.LayerWall, map[string][]content.LookPart{"exit": part(spriteWalls, frameExitRight, blockOffset)})
}

const textID = "text.en"

func text(p *content.Pack) {
	s := map[string]string{
		"msg.lever_stuck": "The lever won't budge.",
		"msg.wrong_item":  "The {item} doesn't fit.",
		"msg.slot_stuck":  "Nothing more fits in the {item}'s place.",
		"msg.no_drop":     "It sinks out of sight.",
		"msg.no_room":     "There is no room for the {item}.",
		"msg.mouth":       "Welcome to the hall of *mechanisms*.",
		"msg.mouth.after": "The *socket* is complete.",
	}
	for _, it := range itemArt {
		s["item."+it.name+".name"] = itemNames[it.name]
	}
	p.Texts[textID] = &content.Text{ID: textID, Strings: s}
	p.Messages = &content.Messages{ID: "messages", Messages: map[string]string{
		content.MsgLeverStuck: "msg.lever_stuck", content.MsgSlotWrongItem: "msg.wrong_item",
		content.MsgSlotStuck: "msg.slot_stuck", content.MsgNoDropHere: "msg.no_drop",
		content.MsgNoRoom: "msg.no_room",
	}}
}

var itemNames = map[string]string{
	"key": "Key", "rock": "Rock", "red_gem": "Red Gem", "blue_gem": "Blue Gem", "sphere": "Sphere",
	"sword": "Sword", "sack": "Sack", "bag": "Bag of Lightness", "potion": "Potion",
}

// Item ids.
const (
	ItemKey     = "item.key"
	ItemRock    = "item.rock"
	ItemRedGem  = "item.red_gem"
	ItemBlueGem = "item.blue_gem"
	ItemSphere  = "item.sphere"
	ItemSword   = "item.sword"
	ItemSack    = "item.sack"
	ItemBag     = "item.bag"
	ItemPotion  = "item.potion"
)

// Thing template ids.
const (
	ThingGuard   = "thing.guard"
	ThingGhost   = "thing.ghost"
	ThingBoulder = "thing.boulder"
	ThingCrate   = "thing.crate"
	ThingPlayer  = "thing.player"
)

// StarterAttributes are the player's attributes, an agile fighter, and its interval bonus,
// standing in for the boots a new character wears: it walks a step every 2 ticks.
var StarterAttributes = map[string]int{
	"strength": 18, "agility": 20, "endurance": 18, "accuracy": 16, "talent": 16, "power": 16,
	"interval_bonus": -2,
}

func records(p *content.Pack) {
	stats := map[string]struct {
		weight, size int
		random       bool
		container    *content.Container
	}{
		"key": {1, 1, false, nil}, "rock": {60, 3, false, nil}, "red_gem": {1, 1, false, nil},
		"blue_gem": {1, 1, false, nil}, "sphere": {6, 2, false, nil}, "sword": {10, 3, true, nil},
		"sack":   {2, 3, false, &content.Container{Capacity: 4, Largest: 2}},
		"bag":    {1, 2, false, &content.Container{Capacity: 4, Largest: 2, Weightless: true}},
		"potion": {2, 1, true, nil},
	}
	for i, it := range itemArt {
		st := stats[it.name]
		id := "item." + it.name
		p.Items[id] = &content.Item{
			ID: id, Name: id + ".name", Weight: st.weight, Size: st.size, Random: st.random,
			Container: st.container,
			Icon:      content.Image{Sprite: spriteIcons, Frame: i}, Floor: content.Image{Sprite: spriteFloor, Frame: i},
		}
	}
	p.Things[ThingGuard] = &content.Thing{ID: ThingGuard, Kind: content.ThingCreature, Sprite: "sprite.guard",
		Facing: 3, Speed: 8, IntervalMS: 330, Weight: 100, HitPoints: content.Dice{Count: 3, Sides: 6, Bonus: 4},
		Behaviors: []string{content.BehaviorHostile, content.BehaviorLeavesCorpse}}
	p.Things[ThingGhost] = &content.Thing{ID: ThingGhost, Kind: content.ThingCreature, Sprite: "sprite.ghost",
		Facing: 3, Speed: 8, IntervalMS: 330, Weight: 100, HitPoints: content.Dice{Bonus: 12},
		Behaviors: []string{content.BehaviorNeverHostile, content.BehaviorResistsTeleport}}
	p.Things[ThingBoulder] = &content.Thing{ID: ThingBoulder, Kind: content.ThingBoulder, Sprite: spriteBoulder,
		Speed: 8, IntervalMS: 110, Weight: 100, Footprint: 16}
	p.Things[ThingCrate] = &content.Thing{ID: ThingCrate, Kind: content.ThingTrap, Sprite: spriteCrate,
		IntervalMS: 2200, Weight: 100, Footprint: 20}
	p.Things[ThingPlayer] = &content.Thing{ID: ThingPlayer, Kind: content.ThingPlayer, Sprite: "sprite.player",
		Facing: 3, Speed: 8, IntervalMS: 275, Weight: 100, Footprint: 16,
		HitPoints: content.Dice{Bonus: 56}, MagicPoints: content.Dice{Bonus: 52}, Stats: StarterAttributes}
}
