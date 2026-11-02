package summoning

import (
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

// unseenShapes are each wall shape's (direction, shape) pairs: the shape a visible wall takes
// while its neighbor in that direction cannot be seen (world.md, Walls out of sight).
var unseenShapes = [wallShapes][][2]int{
	{{2, 0xE}, {4, 0xC}},
	{{2, 0xB}, {6, 0xC}},
	{{6, 0xE}, {4, 0xB}},
	{{0, 0xE}, {4, 0xD}},
	{{0, 0xC}, {2, 0xD}},
	{{0, 0xB}, {6, 0xD}},
	{{2, 2}, {4, 1}, {6, 0}},
	{{0, 0}, {2, 3}, {4, 4}},
	{{0, 2}, {4, 5}, {6, 3}},
	{{0, 1}, {2, 5}, {6, 4}},
	{{0, 6}, {2, 8}, {4, 9}, {6, 7}},
	nil, nil, nil, nil,
}

// cellSight is a cell's floor halves and the directions of the neighbors whose sight shows its
// upper and lower halves, NoFrame for none.
type cellSight struct {
	Halves, Upper, Lower int
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
// half in play (world.md, Visibility). Walls beside doors check as the doors start.
func floorHalves(l *Level, dirs [8]Direction) (halves, upper, lower content.Grid) {
	halves = content.NewGrid(l.Rows, l.Columns)
	upper, lower = content.NewGrid(l.Rows, l.Columns), content.NewGrid(l.Rows, l.Columns)
	set := func(x, y int, s cellSight) {
		halves[x][y], upper[x][y], lower[x][y] = s.Halves, s.Upper, s.Lower
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
			case t <= terrainLastWall:
				// The wall set's shapes give walls their check directions in play.
				halves[x][y] = checkSight(l, dirs, x, y, wallChecksAs(t,
					l.TerrainAt(x+1, y) == terrainDoorHorizontal,
					l.TerrainAt(x, y-1) == terrainDoorVertical,
					l.TerrainAt(x, y+1) == terrainDoorVertical)).Halves
			case t <= terrainDoorVertical:
				set(x, y, checkSight(l, dirs, x, y, t))
			default:
				halves[x][y] = 0
			}
		}
	}
	return halves, upper, lower
}

// checkSight returns the floor halves and check directions of the cell at (x, y) checking as
// terrain t (a wall shape or a closed door), every cell on the map taken as visible.
func checkSight(l *Level, dirs [8]Direction, x, y, t int) cellSight {
	counts := func(dir int) bool {
		if dir == noCheck {
			return false
		}
		nt := int(l.TerrainAt(x+int(dirs[dir].DX), y+int(dirs[dir].DY)))
		return nt != TerrainOutside && (nt < terrainHazard || nt > terrainLastHazard)
	}
	s := cellSight{Upper: content.NoFrame, Lower: content.NoFrame}
	checks := floorChecks[t]
	if counts(checks[0]) {
		s.Halves |= content.HalfUpper
		s.Upper = checks[0]
	}
	if counts(checks[1]) {
		s.Halves |= content.HalfLower
		s.Lower = checks[1]
	}
	return s
}

// doorSight gives every door that is not sealed the check directions of a closed door of its
// orientation, and lets its floor show whole when open, whatever its starting state. Each wall
// beside a door that can change its check directions gets, for every combination of those doors
// closed, the shape each shape checks as (world.md, Import).
func (lb *levelBuilder) doorSight() {
	cl, l := lb.cl, lb.l
	vertical := func(c content.Cell) (bool, bool) {
		f, p := lb.features[c], lb.doors[c]
		if f == nil || f.Kind != content.FeatureDoor || p == nil {
			return false, false
		}
		return p.Extra[0]&doorVertical != 0, true
	}
	for _, f := range cl.Features {
		if f.Kind != content.FeatureDoor || f.SealedBy != "" {
			continue
		}
		checks := floorChecks[terrainDoorHorizontal]
		if v, _ := vertical(f.Cell); v {
			checks = floorChecks[terrainDoorVertical]
		}
		x, y := f.Cell[0], f.Cell[1]
		cl.FloorHalves[x][y] = content.HalvesBoth
		cl.UpperSight[x][y], cl.LowerSight[x][y] = checks[0], checks[1]
	}
	for x := range l.Rows {
		for y := range l.Columns {
			t := int(l.TerrainAt(x, y))
			if t == TerrainOutside || t > terrainLastWall {
				continue
			}
			// The doors that can change this wall's checks, and which way each does.
			type side struct {
				cell               content.Cell
				below, left, right bool
			}
			var sides []side
			if v, ok := vertical(content.Cell{x + 1, y}); ok && !v {
				sides = append(sides, side{cell: content.Cell{x + 1, y}, below: true})
			}
			if v, ok := vertical(content.Cell{x, y - 1}); ok && v {
				sides = append(sides, side{cell: content.Cell{x, y - 1}, left: true})
			}
			if v, ok := vertical(content.Cell{x, y + 1}); ok && v {
				sides = append(sides, side{cell: content.Cell{x, y + 1}, right: true})
			}
			if len(sides) == 0 {
				continue
			}
			ds := content.DoorSight{Cell: content.Cell{x, y}}
			for m := range 1 << len(sides) {
				var below, left, right bool
				for i, sd := range sides {
					if m&(1<<i) != 0 {
						below, left, right = below || sd.below, left || sd.left, right || sd.right
					}
				}
				as := make([]int, wallShapes)
				for s := range as {
					as[s] = wallChecksAs(s, below, left, right)
				}
				ds.As = append(ds.As, as)
			}
			for _, sd := range sides {
				ds.Doors = append(ds.Doors, sd.cell)
			}
			cl.DoorSight = append(cl.DoorSight, ds)
		}
	}
}

// wallChecksAs returns the shape whose check directions wall s uses, with a closed horizontal
// door below it (at (x + 1, y)) and closed vertical doors to its left (at (x, y − 1)) and right
// (at (x, y + 1)) as given.
func wallChecksAs(s int, below, left, right bool) int {
	if below {
		return checkBelowDoor[s]
	}
	if left {
		s = checkLeftDoor[s]
	}
	if right {
		s = checkRightDoor[s]
	}
	return s
}
