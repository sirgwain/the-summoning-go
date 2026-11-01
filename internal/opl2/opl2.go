// Copyright (c) 2026 Bitdancer (github.com/RealBitdancer). SPDX-License-Identifier: MIT.
// Go port for this repository; see LICENSE in this directory.

// Package opl2 emulates the Yamaha YM3812 (OPL2), the FM chip of the AdLib card. It is a port
// of the core of opal (github.com/RealBitdancer/opal), an OPL3 emulator that matches the chip's
// integer arithmetic, cut down to what the OPL2 has: one register array, nine two-operator
// channels, waveforms 0-3 and rhythm mode, with mono output and no timers.
//
// The chip runs at its own rate, Rate. Write queues register writes, which land one per
// sample as the bus allows, so a key-off followed by a key-on restarts a note. Sample gives one
// sample at Rate; Resampler gives samples at another rate.
package opl2

// Rate is the chip's sample rate: its 3.579545 MHz clock divided by 72.
const Rate = 49716

const (
	slotCount    = 18
	channelCount = 9
	voiceStride  = 3 // a channel's carrier slot is its modulator slot + 3

	// The mix is taken after this slot's turn in each sample, so slots past it count with
	// their previous sample's output, as in the chip's pipeline.
	mixSlot = 14
	// opal turns 36 slots a sample, stepping the noise generator at each; the 18 that the
	// OPL2 does not have still step it.
	noiseStepsPerSample = 36

	bassDrumChannel = 6
	rhythmFirstSlot = 12
	hiHatSlot       = 13
	snareSlot       = 16
	cymbalSlot      = 17

	hiHatPhaseNoisy = 0xD0
	hiHatPhaseQuiet = 0x34
	cymbalPhase     = 0x80

	envelopeSilent  = 511
	envelopeOffZone = 504
	envelopePeriod  = 1<<36 - 1
	rateFast        = 48
	rateInstant     = 60
	tremoloSteps    = 210
	tremoloPeak     = 105
	audibleLimit    = 3072
	implicitOne     = 1024
)

// Envelope stages.
const (
	stageAttack = iota
	stageDecay
	stageSustain
	stageRelease
)

// Channel algorithms: how a channel's two operators connect and how loud the result is.
type algorithm struct {
	chain    uint8 // bit p: the operator at position p is modulated by the one before it
	carriers uint8 // bit p: the operator at position p is heard
	gain     int32
	feedback bool // the operator at position 0 modulates itself
}

const (
	algFM = iota
	algAdditive
	algKickFM
	algKickDirect
	algPercussion
)

var algorithms = [...]algorithm{
	algFM:         {chain: 0x2, carriers: 0x2, gain: 1, feedback: true},
	algAdditive:   {chain: 0x0, carriers: 0x3, gain: 1, feedback: true},
	algKickFM:     {chain: 0x2, carriers: 0x2, gain: 2, feedback: true},
	algKickDirect: {chain: 0x0, carriers: 0x2, gain: 2, feedback: true},
	algPercussion: {chain: 0x0, carriers: 0x3, gain: 2, feedback: false},
}

var mulTimes2 = [16]uint32{1, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 20, 24, 24, 30, 30}

var kslROM = [16]int{0, 32, 40, 45, 48, 51, 53, 55, 56, 58, 59, 60, 61, 62, 63, 64}

// The channel each slot belongs to.
var slotChannel = [slotCount]int{0, 1, 2, 0, 1, 2, 3, 4, 5, 3, 4, 5, 6, 7, 8, 6, 7, 8}

// The bit of register BDh that keys each rhythm slot, from slot 12 to 17.
var rhythmKeyBit = [6]uint{4, 0, 2, 4, 3, 1}

type vibratoStep struct {
	halves   int
	negative bool
}

var vibratoSteps = [8]vibratoStep{
	{0, false}, {1, false}, {2, false}, {1, false},
	{0, false}, {1, true}, {2, true}, {1, true},
}

type operator struct {
	stage    int
	key      uint8 // bit 0: keyed by its channel, bit 1: keyed by the rhythm register
	out      int16
	prevOut  int16
	phase    uint32
	envelope uint16

	tremolo      bool
	vibrato      bool
	sustainHold  bool
	keyScaleRate bool
	multiple     uint8
	keyScaleLvl  uint8
	totalLevel   uint8
	attackRate   uint8
	decayRate    uint8
	sustainLevel uint8
	releaseRate  uint8
	waveform     uint8
}

type channel struct {
	fNumber    uint16
	block      uint8
	keyScale   uint8 // the key scale number, latched when the frequency is written
	feedback   uint8
	connection uint8
	algorithm  int
}

type write struct {
	due uint64
	reg uint8
	val uint8
}

// Chip is one OPL2. The zero value is not ready; use New.
type Chip struct {
	ch [channelCount]channel
	op [slotCount]operator

	rhythmMode   bool
	noteSelect   bool
	tremoloDepth bool
	vibratoDepth bool
	waveSelect   bool

	tremoloPosition uint8
	tremoloLevel    uint8
	sampleCounter   uint16
	hiHatPhase      uint16
	cymbalPhase     uint16
	noise           uint32
	chipSamples     uint64

	queue   []write
	nextDue uint64 // the first sample a new write may land on
}

// New returns a chip in its power-on state: all operators silent, all registers 0.
func New() *Chip {
	c := &Chip{noise: 1}
	for i := range c.op {
		c.op[i].stage = stageRelease
		c.op[i].envelope = envelopeSilent
	}
	c.refreshRouting()
	return c
}

// Write queues a write of val to register reg. Queued writes land one per sample, in order.
func (c *Chip) Write(reg int, val byte) {
	due := c.nextDue
	if due < c.chipSamples {
		due = c.chipSamples
	}
	c.queue = append(c.queue, write{due: due, reg: uint8(reg), val: val})
	c.nextDue = due + 1
}

// Pending is the number of queued writes not yet applied.
func (c *Chip) Pending() int { return len(c.queue) }

// Silent reports whether every operator has died away and no write is pending.
func (c *Chip) Silent() bool {
	if len(c.queue) > 0 {
		return false
	}
	for i := range c.op {
		if c.op[i].envelope < envelopeOffZone {
			return false
		}
	}
	return true
}

// Sample advances the chip one sample at Rate and returns its output.
func (c *Chip) Sample() int16 {
	odd, count := c.envelopeTime()
	var mix int32
	for s := 0; s < slotCount; s++ {
		c.turn(s, odd, count)
		if s == mixSlot {
			mix = c.mix()
		}
	}
	for i := slotCount; i < noiseStepsPerSample; i++ {
		c.stepNoise()
	}
	c.closeSample()
	return saturate(mix)
}

func saturate(v int32) int16 {
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return int16(v)
}

func floorShift(v int32, bits uint) int32 {
	if v >= 0 {
		return v >> bits
	}
	return ^(^v >> bits)
}

// The first slot of channel ch: its modulator.
func firstSlot(ch int) int { return ch/3*6 + ch%3 }

func (c *Chip) refreshRouting() {
	for i := range c.ch {
		ch := &c.ch[i]
		switch {
		case c.rhythmMode && i == bassDrumChannel:
			if ch.connection != 0 {
				ch.algorithm = algKickDirect
			} else {
				ch.algorithm = algKickFM
			}
		case c.rhythmMode && i > bassDrumChannel:
			ch.algorithm = algPercussion
		case ch.connection != 0:
			ch.algorithm = algAdditive
		default:
			ch.algorithm = algFM
		}
	}
}

func (c *Chip) apply(reg, val uint8) {
	switch {
	case reg < 0x20:
		switch reg {
		case 0x01:
			c.waveSelect = val&0x20 != 0
		case 0x08:
			c.noteSelect = val&0x40 != 0
		}
	case reg < 0xA0 || reg >= 0xE0:
		row, column := int(reg&0x1F)>>3, int(reg&7)
		if row < 3 && column < 6 {
			c.writeOperator(&c.op[row*6+column], reg&0xE0, val)
		}
	case reg == 0xBD:
		c.writeRhythm(val)
	case reg < 0xD0 && int(reg&0x0F) < channelCount:
		ch := int(reg & 0x0F)
		switch reg & 0xF0 {
		case 0xC0:
			c.ch[ch].feedback = val >> 1 & 7
			c.ch[ch].connection = val & 1
			c.refreshRouting()
		case 0xB0:
			c.writeFrequency(ch, true, val)
		default:
			c.writeFrequency(ch, false, val)
		}
	}
}

func (c *Chip) writeOperator(op *operator, group, val uint8) {
	switch group {
	case 0x20:
		op.tremolo = val&0x80 != 0
		op.vibrato = val&0x40 != 0
		op.sustainHold = val&0x20 != 0
		op.keyScaleRate = val&0x10 != 0
		op.multiple = val & 15
	case 0x40:
		op.keyScaleLvl = val >> 6
		op.totalLevel = val & 63
	case 0x60:
		op.attackRate = val >> 4
		op.decayRate = val & 15
	case 0x80:
		op.sustainLevel = val >> 4
		op.releaseRate = val & 15
	default:
		// With waveform selection off, the OPL2 ignores waveform writes and keeps the
		// waveforms already chosen.
		if c.waveSelect {
			op.waveform = val & 3
		}
	}
}

func (c *Chip) writeRhythm(val uint8) {
	c.tremoloDepth = val&0x80 != 0
	c.vibratoDepth = val&0x40 != 0
	c.rhythmMode = val&0x20 != 0
	for i := range rhythmKeyBit {
		op := &c.op[rhythmFirstSlot+i]
		var key uint8
		if c.rhythmMode {
			key = val >> rhythmKeyBit[i] & 1
		}
		op.key = op.key&1 | key<<1
	}
	c.refreshRouting()
}

func (c *Chip) writeFrequency(i int, high bool, val uint8) {
	ch := &c.ch[i]
	if high {
		ch.fNumber = ch.fNumber&0xFF | uint16(val&3)<<8
		ch.block = val >> 2 & 7
	} else {
		ch.fNumber = ch.fNumber&0x300 | uint16(val)
	}
	shift := 9
	if c.noteSelect {
		shift = 8
	}
	ch.keyScale = 2*ch.block + uint8(ch.fNumber>>shift&1)
	if high {
		key := val >> 5 & 1
		for _, s := range [2]int{firstSlot(i), firstSlot(i) + voiceStride} {
			c.op[s].key = c.op[s].key&2 | key
		}
	}
}

func (c *Chip) closeSample() {
	c.sampleCounter++
	if c.sampleCounter&63 == 0 {
		c.tremoloPosition = uint8((int(c.tremoloPosition) + 1) % tremoloSteps)
	}
	position := int(c.tremoloPosition)
	triangle := position
	if position >= tremoloPeak {
		triangle = tremoloSteps - position
	}
	if c.tremoloDepth {
		c.tremoloLevel = uint8(triangle >> 2)
	} else {
		c.tremoloLevel = uint8(triangle >> 4)
	}

	n := 0
	for n < len(c.queue) && c.queue[n].due <= c.chipSamples {
		c.apply(c.queue[n].reg, c.queue[n].val)
		n++
	}
	if n > 0 {
		c.queue = append(c.queue[:0], c.queue[n:]...)
	}
	c.chipSamples++
}

// envelopeTime is where the envelope generator's global counter stands this sample.
func (c *Chip) envelopeTime() (odd bool, count uint64) {
	pairs := c.chipSamples >> 1
	odd = c.chipSamples&1 != 0
	if pairs >= 2 {
		count = (pairs-2)%envelopePeriod + 1
	}
	return odd, count
}

func slowStepDue(count uint64, group, fraction uint) bool {
	period := uint64(1) << (rateFast/4 - group)
	if count%period == period/2 {
		return true
	}
	if fraction&2 != 0 && count%(2*period) == period {
		return true
	}
	return fraction&1 != 0 && count%(4*period) == 2*period
}

func envelopeSpeed(registerRate, actualRate uint, odd bool, count uint64) uint {
	fraction := actualRate & 3
	if registerRate == 0 {
		return 0
	}
	if actualRate >= rateInstant {
		return 4
	}
	if actualRate >= rateFast {
		pairPosition := uint(count & 3)
		spread := (pairPosition&1)<<1 | pairPosition>>1
		var pairUnits uint = 1
		if spread < fraction {
			pairUnits = 2
		}
		pairUnits <<= (actualRate - rateFast) >> 2
		speed := pairUnits >> 1
		if odd {
			speed |= pairUnits & 1
		}
		return speed
	}
	if odd && slowStepDue(count, actualRate>>2, fraction) {
		return 1
	}
	return 0
}

// envelope steps an operator's envelope and reports whether the operator was just keyed on.
func (c *Chip) envelope(op *operator, ch *channel, odd bool, count uint64) bool {
	keyActive := op.key != 0
	stage := op.stage
	keyOn := keyActive && stage == stageRelease
	level := uint(op.envelope)
	rate := uint(op.releaseRate)
	switch {
	case keyOn || stage == stageAttack:
		rate = uint(op.attackRate)
	case stage == stageDecay:
		rate = uint(op.decayRate)
	case stage == stageSustain && op.sustainHold:
		rate = 0
	}

	rateOffset := uint(ch.keyScale) >> 2
	if op.keyScaleRate {
		rateOffset = uint(ch.keyScale)
	}
	actualRate := 4*rate + rateOffset
	instant := actualRate >= rateInstant
	speed := envelopeSpeed(rate, actualRate, odd, count)
	sustainBand := uint(op.sustainLevel)
	if sustainBand == 15 {
		sustainBand = 31
	}
	offZone := level >= envelopeOffZone

	switch {
	case keyOn:
		if instant {
			level = 0
		}
		stage = stageAttack
	case stage == stageAttack:
		if level == 0 {
			stage = stageDecay
		} else if keyActive && !instant {
			level -= (speed*(level+1) + 7) >> 3
		}
	case stage == stageDecay && level>>4 == sustainBand:
		if offZone {
			level = envelopeSilent
		}
		stage = stageSustain
	case offZone:
		level = envelopeSilent
	default:
		level += speed
	}
	if !keyActive {
		stage = stageRelease
	}
	op.envelope = uint16(level)
	op.stage = stage
	return keyOn
}

func keyScaleAttenuation(op *operator, ch *channel) uint {
	full := 4*kslROM[ch.fNumber>>6] - 32*(8-int(ch.block))
	if full < 0 {
		full = 0
	}
	switch op.keyScaleLvl {
	case 1:
		return uint(full) / 2
	case 2:
		return uint(full) / 4
	case 3:
		return uint(full)
	}
	return 0
}

func (c *Chip) increment(op *operator, ch *channel) uint32 {
	fNumber := int(ch.fNumber)
	if op.vibrato {
		depthShift := 1
		if c.vibratoDepth {
			depthShift = 0
		}
		step := vibratoSteps[c.sampleCounter>>10&7]
		magnitude := (fNumber >> 7 * step.halves >> 1) >> depthShift
		if step.negative {
			fNumber -= magnitude
		} else {
			fNumber += magnitude
		}
	}
	base := uint32(fNumber) << ch.block >> 1
	return base * mulTimes2[op.multiple] >> 1
}

func (c *Chip) stepNoise() uint32 {
	bit := c.noise & 1
	c.noise = c.noise>>1 | (c.noise^c.noise>>14)&1<<22
	return bit
}

func (c *Chip) cymbalBit() uint {
	hat, cym := uint(c.hiHatPhase), uint(c.cymbalPhase)
	return ((hat>>2 ^ hat>>7) | (hat>>3 ^ cym>>5) | (cym>>3 ^ cym>>5)) & 1
}

func (c *Chip) rhythmPhase(slot int, phase uint, noise uint) uint {
	switch slot {
	case hiHatSlot:
		bit := c.cymbalBit()
		if bit^noise != 0 {
			return bit<<9 | hiHatPhaseNoisy
		}
		return bit<<9 | hiHatPhaseQuiet
	case snareSlot:
		bit := uint(c.hiHatPhase) >> 8 & 1
		return bit<<9 | (bit^noise)<<8
	case cymbalSlot:
		c.cymbalPhase = uint16(phase)
		return c.cymbalBit()<<9 | cymbalPhase
	}
	return phase
}

// position is 0 for a channel's modulator slot, 1 for its carrier.
func position(slot int) int {
	if slot == firstSlot(slotChannel[slot]) {
		return 0
	}
	return 1
}

func (c *Chip) modulation(slot int) int32 {
	ch := &c.ch[slotChannel[slot]]
	alg := &algorithms[ch.algorithm]
	pos := position(slot)
	if pos == 0 && alg.feedback {
		if ch.feedback == 0 {
			return 0
		}
		op := &c.op[slot]
		return floorShift(int32(op.out)+int32(op.prevOut), 9-uint(ch.feedback))
	}
	if alg.chain>>pos&1 != 0 {
		return int32(c.op[slot-voiceStride].out)
	}
	return 0
}

type wave struct {
	attenuation uint
	negative    bool
	silent      bool
}

func waveAt(phase uint, waveform uint8) wave {
	secondHalf := phase&0x200 != 0
	secondQuarter := phase&0x100 != 0
	quarter := phase & 0xFF
	sine := uint(logSinROM[quarter])
	if secondQuarter {
		sine = uint(logSinROM[0xFF-quarter])
	}
	switch waveform {
	case 0:
		return wave{attenuation: sine, negative: secondHalf}
	case 1:
		return wave{attenuation: sine, silent: secondHalf}
	case 2:
		return wave{attenuation: sine}
	default:
		return wave{attenuation: uint(logSinROM[quarter]), silent: secondQuarter}
	}
}

func output(w wave, attenuation uint) int16 {
	if w.silent {
		return 0
	}
	level := w.attenuation + 8*attenuation
	magnitude := 0
	if level < audibleLimit {
		mantissa := implicitOne + uint(exponentROM[0xFF-level&0xFF])
		magnitude = int(2 * mantissa >> (level >> 8))
	}
	if w.negative {
		return int16(-magnitude - 1)
	}
	return int16(magnitude)
}

func (c *Chip) turn(slot int, odd bool, count uint64) {
	op := &c.op[slot]
	ch := &c.ch[slotChannel[slot]]

	attenuation := uint(op.envelope) + 4*uint(op.totalLevel) + keyScaleAttenuation(op, ch)
	if op.tremolo {
		attenuation += uint(c.tremoloLevel)
	}

	restart := c.envelope(op, ch, odd, count)

	phase := uint(op.phase>>9) & 0xFFFF
	inc := c.increment(op, ch)
	if restart {
		op.phase = inc
	} else {
		op.phase += inc
	}

	noise := uint(c.stepNoise())
	if slot == hiHatSlot {
		c.hiHatPhase = uint16(phase)
	}
	if c.rhythmMode {
		phase = c.rhythmPhase(slot, phase, noise)
	}

	final := uint(int32(phase)+c.modulation(slot)) & 0x3FF
	op.prevOut = op.out
	op.out = output(waveAt(final, op.waveform), attenuation)
}

func (c *Chip) mix() int32 {
	var sum int32
	for i := range c.ch {
		alg := &algorithms[c.ch[i].algorithm]
		base := firstSlot(i)
		var voice int32
		for pos := 0; pos < 2; pos++ {
			if alg.carriers>>pos&1 != 0 {
				voice += int32(c.op[base+voiceStride*pos].out)
			}
		}
		sum += voice * alg.gain
	}
	return sum
}

// Resampler turns a chip's output into samples at another rate, interpolating linearly
// between the chip's samples.
type Resampler struct {
	chip       *Chip
	rate       int32
	accum      int32
	last, curr int16
}

// NewResampler returns a resampler that takes samples from chip at the given rate.
func NewResampler(chip *Chip, rate int) *Resampler {
	if rate <= 0 {
		rate = Rate
	}
	return &Resampler{chip: chip, rate: int32(rate)}
}

// Next returns the next sample at the resampler's rate.
func (r *Resampler) Next() int16 {
	for r.accum >= r.rate {
		r.last = r.curr
		r.curr = r.chip.Sample()
		r.accum -= r.rate
	}
	fract := (int64(r.accum)*65536 + int64(r.rate)/2) / int64(r.rate)
	out := int32(r.last) + int32(fract*int64(int32(r.curr)-int32(r.last))/65536)
	r.accum += Rate
	return int16(out)
}
