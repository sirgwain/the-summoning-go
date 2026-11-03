package summoning

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/color"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// rawBank builds a raw sprite bank resource whose frame i is filled with value(i).
func rawBank(frames, height, widthUnits int, transparent uint8, value func(i int) uint8) []byte {
	var pixels []byte
	for i := range frames {
		for range height * widthUnits * 8 {
			pixels = append(pixels, value(i))
		}
	}
	return spriteResource(uint16(frames), uint8(height), uint8(widthUnits), transparent, CompressionRaw, pixels)
}

// textRecord encodes s as a TEXT record.
func textRecord(s string) []byte {
	b := make([]byte, 0, len(s)+1)
	for i := range len(s) {
		b = append(b, s[i]^textKey)
	}
	return append(b, 0)
}

// Fixture item templates: names from TEXT record fixtureNameBase on.
const fixtureNameBase = 3

var fixtureItems = []struct {
	name                                       string
	size, weight, container, image, flags, cat byte
}{
	{"Random Object", 0, 2, 0, 0, 0, 6},
	{"Dagger", 2, 1, 0, 1, 0, 0x40 | 9},
	{"Sack", 15, 1, 0x51, 0, 0, 6},
	{"Key", 1, 1, 0, 1, 0x80, 6},
	{"Dagger", 2, 1, 0, 1, 0, 0x40 | 9},
	{"Bag of Lightness", 15, 1, 0x93, 0, 0, 6},
}

// textFixture is TEXT with the fixture's item names and the engine's messages.
func textFixture() []byte {
	recs := make([][]byte, 0xA9)
	for i := range recs {
		recs[i] = textRecord(fmt.Sprintf("text %d", i))
	}
	for i, it := range fixtureItems {
		recs[fixtureNameBase+i] = textRecord(it.name)
	}
	recs[0x8F] = textRecord("The %s doesn't fit")
	return container(append(recs, textRecord("END"))...)
}

func objectsFixture() []byte {
	f := fixture{}.u16(uint16(len(fixtureItems)), fixtureNameBase)
	for i, it := range fixtureItems {
		f = f.u8(it.size, byte(i), it.weight, it.container, 0, 0, 0, 0, 0, 0, it.image, it.flags, 0, it.cat, 0)
	}
	return f
}

// npcFixture has record 0, a walker with eight frames in bank 7 (resource 26), and record 1, an
// animated object.
func npcFixture() []byte {
	f := fixture{}.u16(2)
	f = f.u8(0, 0, 1, 2, 3, 4, 2, 7, 8, 0, 9, 6, 0, 0).u16(behaviorHostile | behaviorCorpse)
	return f.u8(0, 0, 0, 1, 0, 0, 3, 7, patternFixed, 0, 0, 10, 0, 0).u16(0)
}

// importFixture is a small game. INIT's terrain base is 18, its NPC base 19, its wall-decoration
// base 20 and its small and tiny object bases 10 and 12 (see initFixture); the level's banks are
// offsets 0 (walls, 18, caps 19; also traps and boulders), 2 (floors, 20), 3 (plates, 21), 4
// (doors, 22 and 23), 6 (floor overlays, 24), 7 (exit walls, 25), and decoration bank 6 (26).
// songRecord encodes a RESOURCE music record: one instrument and the events.
func songRecord(events ...byte) []byte {
	desc := make([]byte, songDescriptorSize)
	desc[0], desc[1], desc[15], desc[23] = 240, 4, 1, 1
	binary.LittleEndian.PutUint32(desc[6:], uint32(len(events)))
	binary.LittleEndian.PutUint16(desc[16:], 120)
	inst := make([]byte, instrumentSize)
	for k, v := range testInstrument.Modulator {
		inst[2+2*k] = v
	}
	for k, v := range testInstrument.Carrier {
		inst[28+2*k] = v
	}
	rec := []byte{'E', 'H', byte(ResourceMusic), songDescriptorSize, 0}
	rec = append(rec, desc...)
	rec = append(rec, inst...)
	return append(rec, events...)
}

// fixtureMusic is where the import fixture's songs start: tracks 0-16 and sound effects 0-17h.
const fixtureMusic = 60

func importFixture() *Source {
	in := initFixture()
	copy(in[len(in)-4:], []byte{0, 1, 1})                   // start: level 0, cell (1, 1)
	binary.LittleEndian.PutUint16(in[6+2*4:], fixtureMusic) // the music base

	colors := fixture{}.u8(16).zeros(256 * 3)

	plus := func(n int) func(int) uint8 { return func(i int) uint8 { return uint8(n + i) } }
	resources := make([][]byte, 110)
	for _, r := range []int{10, 11, 12, 13, mouthResource, exitMarkerResource} {
		resources[r] = rawBank(1, 2, 1, 0, plus(r))
	}
	resources[18] = rawBank(30, 2, 2, 0, plus(1))
	resources[19] = rawBank(2, 1, 1, 0, plus(100))
	resources[20] = rawBank(4, 2, 1, 0xFF, plus(50))
	resources[21] = rawBank(8, 1, 1, 0, plus(60))
	resources[22] = rawBank(8, 1, 1, 0, plus(70))
	resources[23] = rawBank(8, 1, 1, 0, plus(80))
	resources[24] = rawBank(4, 1, 1, 0, plus(90))
	resources[25] = rawBank(6, 2, 2, 0, plus(110))
	resources[26] = rawBank(10, 1, 1, 0, plus(120))
	// The player's figure: the body (character animation base 16), the legs (17), the body looks
	// (16 + 25h and 26h) and the head looks (18, the wall bank, and 19 standing in for them).
	// Portraits 0 and 1 (people base 13): strips of frames, a row each.
	resources[14] = rawBank(3, 1, 1, 0, plus(14))
	resources[16] = rawBank(48, 2, 1, 0, plus(130))
	resources[17] = rawBank(48, 1, 1, 0, plus(180))
	resources[16+bodyLook] = rawBank(56, 1, 1, 0, plus(150))
	resources[16+bareBodyFemale] = rawBank(56, 1, 1, 0, plus(150))
	for n := range firstEffect + 0x18 {
		resources[fixtureMusic+n] = songRecord(0x00, 0xC0, 0x00, 0x00, 0x90, byte(0x30+n), 0x60, 0x30, 0x80, 0x30, 0x00, 0x00, 0xFC)
	}

	level := fixture{}.u8(3, 4)
	level = level.u16(0, 2, 3, 4, 6, 7)
	level = level.u8(6, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	level = level.u16(LiquidNone, 0, 0).zeros(36)
	level = level.u8(
		0x01, 0x01, 0x03, 0x01,
		0x1B, 0x1C, 0x1F, 0x24,
		0x17, 0x10, 0x24, 0xFF,
	)
	level = level.
		u8(0, 1, 0, 0, uint8(PlaceDecoration), 3).
		u8(0, 2, 0, 0, uint8(PlaceExit), 4, 0, 2, 3).
		u8(2, 1, 0, 0, uint8(PlaceDoor), doorVertical).
		u8(2, 2, 0, 0, uint8(PlaceTeleporter), teleporterArmed, 0, 0xFF, 0, 1).
		u8(1, 3, 0, 0, uint8(PlaceTeleporter), 0, 0, 0, 0, 0).
		// The slot takes the Key and runs procedure 0; the plate runs 1.
		u8(2, 0, 0, 0, uint8(PlaceTrigger), 0, 0, 3, 0, noProcedure).
		u8(1, 2, 0, 0, uint8(PlaceTrigger), 0, 0, 50, 1, noProcedure).
		// A creature carrying a Dagger, with a trap; a shuttling boulder; a Sack holding the Key
		// and a random object; a mouth.
		u8(1, 1, 1, 1, 1, uint8(PlaceNPC), 0).
		u8(1, 1, 0, 0, uint8(PlaceBoulder), boulderShuttling, 1).
		u8(1, 2, 3, 2, 3, 0, 0, uint8(PlaceContainer), 0).
		u8(1, 0, 0, 0, uint8(PlaceMouth), 0).
		u8(0xFF, 0xFF)
	level = level.u16(6).str("Hello\x00")
	// Procedure 0 opens the door; procedure 1 makes a Dagger, sets the plate's threshold and rolls
	// the boulder.
	script := fixture{}.u8(1, 0, 2, 1, 0, 0).u8(3, 11, 1, 0, 1, 0, 10, 1, 2, 99, 0, 5, 1, 1, 4, 0)
	level = level.u16(uint16(len(script)))
	level = append(level, script...)

	return &Source{
		Colors: colors, Init: in, Resource: container(resources...), Levels: container(level),
		Text: textFixture(), Objects: objectsFixture(), NPC: npcFixture(), F: portraitsFixture(),
	}
}

// portraitsFixture is F with two portraits of each other's sex, each the other's other and talk
// portrait: 0, mirrored, and 1, female. One premade character follows.
func portraitsFixture() []byte {
	f := fixture{}.u16(2, 0, 49, 1)
	f = f.u8(1, 5, 6, 7, 8, 1, 1, 0)
	f = f.u8(0, 1, 2, 3, 4, 0, 0, 1)
	// Kordan: portrait 1, attributes 18, 17, 16, 15, 14, 13, school 2, weapon class 3.
	f = f.str("Kordan").zeros(26 - len("Kordan"))
	return f.u8(1, 18, 17, 16, 15, 14, 13, 2, 3)
}

func TestImport(t *testing.T) {
	p, report, err := Import(importFixture(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	// The fixture has six item templates, none of the starting inventory's or creation's, no spell
	// art, no door pictures and no view effect or sparkle art, so its spells, scenes and flipbooks
	// are left out.
	for _, line := range report.Lines() {
		ok := false
		for _, prefix := range []string{"starting inventory:", "spell ", "gesture icons:", "magic wall:", "gateway:", "portrait ", "pointers:", "inventory:", "worn image ", "character creation:", "door picture ", "the door scene:", "the ending scene:", "view effect ", "sparkle "} {
			ok = ok || strings.HasPrefix(line, prefix)
		}
		if !ok {
			t.Errorf("report: %v", line)
		}
	}
	if fx := p.Palettes["palette.0"].Effects; len(fx) != 3 || fx[0].First != 0xE0 || fx[1].Groups != 3 || fx[2].Index != 0xFC || fx[0].PeriodMS != 440 {
		t.Errorf("palette effects %+v", fx)
	}
	if p.Manifest.Start != (content.Start{Level: "level.00", Cell: content.Cell{1, 1}, Music: "music.0"}) {
		t.Errorf("start %+v", p.Manifest.Start)
	}

	l := p.Levels["level.00"]
	if l == nil {
		t.Fatal("no level.00")
	}
	n := content.NoFrame
	if want := (content.Grid{{1, 1, 3, 1}, {n, n, n, n}, {n, n, n, n}}); !reflect.DeepEqual(l.Wall, want) {
		t.Errorf("wall %v, want %v", l.Wall, want)
	}
	if want := (content.Grid{{0, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, n}}); !reflect.DeepEqual(l.Floor, want) {
		t.Errorf("floor %v, want %v", l.Floor, want)
	}
	// Walls show the floor halves whose check neighbors are on the map: shape 1 checks up-left
	// (off the map) and down-right; shape 3 checks the walls on either side.
	if want := (content.Grid{{2, 2, 3, 2}, {3, 3, 3, 3}, {3, 3, 3, n}}); !reflect.DeepEqual(l.FloorHalves, want) {
		t.Errorf("floor halves %v, want %v", l.FloorHalves, want)
	}
	// Shape 3 shows its upper half while (0, 1) is seen and its lower while (0, 3) is, and takes
	// shape Eh with the cell above it unseen; walls take their sight from their shapes. The
	// vertical door at (2, 1) checks as 11h, up (0) and down (4), and shows whole open.
	if s := p.Walls[l.Walls].Shape(3); !reflect.DeepEqual(s.Sight, []int{6, 2}) || !reflect.DeepEqual(s.Unseen, [][2]int{{0, 0xE}, {4, 0xD}}) {
		t.Errorf("shape 3: sight %v, unseen %v", s.Sight, s.Unseen)
	}
	for _, c := range []struct {
		cell         content.Cell
		upper, lower int
	}{{content.Cell{0, 2}, n, n}, {content.Cell{2, 1}, 0, 4}, {content.Cell{1, 1}, n, n}} {
		if u, lo := l.Sight(c.cell); u != c.upper || lo != c.lower {
			t.Errorf("sight of %v: %d, %d, want %d, %d", c.cell, u, lo, c.upper, c.lower)
		}
	}
	if l.Floors != "floors.20" || l.Walls != "walls.18" {
		t.Errorf("sets %s %s", l.Floors, l.Walls)
	}
	if want := []content.FloorType{{Variant: 0}, {Variant: 1}}; !reflect.DeepEqual(l.FloorTypes, want) {
		t.Errorf("floor types %+v", l.FloorTypes)
	}
	if len(l.Overlays) != 0 || len(l.Props) != 0 {
		t.Errorf("fixed art %+v %+v", l.Overlays, l.Props)
	}
	wantDecorations := []content.Placed{{Cell: content.Cell{0, 1}, Sprite: "sprite.res.26", Frame: 3, Offset: content.Offset{8, 8}}}
	if !reflect.DeepEqual(l.Decorations, wantDecorations) {
		t.Errorf("decorations %+v", l.Decorations)
	}
}

func TestImportFeatures(t *testing.T) {
	p, _, err := Import(importFixture(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	l := p.Levels["level.00"]
	cell := func(x, y int) content.Cell { return content.Cell{x, y} }
	want := []*content.Feature{
		{ID: "exit.0.2", Kind: content.FeatureExit, Cell: cell(0, 2), Look: "look.exit.25.3",
			To: &content.Destination{Level: "level.00", Cell: cell(2, 3)}},
		{ID: "mouth.1.0", Kind: content.FeatureMouth, Cell: cell(1, 0), Look: "look.mouth", Message: "msg.l00.0"},
		{ID: "plate.1.2", Kind: content.FeaturePlate, Cell: cell(1, 2), Look: "look.plate.21", Armed: true, Threshold: 50, OnPress: "p1"},
		{ID: "teleporter.1.3", Kind: content.FeatureTeleporter, Cell: cell(1, 3), Look: "look.teleporter.21",
			Destinations: []content.Cell{cell(1, 3), cell(1, 3)}},
		{ID: "slot.2.0", Kind: content.FeatureSlot, Cell: cell(2, 0), Look: "look.slot.24.0", Accepts: "item.key", Keyhole: true, Armed: true, OnInsert: "p0"},
		{ID: "door.2.1", Kind: content.FeatureDoor, Cell: cell(2, 1), Look: "look.door.23.v", State: content.DoorClosed},
		// Jumps (0, 0) and (−1, +1).
		{ID: "teleporter.2.2", Kind: content.FeatureTeleporter, Cell: cell(2, 2), Look: "look.teleporter.21", Armed: true,
			Destinations: []content.Cell{cell(2, 2), cell(1, 3)}},
	}
	if len(l.Features) != len(want) {
		t.Fatalf("%d features, want %d: %+v", len(l.Features), len(want), l.Features)
	}
	for i := range want {
		if !reflect.DeepEqual(l.Features[i], want[i]) {
			t.Errorf("feature %d\n%+v\nwant\n%+v", i, *l.Features[i], *want[i])
		}
	}

	door := p.Looks["look.door.23.v"]
	if door.Layer != content.LayerStand || len(door.States["open"]) != 0 || len(door.States["2"]) != 2 ||
		door.States["2"][0] != (content.LookPart{Sprite: "sprite.res.23", Frame: 4, Offset: content.Offset{-14, -14}}) {
		t.Errorf("door look %+v", door)
	}
	if m := p.Looks["look.mouth"].States["mouth"]; len(m) != 1 || m[0].Offset != (content.Offset{8, -20}) {
		t.Errorf("mouth look %+v", m)
	}

	dagger, key, door2 := "item.dagger", "item.key", "door.2.1"
	wantProcs := map[string][]content.Action{
		"p0": {{On: &door2}},
		"p1": {
			{Create: &content.Create{Cell: cell(1, 0), Item: dagger}},
			{SetThreshold: &content.SetThreshold{Feature: "plate.1.2", Value: 99}},
			{Roll: &content.Roll{Thing: "boulder.1.1", Direction: 4}},
		},
	}
	if !reflect.DeepEqual(l.Procedures, wantProcs) {
		t.Errorf("procedures %+v", l.Procedures)
	}

	offset := content.Offset{-6, 12}
	wantThings := []*content.PlacedThing{
		{ID: "creature.1.1", Thing: "thing.npc.0", Cell: cell(1, 1), Offset: &offset, Items: []content.Object{{Item: dagger}}},
		{ID: "trap.1.1", Thing: "thing.trap.18", Cell: cell(1, 1)},
		{ID: "boulder.1.1", Thing: "thing.boulder.18", Cell: cell(1, 1), Rolling: &content.Rolling{Direction: 2, Shuttle: true}},
	}
	if !reflect.DeepEqual(l.Things, wantThings) {
		t.Errorf("things %+v", l.Things)
	}
	wantObjects := []*content.FloorObject{
		{Cell: cell(1, 2), Object: content.Object{Item: "item.sack", Contents: []content.Object{{Item: key}, {Item: content.RandomItem}}}},
	}
	if !reflect.DeepEqual(l.Objects, wantObjects) {
		t.Errorf("objects %+v", l.Objects)
	}
	if p.Texts["text.en"].Strings["msg.l00.0"] != "Hello" {
		t.Error("level message not in the text")
	}
}

func TestImportCatalog(t *testing.T) {
	p, _, err := Import(importFixture(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for id := range p.Items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if want := []string{"item.bag_of_lightness", "item.dagger", "item.dagger.2", "item.key", "item.random_object", "item.sack"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("items %v", ids)
	}
	dagger := p.Items["item.dagger"]
	if dagger.Name != "msg.4" || !reflect.DeepEqual(dagger.Tags, []string{"weapon"}) || !dagger.Random ||
		dagger.Icon != (content.Image{Sprite: "sprite.res.11", Frame: 0}) || dagger.Floor != (content.Image{Sprite: "sprite.res.13", Frame: 0}) {
		t.Errorf("dagger %+v", dagger)
	}
	if p.Items["item.key"].Random || p.Items["item.random_object"].Random != true {
		t.Error("random items wrong")
	}
	if c := p.Items["item.bag_of_lightness"].Container; c == nil || c.Capacity != 4 || c.Largest != 9 || !c.Weightless {
		t.Errorf("bag %+v", c)
	}
	if c := p.Items["item.sack"].Container; c == nil || c.Weightless {
		t.Errorf("sack %+v", c)
	}
	if p.Messages.Messages[content.MsgSlotWrongItem] != "msg.143" || p.Texts["text.en"].Strings["msg.143"] != "The {item} doesn't fit" {
		t.Errorf("messages %+v", p.Messages.Messages)
	}

	npc := p.Things["thing.npc.0"]
	wantAnims := map[string]content.Animation{
		"idle.0": {Frames: []int{0}}, "walk.0": {Frames: []int{0, 4}, MS: content.Fixed(330)}, "attack.0": {Frames: []int{4}},
		"idle.2": {Frames: []int{1}}, "walk.2": {Frames: []int{1, 5}, MS: content.Fixed(330)}, "attack.2": {Frames: []int{5}},
		"idle.4": {Frames: []int{2}}, "walk.4": {Frames: []int{2, 6}, MS: content.Fixed(330)}, "attack.4": {Frames: []int{6}},
		"idle.6": {Frames: []int{3}}, "walk.6": {Frames: []int{3, 7}, MS: content.Fixed(330)}, "attack.6": {Frames: []int{7}},
		"dead": {Frames: []int{9}}, // the bank's last frame,
	}
	if npc.Kind != content.ThingCreature || npc.Sprite != "sprite.res.26" || npc.Facing != 4 || npc.IntervalMS != 330 ||
		npc.HitPoints != (content.Dice{Count: 2, Sides: 10, Bonus: 4}) ||
		!reflect.DeepEqual(npc.Behaviors, []string{content.BehaviorHostile, content.BehaviorLeavesCorpse}) ||
		!reflect.DeepEqual(npc.Animations, wantAnims) {
		t.Errorf("creature %+v", npc)
	}
	if a := p.Sprites["sprite.res.26"].Anchor; a == nil || *a != (content.Offset{18, 19}) {
		t.Errorf("creature anchor %v", a)
	}
	if b := p.Things["thing.boulder.18"]; b.Kind != content.ThingBoulder || !reflect.DeepEqual(b.Animations["roll.4"].Frames, []int{4, 3, 2, 1, 0}) {
		t.Errorf("boulder %+v", b)
	}
}

func TestImportStillObject(t *testing.T) {
	src := importFixture()
	levels, _ := ReadContainer(src.Levels)
	level := append([]byte(nil), levels[0]...)
	// Place record 1, an animated object showing frame 3, instead of record 0.
	i := bytes.Index(level, []byte{1, 1, 1, 1, 1, uint8(PlaceNPC), 0})
	level[i+6] = 1
	src.Levels = container(level)
	p, _, err := Import(src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	obj := p.Things["thing.npc.1"]
	if obj.Speed != 0 || obj.Facing != 0 || !reflect.DeepEqual(obj.Animations, map[string]content.Animation{"idle": {Frames: []int{3}}}) {
		t.Errorf("animated object %+v", obj)
	}
}

// pixel returns the palette index at (x, y) of a sprite bank's frame.
func pixel(t *testing.T, p *content.Pack, sprite string, frame, x, y int) uint8 {
	t.Helper()
	b := p.Sprites[sprite]
	r := b.FrameRect(frame)
	return b.Sheet.ColorIndexAt(r.Min.X+x, r.Min.Y+y)
}

func TestImportComposesTiles(t *testing.T) {
	p, _, err := Import(importFixture(), Options{})
	if err != nil {
		t.Fatal(err)
	}

	f := p.Floors["floors.20"]
	if f.Variants != 2 || p.Sprites[f.Sprite].FrameSize != (content.Offset{8, 4}) {
		t.Fatalf("floor set %+v", f)
	}
	if pixel(t, p, f.Sprite, 1, 0, 1) != 52 || pixel(t, p, f.Sprite, 1, 7, 2) != 53 {
		t.Error("floor variant 1 is not frames 2 over 3")
	}

	w := p.Walls["walls.18"]
	if w.Offset != (content.Offset{0, -1}) || p.Sprites[w.Sprite].FrameSize != (content.Offset{16, 5}) {
		t.Fatalf("wall set %+v, frame size %v", w, p.Sprites[w.Sprite].FrameSize)
	}
	// Shape 1 has no caps: its top row is transparent, then upper part (frame 1) and lower (16).
	if pixel(t, p, w.Sprite, 1, 3, 0) != 0 || pixel(t, p, w.Sprite, 1, 3, 1) != 2 || pixel(t, p, w.Sprite, 1, 3, 4) != 17 {
		t.Error("shape 1 composed wrong")
	}
	// Shape 5 has both caps: the left cap over the left half, the right over the right half.
	if pixel(t, p, w.Sprite, 5, 0, 0) != 100 || pixel(t, p, w.Sprite, 5, 15, 0) != 101 {
		t.Error("shape 5 caps composed wrong")
	}

	if w.Shapes[0].Diagonal != content.CornerBottom || w.Shapes[1].Diagonal != "" || w.Shapes[5].Diagonal != content.CornerTop || !w.Shapes[1].Face || w.Shapes[2].Face || !w.Shapes[3].Face {
		t.Errorf("wall shape properties %+v %+v %+v %+v", *w.Shapes[0], *w.Shapes[1], *w.Shapes[2], *w.Shapes[3])
	}

	left, right := p.Looks["look.exit.25.1"], p.Looks["look.exit.25.3"]
	if left == nil || right == nil || right.Layer != content.LayerWall {
		t.Fatalf("exit looks %+v %+v", left, right)
	}
	e := right.States[content.FeatureExit][0]
	if e.Frame != 1 || e.Offset != (content.Offset{0, -2}) || left.States[content.FeatureExit][0].Frame != 0 {
		t.Fatalf("exit look parts %+v", e)
	}
	// The exit in shape 3 walls is exit shape 4: frames 3, 4 and 5 stacked.
	if pixel(t, p, e.Sprite, 1, 0, 0) != 113 || pixel(t, p, e.Sprite, 1, 0, 2) != 114 || pixel(t, p, e.Sprite, 1, 0, 5) != 115 {
		t.Error("exit wall composed wrong")
	}
}

func TestImportReportsMissingFrames(t *testing.T) {
	src := importFixture()
	// Give the level a pit (25h), plate frame 6, with a plate bank of only two frames.
	recs, _ := ReadContainer(src.Resource)
	resources := make([][]byte, len(recs))
	copy(resources, recs)
	resources[21] = rawBank(2, 1, 1, 0, func(int) uint8 { return 60 })
	src.Resource = container(resources...)
	levels, _ := ReadContainer(src.Levels)
	level := append([]byte(nil), levels[0]...)
	level[2+levelHeaderSize+5] = 0x25 // cell (1, 1)
	src.Levels = container(level)

	_, report, err := Import(src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Lines()) == 0 {
		t.Error("missing plate frame not reported")
	}
}

func TestWallChecksAs(t *testing.T) {
	// A closed horizontal door below wins over the vertical doors beside.
	if got := wallChecksAs(5, true, true, true); got != checkBelowDoor[5] {
		t.Errorf("below a door: %d, want %d", got, checkBelowDoor[5])
	}
	// Vertical doors on both sides: left, then right of the result.
	if got, want := wallChecksAs(5, false, true, true), checkRightDoor[checkLeftDoor[5]]; got != want {
		t.Errorf("between doors: %d, want %d", got, want)
	}
	if got := wallChecksAs(5, false, false, false); got != 5 {
		t.Errorf("no doors: %d, want 5", got)
	}
}

func TestImportPlayer(t *testing.T) {
	p, _, err := Import(importFixture(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Manifest.Player != playerID {
		t.Fatalf("manifest player %q", p.Manifest.Player)
	}
	tm := p.Things[playerID]
	if tm.Kind != content.ThingPlayer || tm.Speed != 8 || tm.IntervalMS != ms(5) || tm.HitPoints.Bonus != 56 || tm.Stats["agility"] != 20 {
		t.Errorf("player template %+v", tm)
	}
	if got := tm.Animations["walk.0"].Frames; !reflect.DeepEqual(got, []int{9, 17, 25, 33, 41, 1}) {
		t.Errorf("walk.0 frames %v", got)
	}
	// Of the starting inventory the fixture has only template 1, the Palimpsest's place on shelf 1.
	if len(tm.Inventory) != 1 || tm.Inventory[0].Slot != 1 {
		t.Errorf("inventory %v", tm.Inventory)
	}
	if got := tm.Animations["idle.7"].Frames; !reflect.DeepEqual(got, []int{8}) {
		t.Errorf("idle.7 frames %v", got)
	}
	// Facing 1 (direction 0) strikes with the left hand in frame 30h + 8 + 1.
	if got := tm.Animations["attack.left.0"].Frames; !reflect.DeepEqual(got, []int{0x30 + 8 + 1}) {
		t.Errorf("attack.left.0 frames %v", got)
	}
	if tm.Portrait != "portrait.0" {
		t.Errorf("player portrait %q", tm.Portrait)
	}
	want0 := &content.Portrait{ID: "portrait.0", Image: content.Image{Sprite: "sprite.portrait.13"}, Mirrored: true,
		FigureColors: []int{5, 6, 7, 8}, Other: "portrait.1", Talk: "portrait.1"}
	want1 := &content.Portrait{ID: "portrait.1", Image: content.Image{Sprite: "sprite.portrait.14"},
		FigureColors: []int{1, 2, 3, 4}, Female: true, Other: "portrait.0", Talk: "portrait.0"}
	if got := p.Portraits; len(got) != 2 || !reflect.DeepEqual(got["portrait.0"], want0) || !reflect.DeepEqual(got["portrait.1"], want1) {
		t.Errorf("portraits %+v %+v", got["portrait.0"], got["portrait.1"])
	}
	// Portrait 1's three frames of 8×1 stack into one image of 8×3.
	if b := p.Sprites["sprite.portrait.14"]; b == nil || b.Frames != 1 || b.FrameSize != (content.Offset{8, 3}) {
		t.Errorf("portrait sprite %+v", b)
	}
	b := p.Sprites[tm.Sprite]
	if b.FrameSize != (content.Offset{24, 19}) || b.Frames != 72 || *b.Anchor != creatureAnchor {
		t.Fatalf("player sprite %v, %d frames, anchor %v", b.FrameSize, b.Frames, *b.Anchor)
	}
	// Frame 9 (group 1, step 1): the body's frame 9 at row 13, the legs' at row 18, the body
	// look's frame 8 + 1 at row 5, the head's frame 1 at (8, 0).
	at := func(x, y int) uint8 {
		r := b.FrameRect(9)
		return b.Sheet.ColorIndexAt(r.Min.X+x, r.Min.Y+y)
	}
	for _, c := range []struct {
		x, y int
		want uint8
	}{
		{0, 13, 130 + 9}, {0, 18, 180 + 9}, {0, 5, 150 + 9}, {8, 0, 1 + 1},
	} {
		if got := at(c.x, c.y); got != c.want {
			t.Errorf("frame 9 at (%d, %d) = %d, want %d", c.x, c.y, got, c.want)
		}
	}
}

func TestImportFighting(t *testing.T) {
	p, _, err := Import(importFixture(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	d := p.Items["item.dagger"]
	if d.WeaponClass == nil || *d.WeaponClass != 0 || d.Wear != content.WearHand || d.Durability != 250 {
		t.Errorf("dagger %+v", d)
	}
	if s := p.Items["item.sack"]; s.WeaponClass != nil || s.Durability != 0 {
		t.Errorf("sack %+v", s)
	}
	if p.Manifest.Rules != content.RulesSummoning {
		t.Errorf("rules %q", p.Manifest.Rules)
	}
	if got := placeholders("Hit! Rolled %d, damage %d", []string{"score", "damage"}); got != "Hit! Rolled {score}, damage {damage}" {
		t.Errorf("placeholders: %q", got)
	}
	for c, want := range map[int]content.Dice{0: {}, 0x11: {Count: 2, Sides: 2}, 0x25: {Count: 3, Sides: 6}} {
		if got := dice(c); got != want {
			t.Errorf("dice(%02Xh) = %+v, want %+v", c, got, want)
		}
	}
	var tm ObjectTemplate
	tm[4], tm[5], tm[6], tm[7], tm[8], tm[13] = 142, 9, 0x15, 0, 0x21, 0x40|9
	it := &content.Item{}
	(&importer{}).fighting(0, &tm, it)
	want := []content.ItemAction{
		{Kind: content.ActionAttack, Damage: content.Dice{Count: 2, Sides: 6}, Delay: 15, Skill: 2},
		{Kind: content.ActionUseUp, Delay: 12},
	}
	if !reflect.DeepEqual(it.Actions, want) {
		t.Errorf("actions %+v, want %+v", it.Actions, want)
	}
	// Icon 11h uses the object and swings it.
	tm[5] = 0x11
	it = &content.Item{}
	(&importer{}).fighting(0, &tm, it)
	want[1] = content.ItemAction{Kind: content.ActionUse, Delay: 12, Swings: true}
	if !reflect.DeepEqual(it.Actions, want) {
		t.Errorf("actions %+v, want %+v", it.Actions, want)
	}
}

// TestHeldLayers composes a held look from a hand-made table: for hand 0 (table 1) the player's
// frame 0 shows the bank's frame 1 behind the figure at (2, −3), frame 1 its frame 0 in front at
// (−1, 4), and frame 2 nothing.
func TestHeldLayers(t *testing.T) {
	pal := &content.Palette{ID: "p", Colors: [256]color.RGBA{}}
	im := &importer{pack: content.NewPack(content.Manifest{}), init: &Init{}, palette: pal, report: &Report{},
		banks: map[int]*SpriteBank{0: {Width: 2, Height: 2, Transparent: 0,
			Frames: [][]byte{{5, 5, 5, 5}, {6, 6, 6, 6}}}}}
	im.init.Bases.HeldItemAnimation = 0
	set := func(table, frame int, e [4]int8) {
		at := (table*heldEntries + frame) * heldEntrySize
		for i, b := range e {
			im.init.HeldItemFrames[at+i] = uint8(b)
		}
	}
	set(1, 0, [4]int8{0, 1, -3, 2})
	set(1, 1, [4]int8{1, 0, 4, -1})
	set(1, 2, [4]int8{0, 9, 0, 0}) // a frame the bank lacks
	l := im.heldLayers(1, 0, 3)
	if l == nil || l.Behind == "" || l.Front == "" {
		t.Fatalf("layers %+v", l)
	}
	behind, front := im.pack.Sprites[l.Behind], im.pack.Sprites[l.Front]
	// The box covers (−1, −3) to (4, 6): 5×9, and the anchor moves by its corner.
	if behind.FrameSize != (content.Offset{5, 9}) || *behind.Anchor != (content.Offset{creatureAnchor[0] + 1, creatureAnchor[1] + 3}) {
		t.Fatalf("behind sheet %v, anchor %v", behind.FrameSize, *behind.Anchor)
	}
	px := func(b *content.SpriteBank, f, x, y int) uint8 {
		r := b.FrameRect(f)
		return b.Sheet.ColorIndexAt(r.Min.X+x-(-1), r.Min.Y+y-(-3))
	}
	if px(behind, 0, 2, -3) != 6 || px(front, 0, 2, -3) != 0 {
		t.Error("frame 0: the bank's frame 1 is not behind at (2, −3)")
	}
	if px(front, 1, -1, 4) != 5 || px(behind, 1, -1, 4) != 0 {
		t.Error("frame 1: the bank's frame 0 is not in front at (−1, 4)")
	}
	if px(behind, 2, 0, 0) != 0 || px(front, 2, 0, 0) != 0 {
		t.Error("frame 2 draws something")
	}
}

func TestImportCreation(t *testing.T) {
	p, _, err := Import(importFixture(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	c := p.Creation
	if c == nil {
		t.Fatal("no creation record")
	}
	if len(c.Portraits) != 2 || len(c.Methods) != 3 || len(c.WeaponClasses) != 4 || len(c.Schools) != 4 || c.Counter != "" || c.Opening != "" {
		t.Errorf("creation %+v", *c)
	}
	if len(c.Premade) != 1 {
		t.Fatalf("premade %+v", c.Premade)
	}
	pm := c.Premade[0]
	if pm.Name != "Kordan" || pm.Portrait != "portrait.1" || pm.School != 2 || pm.WeaponClass != 3 ||
		pm.Attributes["strength"] != 18 || pm.Attributes["power"] != 13 {
		t.Errorf("premade %+v", pm)
	}
	if err := p.Verify(); err != nil {
		t.Error(err)
	}
}

// TestSpentObjects checks what used-up objects leave (items.md, Using up): the next template for
// an object with a quantity, an Empty Bottle for a bottle, and nothing for an object with no
// quantity or one that uses charges.
func TestSpentObjects(t *testing.T) {
	im := &importer{objects: &Objects{Templates: make([]ObjectTemplate, 14)}, pack: &content.Pack{Items: map[string]*content.Item{}}}
	for i := range im.objects.Templates {
		id := fmt.Sprintf("item.%d", i)
		im.itemIDs = append(im.itemIDs, id)
		im.pack.Items[id] = &content.Item{ID: id}
	}
	useUp := []content.ItemAction{{Kind: content.ActionUseUp}}
	set := func(i, charges int, actions []content.ItemAction) {
		im.pack.Items[im.itemIDs[i]].Charges, im.pack.Items[im.itemIDs[i]].Actions = charges, actions
	}
	set(1, 1, useUp)                                           // an apple: leaves its core
	set(2, 0, useUp)                                           // a rune: leaves nothing
	set(3, 5, []content.ItemAction{{Kind: content.ActionUse}}) // a wand: never replaced
	set(4, 1, useUp)                                           // a potion
	im.objects.Templates[4][13] = categoryBottle
	im.spentAndBroken()
	for i, want := range map[int]string{1: "item.2", 2: "", 3: "", 4: "item.7", 5: ""} {
		if got := im.pack.Items[im.itemIDs[i]].Spent; got != want {
			t.Errorf("template %d spent %q, want %q", i, got, want)
		}
	}
	if got := im.pack.Items["item.4"].BreaksInto; got != "item.12" {
		t.Errorf("the potion breaks into %q", got)
	}
}

// TestSharedChargesImport gives templates 72h–7Bh their charges as a shared count, weapons
// keeping their durability, and other weapons no charges.
func TestSharedChargesImport(t *testing.T) {
	var tm ObjectTemplate
	tm[2], tm[4], tm[5], tm[9], tm[13] = 10, 140, 0x11, 5, 0x40|9
	it := &content.Item{}
	(&importer{}).fighting(firstSharedTemplate+1, &tm, it)
	if !it.SharedCharges || it.Charges != 5 || it.Durability != 6000 {
		t.Errorf("a shared weapon: %+v", it)
	}
	it = &content.Item{}
	(&importer{}).fighting(firstSharedTemplate-1, &tm, it)
	if it.SharedCharges || it.Charges != 0 || it.Durability != 6000 {
		t.Errorf("a weapon: %+v", it)
	}
}

// TestImportConversationOnlyWhenTalks imports NPC record 0, whose script and portrait bytes are 0,
// with INTERACT holding script 0: the hostile record holds no conversation and no portrait, and
// once it talks it holds script 0's.
func TestImportConversationOnlyWhenTalks(t *testing.T) {
	for _, talks := range []bool{false, true} {
		src := importFixture()
		src.Interact = scriptFixture(-2, 7, -4, -3, 0x15, -4, -1) // never: end
		if talks {
			npc := slices.Clone(src.NPC)
			binary.LittleEndian.PutUint16(npc[2+14:], behaviorHostile|behaviorCorpse|behaviorTalks)
			src.NPC = npc
		}
		p, _, err := Import(src, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if p.Conversations["conversation.0"] == nil {
			t.Fatal("script 0 not imported")
		}
		want := ""
		if talks {
			want = "conversation.0"
		}
		if got := p.Things["thing.npc.0"].Conversation; got != want {
			t.Errorf("talks %v: conversation %q, want %q", talks, got, want)
		}
		if got := p.Things["thing.npc.0"].TalkPortrait; !talks && got != "" {
			t.Errorf("a record that does not talk has portrait %q", got)
		}
	}
}
