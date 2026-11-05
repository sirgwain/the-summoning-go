package sim

import (
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// What the world records of the player's travels for the map (docs/spec/world.md, The engine's
// map): the links it has been through, and the order it first visited levels in.

// Link is a teleporter, exit, pit or trapdoor the player has travelled through, and where it
// arrived.
type Link struct {
	// Kind is the feature's kind (content.FeatureTeleporter, FeatureExit, FeaturePit,
	// FeatureTrapdoor).
	Kind string              `json:"kind"`
	From content.Destination `json:"from"`
	To   content.Destination `json:"to"`
}

// Links returns the links the player has travelled, in the order first travelled.
func (w *World) Links() []Link { return w.links }

// Visited returns the levels the player has been on, in the order first visited.
func (w *World) Visited() []string { return w.visited }

// recordLink records that the player went through feature f of level l and arrived where it
// stands now.
func (w *World) recordLink(l *Level, f *Feature) {
	p := w.player
	if p == nil {
		return
	}
	lk := Link{Kind: f.Kind, From: content.Destination{Level: l.ID, Cell: f.Cell},
		To: content.Destination{Level: w.current, Cell: p.Cell()}}
	if !slices.Contains(w.links, lk) {
		w.links = append(w.links, lk)
	}
}

// recordReturn records the way back from exit, pit or trapdoor f of level l, now that the player
// has gone through it: an exit or pit on or next to the arrival cell leading to l, to f's cell or
// next to it. A ladder down is a pit and the ladder up an exit, so each is the other's way back.
func (w *World) recordReturn(l *Level, f *Feature) {
	at := w.Current()
	arrived := w.player.Cell()
	for _, g := range at.Features {
		if g.Kind != content.FeatureExit && g.Kind != content.FeaturePit || g.To == nil || g.To.Level != l.ID ||
			!beside(g.Cell, arrived) || !beside(g.To.Cell, f.Cell) {
			continue
		}
		lk := Link{Kind: g.Kind, From: content.Destination{Level: at.ID, Cell: g.Cell}, To: *g.To}
		if !slices.Contains(w.links, lk) {
			w.links = append(w.links, lk)
		}
		return
	}
}

// beside reports whether cells a and b are the same or next to each other.
func beside(a, b content.Cell) bool {
	return a[0]-b[0] <= 1 && b[0]-a[0] <= 1 && a[1]-b[1] <= 1 && b[1]-a[1] <= 1
}

// visit records level id as visited, the first time.
func (w *World) visit(id string) {
	if !slices.Contains(w.visited, id) {
		w.visited = append(w.visited, id)
	}
}

// restoreVisited returns the visit order a state keeps, or, for a state without one, the levels
// it holds that have seen cells, in the order of their ids, with the current level.
func (w *World) restoreVisited(saved []string) []string {
	if len(saved) > 0 {
		return slices.Clone(saved)
	}
	var out []string
	for _, id := range sortedKeys(w.levels) {
		if id == w.current || slices.Contains(w.levels[id].seen, true) {
			out = append(out, id)
		}
	}
	return out
}

// HeardMouth is a mouth the player has heard and the messages it has said, in the order first
// heard.
type HeardMouth struct {
	Level    string       `json:"level"`
	Cell     content.Cell `json:"cell"`
	Messages []string     `json:"messages"`
}

// SeenTalker is a creature that talks, where the player last saw it.
type SeenTalker struct {
	Level    string       `json:"level"`
	Thing    string       `json:"thing"`
	Template string       `json:"template"`
	Cell     content.Cell `json:"cell"`
}

// Mouths returns the mouths the player has heard.
func (w *World) Mouths() []HeardMouth { return w.mouths }

// Talkers returns the creatures that talk the player has seen, where it last saw them.
func (w *World) Talkers() []SeenTalker { return w.talkers }

// heardMouth records that the player heard mouth f of level l say message.
func (w *World) heardMouth(l *Level, f *Feature, message string) {
	if w.player == nil || message == "" {
		return
	}
	for i := range w.mouths {
		h := &w.mouths[i]
		if h.Level == l.ID && h.Cell == f.Cell {
			if !slices.Contains(h.Messages, message) {
				h.Messages = append(h.Messages, message)
			}
			return
		}
	}
	w.mouths = append(w.mouths, HeardMouth{Level: l.ID, Cell: f.Cell, Messages: []string{message}})
}

// seeTalkers keeps where the player last saw each creature of level l that talks: a creature
// seen talking is recorded where it stands; one seen not talking, or missing from the cell it
// was recorded on now that the cell is seen, is forgotten.
func (w *World) seeTalkers(l *Level) {
	kept := w.talkers[:0]
	for _, s := range w.talkers {
		if s.Level == l.ID {
			var forget bool
			switch t := l.Thing(s.Thing); {
			case t != nil && l.Visible(t.Cell()):
				forget = !w.talks(t) // seen: kept, and moved below, while it talks
			default:
				forget = l.Visible(s.Cell) // gone from where it was seen
			}
			if forget {
				continue
			}
		}
		kept = append(kept, s)
	}
	w.talkers = kept
	for _, t := range l.Things {
		if !w.talks(t) || !l.Visible(t.Cell()) {
			continue
		}
		i := slices.IndexFunc(w.talkers, func(s SeenTalker) bool { return s.Level == l.ID && s.Thing == t.ID })
		s := SeenTalker{Level: l.ID, Thing: t.ID, Template: t.Template.ID, Cell: t.Cell()}
		if i < 0 {
			w.talkers = append(w.talkers, s)
		} else {
			w.talkers[i] = s
		}
	}
}
