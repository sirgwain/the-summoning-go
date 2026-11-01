package summoning

import (
	"encoding/binary"
	"fmt"
)

// Songs are RESOURCE type-3 records: music and sound effects for an OPL2. The song format and
// how the game plays a song are in docs/spec/audio.md. This file decodes songs and plays them
// into a timeline of OPL2 register writes; render.go turns that into sound.

const (
	songDescriptorSize = 24
	instrumentSize     = 58
	operatorParams     = 13
)

// Instrument is a song's instrument: the low bytes of its two operators' parameters, in the
// order of audio.md's parameter table, and their waveforms.
type Instrument struct {
	Modulator, Carrier         [operatorParams]uint8
	ModulatorWave, CarrierWave uint8
}

// Song is a decoded music resource.
type Song struct {
	TicksPerBeat int
	BeatsPerBar  int
	// Length is the song's length in ticks as written; playing does not read it.
	Length    uint32
	Rhythm    bool
	BendRange int
	Tempo     int
	// Instruments are the song's instruments, numbered by program changes.
	Instruments []Instrument
	// Events is the event stream: delays and events, ending with FCh.
	Events []byte
}

// DecodeSong decodes a music resource.
func DecodeSong(r Resource) (*Song, error) {
	if r.Type != ResourceMusic {
		return nil, fmt.Errorf("song: resource type %d, want %d", r.Type, ResourceMusic)
	}
	d := r.Descriptor
	if len(d) < songDescriptorSize {
		return nil, fmt.Errorf("song: descriptor of %d bytes, want %d", len(d), songDescriptorSize)
	}
	s := &Song{
		TicksPerBeat: int(d[0]),
		BeatsPerBar:  int(d[1]),
		Length:       binary.LittleEndian.Uint32(d[2:]),
		Rhythm:       d[14] != 0,
		BendRange:    int(d[15]),
		Tempo:        int(binary.LittleEndian.Uint16(d[16:])),
	}
	streamLen := binary.LittleEndian.Uint32(d[6:])
	count := int(d[23])
	p := r.Payload
	if len(p) < count*instrumentSize {
		return nil, fmt.Errorf("song: %d instruments need %d bytes, payload has %d", count, count*instrumentSize, len(p))
	}
	for i := 0; i < count; i++ {
		b := p[i*instrumentSize:]
		var in Instrument
		for k := 0; k < operatorParams; k++ {
			in.Modulator[k] = b[2+2*k]
			in.Carrier[k] = b[28+2*k]
		}
		in.ModulatorWave = b[54]
		in.CarrierWave = b[56]
		s.Instruments = append(s.Instruments, in)
	}
	events := p[count*instrumentSize:]
	if uint64(len(events)) < uint64(streamLen) {
		return nil, fmt.Errorf("song: event stream of %d bytes, payload has %d", streamLen, len(events))
	}
	s.Events = events[:streamLen]
	return s, nil
}

// The song clock is the PC's timer: the game takes its divisor from pitDivisorBase, and the
// timer counts at PITRate.
const (
	pitDivisorBase = 1193180
	PITRate        = 1193182
)

// SongEvent is one entry of a song's timeline: at clock tick Tick, either a register write
// (Divisor 0) or a change of the clock's rate to PITRate/Divisor ticks a second, from the next
// tick on.
type SongEvent struct {
	Tick     int64
	Reg, Val uint8
	Divisor  int
}

// The operator slots: carrier slots, and the default instruments (audio.md).
var (
	carrierSlot = [18]bool{3: true, 4: true, 5: true, 9: true, 10: true, 11: true, 15: true, 16: true, 17: true}

	melodicModulator = [operatorParams]uint8{1, 1, 3, 15, 5, 0, 1, 3, 15, 0, 0, 0, 1}
	melodicCarrier   = [operatorParams]uint8{0, 1, 1, 15, 7, 0, 2, 4, 0, 0, 0, 1, 0}
	bassDrumMod      = [operatorParams]uint8{0, 0, 0, 10, 4, 0, 8, 12, 11, 0, 0, 0, 1}
	bassDrumCar      = [operatorParams]uint8{0, 0, 0, 13, 4, 0, 6, 15, 0, 0, 0, 0, 1}
	snareDrum        = [operatorParams]uint8{0, 12, 0, 15, 11, 0, 8, 5, 0, 0, 0, 0, 0}
	tomTom           = [operatorParams]uint8{0, 4, 0, 15, 11, 0, 7, 5, 0, 0, 0, 0, 0}
	cymbal           = [operatorParams]uint8{0, 1, 0, 15, 11, 0, 5, 5, 0, 0, 0, 0, 0}
	hiHat            = [operatorParams]uint8{0, 1, 0, 15, 11, 0, 7, 5, 0, 0, 0, 0, 0}

	// For percussion channels 6-10: the slot their volume goes to, and their key bit in BDh.
	percussionVolumeSlot = [5]int{15, 16, 14, 17, 13}
	percussionKey        = [5]uint8{0x10, 0x08, 0x04, 0x02, 0x01}
	// For percussion channels 7-10: the one slot a program change sets.
	percussionSlot = [5]int{1: 16, 2: 14, 3: 17, 4: 13}
)

func slotOffset(s int) uint8  { return uint8(s/6*8 + s%6) }
func modulatorSlot(v int) int { return v/3*6 + v%3 }
func slotVoice(s int) int     { return s/6*3 + s%3 }

// freqRows are the 25 rows of F-numbers, row r tuned 4r cents sharp.
var freqRows = func() (rows [25][12]uint16) {
	for r := range rows {
		v := int64(10000+24*r) * 52088 / 250000
		v = v * 16384 * 9 / 111875
		for k := range rows[r] {
			rows[r][k] = uint16((v + 4) >> 3)
			v = v * 106 / 100
		}
	}
	return rows
}()

const (
	voices       = 11
	songChannels = 11
)

// songPlayer plays songs into register writes as the game's music driver does.
type songPlayer struct {
	timeline []SongEvent
	tick     int64

	params     [18][operatorParams]uint8
	wave       [18]uint8
	opVolume   [18]uint8
	waveSelect bool
	rhythm     bool
	keys       uint8
	bendRange  int
	voiceRow   [voices]int
	voiceSemis [voices]int
	voiceNote  [voices]int
	voiceKey   [voices]bool
	chanVolume [songChannels]uint8
	master     int

	song     *Song
	cursor   int
	position uint32
	delay    int
	wait     int
	running  uint8
	loop     uint32
	playing  bool
	restarts int
	err      error
}

// newSongPlayer returns a player in the state the driver leaves when the game starts.
func newSongPlayer() *songPlayer {
	p := &songPlayer{}
	for s := range p.opVolume {
		p.opVolume[s] = 127
	}
	p.setRhythm(false)
	p.writeRhythm()
	p.out(0x08, 0)
	for v := 0; v < 9; v++ {
		p.voiceOff(v)
	}
	p.bendRange = 1
	for s := 0; s < 18; s++ {
		p.out(0xE0+slotOffset(s), 0)
	}
	p.waveSelect = true
	p.out(0x01, 0x20)
	p.setMaster(127)
	return p
}

func (p *songPlayer) out(reg, val uint8) {
	p.timeline = append(p.timeline, SongEvent{Tick: p.tick, Reg: reg, Val: val})
}

func (p *songPlayer) writeLevel(s int) {
	level := 63 - int(p.params[s][8]&0x3F)
	level = 63 - (level*int(p.opVolume[s])*2+127)/254
	p.out(0x40+slotOffset(s), uint8(level)|p.params[s][0]<<6)
}

func (p *songPlayer) writeRhythm() {
	var v uint8
	if p.rhythm {
		v = 0x20
	}
	p.out(0xBD, v|p.keys)
}

func flag(v uint8, bit uint8) uint8 {
	if v != 0 {
		return bit
	}
	return 0
}

func (p *songPlayer) setOperator(s int, params [operatorParams]uint8, wave uint8) {
	p.params[s] = params
	p.wave[s] = wave & 3
	o := slotOffset(s)
	p.writeRhythm()
	p.out(0x08, 0)
	p.writeLevel(s)
	if !carrierSlot[s] {
		p.out(0xC0+uint8(slotVoice(s)), params[2]<<1|boolByte(params[12] == 0))
	}
	p.out(0x60+o, params[3]<<4|params[6]&0x0F)
	p.out(0x80+o, params[4]<<4|params[7]&0x0F)
	p.out(0x20+o, flag(params[9], 0x80)|flag(params[10], 0x40)|flag(params[5], 0x20)|flag(params[11], 0x10)|params[1]&0x0F)
	var w uint8
	if p.waveSelect {
		w = p.wave[s]
	}
	p.out(0xE0+o, w)
}

func boolByte(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func (p *songPlayer) resetOperators() {
	for s := 0; s < 18; s++ {
		if carrierSlot[s] {
			p.setOperator(s, melodicCarrier, 0)
		} else {
			p.setOperator(s, melodicModulator, 0)
		}
	}
	if !p.rhythm {
		return
	}
	p.setOperator(12, bassDrumMod, 0)
	p.setOperator(15, bassDrumCar, 0)
	p.setOperator(16, snareDrum, 0)
	p.setOperator(14, tomTom, 0)
	p.setOperator(17, cymbal, 0)
	p.setOperator(13, hiHat, 0)
}

func (p *songPlayer) setRhythm(on bool) {
	p.rhythm = on
	p.keys = 0
	if on {
		p.voiceOff(6)
		p.voiceOff(7)
		p.voiceOff(8)
		p.tune(8, 24, false)
		p.tune(7, 31, false)
	}
	p.resetOperators()
	p.writeRhythm()
}

func (p *songPlayer) voiceOff(v int) {
	p.out(0xA0+uint8(v), 0)
	p.out(0xB0+uint8(v), 0)
}

// tune sets voice v to a note, keyed on or off.
func (p *songPlayer) tune(v, note int, key bool) {
	p.voiceKey[v] = key
	p.voiceNote[v] = note
	n := note + p.voiceSemis[v]
	n = max(0, min(95, n))
	f := freqRows[p.voiceRow[v]][n%12]
	p.out(0xA0+uint8(v), uint8(f))
	p.out(0xB0+uint8(v), flag(boolByte(key), 0x20)|uint8(n/12)<<2|uint8(f>>8)&3)
}

func (p *songPlayer) volumeSlot(ch int) int {
	if !p.rhythm || ch < 6 {
		return modulatorSlot(ch) + 3
	}
	return percussionVolumeSlot[ch-6]
}

func (p *songPlayer) setVolume(ch int, v uint8) {
	p.chanVolume[ch] = v
	s := p.volumeSlot(ch)
	p.opVolume[s] = uint8(min(127, p.master*int(v)>>7))
	p.writeLevel(s)
}

func (p *songPlayer) setMaster(v int) {
	p.master = max(0, min(127, v))
	n := 9
	if p.rhythm {
		n = 11
	}
	for ch := 0; ch < n; ch++ {
		p.setVolume(ch, p.chanVolume[ch])
	}
}

func (p *songPlayer) hasVoice(ch int) bool {
	return ch < songChannels && (p.rhythm || ch < 9)
}

func (p *songPlayer) noteOn(ch, note int) {
	note = max(0, note-12)
	if ch < 6 || !p.rhythm {
		p.tune(ch, note, true)
		return
	}
	switch ch {
	case 6:
		p.tune(6, note, false)
	case 8:
		p.tune(8, note, false)
		p.tune(7, note+7, false)
	}
	p.keys |= percussionKey[ch-6]
	p.writeRhythm()
}

func (p *songPlayer) noteOff(ch int) {
	if !p.rhythm || ch < 6 {
		p.tune(ch, p.voiceNote[ch], false)
		return
	}
	p.keys &^= percussionKey[ch-6]
	p.writeRhythm()
}

func (p *songPlayer) bend(v int, bend int) {
	if p.rhythm && v > 6 {
		return
	}
	bend = min(bend, 0x3FFF)
	steps := (bend - 0x2000) * p.bendRange * 25 / 0x2000 // truncated toward zero
	semis := steps / 25
	row := steps % 25
	if row < 0 {
		semis--
		row += 25
	}
	p.voiceSemis[v] = semis
	p.voiceRow[v] = row
	p.tune(v, p.voiceNote[v], p.voiceKey[v])
}

func (p *songPlayer) program(ch, n int) {
	if n >= len(p.song.Instruments) {
		return
	}
	in := p.song.Instruments[n]
	switch {
	case !p.rhythm || ch < 6:
		m := modulatorSlot(ch)
		p.setOperator(m, in.Modulator, in.ModulatorWave)
		p.setOperator(m+3, in.Carrier, in.CarrierWave)
	case ch == 6:
		p.setOperator(12, in.Modulator, in.ModulatorWave)
		p.setOperator(15, in.Carrier, in.CarrierWave)
	default:
		p.setOperator(percussionSlot[ch-6], in.Modulator, in.ModulatorWave)
	}
}

func (p *songPlayer) setTempo(tempo int) {
	rate := p.song.TicksPerBeat * tempo / 60
	divisor := 0
	if rate != 0 {
		divisor = pitDivisorBase / max(rate, 19)
	}
	if divisor == 0 {
		divisor = 1 << 16
	}
	p.timeline = append(p.timeline, SongEvent{Tick: p.tick, Divisor: divisor})
}

// start starts a song, as the game does; loopBars above 0 gives it a loop point.
func (p *songPlayer) start(s *Song, loopBars int) {
	if p.playing {
		p.stop()
	}
	p.song = s
	p.loop = 0
	p.restarts = 0
	p.err = nil
	p.cursor = 0
	p.position = 0
	for ch := range p.chanVolume {
		p.chanVolume[ch] = 0
	}
	p.setRhythm(s.Rhythm)
	p.bendRange = max(1, min(12, s.BendRange))
	p.setTempo(s.Tempo)
	p.delay = int(p.next())
	p.wait = max(p.delay, 1)
	p.playing = true
	p.setMaster(p.master)
	if loopBars > 0 {
		p.loop = uint32(loopBars*s.BeatsPerBar*s.TicksPerBeat) - 1
	}
}

// stop stops the song: every channel's note off, every carrier fully attenuated.
func (p *songPlayer) stop() {
	p.playing = false
	for ch := 0; ch < songChannels; ch++ {
		p.noteOff(ch)
	}
	master := p.master
	p.setMaster(0)
	p.master = master
}

// next reads the next byte of the event stream. Reading past its end stops the song with an
// error.
func (p *songPlayer) next() uint8 {
	if p.cursor >= len(p.song.Events) {
		if p.err == nil {
			p.err = fmt.Errorf("song: event stream ends without FCh")
		}
		p.cursor++
		return 0xFC
	}
	b := p.song.Events[p.cursor]
	p.cursor++
	return b
}

func (p *songPlayer) peek() uint8 {
	if p.cursor >= len(p.song.Events) {
		return 0xFC
	}
	return p.song.Events[p.cursor]
}

// advance waits out the current wait, then takes a step. It reports whether the song still
// plays.
func (p *songPlayer) advance() bool {
	if !p.playing {
		return false
	}
	p.tick += int64(p.wait)
	p.step()
	return p.playing
}

func (p *songPlayer) step() {
	p.position += uint32(p.delay)
	restart := p.loop != 0 && p.position >= p.loop
	for {
		if p.err != nil {
			p.stop()
			return
		}
		if restart {
			restart = false
			p.restarts++
			p.cursor = 0
			p.position = 0
			p.next() // the first delay is skipped
		}
		status := p.peek()
		switch status {
		case 0xF8:
			p.next()
			p.delay = 0xF8
		case 0xFC:
			p.next()
			if p.loop != 0 {
				restart = true
				continue
			}
			p.stop()
			return
		case 0xF0:
			p.next()
			p.sysex()
		default:
			p.event()
		}
		if status != 0xF8 {
			p.delay = int(p.next())
		}
		if p.delay != 0 {
			break
		}
	}
	if p.delay == 0xF8 {
		p.delay = 240
		if p.peek() != 0xF8 {
			p.delay += int(p.next())
		}
	}
	p.wait = p.delay
}

// sysex plays an F0h message: the tempo command, or any other, skipped to F7h.
func (p *songPlayer) sysex() {
	start := p.cursor
	id, sub := p.next(), p.next()
	if id == 0x7F && sub == 0 {
		whole, fraction := int(p.next()), int(p.next())
		t := p.song.Tempo
		p.setTempo(int(uint16(fraction*t>>7 + t*whole)))
		p.next() // F7h
		return
	}
	p.cursor = start
	for p.err == nil && p.next() != 0xF7 {
	}
}

func (p *songPlayer) event() {
	if b := p.peek(); b >= 0x80 {
		p.next()
		p.running = b
	}
	ch := int(p.running & 0x0F)
	switch p.running & 0xF0 {
	case 0x90:
		note, velocity := int(p.next()), p.next()
		if !p.hasVoice(ch) {
			return
		}
		if velocity == 0 {
			p.noteOff(ch)
			return
		}
		if int8(p.chanVolume[ch]) != int8(velocity) || velocity >= 0x80 {
			p.setVolume(ch, velocity)
		}
		p.noteOn(ch, note)
	case 0x80:
		p.next()
		p.next()
		if p.hasVoice(ch) {
			p.noteOff(ch)
		}
	case 0xA0:
		v := p.next()
		if p.hasVoice(ch) {
			p.setVolume(ch, v)
		}
	case 0xC0:
		n := int(p.next())
		if p.hasVoice(ch) {
			p.program(ch, n)
		}
	case 0xE0:
		lo, hi := int(p.next()), int(p.next())
		if p.hasVoice(ch) {
			p.bend(ch, lo+hi<<7)
		}
	case 0xB0:
		p.next()
		p.next()
	case 0xD0:
		p.next()
	default:
		if p.err == nil {
			p.err = fmt.Errorf("song: bad status byte %#02x at %d", p.running, p.cursor-1)
		}
	}
}
