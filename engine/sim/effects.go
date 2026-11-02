package sim

import "github.com/sirgwain/the-summoning-go/engine/content"

// View effects and sparkles (docs/spec/scenes.md, View effects and Sparkles). The world names a
// flipbook in an event and goes on; the presentation pauses it while the flipbook plays. What
// the game does after an effect (a level change, a socket's wall opening, the rest of a spell)
// waits for it: it runs at the start of the next update, which the presentation holds back until
// the flipbook ends.

// playEffect plays view effect flipbook id, placed by thing (or by cell c when the thing is
// gone or there is none). It plays nothing for an empty id.
func (w *World) playEffect(l *Level, id string, c content.Cell, thing string) {
	if id == "" {
		return
	}
	w.emit(Event{Kind: EventEffect, Level: l.ID, Cell: c, Thing: thing, Detail: id})
	w.effectPlayed = true
}

// playSparkle plays sparkle flipbook id, placed by thing, or by feature at cell c.
func (w *World) playSparkle(l *Level, id string, c content.Cell, thing, feature string) {
	if id == "" {
		return
	}
	w.emit(Event{Kind: EventSparkle, Level: l.ID, Cell: c, Thing: thing, Feature: feature, Detail: id})
}

// playerRef is the player's id, or "" when the world has no player.
func (w *World) playerRef() string {
	if w.player == nil {
		return ""
	}
	return PlayerID
}

// playerEffect plays a view effect at the player.
func (w *World) playerEffect(id string) {
	if p := w.player; p != nil {
		w.playEffect(w.Current(), id, p.Cell(), PlayerID)
	}
}

// playerSparkle plays a sparkle at the player.
func (w *World) playerSparkle(id string) {
	if p := w.player; p != nil {
		w.playSparkle(w.Current(), id, p.Cell(), PlayerID, "")
	}
}

// afterEffect runs f now, or at the start of the next update when a view effect played in this
// one, so that it follows the effect.
func (w *World) afterEffect(f func()) {
	if w.effectPlayed {
		w.after = append(w.after, f)
		return
	}
	f()
}

// runAfter runs what waited for the last update's view effects.
func (w *World) runAfter() {
	w.effectPlayed = false
	after := w.after
	w.after = nil
	for _, f := range after {
		f()
	}
}

// Waiting reports whether something waits for a view effect: a level change, a hook, the rest
// of a spell. It runs on the next update.
func (w *World) Waiting() bool { return len(w.after) > 0 }
