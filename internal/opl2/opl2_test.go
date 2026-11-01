package opl2

import (
	"hash/fnv"
	"math"
	"testing"
)

func run(c *Chip, n int) []int16 {
	out := make([]int16, n)
	for i := range out {
		out[i] = c.Sample()
	}
	return out
}

// sineVoice sets channel 0 to a plain carrier: the modulator never attacks (attack rate 0), so
// the carrier sounds alone with the given waveform, attack, sustain level and release.
func sineVoice(c *Chip, waveform, attack, sustain, release uint8) {
	c.Write(0x01, 0x20)
	c.Write(0x20, 0x01) // modulator: multiplier 1
	c.Write(0x40, 0x3F)
	c.Write(0x60, 0x00)
	c.Write(0x80, 0x0F)
	c.Write(0x23, 0x21) // carrier: sustaining, multiplier 1
	c.Write(0x43, 0x00)
	c.Write(0x63, attack<<4)
	c.Write(0x83, sustain<<4|release)
	c.Write(0xE3, waveform)
	c.Write(0xC0, 0x00)
}

func keyOn(c *Chip, ch int, fnum uint16, block uint8) {
	c.Write(0xA0+ch, byte(fnum))
	c.Write(0xB0+ch, 0x20|block<<2|byte(fnum>>8))
}

func keyOff(c *Chip, ch int, fnum uint16, block uint8) {
	c.Write(0xB0+ch, block<<2|byte(fnum>>8))
}

func TestSilentAtPowerOn(t *testing.T) {
	c := New()
	if !c.Silent() {
		t.Fatal("a new chip is not silent")
	}
	for i, v := range run(c, 1000) {
		if v != 0 {
			t.Fatalf("sample %d = %d, want 0", i, v)
		}
	}
}

func TestWritesLandOnePerSample(t *testing.T) {
	c := New()
	c.Write(0xA0, 0x44)
	c.Write(0xB0, 0x11)
	c.Write(0xA1, 0x22)
	if c.Pending() != 3 {
		t.Fatalf("pending %d, want 3", c.Pending())
	}
	for want := 2; want >= 0; want-- {
		c.Sample()
		if c.Pending() != want {
			t.Fatalf("pending %d, want %d", c.Pending(), want)
		}
	}
	if c.ch[0].fNumber != 0x144 || c.ch[0].block != 4 || c.ch[1].fNumber != 0x22 {
		t.Fatalf("channel state %+v %+v", c.ch[0], c.ch[1])
	}
}

func TestSineFrequency(t *testing.T) {
	for _, tc := range []struct {
		fnum  uint16
		block uint8
	}{{0x200, 4}, {0x16B, 5}, {0x287, 3}} {
		c := New()
		sineVoice(c, 0, 15, 0, 0)
		keyOn(c, 0, tc.fnum, tc.block)
		run(c, 100)
		samples := run(c, Rate)
		crossings := 0
		for i := 1; i < len(samples); i++ {
			if samples[i-1] < 0 && samples[i] >= 0 {
				crossings++
			}
		}
		want := float64(tc.fnum) * math.Pow(2, float64(tc.block)) * Rate / (1 << 20)
		if math.Abs(float64(crossings)-want) > 1.5 {
			t.Errorf("fnum %#x block %d: %d cycles a second, want %.1f", tc.fnum, tc.block, crossings, want)
		}
	}
}

func TestInstantAttackAndSustainLevel(t *testing.T) {
	c := New()
	sineVoice(c, 0, 15, 8, 0)
	c.Write(0x63, 0xF4) // attack 15, decay 4
	keyOn(c, 0, 0x200, 4)
	for c.Pending() > 0 {
		c.Sample()
	}
	c.Sample()
	if c.op[3].envelope != 0 {
		t.Fatalf("envelope %d right after key-on at attack rate 15, want 0", c.op[3].envelope)
	}
	run(c, Rate)
	// Sustain level 8 holds the envelope where its top five bits reach 8.
	if c.op[3].stage != stageSustain || c.op[3].envelope != 8<<4 {
		t.Fatalf("stage %d envelope %d, want sustain at %d", c.op[3].stage, c.op[3].envelope, 8<<4)
	}
}

// releaseSamples is how long the carrier takes to die away after a key-off at release rate r.
func releaseSamples(r uint8) int {
	c := New()
	sineVoice(c, 0, 15, 0, r)
	keyOn(c, 0, 0x200, 4)
	run(c, 1000)
	keyOff(c, 0, 0x200, 4)
	run(c, 1)
	n := 0
	for c.op[3].envelope < envelopeOffZone {
		c.Sample()
		n++
		if n > 60*Rate {
			return n
		}
	}
	return n
}

func TestReleaseRatesDouble(t *testing.T) {
	prev := releaseSamples(3)
	for r := uint8(4); r <= 9; r++ {
		n := releaseSamples(r)
		ratio := float64(prev) / float64(n)
		if ratio < 1.8 || ratio > 2.2 {
			t.Errorf("release rate %d takes %d samples, rate %d %d: ratio %.2f, want about 2", r, n, r-1, prev, ratio)
		}
		prev = n
	}
	if !New().Silent() {
		t.Fatal("unreachable")
	}
}

func TestKeyOffThenOnRetriggers(t *testing.T) {
	c := New()
	sineVoice(c, 0, 8, 0, 0)
	keyOn(c, 0, 0x200, 4)
	run(c, Rate/2)
	if c.op[3].stage != stageSustain {
		t.Fatalf("stage %d, want sustain", c.op[3].stage)
	}
	keyOff(c, 0, 0x200, 4)
	keyOn(c, 0, 0x200, 4)
	sawRelease, sawAttack := false, false
	for i := 0; i < 10; i++ {
		c.Sample()
		switch c.op[3].stage {
		case stageRelease:
			sawRelease = true
		case stageAttack:
			sawAttack = sawRelease
		}
	}
	if !sawAttack {
		t.Fatal("a key-off followed by a key-on did not restart the envelope")
	}
}

func TestKeyOnWhileKeyedDoesNotRetrigger(t *testing.T) {
	c := New()
	sineVoice(c, 0, 8, 0, 0)
	keyOn(c, 0, 0x200, 4)
	run(c, Rate/2)
	keyOn(c, 0, 0x180, 4)
	for i := 0; i < 10; i++ {
		c.Sample()
		if c.op[3].stage != stageSustain {
			t.Fatalf("stage %d after a second key-on, want sustain", c.op[3].stage)
		}
	}
	if c.ch[0].fNumber != 0x180 {
		t.Fatalf("fnum %#x, want the new pitch", c.ch[0].fNumber)
	}
}

func TestWaveforms(t *testing.T) {
	for w := uint8(0); w < 4; w++ {
		c := New()
		sineVoice(c, w, 15, 0, 0)
		keyOn(c, 0, 0x200, 4)
		run(c, 100)
		samples := run(c, Rate/10)
		var negative, zero, positive int
		for _, v := range samples {
			switch {
			case v < -1:
				negative++
			case v > 1:
				positive++
			default:
				zero++
			}
		}
		n := len(samples)
		switch w {
		case 0: // sine
			if negative < n/3 || positive < n/3 {
				t.Errorf("waveform 0: %d negative, %d positive of %d", negative, positive, n)
			}
		case 1: // half-sine
			if negative != 0 || zero < n*2/5 || positive < n*2/5 {
				t.Errorf("waveform 1: %d negative, %d near zero, %d positive of %d", negative, zero, positive, n)
			}
		case 2: // absolute sine
			if negative != 0 || positive < n*9/10 {
				t.Errorf("waveform 2: %d negative, %d positive of %d", negative, positive, n)
			}
		case 3: // quarter pulses
			if negative != 0 || zero < n*2/5 || positive < n*2/5 {
				t.Errorf("waveform 3: %d negative, %d near zero, %d positive of %d", negative, zero, positive, n)
			}
		}
	}
}

func TestWaveformWritesNeedWaveSelect(t *testing.T) {
	c := New()
	c.Write(0xE3, 2)
	run(c, 2)
	if c.op[3].waveform != 0 {
		t.Fatalf("waveform %d with selection off, want 0", c.op[3].waveform)
	}
	c.Write(0x01, 0x20)
	c.Write(0xE3, 2)
	run(c, 3)
	if c.op[3].waveform != 2 {
		t.Fatalf("waveform %d with selection on, want 2", c.op[3].waveform)
	}
	c.Write(0x01, 0x00)
	c.Write(0xE3, 1)
	run(c, 3)
	if c.op[3].waveform != 2 {
		t.Fatalf("waveform %d after selection turned off, want the one chosen (2)", c.op[3].waveform)
	}
}

func TestHighConnectionBitsIgnored(t *testing.T) {
	c := New()
	c.Write(0xC0, 0xFF)
	run(c, 2)
	if c.ch[0].feedback != 7 || c.ch[0].connection != 1 {
		t.Fatalf("feedback %d connection %d, want 7 and 1", c.ch[0].feedback, c.ch[0].connection)
	}
}

// energy is the variance of the samples: silent operators output -1 on their negative half,
// so silence has a small offset.
func energy(samples []int16) float64 {
	var sum, sq float64
	for _, v := range samples {
		sum += float64(v)
		sq += float64(v) * float64(v)
	}
	n := float64(len(samples))
	return sq/n - (sum/n)*(sum/n)
}

// drums sets the five rhythm instruments up with short, loud sounds.
func drums(c *Chip) {
	for _, s := range []int{12, 13, 14, 15, 16, 17} {
		off := s/6*8 + s%6
		c.Write(0x20+off, 0x01)
		c.Write(0x40+off, 0x00)
		c.Write(0x60+off, 0xF6)
		c.Write(0x80+off, 0x06)
	}
	keyOff(c, 6, 0x200, 3)
	keyOff(c, 7, 0x2B0, 4)
	keyOff(c, 8, 0x180, 4)
	c.Write(0xBD, 0x20)
}

func TestRhythmKeys(t *testing.T) {
	for _, bit := range []byte{0x10, 0x08, 0x04, 0x02, 0x01} {
		c := New()
		drums(c)
		run(c, 2000)
		if e := energy(run(c, 2000)); e > 50 {
			t.Fatalf("drums sound with no key: energy %.1f", e)
		}
		c.Write(0xBD, 0x20|bit)
		if e := energy(run(c, 2000)); e < 1000 {
			t.Errorf("drum bit %#x: energy %.1f, want it to sound", bit, e)
		}
	}
}

func TestRhythmNeedsRhythmMode(t *testing.T) {
	c := New()
	drums(c)
	c.Write(0xBD, 0x1F) // keys without rhythm mode
	run(c, 100)
	if e := energy(run(c, 2000)); e > 50 {
		t.Fatalf("drum keys sound without rhythm mode: energy %.1f", e)
	}
}

// script plays a fixed set of writes: an FM voice with feedback, vibrato and tremolo, an
// additive voice, and the drums.
func script(c *Chip) []int16 {
	c.Write(0x01, 0x20)
	c.Write(0xBD, 0xC0)
	c.Write(0x20, 0xE2)
	c.Write(0x40, 0x9A)
	c.Write(0x60, 0xA4)
	c.Write(0x80, 0x55)
	c.Write(0x23, 0x61)
	c.Write(0x43, 0x05)
	c.Write(0x63, 0xC3)
	c.Write(0x83, 0x34)
	c.Write(0xE0, 0x01)
	c.Write(0xE3, 0x02)
	c.Write(0xC0, 0x0C)
	c.Write(0x21, 0x31)
	c.Write(0x24, 0x22)
	c.Write(0x41, 0x10)
	c.Write(0x44, 0x08)
	c.Write(0x61, 0xF2)
	c.Write(0x64, 0xF3)
	c.Write(0x81, 0x24)
	c.Write(0x84, 0x26)
	c.Write(0xE4, 0x03)
	c.Write(0xC1, 0x05)
	keyOn(c, 0, 0x16B, 4)
	keyOn(c, 1, 0x287, 3)
	out := run(c, Rate/2)
	drums(c)
	c.Write(0xBD, 0xF5)
	out = append(out, run(c, Rate/4)...)
	keyOff(c, 0, 0x16B, 4)
	c.Write(0xBD, 0xEA)
	out = append(out, run(c, Rate/4)...)
	c.Write(0xBD, 0x20)
	keyOff(c, 1, 0x287, 3)
	return append(out, run(c, Rate)...)
}

// TestGolden pins the output of a fixed script, so that a change to the core shows up. The
// hash matches opal's own build playing the same writes at the same samples.
func TestGolden(t *testing.T) {
	a, b := script(New()), script(New())
	h := fnv.New64a()
	for i, v := range a {
		if b[i] != v {
			t.Fatalf("two runs differ at sample %d", i)
		}
		h.Write([]byte{byte(v), byte(v >> 8)})
	}
	const want uint64 = 0x840c2cdc6f19ce96
	if got := h.Sum64(); got != want {
		t.Fatalf("script hash %#x, want %#x", got, want)
	}
}

func TestResampler(t *testing.T) {
	c := New()
	sineVoice(c, 0, 15, 0, 0)
	keyOn(c, 0, 0x200, 4)
	r := NewResampler(c, 44100)
	samples := make([]int16, 44100)
	for i := range samples {
		samples[i] = r.Next()
	}
	if c.chipSamples < Rate-2 || c.chipSamples > Rate+2 {
		t.Fatalf("one second at 44100 ran %d chip samples, want about %d", c.chipSamples, Rate)
	}
	crossings := 0
	for i := 1; i < len(samples); i++ {
		if samples[i-1] < 0 && samples[i] >= 0 {
			crossings++
		}
	}
	want := 0x200 * 16.0 * Rate / (1 << 20)
	if math.Abs(float64(crossings)-want) > 1.5 {
		t.Fatalf("%d cycles a second at 44100, want %.1f", crossings, want)
	}
}
