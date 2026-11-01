package viewer

import (
	"github.com/sirgwain/the-summoning-go/engine/audio"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// sound plays the cues of the world's sound events.
func (v *Viewer) sound(events []sim.Event) {
	if v.Audio == nil {
		return
	}
	for _, e := range events {
		if e.Kind == sim.EventSound {
			v.Audio.Cue(e.Detail)
		}
	}
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
