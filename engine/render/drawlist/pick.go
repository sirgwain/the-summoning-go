package drawlist

import (
	"fmt"
	"image"
	"slices"
	"sort"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// SimScene returns a simulated level of world w as it is now, with the player when it is on the
// level, showing what the player sees when the level has the player's visibility.
func SimScene(w *sim.World, l *sim.Level) Scene {
	s := Scene{Level: l.Def, Floor: l.Floor, Wall: l.Wall}
	if l.Sighted() {
		s.Visible = l.Visible
	}
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
		th := Thing{ID: t.ID, Template: t.Template.ID, U: t.U, V: t.V, Facing: t.Facing, Frame: t.Frame}
		if w.Step() < t.ShownUntil {
			th.Shown = t.Shown
		}
		switch {
		case t.Rolling != nil:
			th.Anim, th.Phase = fmt.Sprintf("roll.%d", t.Rolling.Direction), t.Phase
		case t.Template.Kind == content.ThingCreature && t.HitPoints <= 0:
			th.Anim = "dead"
		case t.Pose != "":
			th.Anim = fmt.Sprintf("%s.%d", t.Pose, t.Facing)
		case t.Moving:
			th.Anim, th.Phase = fmt.Sprintf("walk.%d", t.Facing), t.Phase
		}
		s.Things = append(s.Things, th)
	}
	for _, f := range l.Flights {
		if f.Spell != nil {
			s.Flights = append(s.Flights, Flight{Sprite: f.Spell.Sprite, Frame: f.Spell.Frame, U: f.U, V: f.V})
			continue
		}
		fl := Flight{Item: f.Object.Item, U: f.U, V: f.V}
		if it := w.Pack().Items[f.Object.Item]; it != nil && it.Flying != nil && (f.Fired || !it.Flying.FiredOnly) {
			fl.Sprite, fl.Frame = it.Flying.Sprite, (f.Direction+1)%8
		}
		s.Flights = append(s.Flights, fl)
	}
	if p := w.Player(); p != nil && w.Current() == l {
		th := Thing{ID: sim.PlayerID, Template: p.Template.ID, U: p.U, V: p.V, Facing: p.Facing, Sprite: p.Look}
		if w.Step() < p.ShownUntil {
			th.Shown = p.Shown
		}
		for hand := range 2 {
			if o := p.Slots[sim.SlotRightHand+hand]; o != nil && !p.Dead && p.Look == "" && !w.DeepWater() {
				th.Held[hand] = o.Item
			}
		}
		switch {
		case p.Pose != "":
			th.Anim = fmt.Sprintf("%s.%d", p.Pose, p.Facing)
		case p.Walking():
			th.Anim, th.Phase = fmt.Sprintf("walk.%d", p.Facing), p.Moves
		}
		s.Things = append(s.Things, th)
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
