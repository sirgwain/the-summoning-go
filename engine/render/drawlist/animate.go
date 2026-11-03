package drawlist

import (
	"fmt"
	"math/rand/v2"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Animator plays the animations that run on time (docs/pack.md, Sprites): a thing standing
// still whose idle animation has a duration and more than one frame steps through it by the
// clock, as its mode says. Animations that step by moves, and frames the simulation chooses,
// are left to the scene.
type Animator struct {
	states map[string]*animState
	rng    *rand.Rand
}

// animState is where one thing's animation is: the animation, its place in the frames, the time
// left on it, and for flicker whether it is flickering.
type animState struct {
	key        string
	anim       content.Animation
	slot, left int
	flickering bool
}

// NewAnimator returns an animator whose random choices follow seed.
func NewAnimator(seed uint64) *Animator {
	return &Animator{states: map[string]*animState{}, rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))}
}

// timedAnim returns the animation thing t shows by time, if any.
func timedAnim(p *content.Pack, t Thing) (content.Animation, string, bool) {
	if t.Anim != "" || t.Frame != nil || t.Sprite != "" || t.Looks != nil {
		return content.Animation{}, "", false
	}
	tmpl, ok := p.Things[t.Template]
	if !ok {
		return content.Animation{}, "", false
	}
	for _, name := range []string{fmt.Sprintf("idle.%d", t.Facing), "idle"} {
		if a, ok := p.Animation(tmpl, name); ok {
			if a.MS.Max <= 0 || len(a.Frames) < 2 {
				return content.Animation{}, "", false
			}
			return a, t.Template + "/" + name, true
		}
	}
	return content.Animation{}, "", false
}

// duration picks how long a frame of a shows, in ms.
func (an *Animator) duration(a content.Animation) int {
	if a.MS.Max > a.MS.Min {
		return a.MS.Min + an.rng.IntN(a.MS.Max-a.MS.Min+1)
	}
	return max(1, a.MS.Min)
}

// Apply gives each thing of s that animates by time the frame it shows now, starting the
// animations of things it has not seen and forgetting those of things that are gone or no
// longer show one.
func (an *Animator) Apply(p *content.Pack, s *Scene) {
	seen := map[string]bool{}
	for i := range s.Things {
		t := &s.Things[i]
		a, key, ok := timedAnim(p, *t)
		if !ok {
			continue
		}
		seen[t.ID] = true
		st := an.states[t.ID]
		if st == nil || st.key != key {
			st = &animState{key: key, anim: a, left: an.duration(a)}
			if a.Mode == content.AnimRandom {
				st.slot = an.rng.IntN(len(a.Frames))
			}
			an.states[t.ID] = st
		}
		frame := a.Frames[st.slot%len(a.Frames)]
		t.Frame = &frame
	}
	for id := range an.states {
		if !seen[id] {
			delete(an.states, id)
		}
	}
}

// Advance moves every animation on by ms, and reports whether any frame changed.
func (an *Animator) Advance(ms int) bool {
	changed := false
	for _, st := range an.states {
		a := st.anim
		if st.flickering {
			// Flickering, the frames after the first change at every advance.
			st.slot = 1 + st.slot%(len(a.Frames)-1)
			changed = true
		}
		for st.left -= ms; st.left <= 0; {
			before := st.slot
			switch a.Mode {
			case content.AnimOnce:
				st.slot = min(st.slot+1, len(a.Frames)-1)
			case content.AnimRandom:
				st.slot = an.rng.IntN(len(a.Frames))
			case content.AnimFlicker:
				st.flickering = !st.flickering
				st.slot = 0
				if st.flickering {
					st.slot = 1
				}
			default:
				st.slot = (st.slot + 1) % len(a.Frames)
			}
			changed = changed || a.Frames[before] != a.Frames[st.slot]
			if a.Mode == content.AnimOnce && st.slot == len(a.Frames)-1 {
				st.left = 1 << 30
				break
			}
			st.left += an.duration(a)
		}
	}
	return changed
}
