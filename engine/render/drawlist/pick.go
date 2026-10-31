package drawlist

import (
	"image"
	"slices"
	"sort"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// SimScene returns a simulated level as it is now.
func SimScene(l *sim.Level) Scene {
	s := Scene{Level: l.Def, Floor: l.Floor, Wall: l.Wall}
	for _, f := range l.Features {
		s.Features = append(s.Features, Feature{Cell: f.Cell, Kind: f.Kind, Look: f.Look, State: f.LookState(), Set: f.Set})
	}
	cells := make([]content.Cell, 0, len(l.Objects))
	for c := range l.Objects {
		cells = append(cells, c)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i][0] != cells[j][0] {
			return cells[i][0] < cells[j][0]
		}
		return cells[i][1] < cells[j][1]
	})
	for _, c := range cells {
		for _, o := range l.Objects[c] {
			s.Objects = append(s.Objects, Object{Cell: c, Item: o.Item})
		}
	}
	for _, t := range l.Things {
		s.Things = append(s.Things, Thing{ID: t.ID, Template: t.Template.ID, U: t.U, V: t.V, Facing: t.Facing})
	}
	return s
}

// CellAt returns the cell whose ground holds view pixel pt: the inverse of RefPoint, taken to
// the cell. A cell's ground is the diamond Diamond returns.
func CellAt(pt image.Point) content.Cell {
	// RefPoint gives x = 16 + v − u and y = (u + v) / 2, so 2u = 2y − (x − 16) and
	// 2v = 2y + (x − 16).
	d := pt.X - 16
	return content.Cell{floorDiv(2*pt.Y-d, 2*CellUnits), floorDiv(2*pt.Y+d, 2*CellUnits)}
}

// Diamond returns the view pixels of cell c's four ground corners, clockwise from the top: the
// reference points of its world corners.
func Diamond(c content.Cell) [4]image.Point {
	u, v := CellUnits*c[0], CellUnits*c[1]
	return [4]image.Point{
		RefPoint(u, v),
		RefPoint(u, v+CellUnits),
		RefPoint(u+CellUnits, v+CellUnits),
		RefPoint(u+CellUnits, v),
	}
}

// ThingAt returns the thing drawn topmost at view pixel pt, judged by its sprite's pixels, and
// the command that draws it.
func ThingAt(p *content.Pack, cmds []Cmd, pt image.Point) (string, Cmd, bool) {
	for _, c := range slices.Backward(cmds) {
		if c.Thing == "" {
			continue
		}
		b := p.Sprites[c.Sprite]
		r := b.FrameRect(c.Frame)
		at := pt.Sub(image.Pt(c.X, c.Y)).Add(r.Min)
		if !at.In(r) || b.Sheet == nil {
			continue
		}
		if b.Transparent != nil && int(b.Sheet.ColorIndexAt(at.X, at.Y)) == *b.Transparent {
			continue
		}
		return c.Thing, c, true
	}
	return "", Cmd{}, false
}
