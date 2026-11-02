package sim

// How far the player's waits have run, for drawing them: a hand's delay before it acts again,
// and the gesture playing before another spell can be cast. A wait counts in the player's
// updates, so it runs out at an update; between updates it counts the part of the interval gone.

// wait is a wait's length in player updates and how far into an interval it began.
type wait struct {
	total int
	from  float64
}

// begin starts a wait of n updates now.
func (w *World) begin(n int) wait { return wait{total: n, from: w.between()} }

// between returns how far the world is from the player's last update to its next, 0..1.
func (w *World) between() float64 {
	p := w.player
	if p == nil || p.next <= p.last {
		return 0
	}
	return min(1, max(0, float64(w.step-p.last)/float64(p.next-p.last)))
}

// left returns how much of wait a is still to run with n of its updates left: 1 as it begins,
// 0 when it has run out.
func (w *World) left(a wait, n int) float64 {
	if n <= 0 {
		return 0
	}
	if a.total < n {
		a = wait{total: n}
	}
	span := float64(a.total) - a.from
	if span <= 0 {
		return 0
	}
	return min(1, max(0, (float64(n)-w.between())/span))
}

// setDelay makes hand wait n updates before it acts again, keeping a longer wait already
// running.
func (w *World) setDelay(hand, n int) {
	p := w.player
	if n > p.Delays[hand] {
		p.Delays[hand], p.delays[hand] = n, w.begin(n)
	}
}

// HandWait returns how much of hand's delay is still to run: 1 just after it acts, falling to 0
// when it can act again.
func (w *World) HandWait(hand int) float64 {
	p := w.player
	if p == nil || hand < 0 || hand > 1 {
		return 0
	}
	return w.left(p.delays[hand], p.Delays[hand])
}

// GestureWait returns how much of the gesture playing is still to run before the spell takes
// effect and another can be cast: 1 as it starts, 0 with none playing.
func (w *World) GestureWait() float64 {
	p := w.player
	if p == nil || p.Magic.Gesture == nil {
		return 0
	}
	g := p.Magic.Gesture
	sp := w.pack.Spells[g.Spell]
	if sp == nil {
		return 0
	}
	// Each update shows one more icon; the update after the last casts the spell.
	return w.left(p.gesture, len(sp.Gesture)+1-g.Shown)
}
