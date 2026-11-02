package summoning

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
)

// The save file's layout (saves.md, The file).
const (
	saveHeaderSize   = 0x100
	saveNameSize     = 50
	saveLevelsAt     = saveHeaderSize + saveNameSize
	saveWorldSize    = 0x54F7
	savePlayerSize   = 0x326
	saveObjectCount  = 2000
	saveObjectSize   = 5
	saveLevelStateAt = 0x0B15 // where the level state starts in the world block
	saveLevelSize    = saveWorldSize - saveLevelStateAt
	saveEntitySize   = 34
	saveCellSize     = 6
	saveGridSide     = 64
	saveNoObject     = 0xFFFF
	saveFreeTemplate = 0xFF
)

// World block offsets (saves.md, The world block).
const (
	swHeld     = 0x0004
	swTimed    = 0x0010
	swVars     = 0x0031
	swNPCs     = 0x0097
	swShared   = 0x08C2
	swVisited  = 0x08D6
	swBound    = 0x08FF
	swPlayer   = 0x0AF3
	timedCount = 14
	varCount   = 100 // 0..98, and 99 the transcript option
	npcCount   = 127
	shareCount = 10
	boundCount = 100
	levelCount = 41
)

// Level state offsets, from the start of the world block (saves.md).
const (
	slEntities    = 0x0B15
	slEntitySlots = 99
	slEntityCount = 0x183B
	slCells       = 0x183C
	slCellCount   = 647
	slLevel       = 0x2766
	slTerrain     = 0x2767
	slView        = 0x3767
	slRows        = 0x4767
	slTriggers    = 0x47A9
	slDoors       = 0x4921
	slTeleporters = 0x4B2F
)

// Player block offsets (saves.md, The player block).
const (
	spName       = 0x0000
	spSchools    = 0x001E
	spWeapons    = 0x0022
	spLevel      = 0x0026
	spSchoolXP   = 0x0027
	spSchoolNext = 0x002F
	spWeaponXP   = 0x0037
	spWeaponNext = 0x003F
	spXP         = 0x0047
	spNextXP     = 0x004B
	spPoints     = 0x004F
	spAttributes = 0x0059
	spMaxAttrs   = 0x005F
	spFatigue    = 0x0067
	spSpells     = 0x006C
	spSelected   = 0x029C
	spConfusion  = 0x02A0
	spPortrait   = 0x02A2
	spStatus     = 0x02A8
	spSlots      = 0x02AA
	spPackOpen   = 0x02FC
	spVision     = 0x02FF
	spBonus      = 0x0304
	spCounted    = 0x0316
	spLook       = 0x0318
	spIcons      = 0x031A
	slotCount    = 40
	spellCount   = 40
	spellSize    = 14
	iconCount    = 12
)

// SaveFile is a saved game of the original (saves.md).
type SaveFile struct {
	Name   string
	Player SavePlayer
	// Held is the object on the pointer.
	Held uint16
	// Timed are the timed counters: the attributes' boosts, then the timed effects.
	Timed [timedCount]uint8
	// Vars are interaction variables 0..99.
	Vars [varCount]uint8
	// NPCs are the NPC records as the game has changed them.
	NPCs [npcCount]NPCRecord
	// Shared are the shared counts of templates 72h–7Bh.
	Shared [shareCount]uint16
	// Bound are the objects bound for another level.
	Bound []BoundObject
	// Body is the player's entity.
	Body SaveEntity
	// Current is the level the player is on; Levels the others visited, in file order.
	Current *SaveLevel
	Levels  []*SaveLevel
	Objects [saveObjectCount]SaveObject
}

// SavePlayer is the player block.
type SavePlayer struct {
	Name                      string
	Schools, Weapons          [4]uint8
	Level                     uint8
	SchoolXP, SchoolNext      [4]uint16
	WeaponXP, WeaponNext      [4]uint16
	XP, NextXP                uint32
	HP, MaxHP, MP, MaxMP, AC  int16
	Attributes, MaxAttributes [6]uint8
	Fatigue                   uint8
	Memorized                 [spellCount]uint8
	Selected                  [4]uint8
	Confusion                 int16
	Portrait, Other, Talk     uint16
	Status                    uint16
	Slots                     [slotCount]uint16
	PackOpen                  bool
	VisionX, VisionY          uint8
	IntervalBonus             int16
	Counted, Look             uint16
	Icons                     [iconCount]bool
}

// BoundObject is an object bound for another level's cell.
type BoundObject struct {
	Object      uint16
	Level, X, Y uint8
}

// SaveEntity is an entity record.
type SaveEntity struct {
	U, V        int16
	Objects     uint16
	Kind        uint8
	Flags       uint8
	Direction   uint8
	State       uint8
	Param, Data uint16
}

// Entity kinds (entities.md, Kinds).
const (
	entityPlayer     = 1
	entityCreature   = 2
	entityFlight     = 3
	entityBoulder    = 4
	entityTrap       = 5
	entityExitMarker = 6
	entityMouth      = 7
	entityConjured   = 8
	entityWallSpell  = 9
)

// Entity and cell feature bits (saves.md).
const (
	entityHostile  = 0x20
	cellUsed       = 0x01
	cellArmed      = 0x02
	cellTeleporter = 0x04
	cellTrigger    = 0x08
	cellDoor       = 0x10
	viewSeen       = 0x80
	viewVisible    = 0x40
	doorInMotion   = 0x10
	doorTowardOpen = 0x08
)

// SaveCell is a cell feature in use.
type SaveCell struct {
	X, Y    uint8
	Bits    uint8
	Objects uint16
	Data    uint8
}

// SaveTrigger is a trigger record.
type SaveTrigger struct {
	Counter      int16
	Threshold    uint8
	ProcA, ProcB uint8
}

// SaveObject is an object of the pool: its template, its value (a container's first object)
// and the next object in its list.
type SaveObject struct {
	Template    uint8
	Value, Next uint16
}

// SaveLevel is a level's state.
type SaveLevel struct {
	Number        int
	Rows, Columns int
	Entities      []SaveEntity
	Cells         []SaveCell
	// Terrain and View are 64 × 64: cell (x, y) at x*64 + y.
	Terrain, View []byte
	Triggers      []SaveTrigger
	// Doors are the doors' state bytes; Teleporters the teleporter records.
	Doors       []uint8
	Teleporters [][5]uint8
}

// TerrainAt returns the terrain of cell (x, y).
func (l *SaveLevel) TerrainAt(x, y int) byte { return l.Terrain[x*saveGridSide+y] }

// ViewAt returns the view flags of cell (x, y).
func (l *SaveLevel) ViewAt(x, y int) byte { return l.View[x*saveGridSide+y] }

// Cell returns the cell feature of (x, y), or nil.
func (l *SaveLevel) Cell(x, y int) *SaveCell {
	for i := range l.Cells {
		if c := &l.Cells[i]; int(c.X) == x && int(c.Y) == y {
			return c
		}
	}
	return nil
}

// ReadSave reads a save file of the original.
func ReadSave(data []byte) (*SaveFile, error) {
	if len(data) < saveLevelsAt+2 {
		return nil, fmt.Errorf("save: %d bytes, too short", len(data))
	}
	s := &SaveFile{Name: saveString(data[saveHeaderSize:saveLevelsAt])}
	at := saveLevelsAt + 2 // the count, which readers ignore
	for {
		if at+2 > len(data) {
			return nil, fmt.Errorf("save: the level states run past the end")
		}
		n := int16(binary.LittleEndian.Uint16(data[at:]))
		at += 2
		if n == -1 {
			break
		}
		if at+4 > len(data) {
			return nil, fmt.Errorf("save: the level states run past the end")
		}
		size := int(binary.LittleEndian.Uint32(data[at:]))
		at += 4
		if size != saveLevelSize || at+size > len(data) {
			return nil, fmt.Errorf("save: level %d's state is %d bytes, want %d", n, size, saveLevelSize)
		}
		// The state is the world block from the level state on: put it where the world has it.
		block := make([]byte, saveWorldSize)
		copy(block[saveLevelStateAt:], data[at:at+size])
		l, err := readSaveLevel(block)
		if err != nil {
			return nil, fmt.Errorf("save: level %d: %w", n, err)
		}
		if l.Number != int(n) {
			return nil, fmt.Errorf("save: level %d's state is of level %d", n, l.Number)
		}
		s.Levels = append(s.Levels, l)
		at += size
	}
	if want := at + saveWorldSize + savePlayerSize + saveObjectCount*saveObjectSize; len(data) != want {
		return nil, fmt.Errorf("save: %d bytes, want %d", len(data), want)
	}
	if err := s.readWorld(data[at : at+saveWorldSize]); err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}
	// The list may hold the current level as it was last left; the world block's is the one
	// that counts.
	s.Levels = slices.DeleteFunc(s.Levels, func(l *SaveLevel) bool { return l.Number == s.Current.Number })
	at += saveWorldSize
	s.Player = readSavePlayer(data[at : at+savePlayerSize])
	at += savePlayerSize
	for i := range s.Objects {
		b := data[at+i*saveObjectSize:]
		s.Objects[i] = SaveObject{Template: b[0], Value: le16(b[1:]), Next: le16(b[3:])}
	}
	return s, nil
}

func (s *SaveFile) readWorld(w []byte) error {
	s.Held = le16(w[swHeld:])
	copy(s.Timed[:], w[swTimed:])
	copy(s.Vars[:], w[swVars:])
	for i := range s.NPCs {
		copy(s.NPCs[i][:], w[swNPCs+i*npcRecordSize:])
	}
	for i := range s.Shared {
		s.Shared[i] = le16(w[swShared+2*i:])
	}
	for i := range boundCount {
		b := w[swBound+5*i:]
		if o := le16(b); o != saveNoObject {
			s.Bound = append(s.Bound, BoundObject{Object: o, Level: b[2], X: b[3], Y: b[4]})
		}
	}
	s.Body = readSaveEntity(w[swPlayer:])
	l, err := readSaveLevel(w)
	if err != nil {
		return err
	}
	s.Current = l
	return nil
}

// readSaveLevel reads the level state from a world block.
func readSaveLevel(w []byte) (*SaveLevel, error) {
	l := &SaveLevel{Number: int(w[slLevel]), Rows: int(w[slRows]), Columns: int(w[slRows+1])}
	if l.Rows > saveGridSide || l.Columns > saveGridSide {
		return nil, fmt.Errorf("%d×%d cells, over the maximum", l.Rows, l.Columns)
	}
	if l.Number >= levelCount {
		return nil, fmt.Errorf("level %d does not exist", l.Number)
	}
	// Entries past the count are left over from levels loaded before.
	n := min(int(w[slEntityCount])-1, slEntitySlots)
	for i := range max(n, 0) {
		if e := readSaveEntity(w[slEntities+i*saveEntitySize:]); e.Kind != 0 {
			l.Entities = append(l.Entities, e)
		}
	}
	for i := range slCellCount {
		b := w[slCells+i*saveCellSize:]
		if b[2]&cellUsed != 0 {
			l.Cells = append(l.Cells, SaveCell{X: b[0], Y: b[1], Bits: b[2], Objects: le16(b[3:]), Data: b[5]})
		}
	}
	l.Terrain = bytes.Clone(w[slTerrain : slTerrain+saveGridSide*saveGridSide])
	l.View = bytes.Clone(w[slView : slView+saveGridSide*saveGridSide])
	if n = int(w[slTriggers]); n > 75 {
		return nil, fmt.Errorf("%d triggers, over the maximum", n)
	}
	for i := range n {
		b := w[slTriggers+1+5*i:]
		l.Triggers = append(l.Triggers, SaveTrigger{Counter: int16(le16(b)), Threshold: b[2], ProcA: b[3], ProcB: b[4]})
	}
	if n = int(w[slDoors]); n > 75 {
		return nil, fmt.Errorf("%d doors, over the maximum", n)
	}
	for i := range n {
		l.Doors = append(l.Doors, w[slDoors+1+7*i+4])
	}
	if n = int(w[slTeleporters]); n > 100 {
		return nil, fmt.Errorf("%d teleporters, over the maximum", n)
	}
	for i := range n {
		var t [5]uint8
		copy(t[:], w[slTeleporters+1+5*i:])
		l.Teleporters = append(l.Teleporters, t)
	}
	return l, nil
}

func readSaveEntity(b []byte) SaveEntity {
	return SaveEntity{
		U: int16(le16(b[0x00:])), V: int16(le16(b[0x02:])), Objects: le16(b[0x08:]), Kind: b[0x0A],
		Flags: b[0x12], Direction: b[0x17], State: b[0x18], Param: le16(b[0x1C:]), Data: le16(b[0x1E:]),
	}
}

func readSavePlayer(b []byte) SavePlayer {
	p := SavePlayer{Name: saveString(b[spName : spName+26]), Level: b[spLevel], XP: le32(b[spXP:]), NextXP: le32(b[spNextXP:]),
		Fatigue: b[spFatigue], Confusion: int16(le16(b[spConfusion:])), Status: le16(b[spStatus:]),
		PackOpen: b[spPackOpen] != 0, VisionX: b[spVision], VisionY: b[spVision+1], IntervalBonus: int16(le16(b[spBonus:])),
		Counted: le16(b[spCounted:]), Look: le16(b[spLook:])}
	copy(p.Schools[:], b[spSchools:])
	copy(p.Weapons[:], b[spWeapons:])
	for i := range 4 {
		p.SchoolXP[i], p.SchoolNext[i] = le16(b[spSchoolXP+2*i:]), le16(b[spSchoolNext+2*i:])
		p.WeaponXP[i], p.WeaponNext[i] = le16(b[spWeaponXP+2*i:]), le16(b[spWeaponNext+2*i:])
	}
	pts := [5]*int16{&p.HP, &p.MaxHP, &p.MP, &p.MaxMP, &p.AC}
	for i, v := range pts {
		*v = int16(le16(b[spPoints+2*i:]))
	}
	copy(p.Attributes[:], b[spAttributes:])
	copy(p.MaxAttributes[:], b[spMaxAttrs:])
	for i := range p.Memorized {
		p.Memorized[i] = b[spSpells+i*spellSize]
	}
	copy(p.Selected[:], b[spSelected:])
	p.Portrait, p.Other, p.Talk = le16(b[spPortrait:]), le16(b[spPortrait+2:]), le16(b[spPortrait+4:])
	for i := range p.Slots {
		p.Slots[i] = le16(b[spSlots+2*i:])
	}
	for i := range p.Icons {
		p.Icons[i] = b[spIcons+i] != 0
	}
	return p
}

func le16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
func le32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }

// saveString returns b up to its first NUL, decoded from code page 437.
func saveString(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return decode437(b)
}
