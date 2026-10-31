package devpack

import "github.com/sirgwain/the-summoning-go/engine/content"

// Level ids and the hall's start cell.
const (
	Hall   = "level.hall"
	Cellar = "level.cellar"
)

// HallStart is where a game starts, in the hall.
var HallStart = content.Cell{10, 4}

// Floor types of both levels.
const (
	typePlain = iota
	typeFlags
	typeWater
	typeLava
)

func floorTypes() []content.FloorType {
	return []content.FloorType{
		typePlain: {Variant: floorPlain},
		typeFlags: {Variant: floorFlags},
		typeWater: {Variant: floorWater, Depth: 4, Sinks: true},
		typeLava:  {Variant: floorLava, Enter: &content.Effect{Damage: content.Dice{Count: 2, Sides: 2}, Type: "fire"}, Avoided: true},
	}
}

// grid makes a level's floor and wall grids from a map, a string per row x, a character per
// column y: '#' a wall, 'e' a wall with a face to the right, '.' floor, ',' flagstones, '~' deep
// water, '^' lava, ' ' nothing. Walls have plain floor under them.
func grid(l *content.Level, rows []string) {
	l.Size = [2]int{len(rows), len(rows[0])}
	l.Floor, l.Wall = content.NewGrid(l.Size[0], l.Size[1]), content.NewGrid(l.Size[0], l.Size[1])
	for x, row := range rows {
		if len(row) != l.Size[1] {
			panic("devpack: ragged map row")
		}
		for y, ch := range row {
			floor, wall := typePlain, content.NoFrame
			switch ch {
			case '#':
				wall = shapeBlock
			case 'e':
				wall = shapeFaceRight
			case '.':
			case ',':
				floor = typeFlags
			case '~':
				floor = typeWater
			case '^':
				floor = typeLava
			case ' ':
				floor = content.NoFrame
			default:
				panic("devpack: unknown map character " + string(ch))
			}
			l.Floor[x][y], l.Wall[x][y] = floor, wall
		}
	}
}

func ptr[T any](v T) *T { return &v }

// The hall's cells, for tests and scripts.
var (
	LeverA        = content.Cell{7, 3}
	DoorA         = content.Cell{5, 3}
	PlateB        = content.Cell{7, 9}
	DoorB         = content.Cell{5, 9}
	SlotC         = content.Cell{7, 15}
	DoorC         = content.Cell{5, 15}
	DoorD         = content.Cell{5, 22}
	ExitD         = content.Cell{2, 22}
	SealedDoor    = content.Cell{5, 25}
	LeverD        = content.Cell{16, 22}
	Sphere        = content.Cell{15, 22}
	Teleporter    = content.Cell{12, 3}
	LeverTele     = content.Cell{13, 1}
	Pit           = content.Cell{12, 7}
	Trapdoor      = content.Cell{12, 10}
	LeverTrapdoor = content.Cell{10, 10}
	Mouth         = content.Cell{10, 13}
	Mark          = content.Cell{12, 13}
	Socket        = content.Cell{13, 22}
	SocketWall    = content.Cell{14, 22}
	LeverBoulder  = content.Cell{12, 26}
	WestExit      = content.Cell{9, 0}
	RocksB        = [2]content.Cell{{8, 8}, {8, 10}}
	GemB          = content.Cell{2, 9}
	GemC          = content.Cell{2, 15}
	Sack          = content.Cell{8, 17}
	CrateStart    = content.Cell{8, 6}
)

// hall is the main level: a hall with a closet behind each door along its back wall and the
// other mechanisms laid out across its floor.
//
//	Closet A: lever A opens and closes door A; a sword inside.
//	Closet B: plate B (100) holds door B open; two rocks (60 each) and things can press it.
//	Closet C: slot C takes the key (in the sack) and opens door C.
//	The socket takes the two gems from closets B and C and opens the wall behind it into a nook,
//	and changes what the mouth says. Lever D in the nook opens door D.
//	Closet D: the exit down to the cellar.
//	Closet E: a door sealed by the sphere, which lies in the nook; a potion inside. The sphere
//	thrown at the door opens it.
//	Elsewhere: a teleporter and the lever that arms it, a pit and a trapdoor down to the cellar
//	(a lever opens the trapdoor), a mouth, a mark, deep water that swallows objects, lava, a
//	guard, a ghost that teleporters leave alone, a crate to push, a boulder and a lever that
//	rolls it, and an exit in the west wall.
//
// The player starts by closet A, wearing boots of speed. Every closet opens by playing.
func hall(p *content.Pack) {
	l := &content.Level{ID: Hall, Floors: floorSet, Walls: wallSet, FloorTypes: floorTypes()}
	grid(l, []string{
		"############################",
		"#,,,,,#,,,,,#,,,,,#,,,,#,,,#",
		"#,,,,,#,,,,,#,,,,,#,,,,#,,,#",
		"#,,,,,#,,,,,#,,,,,#,,,,#,,,#",
		"#,,,,,#,,,,,#,,,,,#,,,,#,,,#",
		"###.#####.#####.######.##.##",
		"#..........................#",
		"#..........................#",
		"#..........................#",
		"e..........................#",
		"#..........................#",
		"#...............~~~........#",
		"#...............~~~........#",
		"#...............^^^........#",
		"############################",
		"                    #,,,#   ",
		"                    #,,,#   ",
		"                    #####   ",
	})
	door := func(id string, c content.Cell) *content.Feature {
		return &content.Feature{ID: id, Kind: content.FeatureDoor, Cell: c, Look: lookDoor, State: content.DoorClosed}
	}
	lever := func(id string, c content.Cell, on, off string) *content.Feature {
		return &content.Feature{ID: id, Kind: content.FeatureLever, Cell: c, Look: lookLever, Armed: true, OnTurnOn: on, OnTurnOff: off}
	}
	sealed := door("door.e", SealedDoor)
	sealed.SealedBy = ItemSphere
	l.Features = []*content.Feature{
		lever("lever.a", LeverA, "open_a", "close_a"),
		door("door.a", DoorA),
		{ID: "plate.b", Kind: content.FeaturePlate, Cell: PlateB, Look: lookPlate, Armed: true, Threshold: 100, OnPress: "open_b", OnRelease: "close_b"},
		door("door.b", DoorB),
		{ID: "slot.c", Kind: content.FeatureSlot, Cell: SlotC, Look: lookSlot, Armed: true, Accepts: ItemKey, Keyhole: true, OnInsert: "open_c"},
		door("door.c", DoorC),
		door("door.d", DoorD),
		lever("lever.d", LeverD, "open_d", "close_d"),
		sealed,
		{ID: "exit.d", Kind: content.FeatureExit, Cell: ExitD, Look: lookExitFloor, To: &content.Destination{Level: Cellar, Cell: content.Cell{6, 4}}},
		{ID: "teleporter", Kind: content.FeatureTeleporter, Cell: Teleporter, Look: lookTeleporter, Armed: true,
			Destinations: []content.Cell{{7, 24}, {12, 24}}},
		lever("lever.teleporter", LeverTele, "toggle_teleporter", "toggle_teleporter"),
		{ID: "pit", Kind: content.FeaturePit, Cell: Pit, Look: lookPit, To: &content.Destination{Level: Cellar, Cell: content.Cell{3, 3}}},
		{ID: "trapdoor", Kind: content.FeatureTrapdoor, Cell: Trapdoor, Look: lookTrapdoor, To: &content.Destination{Level: Cellar, Cell: content.Cell{3, 6}}},
		lever("lever.trapdoor", LeverTrapdoor, "open_trapdoor", "close_trapdoor"),
		{ID: "mouth", Kind: content.FeatureMouth, Cell: Mouth, Look: lookMouth, Message: "msg.mouth"},
		{ID: "mark", Kind: content.FeatureMark, Cell: Mark, Look: lookMark, Rune: "rune.dev"},
		{ID: "socket", Kind: content.FeatureSocket, Cell: Socket, Look: lookSocket, Places: []string{ItemRedGem, ItemBlueGem}, OnComplete: "reveal"},
		lever("lever.boulder", LeverBoulder, "roll", "stop"),
		{ID: "exit.west", Kind: content.FeatureExit, Cell: WestExit, Look: lookExitRight, To: &content.Destination{Level: Cellar, Cell: content.Cell{5, 2}}},
	}
	l.Procedures = map[string][]content.Action{
		"open_a":            {{On: ptr("door.a")}},
		"close_a":           {{Off: ptr("door.a")}},
		"open_b":            {{On: ptr("door.b")}},
		"close_b":           {{Off: ptr("door.b")}},
		"open_c":            {{On: ptr("door.c")}},
		"open_d":            {{On: ptr("door.d")}},
		"close_d":           {{Off: ptr("door.d")}},
		"toggle_teleporter": {{Toggle: ptr("teleporter")}},
		"open_trapdoor":     {{On: ptr("trapdoor")}},
		"close_trapdoor":    {{Off: ptr("trapdoor")}},
		"reveal": {
			{SetFloor: &content.SetFloor{Cell: SocketWall, Type: typePlain}},
			{SetMessage: &content.SetMessage{Feature: "mouth", Message: "msg.mouth.after"}},
		},
		"roll": {{Roll: &content.Roll{Thing: "boulder", Direction: 0}}},
		"stop": {{Stop: ptr("boulder")}},
	}
	l.Things = []*content.PlacedThing{
		{ID: "guard", Thing: ThingGuard, Cell: content.Cell{9, 20}, Items: []content.Object{{Item: ItemPotion}}},
		{ID: "ghost", Thing: ThingGhost, Cell: content.Cell{11, 21}, Facing: ptr(1)},
		{ID: "boulder", Thing: ThingBoulder, Cell: content.Cell{9, 24}},
		{ID: "crate", Thing: ThingCrate, Cell: CrateStart},
	}
	object := func(c content.Cell, item string, contents ...string) *content.FloorObject {
		o := &content.FloorObject{Cell: c, Object: content.Object{Item: item}}
		for _, it := range contents {
			o.Contents = append(o.Contents, content.Object{Item: it})
		}
		return o
	}
	l.Objects = []*content.FloorObject{
		object(content.Cell{2, 3}, ItemSword),
		object(GemB, ItemRedGem),
		object(GemC, ItemBlueGem),
		object(RocksB[0], ItemRock),
		object(RocksB[1], ItemRock),
		object(Sack, ItemSack, ItemKey),
		object(content.Cell{9, 18}, ItemBag),
		object(Sphere, ItemSphere),
		object(content.Cell{2, 25}, ItemPotion),
		object(content.Cell{13, 8}, content.RandomItem),
	}
	p.Levels[l.ID] = l
}

// BruteStart is where the cellar's brute waits.
var BruteStart = content.Cell{2, 2}

// cellar is the level below the hall, where its pit, trapdoor and exits lead, with an exit back
// up: a hostile brute waits there, and boots of speed and a mail shirt lie by the west exit.
func cellar(p *content.Pack) {
	l := &content.Level{ID: Cellar, Floors: floorSet, Walls: wallSet, FloorTypes: floorTypes()}
	grid(l, []string{
		"##########",
		"#........#",
		"#........#",
		"#.,,,,,,.#",
		"e.,,,,,,.#",
		"#.,,,,,,.#",
		"#........#",
		"#........#",
		"#........#",
		"##########",
	})
	l.Features = []*content.Feature{
		{ID: "exit.up", Kind: content.FeatureExit, Cell: content.Cell{7, 7}, Look: lookExitFloor, To: &content.Destination{Level: Hall, Cell: HallStart}},
		{ID: "exit.west", Kind: content.FeatureExit, Cell: content.Cell{4, 0}, Look: lookExitRight, To: &content.Destination{Level: Hall, Cell: content.Cell{9, 1}}},
	}
	l.Objects = []*content.FloorObject{
		{Cell: content.Cell{5, 5}, Object: content.Object{Item: ItemPotion}},
		{Cell: content.Cell{6, 2}, Object: content.Object{Item: ItemBoots}},
		{Cell: content.Cell{6, 3}, Object: content.Object{Item: ItemMail}},
	}
	l.Things = []*content.PlacedThing{
		{ID: "brute", Thing: ThingBrute, Cell: BruteStart, Items: []content.Object{{Item: ItemKey}}},
	}
	p.Levels[l.ID] = l
}

// Arena is a level for fighting: the player arrives at ArenaStart, where a sword lies, and a
// hostile goblin, about as strong as a first level's monsters, waits along the same row.
const Arena = "level.arena"

// ArenaStart and GoblinStart are where the arena's player and goblin start.
var (
	ArenaStart  = content.Cell{1, 1}
	GoblinStart = content.Cell{1, 7}
)

func arena(p *content.Pack) {
	l := &content.Level{ID: Arena, Floors: floorSet, Walls: wallSet, FloorTypes: floorTypes()}
	grid(l, []string{
		"##########",
		"#........#",
		"#.,,,,,,.#",
		"#........#",
		"##########",
	})
	l.Things = []*content.PlacedThing{{ID: "goblin", Thing: ThingGoblin, Cell: GoblinStart}}
	l.Objects = []*content.FloorObject{{Cell: ArenaStart, Object: content.Object{Item: ItemSword}}}
	p.Levels[l.ID] = l
}
