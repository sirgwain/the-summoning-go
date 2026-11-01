// Package audio plays a pack's music and sound effects. The mixer decides what plays; a Backend
// plays it. The Ebitengine backend is in engine/audio/speaker, so the rules test without a
// display or a sound card.
//
// The game played one song at a time, music and effects alike (docs/spec/audio.md, Songs). The
// mixer does not keep that limit: music has a channel of its own, and effects sound over it and
// over each other.
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
	// ModeEffects plays effects only.
	ModeEffects
	// ModeSilent plays nothing.
	ModeSilent
)

// The fades lower the music's master volume, 0..127, in steps of fadeDrop. The game's slow fade
// steps with the song's events; the mixer steps it at slowStep, about the mean of the game's
// looping tracks. The quick fade steps at quickStep, as the game does.
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

type playing struct {
	voice Voice
	sound *content.Sound
}

// Mixer plays one music track at a time, and any number of effects over it. Its zero value is
// not ready; use New.
type Mixer struct {
	backend Backend
	pack    *content.Pack
	// Music and Effects are the volumes, 0 to 1, of music and of effects.
	Music, Effects float64
	mode           Mode

	music *playing
	// effects are the effects sounding, by sound id: an effect started again restarts.
	effects map[string]*playing
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
	return &Mixer{backend: b, pack: pack, Music: 1, Effects: 1, master: maxMaster, effects: map[string]*playing{}}
}

// Mode returns the music key's setting.
func (m *Mixer) Mode() Mode { return m.mode }

// MusicPlaying returns the id of the music track playing, or "".
func (m *Mixer) MusicPlaying() string {
	if m.music == nil || !m.music.voice.Playing() {
		return ""
	}
	return m.music.sound.ID
}

// EffectPlaying reports whether effect id is sounding.
func (m *Mixer) EffectPlaying(id string) bool {
	e := m.effects[id]
	return e != nil && e.voice.Playing()
}

// Cue plays the sound the pack gives an engine cue, if any, as an effect: a fanfare sounds over
// the music rather than stopping it.
func (m *Mixer) Cue(cue string) {
	if c := m.pack.Cues; c != nil {
		if id, ok := c.Cues[cue]; ok {
			m.PlayEffect(id)
		}
	}
}

// Play plays sound id: a music track replaces the one playing; anything else is an effect.
func (m *Mixer) Play(id string) {
	if strings.HasPrefix(id, content.MusicPrefix) {
		m.PlayMusic(id)
	} else {
		m.PlayEffect(id)
	}
}

// PlayMusic plays sound id on the music channel, stopping the track playing.
func (m *Mixer) PlayMusic(id string) {
	s := m.pack.Sounds[id]
	if s == nil || m.mode != ModeAll {
		return
	}
	m.startMusic(s)
}

// PlayEffect plays sound id over whatever plays, restarting it if it is already sounding.
func (m *Mixer) PlayEffect(id string) {
	s := m.pack.Sounds[id]
	if s == nil || m.mode == ModeSilent {
		return
	}
	if e := m.effects[id]; e != nil {
		e.voice.Stop()
		delete(m.effects, id)
	}
	if p := m.start(s, m.Effects); p != nil {
		m.effects[id] = p
	}
}

func (m *Mixer) start(s *content.Sound, volume float64) *playing {
	v, err := m.backend.Play(s)
	if err != nil {
		m.Err = err
		return nil
	}
	v.SetVolume(volume)
	return &playing{v, s}
}

func (m *Mixer) startMusic(s *content.Sound) {
	m.haltMusic()
	m.music = m.start(s, m.Music)
}

// haltMusic stops the music at once, putting a fade's volume back.
func (m *Mixer) haltMusic() {
	if m.music != nil {
		m.music.voice.Stop()
	}
	m.music = nil
	m.fade, m.master = noFade, maxMaster
}

func (m *Mixer) haltEffects() {
	for id, e := range m.effects {
		e.voice.Stop()
		delete(m.effects, id)
	}
}

// StopMusic fades the music out slowly: the game's way of stopping a track while play goes on.
func (m *Mixer) StopMusic() { m.startFade(slowFade, slowStep) }

// FadeOutMusic fades the music out quickly: the game's way before a scene or a change of level.
func (m *Mixer) FadeOutMusic() { m.startFade(quickFade, quickStep) }

func (m *Mixer) startFade(f fade, step time.Duration) {
	if m.MusicPlaying() == "" || m.fade == quickFade || (m.fade == f && f == slowFade) {
		return
	}
	m.fade, m.fadeNext = f, m.clock+step
}

// CycleMode moves the music key to its next setting. Leaving ModeAll stops the music, keeping a
// looping track to start again on coming back; ModeSilent stops the effects too.
func (m *Mixer) CycleMode() Mode {
	m.mode = (m.mode + 1) % 3
	switch m.mode {
	case ModeEffects:
		if m.MusicPlaying() != "" && m.music.sound.Loop > 0 {
			m.paused = m.music.sound
		}
		m.haltMusic()
	case ModeSilent:
		m.haltEffects()
	case ModeAll:
		if s := m.paused; s != nil {
			m.paused = nil
			m.startMusic(s)
		}
	}
	return m.mode
}

// Update advances the fades by dt, lets go of ended sounds, and takes up a change of volume.
func (m *Mixer) Update(dt time.Duration) {
	m.clock += dt
	for id, e := range m.effects {
		if !e.voice.Playing() {
			delete(m.effects, id)
			continue
		}
		e.voice.SetVolume(m.Effects)
	}
	if m.music != nil && !m.music.voice.Playing() {
		m.haltMusic()
	}
	for m.fade != noFade && m.clock >= m.fadeNext {
		switch m.fade {
		case slowFade:
			if m.master <= slowStop {
				m.haltMusic()
				return
			}
			m.master = max(0, m.master-fadeDrop)
			m.fadeNext += slowStep
		case quickFade:
			if m.master < fadeDrop {
				m.haltMusic()
				return
			}
			m.master -= fadeDrop
			m.fadeNext += quickStep
		}
	}
	if m.music != nil {
		m.music.voice.SetVolume(m.Music * masterGain(m.master))
	}
}

// masterGain is the gain of master volume n.
func masterGain(n int) float64 {
	if n >= maxMaster {
		return 1
	}
	return math.Pow(10, -fullScaleDB*float64(maxMaster-n)/maxMaster/20)
}
