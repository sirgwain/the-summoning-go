package sim

import (
	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Dirs are the world steps of the eight directions, in cells (docs/spec/README.md, Directions).
var Dirs = [8][2]int{{-1, 0}, {-1, 1}, {0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}}

// Movement constants (docs/spec/entities.md).
const (
	// backOff is how far a move shrinks per axis each time it overlaps another thing.
	backOff = 2
	// slide is how far the player slides along a diagonal wall.
	slide = 4
	// pushTicks is how much longer the player waits after a push.
	pushTicks = 3
)

// stepsMS converts a duration in milliseconds to steps, at least 1.
func stepsMS(ms int) int64 {
	return max(1, int64(ms*StepsPerSecond+500)/1000)
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// toward0 moves n by d toward zero, stopping at zero.
func toward0(n, d int) int {
	switch {
	case n > d:
		return n - d
	case n < -d:
		return n + d
	}
	return 0
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// cellOf returns the cell holding world point (u, v).
func cellOf(u, v int) content.Cell {
	return content.Cell{floorDiv(u, CellUnits), floorDiv(v, CellUnits)}
}

// body is something that takes up room in a level: a thing, or the player when thing is nil.
type body struct {
	thing *Thing
	u, v  int
	box   content.Box
}

// viewPoint returns where world point (u, v) is drawn, in view pixels, up to the view's origin
// (docs/spec/world.md, Projection).
func viewPoint(u, v int) (x, y int) {
	return v - u, floorDiv(u, 2) + floorDiv(v, 2)
}

// overlaps reports whether box a with its ground point at (au, av) overlaps box b with its ground
// point at (bu, bv) (docs/spec/entities.md, Collision boxes).
func overlaps(au, av int, a content.Box, bu, bv int, b content.Box) bool {
	ax, ay := viewPoint(au, av)
	bx, by := viewPoint(bu, bv)
	return a.Overlaps(ax, ay, b, bx, by)
}

// bodies returns what takes up room in level l, the player first.
func (w *World) bodies(l *Level) []body {
	var out []body
	if p := w.player; p != nil && w.current == l.ID {
		out = append(out, body{u: p.U, v: p.V, box: p.Template.Bounds()})
	}
	for _, t := range l.Things {
		out = append(out, body{thing: t, u: t.U, v: t.V, box: t.Template.Bounds()})
	}
	return out
}

// blocker returns the first body of l other than self (nil self is the player) whose box
// overlaps box at (u, v), leaving out bodies in skip.
func (w *World) blocker(l *Level, self *Thing, playerSelf bool, u, v int, box content.Box, skip map[*Thing]bool) *body {
	for _, b := range w.bodies(l) {
		switch {
		case b.thing == nil && playerSelf, b.thing != nil && b.thing == self, b.thing != nil && skip[b.thing]:
			continue
		}
		if overlaps(u, v, box, b.u, b.v, b.box) {
			return &b
		}
	}
	return nil
}

// thingFits reports whether thing t can stand at (u, v) as far as the terrain goes: boulders and
// traps test their footprint's corners, others their ground point.
func (w *World) thingFits(l *Level, t *Thing, u, v int) bool {
	switch t.Template.Kind {
	case content.ThingBoulder, content.ThingTrap:
		r := t.Template.Size()/2 - 1
		for _, d := range [4][2]int{{-r, -r}, {-r, r}, {r, -r}, {r, r}} {
			if l.Blocked(cellOf(u+d[0], v+d[1])) {
				return false
			}
		}
		return true
	}
	return !l.Blocked(cellOf(u, v))
}

// pushable reports whether the player pushes t: a trap or a corpse.
func pushable(t *Thing) bool {
	return t.Template.Kind == content.ThingTrap || t.Template.Kind == content.ThingConjured || t.Template.Kind == content.ThingCreature && t.HitPoints <= 0
}

// push moves thing t by (du, dv), first pushing the pushable things in its way, and reports
// whether it moved (docs/spec/entities.md, Pushing). What moves is drawn gliding there by step
// end, with the player pushing it.
func (w *World) push(l *Level, t *Thing, du, dv, depth int, end int64) bool {
	if depth >= maxDepth || !w.thingFits(l, t, t.U+du, t.V+dv) {
		return false
	}
	for range maxDepth {
		b := w.blocker(l, t, true, t.U+du, t.V+dv, t.Template.Bounds(), nil)
		if b == nil {
			t.glide.start(t.U, t.V, w.step, end)
			w.moveThingBy(l, t, du, dv)
			return true
		}
		if b.thing == nil || !pushable(b.thing) || !w.push(l, b.thing, du, dv, depth+1, end) {
			return false
		}
	}
	return false
}

// moveThingBy moves thing t by (du, dv) and runs the effects of the cells it leaves and enters.
func (w *World) moveThingBy(l *Level, t *Thing, du, dv int) {
	from := t.Cell()
	t.U, t.V = t.U+du, t.V+dv
	t.Phase++
	to := t.Cell()
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: to, Thing: t.ID, Detail: "moved"})
	if from != to {
		w.weigh(l, from, -t.weight(), 0)
		w.weigh(l, to, t.weight(), 0)
		w.entered(l, t, 0)
	}
}

// startBoulder starts boulder t rolling along direction d; a resting one stops when blocked, a
// rolling one only turns.
func (w *World) startBoulder(t *Thing, d int, shuttle bool) {
	if t.Rolling != nil {
		t.Rolling.Direction = d
		return
	}
	t.Rolling = &content.Rolling{Direction: d, Shuttle: shuttle}
	t.next = w.step
}

// rollBoulders moves the rolling boulders of l whose update has come (docs/spec/entities.md,
// Boulders).
func (w *World) rollBoulders(l *Level) {
	for _, t := range append([]*Thing(nil), l.Things...) {
		if t.Rolling == nil || w.step < t.next || l.Thing(t.ID) != t {
			continue
		}
		t.next = w.step + stepsMS(t.Template.IntervalMS)
		t.glide.start(t.U, t.V, w.step, t.next)
		d := t.Rolling.Direction
		speed := t.Template.Speed
		if speed == 0 {
			speed = 8
		}
		du, dv := Dirs[d][0]*speed, Dirs[d][1]*speed
		b := w.blocker(l, t, false, t.U+du, t.V+dv, t.Template.Bounds(), nil)
		if b == nil && w.thingFits(l, t, t.U+du, t.V+dv) {
			w.moveThingBy(l, t, du, dv)
			continue
		}
		if b != nil {
			hurt := w.roll(content.Dice{Count: 3, Sides: 4})
			if b.thing == nil {
				w.hurtPlayerAloud(hurt, combat.Physical, "boulder")
			} else {
				w.hurtThing(l, b.thing, hurt, combat.Physical)
				if b.thing.Template.Kind == content.ThingBoulder {
					w.startBoulder(b.thing, d, false)
				}
			}
		}
		if t.Rolling.Shuttle {
			t.Rolling.Direction = (d + 4) % 8
		} else {
			t.Rolling = nil
			w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "stopped"})
		}
	}
}
