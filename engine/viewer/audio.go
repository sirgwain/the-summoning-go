package viewer

import (
	"github.com/sirgwain/the-summoning-go/engine/audio"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// sound plays the world's sound and music events. A level's own footsteps take the place of
// the footstep cue.
func (v *Viewer) sound(events []sim.Event) {
	if v.Audio == nil {
		return
	}
	for _, e := range events {
		switch {
		case e.Kind == sim.EventMusic && e.Detail == "":
			v.Audio.StopMusic()
		case e.Kind == sim.EventMusic:
			v.Audio.PlayMusic(e.Detail)
		case e.Kind == sim.EventSound && e.Detail == content.CueFootstep && v.levelFootsteps(e.Level) != "":
			v.Audio.PlayEffect(v.levelFootsteps(e.Level))
		case e.Kind == sim.EventSound:
			v.Audio.Cue(e.Detail)
		}
	}
}

// levelFootsteps is level id's footstep sound, or "".
func (v *Viewer) levelFootsteps(id string) string {
	if l := v.Pack.Levels[id]; l != nil {
		return l.Footsteps
	}
	return ""
}

// musicMessages announce the music key's settings.
var musicMessages = map[audio.Mode]string{
	audio.ModeAll: content.MsgMusicAll, audio.ModeEffects: content.MsgMusicEffects, audio.ModeSilent: content.MsgMusicSilent,
}

// cycleMusic moves the music key to its next setting and announces it (docs/spec/audio.md,
// Songs).
func (v *Viewer) cycleMusic() {
	mode := v.Audio.CycleMode()
	if m := v.Pack.Messages; m != nil {
		if key, ok := m.Messages[musicMessages[mode]]; ok {
			v.notice(inspect.Text(v.Pack, key))
		}
	}
}
