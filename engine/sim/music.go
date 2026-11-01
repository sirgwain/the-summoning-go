package sim

import "github.com/sirgwain/the-summoning-go/engine/content"

// Music in play (docs/spec/audio.md, Music in play and Footsteps).

// EventMusic starts a music track (Detail its sound id) or, with Detail empty, fades out the
// track playing.
const EventMusic = "music"

// footstepCount is the step count a stride wraps at; the counts that sound are footstepOn.
const footstepCount = 6

var footstepOn = [footstepCount]bool{1: true, 4: true}

// startMusic starts track id.
func (w *World) startMusic(id string) {
	w.emit(Event{Kind: EventMusic, Level: w.current, Detail: id})
}

// stopMusic fades out the track playing, forgetting the theme.
func (w *World) stopMusic() {
	w.theme = ""
	w.emit(Event{Kind: EventMusic, Level: w.current})
}

// updateTheme starts the theme of a creature the player sees, once one first is, and fades it
// out once none is.
func (w *World) updateTheme(l *Level) {
	theme := ""
	for _, t := range l.Things {
		if t.Template.Theme != "" && alive(t) && l.Visible(t.Cell()) {
			theme = t.Template.Theme
			break
		}
	}
	switch {
	case theme == w.theme:
	case theme == "":
		w.stopMusic()
	default:
		w.theme = theme
		w.startMusic(theme)
	}
}

// stride counts a walk step that moved the player, sounding a footstep on the counts that do.
func (w *World) stride() {
	p := w.player
	if p.Status.Levitating {
		p.StepCount = 1
		return
	}
	p.StepCount = (p.StepCount + 1) % footstepCount
	if footstepOn[p.StepCount] {
		w.emit(Event{Kind: EventSound, Level: w.current, Cell: p.Cell(), Detail: content.CueFootstep})
	}
}
