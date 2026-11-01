package devpack

import "github.com/sirgwain/the-summoning-go/engine/content"

// Sound ids: a short tune that loops after its first note, and a blip.
const (
	MusicTune = "music.tune"
	SFXBlip   = "sfx.blip"
)

const soundRate = 22050

// tone is n samples of a triangle wave of the given period, fading out over its last quarter.
func tone(n, period, amplitude int) []int16 {
	out := make([]int16, n)
	for i := range out {
		phase := i % period
		v := 4*amplitude*phase/period - amplitude
		if phase >= period/2 {
			v = 3*amplitude - 4*amplitude*phase/period
		}
		if fade := n / 4; i >= n-fade {
			v = v * (n - i) / fade
		}
		out[i] = int16(v)
	}
	return out
}

func sounds(p *content.Pack) {
	intro := tone(soundRate/4, 50, 6000) // 441 Hz
	loop := append(tone(soundRate/4, 40, 6000), tone(soundRate/4, 33, 6000)...)
	p.Sounds[MusicTune] = &content.Sound{
		ID: MusicTune, File: "tune.wav", Intro: len(intro), Loop: len(loop),
		Data: content.EncodeWAV(append(intro, loop...), soundRate),
	}
	p.Sounds[SFXBlip] = &content.Sound{
		ID: SFXBlip, File: "blip.wav", Data: content.EncodeWAV(tone(soundRate/20, 25, 8000), soundRate),
	}
	p.Cues = &content.Cues{ID: "cues", Cues: map[string]string{
		content.CueLever: SFXBlip, content.CueDoor: SFXBlip, content.CueUnseal: SFXBlip, content.CueTeleport: SFXBlip,
	}}
}
