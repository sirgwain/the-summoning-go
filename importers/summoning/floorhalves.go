package summoning

import (
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Floor halves (world.md, Floor halves).

const noCheck = -1

// floorChecks are the check directions, upper then lower, of walls 00h–0Eh, 0Fh and the closed
// doors 10h and 11h.
var floorChecks = [0x12][2]int{
	{7, 3}, {0, 4}, {1, 1}, {6, 2}, {5, 4}, {7, 3}, {0, 3}, {6, 3},
	{noCheck, 2}, {noCheck, 4}, {noCheck, 3}, {0, 3}, {7, 4}, {6, 3}, {7, 2},
	{noCheck, noCheck},
	{6, 2}, {0, 4},
}

// The shape a wall next to a closed door checks as.
var (
	checkBelowDoor = [wallShapes]int{0, 6, 2, 3, 7, 8, 6, 7, 8, 0xA, 0xA, 2, 0, 0xD, 0xE}
	checkLeftDoor  = [wallShapes]int{6, 1, 2, 8, 9, 5, 6, 0xA, 8, 9, 0xA, 0xB, 0xC, 5, 2}
	checkRightDoor = [wallShapes]int{0, 1, 6, 7, 4, 9, 6, 7, 0xA, 9, 0xA, 0xB, 0xC, 4, 0}
)

const (
	terrainDoorHorizontal = 0x10
	terrainDoorVertical   = 0x11
	terrainDoorMoving     = 0x15
)

// hazardShowsLower reports whether a hazard tile's lower floor half is drawn.
func hazardShowsLower(t int) bool {
	switch t {
	case 0x2B, 0x2E, 0x30, 0x33:
		return true
	}
	return false
}

// floorHalves computes which halves of each cell's floor are drawn, taking every cell on the
// map as visible, and for walls and closed doors the check directions whose sight shows each
// half in play (world.md, Visibility).
func floorHalves(l *Level, dirs [8]Direction) (halves, upper, lower content.Grid) {
	halves = content.NewGrid(l.Rows, l.Columns)
	upper, lower = content.NewGrid(l.Rows, l.Columns), content.NewGrid(l.Rows, l.Columns)
	counts := func(x, y, dir int) bool {
		if dir == noCheck {
			return false
		}
		nx, ny := x+int(dirs[dir].DX), y+int(dirs[dir].DY)
		t := int(l.TerrainAt(nx, ny))
		return t != TerrainOutside && (t < terrainHazard || t > terrainLastHazard)
	}
	for x := range l.Rows {
		for y := range l.Columns {
			t := int(l.TerrainAt(x, y))
			switch {
			case t == TerrainOutside:
				continue
			case t >= terrainDoorMoving && t <= terrainGemSocket:
				halves[x][y] = content.HalvesBoth
			case t >= terrainHazard && t <= terrainLastHazard:
				halves[x][y] = 0
				if hazardShowsLower(t) {
					halves[x][y] = content.HalfLower
				}
			case t <= terrainDoorVertical:
				checks := floorChecks[t]
				if t <= terrainLastWall {
					checks = floorChecks[wallChecksAs(l, x, y, t)]
				}
				h := 0
				if counts(x, y, checks[0]) {
					h |= content.HalfUpper
					upper[x][y] = checks[0]
				}
				if counts(x, y, checks[1]) {
					h |= content.HalfLower
					lower[x][y] = checks[1]
				}
				halves[x][y] = h
			default:
				halves[x][y] = 0
			}
		}
	}
	return halves, upper, lower
}

// doorSight gives every door that is not sealed the check directions of a closed door of its
// orientation, and lets its floor show whole when open, whatever its starting state.
func doorSight(cl *content.Level) {
	for _, f := range cl.Features {
		if f.Kind != content.FeatureDoor || f.SealedBy != "" {
			continue
		}
		checks := floorChecks[terrainDoorHorizontal]
		if strings.HasSuffix(f.Look, ".v") {
			checks = floorChecks[terrainDoorVertical]
		}
		x, y := f.Cell[0], f.Cell[1]
		cl.FloorHalves[x][y] = content.HalvesBoth
		cl.UpperSight[x][y], cl.LowerSight[x][y] = checks[0], checks[1]
	}
}

// wallChecksAs returns the shape whose check directions wall s at (x, y) uses.
func wallChecksAs(l *Level, x, y, s int) int {
	if l.TerrainAt(x+1, y) == terrainDoorHorizontal {
		return checkBelowDoor[s]
	}
	if l.TerrainAt(x, y-1) == terrainDoorVertical {
		s = checkLeftDoor[s]
	}
	if l.TerrainAt(x, y+1) == terrainDoorVertical {
		s = checkRightDoor[s]
	}
	return s
}
