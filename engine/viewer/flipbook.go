package viewer

import (
	"github.com/sirgwain/the-summoning-go/engine/audio"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// FlipbookRun plays a flipbook's steps by the clock (docs/pack.md, Flipbooks).
type FlipbookRun struct {
	Flipbook *content.Flipbook
	// Step is the step showing and MS how long it has shown.
	Step, MS int
	started  bool
	// Done is set when the last step has shown its time, or the run was cancelled.
	Done bool
}

// NewFlipbookRun starts f at its first step.
func NewFlipbookRun(f *content.Flipbook) *FlipbookRun { return &FlipbookRun{Flipbook: f} }

// Update advances the run by ms, calling enter for each step as it starts, the first included.
func (r *FlipbookRun) Update(ms int, enter func(*content.FlipbookStep)) {
	steps := r.Flipbook.Steps
	if !r.started {
		r.started = true
		if len(steps) == 0 {
			r.Done = true
			return
		}
		enter(&steps[0])
	}
	if r.Done {
		return
	}
	r.MS += ms
	for r.MS >= steps[r.Step].MS {
		r.MS -= steps[r.Step].MS
		if r.Step+1 == len(steps) {
			r.MS, r.Done = steps[r.Step].MS, true
			return
		}
		r.Step++
		enter(&steps[r.Step])
	}
}

// Frame is the frame showing.
func (r *FlipbookRun) Frame() int {
	if len(r.Flipbook.Steps) == 0 {
		return 0
	}
	return r.Flipbook.Steps[r.Step].Frame
}

// Cancellable reports whether the step showing can be cut short.
func (r *FlipbookRun) Cancellable() bool {
	return !r.Done && len(r.Flipbook.Steps) > 0 && r.Flipbook.Steps[r.Step].Cancel
}

// Cancel ends the run if the step showing allows it, and reports whether it did.
func (r *FlipbookRun) Cancel() bool {
	if !r.Cancellable() {
		return false
	}
	r.Done = true
	return true
}

// PlayFlipbookStep starts a step's music and sounds on the mixer.
func PlayFlipbookStep(m *audio.Mixer, s *content.FlipbookStep) {
	if m == nil {
		return
	}
	if s.MusicStop {
		m.HaltMusic()
	}
	if s.MusicFade {
		m.FadeOutMusic()
	}
	if s.Music != "" {
		m.PlayMusic(s.Music)
	}
	for _, id := range s.Sounds {
		m.PlayEffect(id)
	}
}
