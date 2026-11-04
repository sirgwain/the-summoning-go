package viewer

import (
	"fmt"

	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// Napping (docs/spec/player.md, Napping): holding N runs the world napFactor times as fast, to
// wait for magic points, until something needs the player.

// napFactor is how many world updates a napping update runs.
const napFactor = 64

// napShowFactor is how much faster the presentation's clock and the animations that run on time
// go while napping: slower than the world, which would show them a second or more apart.
const napShowFactor = 8

// updateMS is an update's length in ms.
const updateMS = 1000 / sim.StepsPerSecond

// KeyNap is the key held to nap.
const KeyNap Key = "N"

// napRefusal says why a nap cannot run now, or "" when it can.
func (v *Viewer) napRefusal() string {
	p := v.World.Player()
	switch {
	case p.MagicPoints >= p.MaxMagicPoints:
		return "Your magic points are full."
	case v.World.CreatureNear():
		return "A creature is near."
	}
	return ""
}

// nap handles the nap key in play, reporting whether napping took the update. cmds are what the
// update's other input asked for: any of it, another key or a click stops a nap.
func (v *Viewer) nap(in Input, cmds []input.Command) bool {
	p := v.World.Player()
	if !v.Play || v.Paused || p == nil || p.Dead {
		v.Napping = false
		return false
	}
	if !in.held(KeyNap) {
		v.Napping, v.napHeldOff = false, false
		return false
	}
	if !v.Napping {
		if !in.pressed(KeyNap) || v.napHeldOff {
			return false
		}
		v.napHeldOff = true
		if why := v.napRefusal(); why != "" {
			v.notice(why)
			return false
		}
		v.Napping = true
	} else if v.interrupts(in, cmds) {
		v.Napping = false
		return false
	}
	// The frame is drawn once, so the scene is rebuilt once for all of its world updates.
	stopped := false
	v.batch(func() {
		for range napFactor {
			hp := p.HitPoints
			events := v.updateWorld(nil)
			if stop, why := v.napStop(events, hp); stop {
				v.Napping, stopped = false, true
				if why != "" {
					v.notice(why)
				}
				return
			}
		}
	})
	if stopped {
		return true
	}
	p = v.World.Player()
	played := v.World.Step() / sim.StepsPerSecond
	v.notice(fmt.Sprintf("Waiting: %d/%d magic points, played %d:%02d:%02d",
		p.MagicPoints, p.MaxMagicPoints, played/3600, played/60%60, played%60))
	return true
}

// interrupts reports whether the update's input, besides the nap key, stops a nap.
func (v *Viewer) interrupts(in Input, cmds []input.Command) bool {
	if len(cmds) > 0 || in.Left.Released || in.Right.Released {
		return true
	}
	for k, down := range in.Pressed {
		if down && k != KeyNap && k != KeyShift && k != KeyControl {
			return true
		}
	}
	return false
}

// napStop reports whether a nap stops after a world update with events, the player having had
// hp before it, and what to say: why it cannot go on, or nothing when the hurt, the message or
// the conversation that stopped it shows for itself.
func (v *Viewer) napStop(events []sim.Event, hp int) (bool, string) {
	p := v.World.Player()
	if p == nil || p.Dead || p.HitPoints < hp || v.World.Talk() != nil || v.Front != nil {
		return true, ""
	}
	for _, e := range events {
		if e.Kind == sim.EventMessage {
			return true, ""
		}
	}
	why := v.napRefusal()
	return why != "", why
}
