package summoning

import (
	"reflect"
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

// importFixture is a small game. INIT's terrain base is 18 and its wall-decoration base 20 (see
// initFixture); the level's banks are offsets 0 (walls, 18, caps 19), 2 (floors, 20), 3 (plates,
// 21), 4 (doors, 22 and 23), 6 (floor overlays, 24), 7 (exit walls, 25), and decoration bank 6
// (26).
func importFixture() *Source {
	in := initFixture()
	copy(in[len(in)-4:], []byte{0, 1, 1}) // start: level 0, cell (1, 1)

	colors := fixture{}.u8(16).zeros(256 * 3)

	plus := func(n int) func(int) uint8 { return func(i int) uint8 { return uint8(n + i) } }
	resources := make([][]byte, 27)
	resources[18] = rawBank(30, 2, 2, 0, plus(1))
	resources[19] = rawBank(2, 1, 1, 0, plus(100))
	resources[20] = rawBank(4, 2, 1, 0xFF, plus(50))
	resources[21] = rawBank(8, 1, 1, 0, plus(60))
	resources[22] = rawBank(8, 1, 1, 0, plus(70))
	resources[23] = rawBank(8, 1, 1, 0, plus(80))
	resources[24] = rawBank(4, 1, 1, 0, plus(90))
	resources[25] = rawBank(6, 2, 2, 0, plus(110))
	resources[26] = rawBank(10, 1, 1, 0, plus(120))

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
		u8(0, 2, 0, 0, uint8(PlaceExit), 0, 1, 1, 1).
		u8(2, 1, 0, 0, uint8(PlaceDoor), doorVertical).
		u8(2, 2, 0, 0, uint8(PlaceTeleporter), teleporterArmed, 0, 0, 0, 0).
		u8(1, 3, 0, 0, uint8(PlaceTeleporter), 0, 0, 0, 0, 0).
		u8(0xFF, 0xFF).
		u16(0, 0)

	return &Source{Colors: colors, Init: in, Resource: container(resources...), Levels: container(level)}
}

func TestImport(t *testing.T) {
	p, report, err := Import(importFixture())
	if err != nil {
		t.Fatal(err)
	}
	if lines := report.Lines(); len(lines) != 0 {
		t.Errorf("report: %v", lines)
	}
	if p.Manifest.Start != (content.Start{Level: "level.00", Cell: content.Cell{1, 1}}) {
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
	if l.Floors != "floors.20" || l.Walls != "walls.18" || l.ExitWalls != "exitwalls.25" {
		t.Errorf("sets %s %s %s", l.Floors, l.Walls, l.ExitWalls)
	}
	if want := []content.Cell{{0, 2}}; !reflect.DeepEqual(l.Exits, want) {
		t.Errorf("exits %v", l.Exits)
	}

	wantOverlays := []content.Placed{
		{Cell: content.Cell{1, 2}, Sprite: "sprite.res.21", Frame: 0, Offset: content.Offset{-16, 8}},
		{Cell: content.Cell{2, 0}, Sprite: "sprite.res.24", Frame: 0, Offset: content.Offset{4, 11}},
		{Cell: content.Cell{2, 2}, Sprite: "sprite.res.21", Frame: 5, Offset: content.Offset{-16, 8}},
	}
	if !reflect.DeepEqual(l.Overlays, wantOverlays) {
		t.Errorf("overlays %+v\nwant %+v", l.Overlays, wantOverlays)
	}
	wantDecorations := []content.Placed{{Cell: content.Cell{0, 1}, Sprite: "sprite.res.26", Frame: 3, Offset: content.Offset{8, 8}}}
	if !reflect.DeepEqual(l.Decorations, wantDecorations) {
		t.Errorf("decorations %+v", l.Decorations)
	}
	wantProps := []content.Placed{
		{Cell: content.Cell{2, 1}, Sprite: "sprite.res.23", Frame: 0, Offset: content.Offset{-14, -14}},
		{Cell: content.Cell{2, 1}, Sprite: "sprite.res.23", Frame: 1, Offset: content.Offset{18, -3}},
	}
	if !reflect.DeepEqual(l.Props, wantProps) {
		t.Errorf("props %+v", l.Props)
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
	p, _, err := Import(importFixture())
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

	e := p.Walls["exitwalls.25"]
	if e.Shapes[1] != 0 || e.Shapes[3] != 1 || e.Shapes[0] != content.NoFrame || e.Offset != (content.Offset{0, -2}) {
		t.Fatalf("exit wall set %+v", e)
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

	_, report, err := Import(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Lines()) == 0 {
		t.Error("missing plate frame not reported")
	}
}

func TestWallChecksAs(t *testing.T) {
	l := &Level{Rows: 2, Columns: 3, Terrain: []byte{
		0x11, 0x05, 0x11,
		0x1B, 0x10, 0x1B,
	}}
	// A closed horizontal door below wins over the vertical doors beside.
	if got := wallChecksAs(l, 0, 1, 5); got != checkBelowDoor[5] {
		t.Errorf("below a door: %d, want %d", got, checkBelowDoor[5])
	}
	l.Terrain[4] = 0x1B
	// Vertical doors on both sides: left, then right of the result.
	if got, want := wallChecksAs(l, 0, 1, 5), checkRightDoor[checkLeftDoor[5]]; got != want {
		t.Errorf("between doors: %d, want %d", got, want)
	}
}
