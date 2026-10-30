package summoning

import "testing"

func initFixture() fixture {
	f := fixture{}.u16(1, 2, 3)
	f = f.u16(10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
	f = f.u8(2, 5, 6, 7, 8) // two worn-image offsets
	f = f.zeros(256 + 256 + figureFrameShiftLen + heldItemFramesLen)
	costs := make([]byte, SpellSchools*SpellsPerSchool)
	costs[SpellsPerSchool+2] = 25 // school 1, spell 2
	f = append(f, costs...)
	power := make([]byte, SpellSchools*SpellsPerSchool*3)
	copy(power[3:], []byte{6, 10, 4}) // school 0, spell 1
	f = append(f, power...)
	for s := range SpellSchools * SpellsPerSchool {
		g := make([]byte, spellGestureSize)
		if s == 0 {
			copy(g, "DJ")
		}
		f = append(f, g...)
	}
	f = f.u8(0xFF, 0, 0xFF, 1, 0, 1, 1, 1, 1, 0, 1, 0xFF, 0, 0xFF, 0xFF, 0xFF)
	f = f.u8(7, 0, 1, 8, 6, 8, 2, 8, 5, 4, 3)
	tops := make([]byte, WallTopShapes)
	tops[5] = WallTopLeftCap | WallTopRightCap
	f = append(f, tops...)
	return f.u8(4, 16, 15, 0)
}

func TestReadInit(t *testing.T) {
	in, err := ReadInit(initFixture())
	if err != nil {
		t.Fatal(err)
	}
	if in.Bases.Frontend != 1 || in.Bases.EndScene != 3 || in.Bases.SmallObject != 10 || in.Bases.Terrain != 18 || in.Bases.WallDecoration != 20 {
		t.Errorf("bases %+v", in.Bases)
	}
	if len(in.WornOffsets) != 2 || in.WornOffsets[1] != [2]uint8{7, 8} {
		t.Errorf("worn offsets %v", in.WornOffsets)
	}
	if in.SpellCosts[1][2] != 25 {
		t.Errorf("spell cost = %d", in.SpellCosts[1][2])
	}
	if in.SpellPower[0][1] != (SpellPower{6, 10, 4}) {
		t.Errorf("spell power = %+v", in.SpellPower[0][1])
	}
	if in.SpellGestures[0][0] != "DJ" || in.SpellGestures[0][1] != "" {
		t.Errorf("gestures %q %q", in.SpellGestures[0][0], in.SpellGestures[0][1])
	}
	if in.Directions[0] != (Direction{-1, 0}) || in.Directions[5] != (Direction{1, -1}) {
		t.Errorf("directions %v", in.Directions)
	}
	if in.KeypadFacings[3] != KeypadFacingNone || in.KeypadFacings[0] != 7 {
		t.Errorf("keypad %v", in.KeypadFacings)
	}
	if in.WallTops[5] != 3 {
		t.Errorf("wall tops %v", in.WallTops)
	}
	if in.StartLevel != 4 || in.StartX != 16 || in.StartY != 15 {
		t.Errorf("start %d (%d, %d)", in.StartLevel, in.StartX, in.StartY)
	}
}

func TestReadInitShort(t *testing.T) {
	f := initFixture()
	if _, err := ReadInit(f[:len(f)-3]); err == nil {
		t.Error("no error")
	}
}
