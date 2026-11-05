package sim

import "github.com/sirgwain/the-summoning-go/engine/content"

// Tactics constants (docs/spec/creatures.md, Tactics).
const (
	// wayWindow is how many cells, on each axis, a creature looks for its way around walls.
	wayWindow = 24
	// keepBand is how many cells past its keep_distance a creature lets the player be before it
	// closes in.
	keepBand = 2
	// keepCastChance is the 1 in n chance a lined-up creature keeping its distance casts.
	keepCastChance = 4
	// backOffSteps and backOffRange: a creature backing off after its blow backs away
	// backOffSteps to backOffSteps+backOffRange-1 steps.
	backOffSteps = 2
	backOffRange = 2
	// flankSteps is the most steps a creature spends making for the cell behind the player.
	flankSteps = 20
)

// recover starts creature t's recovery from its blow, as its template's after_blow says; a
// flanker whose blow was already from behind rests.
func (w *World) recover(t *Thing) {
	switch t.Template.AfterBlow {
	case content.AfterBlowBackOff:
		t.Recovery, t.RecoverySteps = content.AfterBlowBackOff, backOffSteps+w.rng.IntN(backOffRange)
		t.walkDir, t.walkCount = restDir, 0
	case content.AfterBlowFlank:
		if p := w.player; p != nil && behind(t.Facing, p.Facing) {
			t.Recovery, t.RecoverySteps = "", 0
			t.walkDir, t.walkCount = restDir, 10+w.rng.IntN(5)
			return
		}
		t.Recovery, t.RecoverySteps = content.AfterBlowFlank, flankSteps
		t.walkDir, t.walkCount = restDir, 0
	default:
		t.Recovery, t.RecoverySteps = "", 0
		t.walkDir, t.walkCount = restDir, 10+w.rng.IntN(5)
	}
}

// hostileWalk picks hostile creature t's walk of one step while the player is visible: its
// recovery from a blow, keeping its distance, or heading for the player by the way around walls.
// It sets t's walk and whether it steps facing the player.
func (w *World) hostileWalk(l *Level, t *Thing) {
	p := w.player
	t.walkCount, t.facePlayer = 1, false
	face := func(d int) { t.walkDir, t.facePlayer = d, true }
	switch t.Recovery {
	case content.AfterBlowBackOff:
		if t.RecoverySteps--; t.RecoverySteps <= 0 {
			t.Recovery, t.RecoverySteps = "", 0
		}
		if d, ok := w.awayFrom(l, t); ok {
			face(d)
		} else {
			t.walkDir = restDir
		}
		return
	case content.AfterBlowFlank:
		if t.RecoverySteps--; t.RecoverySteps <= 0 {
			t.Recovery, t.RecoverySteps = "", 0
		}
		pc := p.Cell()
		f := Dirs[2*(p.Facing/2)]
		goal := content.Cell{pc[0] - f[0], pc[1] - f[1]}
		if t.Cell() == goal {
			if d, ok := w.behindLine(t); ok {
				t.walkDir = d
				return
			}
			t.Recovery, t.RecoverySteps = "", 0
			break
		}
		if w.creatureStands(l, t, goal) {
			if d, ok := w.wayStep(l, t, goal, goal[0]*CellUnits+CellUnits/2, goal[1]*CellUnits+CellUnits/2, &pc); ok {
				t.walkDir = d
				return
			}
		}
		t.Recovery, t.RecoverySteps = "", 0
		t.walkDir, t.walkCount = restDir, 10+w.rng.IntN(5)
		return
	}
	if n := t.Template.KeepDistance; n > 0 {
		du, dv := p.U-t.U, p.V-t.V
		gap, small := max(abs(du), abs(dv)), min(abs(du), abs(dv))
		switch {
		case gap < CellUnits*n:
			if d, ok := w.awayFrom(l, t); ok {
				face(d)
				return
			}
		case gap > CellUnits*(n+keepBand):
		case small < stepOf(t):
			t.walkDir, t.Facing = restDir, 2*towards(t.U, t.V, p.U, p.V)
			if sp := w.pack.Spells[t.Template.Spell]; sp != nil && w.chance(1, keepCastChance) {
				t.Pose = PoseAttack
				w.castSpellOn(l, creatureCaster(t), sp, w.roll(t.Template.SpellPower))
				t.walkCount = 10 + w.rng.IntN(5)
			}
			return
		default:
			face(lineUp(du, dv))
			return
		}
	}
	t.walkDir, _ = w.wayStep(l, t, p.Cell(), p.U, p.V, nil)
}

// behindLine returns the four-way direction creature t, behind the player, steps in to come onto
// the line of the player's facing; false when it is on it, its gap across that line below its
// step.
func (w *World) behindLine(t *Thing) (int, bool) {
	p := w.player
	du, dv := p.U-t.U, p.V-t.V
	if Dirs[2*(p.Facing/2)][0] != 0 {
		if abs(dv) < stepOf(t) {
			return 0, false
		}
		return awayDir(-dv, 1, 3), true
	}
	if abs(du) < stepOf(t) {
		return 0, false
	}
	return awayDir(du, 0, 2), true
}

// lineUp is the four-way direction that closes the smaller of the gaps (du, dv).
func lineUp(du, dv int) int {
	if abs(du) >= abs(dv) {
		if dv > 0 {
			return 1
		}
		return 3
	}
	if du < 0 {
		return 0
	}
	return 2
}

// awayFrom returns the four-way direction creature t backs away from the player in: directly
// away along the larger gap's axis, else away along the other axis (the positive way first when
// that gap is 0), the first whose neighboring cell it may stand in; false when none.
func (w *World) awayFrom(l *Level, t *Thing) (int, bool) {
	p := w.player
	du, dv := p.U-t.U, p.V-t.V
	var ways []int
	if abs(du) >= abs(dv) {
		ways = append(ways, awayDir(du, 2, 0))
		switch {
		case dv > 0:
			ways = append(ways, 3)
		case dv < 0:
			ways = append(ways, 1)
		default:
			ways = append(ways, 1, 3)
		}
	} else {
		ways = append(ways, awayDir(dv, 1, 3))
		switch {
		case du > 0:
			ways = append(ways, 0)
		case du < 0:
			ways = append(ways, 2)
		default:
			ways = append(ways, 2, 0)
		}
	}
	own := t.Cell()
	for _, d := range ways {
		if w.creatureStands(l, t, content.Cell{own[0] + Dirs[2*d][0], own[1] + Dirs[2*d][1]}) {
			return d, true
		}
	}
	return 0, false
}

// awayDir is up when the gap is below 0 (the player on the negative side), else down.
func awayDir(gap, up, down int) int {
	if gap < 0 {
		return up
	}
	return down
}

// wayStep returns the four-way direction creature t steps in toward goal, whose point is
// (gu, gv), by the shortest way around walls, not standing in avoid; false, with the plain
// heading, when no way reaches it (docs/spec/creatures.md, Tactics, Finding the way).
func (w *World) wayStep(l *Level, t *Thing, goal content.Cell, gu, gv int, avoid *content.Cell) (int, bool) {
	h := towards(t.U, t.V, gu, gv)
	own := t.Cell()
	if own == goal {
		return h, true
	}
	dist := w.wayTo(l, t, goal, avoid)
	d0 := dist(own)
	if d0 < 0 {
		return h, false
	}
	if dist(content.Cell{own[0] + Dirs[2*h][0], own[1] + Dirs[2*h][1]}) == d0-1 {
		return h, true
	}
	for _, d := range []int{t.walkDir, lineUp(gu-t.U, gv-t.V), 0, 1, 2, 3} {
		if d == restDir {
			continue
		}
		c := content.Cell{own[0] + Dirs[2*d][0], own[1] + Dirs[2*d][1]}
		if dist(c) == d0-1 {
			return d, true
		}
	}
	return h, false
}

// wayTo counts the way, in four-way cells, from each cell within wayWindow of creature t to goal,
// over the cells t's ground point may stand in but avoid; it returns the count of a cell, or −1
// for a cell with no way or outside the window. It counts out from goal until it reaches t.
func (w *World) wayTo(l *Level, t *Thing, goal content.Cell, avoid *content.Cell) func(content.Cell) int {
	own := t.Cell()
	const side = 2*wayWindow + 1
	var counts [side * side]int16
	index := func(c content.Cell) int {
		x, y := c[0]-own[0]+wayWindow, c[1]-own[1]+wayWindow
		if x < 0 || y < 0 || x >= side || y >= side {
			return -1
		}
		return x*side + y
	}
	get := func(c content.Cell) int {
		if i := index(c); i >= 0 {
			return int(counts[i]) - 1
		}
		return -1
	}
	gi := index(goal)
	if gi < 0 {
		return get
	}
	counts[gi] = 1
	queue := []content.Cell{goal}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == own {
			break
		}
		n := counts[index(c)]
		for d := range 4 {
			nc := content.Cell{c[0] + Dirs[2*d][0], c[1] + Dirs[2*d][1]}
			i := index(nc)
			if i < 0 || counts[i] != 0 {
				continue
			}
			if nc != own && (avoid != nil && nc == *avoid || !w.creatureStands(l, t, nc)) {
				continue
			}
			counts[i] = n + 1
			queue = append(queue, nc)
		}
	}
	return get
}
