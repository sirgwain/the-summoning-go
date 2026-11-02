package devpack

import "github.com/sirgwain/the-summoning-go/engine/content"

// GameArt dresses the dev pack in the art of The Summoning, from a pack its importer wrote
// (importers/summoning; docs/pack.md gives the ids it writes). It names that pack's records by
// id and holds none of their art: the dressed pack is made on the machine that has the game, and
// is not checked in.
//
//	summoning devpack --art packs/summoning
//
// writes it to packs/dev.game.
var GameArt = Art{
	Floors: "floors.1036",
	Walls:  "walls.1056",
	FloorVariants: map[int]int{
		floorPlain: 0, floorFlags: 1, floorWater: 2, floorLava: 3,
	},
	// The game shapes a wall by its neighbors (docs/spec/world.md, Wall shapes): faces along a
	// run, diagonals at corners, T pieces and ends; a lone wall takes the four-way piece.
	WallShapes: [16]int{
		0:                                      10,
		wallUBefore:                            13,
		wallUAfter:                             14,
		wallUBefore | wallUAfter:               3,
		wallVBefore:                            11,
		wallVAfter:                             12,
		wallVBefore | wallVAfter:               1,
		wallUAfter | wallVAfter:                0,
		wallUAfter | wallVBefore:               2,
		wallUBefore | wallVAfter:               4,
		wallUBefore | wallVBefore:              5,
		wallUAfter | wallVBefore | wallVAfter:  6,
		wallUBefore | wallUAfter | wallVAfter:  7,
		wallUBefore | wallUAfter | wallVBefore: 8,
		wallUBefore | wallVBefore | wallVAfter: 9,
		wallUBefore | wallUAfter | wallVBefore | wallVAfter: 10,
	},
	WallChecks: [][2]int{
		{7, 3}, {0, 4}, {1, 1}, {6, 2}, {5, 4}, {7, 3}, {0, 3}, {6, 3},
		{-1, 2}, {-1, 4}, {-1, 3}, {0, 3}, {7, 4}, {6, 3}, {7, 2},
	},
	DoorChecks:        [2][2]int{{6, 2}, {0, 4}},
	ChecksBelowDoor:   []int{0, 6, 2, 3, 7, 8, 6, 7, 8, 10, 10, 2, 0, 13, 14},
	ChecksLeftOfDoor:  []int{6, 1, 2, 8, 9, 5, 6, 10, 8, 9, 10, 11, 12, 5, 2},
	ChecksRightOfDoor: []int{0, 1, 6, 7, 4, 9, 6, 7, 10, 9, 10, 11, 12, 4, 0},
	Doors:             [2]string{"look.door.1030.h", "look.door.1031.v"},
	SealedDoor:        "look.door.1030.h.sealed5",
	WallExits:         map[int]string{1: "look.exit.1014.1", 3: "look.exit.1014.3"},
	Features: map[string]string{
		content.FeatureExit:       "look.exit_marker",
		content.FeatureLever:      "look.lever.1066",
		content.FeaturePlate:      "look.plate.1045",
		content.FeatureSlot:       "look.slot.1066.0",
		content.FeatureTeleporter: "look.teleporter.1045",
		content.FeaturePit:        "look.pit.1045",
		content.FeatureTrapdoor:   "look.trapdoor.1045",
		content.FeatureSocket:     "look.socket.1067",
		content.FeatureMouth:      "look.mouth",
		content.FeatureMark:       "look.mark.1045.1",
	},
	Things: map[string]string{
		ThingPlayer:    "thing.player",
		ThingGuard:     "thing.npc.14",
		ThingGhost:     "thing.npc.117",
		ThingBrute:     "thing.npc.113",
		ThingGoblin:    "thing.npc.124",
		ThingSage:      "thing.npc.2",
		ThingTarget:    "thing.npc.97",
		ThingBoulder:   "thing.boulder.1076",
		ThingMagicWall: "thing.magic_wall",
		ThingPortal:    "thing.gateway",
	},
	Items: map[string]string{
		ItemKey:     "item.iron_key",
		ItemRock:    "item.rock",
		ItemRedGem:  "item.ruby",
		ItemBlueGem: "item.moonstone",
		ItemSphere:  "item.energy_sphere",
		ItemSword:   "item.broadsword",
		ItemSack:    "item.sack",
		ItemBag:     "item.bag_of_lightness",
		ItemPotion:  "item.jera_potion",
		ItemBoots:   "item.boots_of_speed",
		ItemMail:    "item.chainmail",
		ItemScroll:  "item.fireball_parchment",
	},
	Portraits: map[string]string{
		PortraitPlayer: "portrait.0",
		PortraitSage:   "portrait.50",
	},
	Flipbooks: map[string]string{
		FlipbookThrough: "flipbook.effect.1",
		FlipbookSocket:  "flipbook.effect.8",
		FlipbookTalk:    "flipbook.effect.9",
		SparkleSeal:     "flipbook.sparkle.seal.5",
	},
}
