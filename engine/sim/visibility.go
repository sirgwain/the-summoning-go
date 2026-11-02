package sim

import "github.com/sirgwain/the-summoning-go/engine/content"

// Visible reports whether cell c of the level is visible to the player. A level with no player
// on it has every cell visible.
func (l *Level) Visible(c content.Cell) bool {
	if l.visible == nil {
		return true
	}
	return l.InBounds(c) && l.visible[l.index(c)]
}

// Seen reports whether cell c of the level has ever been visible to the player.
func (l *Level) Seen(c content.Cell) bool {
	return l.seen != nil && l.InBounds(c) && l.seen[l.index(c)]
}

// Sighted reports whether the level's visibility is the player's, rather than every cell.
func (l *Level) Sighted() bool { return l.visible != nil }

func (l *Level) index(c content.Cell) int { return c[0]*l.Def.Size[1] + c[1] }

// see brings the player's vision cell and the current level's visibility up to date: the vision
// cell follows the player, except while it stands in a door's cell (docs/spec/world.md,
// Visibility).
func (w *World) see() {
	p := w.player
	if p == nil {
		return
	}
	l := w.Current()
	c := p.Cell()
	if f := l.byCell[c]; f == nil || f.Kind != content.FeatureDoor {
		p.vision = c
	}
	l.computeVisibility(p.vision)
	l.takeShapes(w.pack.Walls[l.Def.Walls])
}

// takeShapes finds the shape each visible wall is taken as while walls beside it cannot be seen
// (docs/spec/world.md, Walls out of sight). The level's own shapes stay as they are.
func (l *Level) takeShapes(set *content.WallSet) {
	clear(l.taken)
	if set == nil || l.visible == nil {
		return
	}
	for x := range l.Def.Size[0] {
		for y := range l.Def.Size[1] {
			c := content.Cell{x, y}
			own := l.Wall[x][y]
			if own == content.NoFrame || !l.visible[l.index(c)] {
				continue
			}
			if s := set.Taken(own, c, Dirs, l.InBounds, l.Visible); s != own {
				if l.taken == nil {
					l.taken = map[content.Cell]int{}
				}
				l.taken[c] = s
			}
		}
	}
}

// Shape returns the shape cell c's wall is taken as: its own, unless walls beside it cannot be
// seen (docs/spec/world.md, Walls out of sight).
func (l *Level) Shape(c content.Cell) int {
	if s, ok := l.taken[c]; ok {
		return s
	}
	return l.Wall[c[0]][c[1]]
}

// Taken returns the visible walls taken as another shape than their own, by cell.
func (l *Level) Taken() map[content.Cell]int { return l.taken }

// computeVisibility marks the cells visible from cell from: every open cell joined to it through
// open cells, each step to any of the eight neighbors, and every blocked cell next to one of
// them. The cell from counts as open.
func (l *Level) computeVisibility(from content.Cell) {
	n := l.Def.Size[0] * l.Def.Size[1]
	if l.visible == nil {
		l.visible, l.seen = make([]bool, n), make([]bool, n)
	} else {
		clear(l.visible)
	}
	if !l.InBounds(from) {
		return
	}
	open := make([]bool, n)
	stack := []content.Cell{from}
	open[l.index(from)] = true
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		l.visible[l.index(c)] = true
		for _, d := range Dirs {
			nb := content.Cell{c[0] + d[0], c[1] + d[1]}
			if !l.InBounds(nb) {
				continue
			}
			i := l.index(nb)
			if l.Blocked(nb) {
				l.visible[i] = true
				continue
			}
			if !open[i] {
				open[i] = true
				stack = append(stack, nb)
			}
		}
	}
	for i, v := range l.visible {
		if v {
			l.seen[i] = true
		}
	}
}
