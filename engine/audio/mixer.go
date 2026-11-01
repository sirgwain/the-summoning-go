// Package audio plays a pack's music and sound effects by the game's rules
// (docs/spec/audio.md, Songs). The mixer decides what plays; a Backend plays it. The Ebitengine
// backend is in engine/audio/ebiten, so the rules test without a display or a sound card.
package audio

import (
	"math"
	"strings"
	"time"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Voice is a sound a Backend is playing.
type Voice interface {
	// Playing reports whether the sound is still sounding.
	Playing() bool
	// SetVolume sets the sound's gain, 0 to 1.
	SetVolume(v float64)
	// Stop stops the sound for good.
	Stop()
}

// Backend starts sounds.
type Backend interface {
	Play(s *content.Sound) (Voice, error)
}

// Mode is the music key's setting.
type Mode int

const (
	// ModeAll plays music and effects.
	ModeAll Mode = iota
	// ModeEffects plays effects only: music tracks do not start.
	ModeEffects
	// ModeSilent plays nothing.
	ModeSilent
)

// The fades lower the master volume, 0..127, in steps of fadeDrop. The game's slow fade steps
// with the song's events; the mixer steps it at slowStep, about the mean of the game's looping
// tracks. The quick fade steps at quickStep, as the game does.
const (
	maxMaster = 127
	fadeDrop  = 3
	slowStop  = 30
	slowStep  = 125 * time.Millisecond
	quickStep = time.Second * 10 / 182
	// fullScaleDB is the attenuation of a carrier at full output level when the master volume
	// is 0: 63 steps of 0.75 dB. The mixer applies the master volume as this much attenuation
	// times (1 − master/127).
	fullScaleDB = 63 * 0.75
)

type fade int

const (
	noFade fade = iota
	slowFade
	quickFade
)

// Mixer plays one sound at a time: a music track, or an effect while no track is playing. Its
// zero value is not ready; use New.
type Mixer struct {
	backend Backend
	pack    *content.Pack
	// Music and Effects are the volumes, 0 to 1, of music tracks and of effects.
	Music, Effects float64
	mode           Mode

	voice Voice
	sound *content.Sound
	// paused is the looping track the music key stopped, started again when it comes back to
	// ModeAll.
	paused *content.Sound

	master   int
	fade     fade
	fadeNext time.Duration
	clock    time.Duration
	// Err is the last error the backend gave.
	Err error
}

// New returns a mixer for pack's sounds, playing through b.
func New(b Backend, pack *content.Pack) *Mixer {
	return &Mixer{backend: b, pack: pack, Music: 1, Effects: 1, master: maxMaster}
}

// Mode returns the music key's setting.
func (m *Mixer) Mode() Mode { return m.mode }

// Playing returns the id of the sound playing, or "".
func (m *Mixer) Playing() string {
	if m.voice == nil || !m.voice.Playing() {
		return ""
	}
	return m.sound.ID
}

// Cue plays the sound the pack gives an engine cue, if any.
func (m *Mixer) Cue(cue string) {
	if c := m.pack.Cues; c != nil {
		if id, ok := c.Cues[cue]; ok {
			m.Play(id)
		}
	}
}

// Play plays sound id: a music track, which stops whatever plays, or an effect, which plays
// only while no track and no effect of higher priority is sounding.
func (m *Mixer) Play(id string) {
	s := m.pack.Sounds[id]
	if s == nil || m.mode == ModeSilent {
		return
	}
	music := strings.HasPrefix(id, content.MusicPrefix)
	if music && m.mode != ModeAll {
		return
	}
	if !music && m.Playing() != "" {
		if !strings.HasPrefix(m.sound.ID, content.SFXPrefix) || m.sound.Priority > s.Priority {
			return
		}
	}
	m.start(s)
}

func (m *Mixer) start(s *content.Sound) {
	m.halt()
	v, err := m.backend.Play(s)
	if err != nil {
		m.Err = err
		return
	}
	m.voice, m.sound = v, s
	m.apply()
}

// halt stops the sound at once, putting a fade's volume back.
func (m *Mixer) halt() {
	if m.voice != nil {
		m.voice.Stop()
	}
	m.voice, m.sound = nil, nil
	m.fade, m.master = noFade, maxMaster
}

// Stop fades the sound out slowly: the game's way of stopping a track while play goes on.
func (m *Mixer) Stop() { m.startFade(slowFade, slowStep) }

// FadeOut fades the sound out quickly: the game's way before a scene or a change of level.
func (m *Mixer) FadeOut() { m.startFade(quickFade, quickStep) }

func (m *Mixer) startFade(f fade, step time.Duration) {
	if m.Playing() == "" || m.fade == quickFade || (m.fade == f && f == slowFade) {
		return
	}
	m.fade, m.fadeNext = f, m.clock+step
}

// CycleMode moves the music key to its next setting. Leaving ModeAll stops the sound, keeping
// a looping track to start again on coming back.
func (m *Mixer) CycleMode() Mode {
	m.mode = (m.mode + 1) % 3
	if m.mode != ModeAll {
		if s := m.sound; s != nil && m.Playing() != "" && s.Loop > 0 && strings.HasPrefix(s.ID, content.MusicPrefix) {
			m.paused = s
		}
		m.halt()
		return m.mode
	}
	if s := m.paused; s != nil {
		m.paused = nil
		m.start(s)
	}
	return m.mode
}

// Update advances the fades by dt, and takes up a change of volume.
func (m *Mixer) Update(dt time.Duration) {
	m.clock += dt
	if m.voice != nil && !m.voice.Playing() {
		m.halt()
	}
	for m.fade != noFade && m.clock >= m.fadeNext {
		switch m.fade {
		case slowFade:
			if m.master <= slowStop {
				m.halt()
				return
			}
			m.master = max(0, m.master-fadeDrop)
			m.fadeNext += slowStep
		case quickFade:
			if m.master < fadeDrop {
				m.halt()
				return
			}
			m.master -= fadeDrop
			m.fadeNext += quickStep
		}
	}
	m.apply()
}

func (m *Mixer) apply() {
	if m.voice == nil {
		return
	}
	v := m.Effects
	if strings.HasPrefix(m.sound.ID, content.MusicPrefix) {
		v = m.Music
	}
	m.voice.SetVolume(v * masterGain(m.master))
}

// masterGain is the gain of master volume n.
func masterGain(n int) float64 {
	if n >= maxMaster {
		return 1
	}
	return math.Pow(10, -fullScaleDB*float64(maxMaster-n)/maxMaster/20)
}
