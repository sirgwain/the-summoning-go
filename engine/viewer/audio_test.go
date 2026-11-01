package viewer

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/audio"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

type testVoice struct{ stopped bool }

func (v *testVoice) Playing() bool     { return !v.stopped }
func (v *testVoice) SetVolume(float64) {}
func (v *testVoice) Stop()             { v.stopped = true }

type testSpeaker struct{ played []string }

func (s *testSpeaker) Play(snd *content.Sound) (audio.Voice, error) {
	s.played = append(s.played, snd.ID)
	return &testVoice{}, nil
}

// TestLeverSound pulls the hall's lever: its cue plays. F9 then turns sound off, announcing it.
func TestLeverSound(t *testing.T) {
	v := playIn(t, devpack.Hall, devpack.LeverA)
	sp := &testSpeaker{}
	v.Audio = audio.New(sp, v.Pack)
	pressKey(v, KeyEnter)
	if len(sp.played) == 0 || sp.played[0] != devpack.SFXBlip {
		t.Fatalf("played %v, want %s", sp.played, devpack.SFXBlip)
	}
	pressKey(v, KeyF9)
	if v.Audio.Mode() != audio.ModeEffects || v.Notice != "Sound effects only." {
		t.Errorf("after F9: mode %v, notice %q", v.Audio.Mode(), v.Notice)
	}
	pressKey(v, KeyF9)
	if v.Audio.Mode() != audio.ModeSilent || v.Notice != "Sound off." {
		t.Errorf("after F9 twice: mode %v, notice %q", v.Audio.Mode(), v.Notice)
	}
}
