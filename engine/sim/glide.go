package sim

// maxGlide is the farthest, in world units along either axis, a thing is drawn gliding over in
// one update; a longer move (a teleporter, a placement) is drawn as a jump.
const maxGlide = CellUnits / 2

// hurrySteps is how many steps the player's glide takes to finish once no walk is asked for.
const hurrySteps = 2

// Glide is how a thing is drawn between its updates (docs/spec/entities.md, Drawn between
// updates): Left of Span steps remain of its way from (FromU, FromV) to where it is, so it is
// drawn Left / Span of the way back toward there. Span 0 draws it where it is.
type Glide struct {
	FromU, FromV int
	Left, Span   int64
}

// glide is where a thing was at the start of its last update, at step at, and the step end its
// glide to where it is ends. The zero glide draws the thing where it is.
type glide struct {
	u, v    int
	at, end int64
	set     bool
}

// start records (u, v) as where the thing is at the start of its update at step, its glide
// ending at step end.
func (g *glide) start(u, v int, step, end int64) {
	*g = glide{u: u, v: v, at: step, end: end, set: true}
}

// state returns how a thing now at (u, v) is drawn at step.
func (g glide) state(u, v int, step int64) Glide {
	if !g.set || g.end <= g.at || step >= g.end || (u == g.u && v == g.v) ||
		abs(u-g.u) > maxGlide || abs(v-g.v) > maxGlide {
		return Glide{}
	}
	return Glide{FromU: g.u, FromV: g.v, Left: min(g.end-step, g.end-g.at), Span: g.end - g.at}
}

// hurry has a glide under way at step finish within hurrySteps, going on from where it is drawn.
func (g *glide) hurry(u, v int, step int64) {
	s := g.state(u, v, step)
	if s.Span == 0 || s.Left <= hurrySteps {
		return
	}
	// The same part of the way is left, over fewer steps.
	span := (hurrySteps*s.Span + s.Left/2) / s.Left
	g.end, g.at = step+hurrySteps, step+hurrySteps-span
}

// Glide returns how thing t is drawn at step.
func (t *Thing) Glide(step int64) Glide { return t.glide.state(t.U, t.V, step) }

// Glide returns how the player is drawn at step.
func (p *Player) Glide(step int64) Glide { return p.glide.state(p.U, p.V, step) }

// Glide returns how flight f is drawn at step.
func (f *Flight) Glide(step int64) Glide { return f.glide.state(f.U, f.V, step) }

// Gliding reports whether anything on the current level is drawn, at the world's step, between
// where it was and where it is.
func (w *World) Gliding() bool {
	l, s := w.Current(), w.step
	if p := w.player; p != nil && p.Glide(s).Span > 0 {
		return true
	}
	for _, t := range l.Things {
		if t.Glide(s).Span > 0 {
			return true
		}
	}
	for _, f := range l.Flights {
		if f.Glide(s).Span > 0 {
			return true
		}
	}
	return false
}
