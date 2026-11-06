package summoning

import (
	"bytes"
	"fmt"
)

// Level dimensions.
const (
	MaxLevelRows    = 64
	MaxLevelColumns = 64
	levelHeaderSize = 64
	// deepWaterFloorBank is the floor bank offset that marks a deep-water level.
	deepWaterFloorBank = 0x1B
)

// LevelHeader is a level's 64-byte header. Bank fields are offsets from INIT's terrain base;
// decoration banks are offsets from its wall-decoration base.
type LevelHeader struct {
	WallBank         uint16
	FloorBank        uint16
	PlateBank        uint16
	DoorBank         uint16
	FloorOverlayBank uint16
	ExitWallBank     uint16
	// DecorationBanks are the five decoration banks; DecorationUnknown the second byte of each
	// pair.
	DecorationBanks   [5]uint8
	DecorationUnknown [5]uint8
	LiquidKind        uint16
	TrapBank          uint16
	BoulderBank       uint16
	Unknown           [18]uint16
}

// Liquid kinds.
const (
	LiquidNone  = 0
	LiquidWater = 1
	LiquidFire  = 2
)

// DeepWater reports whether liquid depth applies on the level.
func (h *LevelHeader) DeepWater() bool { return h.FloorBank == deepWaterFloorBank }

// PlacementType is the type of an entity stream entry.
type PlacementType uint8

const (
	PlaceObjects     PlacementType = 0
	PlaceDecoration  PlacementType = 1
	PlaceRuneMark    PlacementType = 2
	PlaceTeleporter  PlacementType = 3
	PlaceDoor        PlacementType = 4
	PlaceUnused      PlacementType = 5
	PlaceBoulder     PlacementType = 6
	PlaceTrigger     PlacementType = 7
	PlaceExit        PlacementType = 8
	PlaceNPC         PlacementType = 9
	PlaceWeightPlate PlacementType = 10
	PlaceMouth       PlacementType = 11
)

// placementExtra is the number of extra bytes each placement type carries.
var placementExtra = [...]int{0, 1, 1, 5, 1, 0, 2, 5, 4, 1, 1, 1}

// Placement is one entry of a level's entity stream.
type Placement struct {
	X, Y uint8
	// Objects are object template ids.
	Objects []uint8
	// Trap places a trap entity at the cell.
	Trap  bool
	Type  PlacementType
	Extra []byte
}

// Decoration returns a decoration placement's ornament frame and decoration index.
func (p *Placement) Decoration() (ornament, decoration int) {
	b := int(p.Extra[0])
	return b % 10, b / 10
}

// Instruction is one instruction of a level procedure.
type Instruction struct {
	Op   uint8
	Args [4]uint8
}

// Level is one record of LEVELS.
type Level struct {
	Rows, Columns int
	Header        LevelHeader
	// Terrain is the terrain byte of each cell, row-major: cell (x, y) is Terrain[x*Columns+y].
	Terrain    []byte
	Placements []Placement
	// Messages are the text pool's strings, including empty ones.
	Messages [][]byte
	// Procedures are the script pool's level procedures.
	Procedures [][]Instruction
}

// TerrainAt returns the terrain of cell (x, y); cells outside the grid are TerrainOutside.
func (l *Level) TerrainAt(x, y int) byte {
	if x < 0 || y < 0 || x >= l.Rows || y >= l.Columns {
		return TerrainOutside
	}
	return l.Terrain[x*l.Columns+y]
}

// TerrainOutside is the terrain of cells off the map.
const TerrainOutside = 0xFF

// ReadLevels reads every level of LEVELS.
func ReadLevels(data []byte) ([]*Level, error) {
	records, err := ReadContainer(data)
	if err != nil {
		return nil, fmt.Errorf("levels: %w", err)
	}
	levels := make([]*Level, len(records))
	for i, rec := range records {
		if levels[i], err = ParseLevel(rec); err != nil {
			return nil, fmt.Errorf("level %d: %w", i, err)
		}
	}
	return levels, nil
}

// ParseLevel parses one LEVELS record.
func ParseLevel(rec []byte) (*Level, error) {
	r := &reader{data: rec}
	l := &Level{Rows: int(r.u8()), Columns: int(r.u8())}
	if l.Rows > MaxLevelRows || l.Columns > MaxLevelColumns {
		return nil, fmt.Errorf("%d×%d cells, over the maximum", l.Rows, l.Columns)
	}

	h := &l.Header
	for _, p := range []*uint16{&h.WallBank, &h.FloorBank, &h.PlateBank, &h.DoorBank, &h.FloorOverlayBank, &h.ExitWallBank} {
		*p = r.u16()
	}
	for i := range 5 {
		h.DecorationBanks[i], h.DecorationUnknown[i] = r.u8(), r.u8()
	}
	h.LiquidKind, h.TrapBank, h.BoulderBank = r.u16(), r.u16(), r.u16()
	for i := range h.Unknown {
		h.Unknown[i] = r.u16()
	}

	l.Terrain = r.bytes(l.Rows * l.Columns)

	for r.err == nil {
		x, y := r.u8(), r.u8()
		if x == 0xFF || y == 0xFF {
			break
		}
		p := Placement{X: x, Y: y}
		p.Objects = r.bytes(int(r.u8()))
		p.Trap = r.u8() != 0
		p.Type = PlacementType(r.u8())
		if r.err == nil && int(p.Type) >= len(placementExtra) {
			return nil, fmt.Errorf("placement at (%d, %d): unknown type %d", x, y, p.Type)
		}
		p.Extra = r.bytes(placementExtra[p.Type])
		l.Placements = append(l.Placements, p)
	}

	textPool := r.bytes(int(r.u16()))
	scriptPool := r.bytes(int(r.u16()))
	if r.err != nil {
		return nil, r.err
	}
	if r.remaining() != 0 {
		return nil, fmt.Errorf("%d bytes after the script pool", r.remaining())
	}

	l.Messages = bytes.Split(textPool, []byte{0})
	procs, err := parseProcedures(scriptPool)
	if err != nil {
		return nil, err
	}
	l.Procedures = procs
	return l, nil
}

const instructionSize = 5

// parseProcedures reads the level procedures of a script pool. A zero count is an empty
// procedure; the pool's zero padding reads as empty procedures, which are dropped from the end.
func parseProcedures(pool []byte) ([][]Instruction, error) {
	var procs [][]Instruction
	pos := 0
	for pos < len(pool) {
		n := int(pool[pos])
		pos++
		if pos+n*instructionSize > len(pool) {
			return nil, fmt.Errorf("script pool: procedure %d runs past the end", len(procs))
		}
		proc := make([]Instruction, n)
		for i := range proc {
			b := pool[pos : pos+instructionSize]
			proc[i] = Instruction{Op: b[0], Args: [4]uint8{b[1], b[2], b[3], b[4]}}
			pos += instructionSize
		}
		procs = append(procs, proc)
	}
	for len(procs) > 0 && len(procs[len(procs)-1]) == 0 {
		procs = procs[:len(procs)-1]
	}
	return procs, nil
}
