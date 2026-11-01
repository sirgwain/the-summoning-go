package summoning

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

// testInstrument has distinct values in every parameter, so a write shows which ones it took.
var testInstrument = Instrument{
	Modulator:     [13]uint8{2, 3, 5, 10, 6, 1, 4, 7, 20, 1, 0, 1, 0},
	Carrier:       [13]uint8{1, 2, 0, 12, 3, 0, 5, 8, 4, 0, 1, 0, 0},
	ModulatorWave: 1,
	CarrierWave:   2,
}

func testSong(rhythm bool, events ...byte) *Song {
	return &Song{
		TicksPerBeat: 240, BeatsPerBar: 4, Rhythm: rhythm, BendRange: 1, Tempo: 120,
		Instruments: []Instrument{testInstrument}, Events: events,
	}
}

// play starts a song on a player fresh from the game's start and plays it to its end. It
// returns the player and the index of the first timeline entry after the game's start.
func play(t *testing.T, s *Song, loopBars, maxSteps int) (*songPlayer, int) {
	t.Helper()
	p := newSongPlayer()
	first := len(p.timeline)
	p.start(s, loopBars)
	for i := 0; p.advance(); i++ {
		if i == maxSteps {
			break
		}
	}
	return p, first
}

// writesAt formats the register writes at a tick as "reg=val" pairs.
func writesAt(p *songPlayer, tick int64) string {
	var b []string
	for _, e := range p.timeline {
		if e.Tick == tick && e.Divisor == 0 {
			b = append(b, fmt.Sprintf("%02X=%02X", e.Reg, e.Val))
		}
	}
	return strings.Join(b, " ")
}

func hasWrites(got, want string) bool { return strings.Contains(" "+got+" ", " "+want+" ") }

func TestFreqRows(t *testing.T) {
	want := [12]uint16{343, 364, 385, 408, 433, 459, 486, 515, 546, 579, 614, 650}
	if freqRows[0] != want {
		t.Fatalf("row 0 %v, want %v", freqRows[0], want)
	}
	for r := 1; r < 25; r++ {
		for k := 0; k < 12; k++ {
			if freqRows[r][k] < freqRows[r-1][k] {
				t.Fatalf("row %d note %d %d is below row %d's %d", r, k, freqRows[r][k], r-1, freqRows[r-1][k])
			}
		}
	}
	// Row 24 is 96 cents sharp of row 0.
	if got := float64(freqRows[24][0]) / float64(freqRows[0][0]); got < 1.054 || got > 1.059 {
		t.Fatalf("row 24 / row 0 = %.4f, want about 1.057", got)
	}
}

func TestDecodeSong(t *testing.T) {
	desc := make([]byte, songDescriptorSize)
	desc[0], desc[1] = 240, 4
	binary.LittleEndian.PutUint32(desc[2:], 960)
	binary.LittleEndian.PutUint32(desc[6:], 3)
	desc[14], desc[15] = 1, 2
	binary.LittleEndian.PutUint16(desc[16:], 140)
	desc[23] = 1
	inst := make([]byte, instrumentSize)
	for k := 0; k < 13; k++ {
		inst[2+2*k] = byte(k + 1)
		inst[3+2*k] = 0xFF // high bytes do not count
		inst[28+2*k] = byte(k + 20)
	}
	inst[54], inst[56] = 3, 1
	payload := append(inst, 0x00, 0xFC, 0x00, 0xEE) // the stream is 3 bytes; the rest is not
	s, err := DecodeSong(Resource{Type: ResourceMusic, Descriptor: desc, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if s.TicksPerBeat != 240 || s.BeatsPerBar != 4 || s.Length != 960 || !s.Rhythm || s.BendRange != 2 || s.Tempo != 140 {
		t.Fatalf("header %+v", s)
	}
	if len(s.Instruments) != 1 || s.Instruments[0].Modulator[12] != 13 || s.Instruments[0].Carrier[0] != 20 ||
		s.Instruments[0].ModulatorWave != 3 || s.Instruments[0].CarrierWave != 1 {
		t.Fatalf("instrument %+v", s.Instruments)
	}
	if string(s.Events) != "\x00\xFC\x00" {
		t.Fatalf("events % X", s.Events)
	}

	desc[23] = 2
	if _, err := DecodeSong(Resource{Type: ResourceMusic, Descriptor: desc, Payload: payload}); err == nil {
		t.Error("two instruments in a payload of one: no error")
	}
	desc[23] = 1
	binary.LittleEndian.PutUint32(desc[6:], 9)
	if _, err := DecodeSong(Resource{Type: ResourceMusic, Descriptor: desc, Payload: payload}); err == nil {
		t.Error("a stream longer than the payload: no error")
	}
}

func TestGameStartWrites(t *testing.T) {
	p := newSongPlayer()
	got := writesAt(p, 0)
	// Slot 0 takes the melodic modulator default first: level 15 at volume 127, key scale 1.
	if want := "BD=00 08=00 40=4F C0=06 60=F1 80=53 20=01 E0=00"; !strings.HasPrefix(got, want) {
		t.Fatalf("game start begins %q, want %q", got[:min(len(got), 60)], want)
	}
	// Slot 3 takes the melodic carrier default: no C0h write.
	if !hasWrites(got, "BD=00 08=00 43=00 63=F2 83=74 23=11 E3=00") {
		t.Fatal("slot 3 does not take the melodic carrier default")
	}
	// Waveform selection goes on, then the master volume with channel volumes 0 fully
	// attenuates every carrier.
	if !strings.HasSuffix(got, "F5=00 01=20 43=3F 44=3F 45=3F 4B=3F 4C=3F 4D=3F 53=3F 54=3F 55=3F") {
		t.Fatalf("game start ends %q", got[len(got)-80:])
	}
}

func TestSongStartSilencesCarriers(t *testing.T) {
	p, first := play(t, testSong(false, 0x00, 0xFC), 0, 10)
	start := p.timeline[first:]
	var sb []string
	for _, e := range start {
		if e.Tick == 0 && e.Divisor == 0 {
			sb = append(sb, fmt.Sprintf("%02X=%02X", e.Reg, e.Val))
		}
	}
	got := strings.Join(sb, " ")
	// The melodic carrier default has level 0; at volume 0 it is fully attenuated (3Fh).
	if !strings.HasSuffix(got, "43=3F 44=3F 45=3F 4B=3F 4C=3F 4D=3F 53=3F 54=3F 55=3F") {
		t.Fatalf("song start ends %q", got[max(0, len(got)-80):])
	}
}

func TestNoteOnAndOff(t *testing.T) {
	s := testSong(false,
		0x00, 0xC0, 0x00, // program 0 on channel 0
		0x00, 0x90, 0x3C, 0x64, // note 60, velocity 100
		0x0A, 0x80, 0x3C, 0x00, // ten ticks later, note off
		0x00, 0xFC)
	p, _ := play(t, s, 0, 100)
	at1 := writesAt(p, 1)
	// The program change: the modulator to slot 0, the carrier to slot 3.
	if !hasWrites(at1, "BD=00 08=00 40=94 C0=0B 60=A4 80=67 20=B3 E0=01") {
		t.Errorf("modulator writes missing from %q", at1)
	}
	if !hasWrites(at1, "BD=00 08=00 43=7F 63=C5 83=38 23=42 E3=02") {
		t.Errorf("carrier writes missing from %q", at1)
	}
	// Velocity 100 sets the channel volume: carrier level 4 at volume 100*127/128 = 99.
	// level = 63 - ((63-4)*99*2+127)/254 = 17; key scale 1.
	// Then note 60 sounds as 48: block 4, F-number 343 (157h).
	if !strings.HasSuffix(at1, "43=51 A0=57 B0=31") {
		t.Errorf("note on: tick 1 ends %q, want 43=51 A0=57 B0=31", at1)
	}
	at11 := writesAt(p, 11)
	if !strings.HasPrefix(at11, "A0=57 B0=11") {
		t.Errorf("note off: tick 11 is %q, want A0=57 B0=11 first", at11)
	}
	// The song ends: note off on all eleven channels (9 and 10 on registers the OPL2 lacks),
	// then every carrier fully attenuated.
	if !hasWrites(at11, "A9=57 B9=01 AA=57 BA=01 43=7F") {
		t.Errorf("stop writes missing from %q", at11)
	}
	if p.playing || p.err != nil {
		t.Fatalf("playing %v err %v after FCh", p.playing, p.err)
	}
}

func TestNoteOnWithoutKeyOff(t *testing.T) {
	s := testSong(false,
		0x00, 0x90, 0x3C, 0x40,
		0x05, 0x90, 0x3E, 0x40, // a second note on the same channel, no note off
		0x05, 0xFC)
	p, _ := play(t, s, 0, 10)
	// The second note rewrites the pitch keyed on, with no key-off between.
	if got := writesAt(p, 6); got != "A0=81 B0=31" {
		t.Fatalf("tick 6 is %q, want A0=81 B0=31", got)
	}
}

func TestVolumeEvent(t *testing.T) {
	s := testSong(false,
		0x00, 0xC0, 0x00,
		0x00, 0xA0, 0x40, // channel volume 64
		0x00, 0xFC)
	p, _ := play(t, s, 0, 10)
	// Scaled by the master volume: 127*64/128 = 63. level = 63 - (59*63*2+127)/254 = 34 (22h),
	// with key scale 1 (40h).
	if got := writesAt(p, 1); !hasWrites(got, "E3=02 43=62") {
		t.Fatalf("tick 1 %q lacks the volume write 43=62 after the program change", got)
	}
}

func TestPitchBend(t *testing.T) {
	for _, tc := range []struct {
		lo, hi byte
		rangeS int
		want   string
	}{
		{0x00, 0x60, 2, "A0=6C B0=31"}, // 3000h, range 2: +1 semitone, row 0: 364
		{0x00, 0x20, 2, "A0=8A B0=2E"}, // 1000h: -1 semitone: note 47, block 3, F-number 650
		{0x00, 0x42, 2, "A0=58 B0=31"}, // 2100h: step 1, row 1
		{0x00, 0x3E, 2, "A0=B0 B0=2E"}, // 1F00h: step -1, a semitone down on row 24: 2B0h
		{0x00, 0x40, 2, "A0=57 B0=31"}, // centered: row 0
	} {
		s := testSong(false,
			0x00, 0x90, 0x3C, 0x40,
			0x05, 0xE0, tc.lo, tc.hi,
			0x05, 0xFC)
		s.BendRange = tc.rangeS
		p, _ := play(t, s, 0, 10)
		// The bend retunes the sounding note at once, keyed on.
		if got := writesAt(p, 6); got != tc.want {
			t.Errorf("bend %02X%02X: tick 6 is %q, want %q", tc.hi, tc.lo, got, tc.want)
		}
	}
}

func TestRhythm(t *testing.T) {
	s := testSong(true,
		0x00, 0xCA, 0x00, // program 0 on the hi-hat
		0x00, 0x98, 0x3C, 0x40, // tom-tom note 60
		0x00, 0x9A, 0x3C, 0x40, // hi-hat
		0x05, 0x88, 0x3C, 0x00, // tom-tom off
		0x00, 0xFC)
	p, first := play(t, s, 0, 10)
	var start []string
	for _, e := range p.timeline[first:] {
		if e.Tick == 0 && e.Divisor == 0 {
			start = append(start, fmt.Sprintf("%02X=%02X", e.Reg, e.Val))
		}
	}
	// Rhythm mode tunes voice 8 to note 24 and voice 7 to note 31, keyed off.
	if got := strings.Join(start, " "); !strings.HasPrefix(got, "A6=00 B6=00 A7=00 B7=00 A8=00 B8=00 A8=57 B8=09 A7=03 B7=0A") {
		t.Fatalf("rhythm song start begins %q", got[:60])
	}
	at1 := writesAt(p, 1)
	// The hi-hat is slot 13, a modulator slot of voice 7: its program change writes C7h. Its
	// volume is 0 from the song start, so its level is fully attenuated.
	if !hasWrites(at1, "BD=20 08=00 51=BF C7=0B 71=A4 91=67 31=B3 F1=01") {
		t.Errorf("hi-hat program change missing from %q", at1)
	}
	// The tom-tom tunes voice 8 to the note and voice 7 a fifth up, keyed off, then keys BDh.
	if !hasWrites(at1, "A8=57 B8=11 A7=03 B7=12 BD=24") {
		t.Errorf("tom-tom note on missing from %q", at1)
	}
	// The hi-hat sets its bit only.
	if !strings.HasSuffix(at1, "BD=25") {
		t.Errorf("hi-hat note on: tick 1 ends %q, want BD=25", at1)
	}
	if got := writesAt(p, 6); !strings.HasPrefix(got, "BD=21") {
		t.Errorf("tom-tom off: tick 6 is %q, want BD=21 first", got)
	}
}

func TestTempo(t *testing.T) {
	s := testSong(false,
		0x00, 0xF0, 0x7F, 0x00, 0x02, 0x40, 0xF7, // tempo 120*2 + 120*64/128 = 300
		0x05, 0xFC)
	p, _ := play(t, s, 0, 10)
	var divisors []string
	for _, e := range p.timeline {
		if e.Divisor != 0 {
			divisors = append(divisors, fmt.Sprintf("%d@%d", e.Divisor, e.Tick))
		}
	}
	// 120 BPM: 480 ticks a second, divisor 1193180/480 = 2485. 300 BPM: 1200, divisor 994.
	if got := strings.Join(divisors, " "); got != "2485@0 994@1" {
		t.Fatalf("divisors %q, want 2485@0 994@1", got)
	}

	slow := testSong(false, 0x00, 0xFC)
	slow.Tempo = 4 // 16 ticks a second, raised to 19
	p, _ = play(t, slow, 0, 10)
	for _, e := range p.timeline {
		if e.Divisor != 0 && e.Divisor != 1193180/19 {
			t.Fatalf("divisor %d at tempo 4, want %d", e.Divisor, 1193180/19)
		}
	}
}

func TestDelays(t *testing.T) {
	for _, tc := range []struct {
		name   string
		events []byte
		want   int64 // the tick of the note
		note   string
	}{
		{"first delay 0 waits a tick", []byte{0x00, 0x90, 0x3C, 0x40, 0x05, 0xFC}, 1, "A0=57 B0=31"},
		{"first delay 5", []byte{0x05, 0x90, 0x3C, 0x40, 0x05, 0xFC}, 5, "A0=57 B0=31"},
		{"F8h adds 240", []byte{0x00, 0x90, 0x3C, 0x40, 0xF8, 0x10, 0x80, 0x3C, 0x00, 0x05, 0xFC}, 1 + 240 + 16, "A0=57 B0=11"},
		{"F8h F8h", []byte{0x00, 0x90, 0x3C, 0x40, 0xF8, 0xF8, 0x05, 0x80, 0x3C, 0x00, 0x05, 0xFC}, 1 + 240 + 245, "A0=57 B0=11"},
	} {
		p, _ := play(t, testSong(false, tc.events...), 0, 10)
		if got := writesAt(p, tc.want); !strings.HasSuffix(got, tc.note) {
			t.Errorf("%s: tick %d is %q, want it to end with %s", tc.name, tc.want, got, tc.note)
		}
	}
}

func TestRunningStatusAndIgnoredEvents(t *testing.T) {
	s := testSong(false,
		0x00, 0x90, 0x3C, 0x40,
		0x00, 0x3E, 0x40, // running status: another note on, on channel 0
		0x00, 0xB0, 0x07, 0x64, // control change: ignored
		0x00, 0xD0, 0x10, // channel pressure: ignored
		0x00, 0xF0, 0x43, 0x12, 0x00, 0xF7, // another F0h message: skipped
		0x00, 0x91, 0x3C, 0x40, // note on channel 1
		0x00, 0x99, 0x3C, 0x40, // channel 9 has no voice without rhythm mode
		0x00, 0xC0, 0x05, // instrument 5 does not exist
		0x05, 0xFC)
	p, _ := play(t, s, 0, 10)
	at1 := writesAt(p, 1)
	if !strings.Contains(at1, "A0=57 B0=31 A0=81 B0=31") {
		t.Errorf("running status: %q lacks both notes", at1)
	}
	if !strings.HasSuffix(at1, "B0=31 44=20 A1=57 B1=31") {
		t.Errorf("channel 1's note missing after the ignored events: %q", at1)
	}
	if strings.Contains(at1, "A9=") || strings.Contains(at1, "B9=") {
		t.Errorf("channel 9 played without rhythm mode: %q", at1)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
}

func TestBadStatusStops(t *testing.T) {
	p, _ := play(t, testSong(false, 0x00, 0xF3, 0x00, 0xFC), 0, 10)
	if p.err == nil || p.playing {
		t.Fatalf("err %v playing %v after status F3h", p.err, p.playing)
	}
	p, _ = play(t, testSong(false, 0x00, 0x90, 0x3C, 0x40, 0x05), 0, 10)
	if p.err == nil || p.playing {
		t.Fatalf("err %v playing %v for a stream without FCh", p.err, p.playing)
	}
}

func TestLoop(t *testing.T) {
	// One bar of 4 beats at 240 ticks: the loop point is 959.
	s := testSong(false,
		0x00, 0x90, 0x3C, 0x40,
		0x00, 0x91, 0x3C, 0x40,
		0xF8, 0xF8, 0xF8, 0xF0, // a delay of 960: past the loop point
		0x80, 0x3C, 0x00, // dropped by the restart
		0x00, 0xFC)
	p, _ := play(t, s, 1, 6)
	if p.restarts != 2 || !p.playing {
		t.Fatalf("restarts %d playing %v, want 2 restarts, still playing", p.restarts, p.playing)
	}
	// The restart plays the first events in its own tick, without stopping the notes.
	if got := writesAt(p, 961); got != "A0=57 B0=31 A1=57 B1=31" {
		t.Fatalf("tick 961 is %q, want both notes again and nothing else", got)
	}

	// A song shorter than its loop point restarts at FCh.
	short := testSong(false, 0x00, 0x90, 0x3C, 0x40, 0x10, 0xFC)
	p, _ = play(t, short, 4, 3)
	if p.restarts != 3 || !p.playing {
		t.Fatalf("restarts %d playing %v at FCh with a loop point", p.restarts, p.playing)
	}
	if got := writesAt(p, 17); got != "A0=57 B0=31" {
		t.Fatalf("tick 17 is %q, want the note again", got)
	}
}
