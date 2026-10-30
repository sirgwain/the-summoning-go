package summoning

import (
	"bytes"
	"testing"
)

// levelFixture builds a 2×3 level with the given entity stream, text pool and script pool.
func levelFixture(stream, text, script []byte) []byte {
	f := fixture{}.u8(2, 3)
	f = f.u16(44, 0x1B, 33, 18, 54, 2)
	f = f.u8(0, 9, 1, 0, 2, 0, 10, 0, 0, 0)
	f = f.u16(LiquidWater, 58, 65)
	f = f.zeros(36)
	f = f.u8(0x01, 0x1B, 0x03, 0x1B, 0x1F, 0x24)
	f = append(f, stream...)
	f = f.u16(uint16(len(text)))
	f = append(f, text...)
	f = f.u16(uint16(len(script)))
	return append(f, script...)
}

func TestParseLevel(t *testing.T) {
	stream := fixture{}.
		u8(0, 0, 0, 0, uint8(PlaceDecoration), 23). // decoration: ornament 3 of decoration 2
		u8(1, 2, 2, 7, 9, 1, uint8(PlaceObjects)).  // two objects and a trap
		u8(1, 1, 0, 0, uint8(PlaceExit), 0, 5, 16, 15).
		u8(0xFF, 0x00)
	text := []byte("first\x00\x00third\x00\x00\x00")
	script := fixture{}.
		u8(1, 0, 1, 2, 0, 0). // procedure 0: one instruction
		u8(0).                // procedure 1: empty
		u8(2, 3, 1, 2, 1, 4, 10, 1, 1, 50, 0).
		zeros(4) // padding

	l, err := ParseLevel(levelFixture(stream, text, script))
	if err != nil {
		t.Fatal(err)
	}
	if l.Rows != 2 || l.Columns != 3 {
		t.Fatalf("size %d×%d", l.Rows, l.Columns)
	}
	h := l.Header
	if h.WallBank != 44 || h.ExitWallBank != 2 || h.LiquidKind != LiquidWater || h.TrapBank != 58 || h.BoulderBank != 65 {
		t.Errorf("header %+v", h)
	}
	if h.DecorationBanks != [5]uint8{0, 1, 2, 10, 0} || h.DecorationUnknown[0] != 9 {
		t.Errorf("decorations %v %v", h.DecorationBanks, h.DecorationUnknown)
	}
	if !h.DeepWater() {
		t.Error("not deep water")
	}
	if l.TerrainAt(1, 2) != 0x24 || l.TerrainAt(0, 1) != 0x1B || l.TerrainAt(2, 0) != TerrainOutside || l.TerrainAt(-1, 0) != TerrainOutside {
		t.Errorf("terrain %v", l.Terrain)
	}

	if len(l.Placements) != 3 {
		t.Fatalf("%d placements", len(l.Placements))
	}
	if orn, dec := l.Placements[0].Decoration(); orn != 3 || dec != 2 {
		t.Errorf("decoration %d %d", orn, dec)
	}
	p := l.Placements[1]
	if p.X != 1 || p.Y != 2 || !bytes.Equal(p.Objects, []byte{7, 9}) || !p.Trap || p.Type != PlaceObjects || len(p.Extra) != 0 {
		t.Errorf("objects placement %+v", p)
	}
	if p := l.Placements[2]; p.Type != PlaceExit || !bytes.Equal(p.Extra, []byte{0, 5, 16, 15}) {
		t.Errorf("exit placement %+v", p)
	}

	if len(l.Messages) < 3 || string(l.Messages[0]) != "first" || len(l.Messages[1]) != 0 || string(l.Messages[2]) != "third" {
		t.Errorf("messages %q", l.Messages)
	}

	if len(l.Procedures) != 3 {
		t.Fatalf("%d procedures, want 3", len(l.Procedures))
	}
	if len(l.Procedures[1]) != 0 {
		t.Errorf("procedure 1 has %d instructions", len(l.Procedures[1]))
	}
	if got, want := l.Procedures[2][1], (Instruction{10, [4]uint8{1, 1, 50, 0}}); got != want {
		t.Errorf("procedure 2 instruction 1 = %+v, want %+v", got, want)
	}
}

func TestParseLevelRejects(t *testing.T) {
	end := fixture{}.u8(0xFF, 0xFF)
	tests := map[string][]byte{
		"unknown type":   levelFixture(fixture{}.u8(0, 0, 0, 0, 12).u8(0xFF, 0xFF), nil, nil),
		"trailing bytes": append(levelFixture(end, nil, nil), 0),
		"short pool":     levelFixture(end, nil, fixture{}.u8(2, 0, 0, 0, 0, 0)),
		"too large":      fixture{}.u8(65, 1),
		"truncated":      levelFixture(end, nil, nil)[:70],
	}
	for name, rec := range tests {
		if _, err := ParseLevel(rec); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestReadLevels(t *testing.T) {
	end := fixture{}.u8(0xFF, 0xFF)
	levels, err := ReadLevels(container(levelFixture(end, nil, nil), levelFixture(end, []byte("hi\x00"), nil)))
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 2 || string(levels[1].Messages[0]) != "hi" {
		t.Errorf("levels %v", levels)
	}
}
