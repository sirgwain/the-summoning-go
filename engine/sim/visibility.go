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
}

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
