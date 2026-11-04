package sim

// maxGlide is the farthest, in world units along either axis, a thing is drawn gliding over in
// one update; a longer move (a teleporter, a placement) is drawn as a jump.
const maxGlide = CellUnits / 2

// glide is where a thing was at the start of its last update, at step at, so that it can be
// drawn gliding from there to where it is by its next update (docs/spec/entities.md, Drawn
// between updates). The zero glide draws the thing where it is.
type glide struct {
	u, v int
	at   int64
	set  bool
}

// start records (u, v) as where the thing is at the start of its update at step.
func (g *glide) start(u, v int, step int64) { *g = glide{u: u, v: v, at: step, set: true} }

// drawn returns where a thing now at (u, v), next updated at step next, is drawn at step: the
// part of the way from where it was to (u, v) that the steps since its last update are of the
// steps between its updates, each coordinate rounded toward where it was.
func (g glide) drawn(u, v int, next, step int64) (int, int) {
	span := next - g.at
	if !g.set || span <= 0 || step >= next || abs(u-g.u) > maxGlide || abs(v-g.v) > maxGlide {
		return u, v
	}
	k := max(0, step-g.at)
	toward := func(from, to int) int { return from + int(int64(to-from)*k/span) }
	return toward(g.u, u), toward(g.v, v)
}

// gliding reports whether a thing at (u, v) is drawn elsewhere at step.
func (g glide) gliding(u, v int, next, step int64) bool {
	du, dv := g.drawn(u, v, next, step)
	return du != u || dv != v
}

// Drawn returns where thing t is drawn at step (docs/spec/entities.md, Drawn between updates).
func (t *Thing) Drawn(step int64) (u, v int) { return t.glide.drawn(t.U, t.V, t.next, step) }

// Drawn returns where the player is drawn at step.
func (p *Player) Drawn(step int64) (u, v int) { return p.glide.drawn(p.U, p.V, p.next, step) }

// Drawn returns where flight f is drawn at step.
func (f *Flight) Drawn(step int64) (u, v int) { return f.glide.drawn(f.U, f.V, f.next, step) }

// Gliding reports whether anything on the current level is drawn, at the world's step, between
// where it was and where it is.
func (w *World) Gliding() bool {
	l, s := w.Current(), w.step
	if p := w.player; p != nil && p.glide.gliding(p.U, p.V, p.next, s) {
		return true
	}
	for _, t := range l.Things {
		if t.glide.gliding(t.U, t.V, t.next, s) {
			return true
		}
	}
	for _, f := range l.Flights {
		if f.glide.gliding(f.U, f.V, f.next, s) {
			return true
		}
	}
	return false
}
