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
		{ID: "music.1", Loop: 10}, {ID: "music.2"},
		{ID: "sfx.3", Priority: 3}, {ID: "sfx.5", Priority: 5}, {ID: "sfx.0"},
	} {
		p.Sounds[s.ID] = s
	}
	p.Cues = &content.Cues{ID: "cues", Cues: map[string]string{content.CueLever: "sfx.3"}}
	b := &fakeBackend{}
	return New(b, p), b
}

func TestEffectRule(t *testing.T) {
	m, b := testMixer()
	m.Play("music.2")
	m.Play("sfx.5")
	if m.Playing() != "music.2" || len(b.started) != 1 {
		t.Fatalf("an effect over a track: playing %q", m.Playing())
	}
	b.last().ended = true
	m.Cue(content.CueLever)
	if m.Playing() != "sfx.3" {
		t.Fatalf("an effect after the track ended: playing %q", m.Playing())
	}
	m.Play("sfx.0")
	if m.Playing() != "sfx.3" {
		t.Fatalf("a lower effect over a higher: playing %q", m.Playing())
	}
	m.Play("sfx.3")
	if m.Playing() != "sfx.3" || len(b.started) != 3 || !b.started[1].stopped {
		t.Fatalf("an equal effect replaces: playing %q, %d started", m.Playing(), len(b.started))
	}
	m.Play("sfx.5")
	if m.Playing() != "sfx.5" {
		t.Fatalf("a higher effect over a lower: playing %q", m.Playing())
	}
	m.Play("music.1")
	if m.Playing() != "music.1" || !b.started[3].stopped {
		t.Fatalf("a track over an effect: playing %q", m.Playing())
	}
	m.Play("sfx.9")
	m.Cue(content.CueDoor)
	if m.Playing() != "music.1" {
		t.Fatalf("unknown sounds: playing %q", m.Playing())
	}
}

func TestMode(t *testing.T) {
	m, b := testMixer()
	m.Play("music.1")
	if m.CycleMode() != ModeEffects || m.Playing() != "" || !b.last().stopped {
		t.Fatalf("effects only: playing %q", m.Playing())
	}
	m.Play("music.2")
	if m.Playing() != "" {
		t.Fatal("a track started with effects only")
	}
	m.Play("sfx.0")
	if m.Playing() != "sfx.0" {
		t.Fatal("no effect with effects only")
	}
	if m.CycleMode() != ModeSilent || m.Playing() != "" {
		t.Fatal("silent: still playing")
	}
	m.Play("sfx.0")
	if m.Playing() != "" {
		t.Fatal("an effect played in silence")
	}
	if m.CycleMode() != ModeAll || m.Playing() != "music.1" {
		t.Fatalf("back on: playing %q, want the looping track again", m.Playing())
	}
	m.Play("music.2")
	m.CycleMode()
	m.CycleMode()
	if m.CycleMode(); m.Playing() != "" {
		t.Fatalf("back on: playing %q, but the track did not loop", m.Playing())
	}
}

func TestFades(t *testing.T) {
	const frame = time.Second / 60
	run := func(m *Mixer, limit time.Duration) time.Duration {
		var d time.Duration
		for ; m.Playing() != "" && d < limit; d += frame {
			m.Update(frame)
		}
		return d
	}
	m, b := testMixer()
	m.Play("music.1")
	m.Update(frame)
	if v := b.last().volume; v != 1 {
		t.Fatalf("volume %v", v)
	}
	m.FadeOut()
	m.Update(quickStep)
	if v := b.last().volume; math.Abs(v-masterGain(124)) > 1e-9 {
		t.Fatalf("after a quick step: volume %v", v)
	}
	// 127 to 1 in 42 drops, and the stop a step later.
	if d := run(m, 10*time.Second) + quickStep; d < 43*quickStep || d > 43*quickStep+frame {
		t.Errorf("quick fade took %v, want %v", d, 43*quickStep)
	}
	m.Play("music.1")
	if v := b.last().volume; v != 1 {
		t.Fatalf("after a fade: volume %v", v)
	}
	m.Stop()
	// 127 to 28 in 33 drops, and the stop a step later.
	if d := run(m, 10*time.Second); d < 34*slowStep || d > 34*slowStep+frame {
		t.Errorf("slow fade took %v, want %v", d, 34*slowStep)
	}
	m.Effects = 0.5
	m.Play("sfx.0")
	if v := b.last().volume; v != 0.5 {
		t.Errorf("effect volume %v", v)
	}
	if g := masterGain(0); math.Abs(20*math.Log10(g)+fullScaleDB) > 1e-9 {
		t.Errorf("gain at 0: %v", g)
	}
}
