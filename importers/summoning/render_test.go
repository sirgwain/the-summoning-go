package summoning

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/internal/opl2"
)

// chipSamplesAt is the chip sample of a tick at 120 BPM (divisor 2485).
func chipSamplesAt(tick int64) int64 { return tick * 2485 * opl2.Rate / PITRate }

func peak(samples []int16) int {
	p := 0
	for _, v := range samples {
		p = max(p, int(v), -int(v))
	}
	return p
}

func TestRenderOnce(t *testing.T) {
	s := testSong(false,
		0x00, 0xC0, 0x00,
		0x00, 0x90, 0x3C, 0x7F,
		0x60, 0x80, 0x3C, 0x00, // 96 ticks later, note off
		0x00, 0xFC)
	r, err := RenderSong(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Loop != 0 || r.Intro != 0 {
		t.Fatalf("intro %d loop %d for a song that plays once", r.Intro, r.Loop)
	}
	end := int(chipSamplesAt(97) * SampleRate / opl2.Rate)
	if len(r.Samples) < end || len(r.Samples) > end+maxTailSeconds*SampleRate {
		t.Fatalf("%d samples, want the song's %d and a tail", len(r.Samples), end)
	}
	if p := peak(r.Samples[:end]); p < 1000 {
		t.Fatalf("peak %d while the note sounds", p)
	}
	// The note sounds from tick 1, after the song start's writes land.
	if p := peak(r.Samples[:SampleRate/1000]); p > 2 {
		t.Fatalf("peak %d in the first millisecond, before the note", p)
	}
}

func TestRenderLoop(t *testing.T) {
	s := testSong(false,
		0x00, 0xC0, 0x00,
		0x00, 0x90, 0x3C, 0x7F,
		0x30, 0x80, 0x3C, 0x00,
		0x00, 0xFC)
	r, err := RenderSong(s, 4)
	if err != nil {
		t.Fatal(err)
	}
	// The song restarts at FCh, tick 49, and again 48 ticks later.
	intro := int(chipSamplesAt(49) * SampleRate / opl2.Rate)
	end := int(chipSamplesAt(97) * SampleRate / opl2.Rate)
	if r.Intro != intro || r.Intro+r.Loop != end || len(r.Samples) != end {
		t.Fatalf("intro %d loop %d samples %d, want intro %d ending at %d", r.Intro, r.Loop, len(r.Samples), intro, end)
	}
	if p := peak(r.Samples[r.Intro:]); p < 1000 {
		t.Fatalf("the loop is silent: peak %d", p)
	}
}

func TestRenderError(t *testing.T) {
	if _, err := RenderSong(testSong(false, 0x00, 0x90, 0x3C, 0x40, 0x05), 0); err == nil {
		t.Fatal("a song without FCh: no error")
	}
}

func TestImportAudio(t *testing.T) {
	p, _, err := Import(importFixture(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	for n := range firstEffect {
		if p.Sounds[soundID(n)] == nil {
			t.Errorf("no %s", soundID(n))
		}
	}
	sfx := p.Sounds["sfx.0"]
	if sfx == nil || sfx.File != "0.wav" || sfx.Loop != 0 {
		t.Fatalf("sfx.0 %+v", sfx)
	}
	// Track 1 loops (48 bars): longer than the song, so it restarts at FCh.
	m := p.Sounds["music.1"]
	if m.Intro == 0 || m.Loop == 0 {
		t.Fatalf("music.1 intro %d loop %d", m.Intro, m.Loop)
	}
	w, err := content.ReadWAV(m.Data)
	if err != nil || w.Rate != SampleRate || w.Channels != 1 || w.Frames != m.Intro+m.Loop {
		t.Fatalf("music.1 WAV %+v %v, want %d samples", w, err, m.Intro+m.Loop)
	}
	if p.Cues == nil || p.Cues.Cues[content.CueDoor] != "sfx.23" ||
		p.Cues.Cues[content.CueLevelUp] != "music.10" || p.Cues.Cues[content.CueFootstep] != "sfx.0" {
		t.Fatalf("cues %+v", p.Cues)
	}
	if p.Sounds["music.2"].Loop != 0 {
		t.Fatal("music.2 loops")
	}

	p, _, err = Import(importFixture(), Options{NoAudio: true})
	if err != nil || len(p.Sounds) != 0 {
		t.Fatalf("no audio: %d sounds, err %v", len(p.Sounds), err)
	}
}

func TestMissileTrack(t *testing.T) {
	for _, tc := range []struct {
		m    content.MissileSpell
		want int
	}{
		{content.MissileSpell{Hurts: content.HurtsFire}, firstEffect + 0x0C},
		{content.MissileSpell{Hurts: content.HurtsPoison}, firstEffect + 8},
		{content.MissileSpell{Hurts: content.HurtsLightning}, firstEffect + 0x0D},
		{content.MissileSpell{Scatters: true}, firstEffect + 0x0D},
		{content.MissileSpell{Opens: true}, 0},
	} {
		got, ok := missileTrack(&tc.m)
		if got != tc.want || ok != (tc.want != 0) {
			t.Errorf("%+v: track %d %v, want %d", tc.m, got, ok, tc.want)
		}
	}
}
