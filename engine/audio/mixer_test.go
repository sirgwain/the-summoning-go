package audio

import (
	"math"
	"testing"
	"time"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

type fakeVoice struct {
	id      string
	volume  float64
	stopped bool
	ended   bool
}

func (v *fakeVoice) Playing() bool         { return !v.stopped && !v.ended }
func (v *fakeVoice) SetVolume(vol float64) { v.volume = vol }
func (v *fakeVoice) Stop()                 { v.stopped = true }

type fakeBackend struct{ started []*fakeVoice }

func (b *fakeBackend) Play(s *content.Sound) (Voice, error) {
	v := &fakeVoice{id: s.ID}
	b.started = append(b.started, v)
	return v, nil
}

func (b *fakeBackend) last() *fakeVoice { return b.started[len(b.started)-1] }

func testMixer() (*Mixer, *fakeBackend) {
	p := content.NewPack(content.Manifest{})
	for _, s := range []*content.Sound{
		{ID: "music.1", Loop: 10}, {ID: "music.2"}, {ID: "music.10"},
		{ID: "sfx.3"}, {ID: "sfx.5"},
	} {
		p.Sounds[s.ID] = s
	}
	p.Cues = &content.Cues{ID: "cues", Cues: map[string]string{content.CueLever: "sfx.3", content.CueLevelUp: "music.10"}}
	b := &fakeBackend{}
	return New(b, p), b
}

func TestLayers(t *testing.T) {
	m, b := testMixer()
	m.Play("music.1")
	m.Play("sfx.5")
	m.Cue(content.CueLever)
	if m.MusicPlaying() != "music.1" || !m.EffectPlaying("sfx.5") || !m.EffectPlaying("sfx.3") {
		t.Fatalf("effects over music: music %q, sfx.5 %v, sfx.3 %v", m.MusicPlaying(), m.EffectPlaying("sfx.5"), m.EffectPlaying("sfx.3"))
	}
	first := b.last()
	m.Cue(content.CueLever)
	if !first.stopped || !m.EffectPlaying("sfx.3") || len(b.started) != 4 {
		t.Fatalf("an effect started again: the first stopped %v, %d started", first.stopped, len(b.started))
	}
	m.Cue(content.CueLevelUp)
	if m.MusicPlaying() != "music.1" || !m.EffectPlaying("music.10") {
		t.Fatalf("a fanfare: music %q, fanfare %v", m.MusicPlaying(), m.EffectPlaying("music.10"))
	}
	m.Play("music.2")
	if m.MusicPlaying() != "music.2" || !b.started[0].stopped || !m.EffectPlaying("sfx.5") {
		t.Fatalf("a new track: music %q, the old stopped %v", m.MusicPlaying(), b.started[0].stopped)
	}
	b.last().ended = true
	m.Update(time.Millisecond)
	if m.MusicPlaying() != "" {
		t.Fatalf("an ended track: music %q", m.MusicPlaying())
	}
	m.Play("sfx.9")
	m.Cue(content.CueDoor)
	if len(b.started) != 6 {
		t.Fatalf("unknown sounds started: %d", len(b.started))
	}
}

func TestMode(t *testing.T) {
	m, b := testMixer()
	m.Play("music.1")
	if m.CycleMode() != ModeEffects || m.MusicPlaying() != "" || !b.last().stopped {
		t.Fatalf("effects only: music %q", m.MusicPlaying())
	}
	m.Play("music.2")
	if m.MusicPlaying() != "" {
		t.Fatal("a track started with effects only")
	}
	m.Play("sfx.5")
	if !m.EffectPlaying("sfx.5") {
		t.Fatal("no effect with effects only")
	}
	if m.CycleMode() != ModeSilent || m.EffectPlaying("sfx.5") {
		t.Fatal("silent: an effect still sounds")
	}
	m.Play("sfx.5")
	if m.EffectPlaying("sfx.5") {
		t.Fatal("an effect played in silence")
	}
	if m.CycleMode() != ModeAll || m.MusicPlaying() != "music.1" {
		t.Fatalf("back on: music %q, want the looping track again", m.MusicPlaying())
	}
	m.Play("music.2")
	m.CycleMode()
	m.CycleMode()
	if m.CycleMode(); m.MusicPlaying() != "" {
		t.Fatalf("back on: music %q, but the track did not loop", m.MusicPlaying())
	}
}

func TestFades(t *testing.T) {
	const frame = time.Second / 60
	run := func(m *Mixer, limit time.Duration) time.Duration {
		var d time.Duration
		for ; m.MusicPlaying() != "" && d < limit; d += frame {
			m.Update(frame)
		}
		return d
	}
	m, b := testMixer()
	m.Play("music.1")
	music := b.last()
	m.Play("sfx.5")
	m.Update(frame)
	if music.volume != 1 {
		t.Fatalf("volume %v", music.volume)
	}
	m.FadeOutMusic()
	m.Update(quickStep)
	if math.Abs(music.volume-masterGain(124)) > 1e-9 || b.last().volume != 1 {
		t.Fatalf("after a quick step: music %v, effect %v", music.volume, b.last().volume)
	}
	// 127 to 1 in 42 drops, and the stop a step later.
	if d := run(m, 10*time.Second) + quickStep; d < 43*quickStep || d > 43*quickStep+frame {
		t.Errorf("quick fade took %v, want %v", d, 43*quickStep)
	}
	m.Play("music.1")
	if v := b.last().volume; v != 1 {
		t.Fatalf("after a fade: volume %v", v)
	}
	m.StopMusic()
	// 127 to 28 in 33 drops, and the stop a step later.
	if d := run(m, 10*time.Second); d < 34*slowStep || d > 34*slowStep+frame {
		t.Errorf("slow fade took %v, want %v", d, 34*slowStep)
	}
	m.Effects = 0.5
	m.Play("sfx.3")
	if v := b.last().volume; v != 0.5 {
		t.Errorf("effect volume %v", v)
	}
	if g := masterGain(0); math.Abs(20*math.Log10(g)+fullScaleDB) > 1e-9 {
		t.Errorf("gain at 0: %v", g)
	}
}
