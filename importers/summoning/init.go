package summoning

import (
	"encoding/binary"
	"fmt"
)

// ResourceBases are INIT's resource group bases. A group's resources are addressed as its base
// plus an offset.
type ResourceBases struct {
	Frontend           uint16
	ViewEffect         uint16
	EndScene           uint16
	SmallObject        uint16
	Worn               uint16
	TinyObject         uint16
	People             uint16
	Music              uint16
	Scroll             uint16
	CharacterAnimation uint16
	HeldItemAnimation  uint16
	Terrain            uint16
	NPC                uint16
	WallDecoration     uint16
}

// SpellPower is a spell's power entry in INIT.
type SpellPower struct {
	Dice, Sides, BonusSides uint8
}

// Direction is one step in a direction, in cells.
type Direction struct {
	DX, DY int8
}

// Spell table dimensions in INIT.
const (
	SpellSchools        = 4
	SpellsPerSchool     = 10
	spellGestureSize    = 9
	keypadFacingCount   = 11
	WallTopShapes       = 20
	WallTopLeftCap      = 1 << 1
	WallTopRightCap     = 1 << 0
	KeypadFacingNone    = 8
	figureFrameShiftLen = 96
	heldItemFramesLen   = 1536
)

// Init holds INIT's tables, in file order. Tables whose use is not yet specified are kept as
// raw bytes.
type Init struct {
	Bases ResourceBases
	// WornOffsets are the worn-image offsets, (col, row) pairs.
	WornOffsets       [][2]uint8
	TextureShades     [256]uint8
	BaseColorMap      [256]uint8
	FigureFrameShifts [figureFrameShiftLen]uint8
	HeldItemFrames    [heldItemFramesLen]uint8
	SpellCosts        [SpellSchools][SpellsPerSchool]uint8
	SpellPower        [SpellSchools][SpellsPerSchool]SpellPower
	// SpellGestures are the gestures as letters A–L.
	SpellGestures [SpellSchools][SpellsPerSchool]string
	Directions    [8]Direction
	// KeypadFacings are the facings walked by Home, Up, PgUp, (−), Left, (5), Right, (+), End,
	// Down and PgDn; KeypadFacingNone for none.
	KeypadFacings [keypadFacingCount]uint8
	// WallTops are the caps of each wall shape: WallTopLeftCap and WallTopRightCap.
	WallTops [WallTopShapes]uint8
	// Start is the new game's level and cell.
	StartLevel, StartX, StartY uint8
}

// ReadInit reads INIT.
func ReadInit(data []byte) (*Init, error) {
	r := &reader{data: data}
	in := &Init{}
	b := &in.Bases
	for _, p := range []*uint16{
		&b.Frontend, &b.ViewEffect, &b.EndScene,
		&b.SmallObject, &b.Worn, &b.TinyObject, &b.People, &b.Music, &b.Scroll,
		&b.CharacterAnimation, &b.HeldItemAnimation, &b.Terrain, &b.NPC, &b.WallDecoration,
	} {
		*p = r.u16()
	}
	n := int(r.u8())
	in.WornOffsets = make([][2]uint8, n)
	for i := range in.WornOffsets {
		in.WornOffsets[i] = [2]uint8{r.u8(), r.u8()}
	}
	r.copy(in.TextureShades[:])
	r.copy(in.BaseColorMap[:])
	r.copy(in.FigureFrameShifts[:])
	r.copy(in.HeldItemFrames[:])
	for s := range SpellSchools {
		r.copy(in.SpellCosts[s][:])
	}
	for s := range SpellSchools {
		for i := range SpellsPerSchool {
			in.SpellPower[s][i] = SpellPower{r.u8(), r.u8(), r.u8()}
		}
	}
	for s := range SpellSchools {
		for i := range SpellsPerSchool {
			in.SpellGestures[s][i] = cString(r.bytes(spellGestureSize))
		}
	}
	for i := range in.Directions {
		in.Directions[i] = Direction{int8(r.u8()), int8(r.u8())}
	}
	r.copy(in.KeypadFacings[:])
	r.copy(in.WallTops[:])
	in.StartLevel, in.StartX, in.StartY = r.u8(), r.u8(), r.u8()
	if r.err != nil {
		return nil, fmt.Errorf("init: %w", r.err)
	}
	return in, nil
}

// reader reads little-endian values from a byte slice. After the first read past the end,
// every read returns zeros and err is set.
type reader struct {
	data []byte
	pos  int
	err  error
}

func (r *reader) bytes(n int) []byte {
	if r.err != nil || n < 0 || r.pos+n > len(r.data) {
		if r.err == nil {
			r.err = fmt.Errorf("unexpected end of data at offset %d, reading %d bytes", r.pos, n)
		}
		return make([]byte, max(n, 0))
	}
	b := r.data[r.pos : r.pos+n : r.pos+n]
	r.pos += n
	return b
}

func (r *reader) copy(dst []byte) { copy(dst, r.bytes(len(dst))) }

func (r *reader) u8() uint8 { return r.bytes(1)[0] }

func (r *reader) u16() uint16 { return binary.LittleEndian.Uint16(r.bytes(2)) }

func (r *reader) remaining() int { return len(r.data) - r.pos }

// cString returns b up to its first NUL.
func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
