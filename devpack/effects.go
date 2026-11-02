package devpack

import (
	"math"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The dev pack's view effects and sparkles (docs/spec/scenes.md, View effects and Sparkles):
// plain animations drawn in code, each a pattern and a color of its own so it is plain which one
// plays. The hall's floor exit, pit, trapdoor, socket and sealed door play them; in the study,
// killing the target, the Heal spell and the sage's word on fireballs do.

// Flipbook ids.
const (
	FlipbookClimbDown = "flipbook.dev.climb_down"
	FlipbookFall      = "flipbook.dev.fall"
	FlipbookThrough   = "flipbook.dev.through"
	FlipbookSocket    = "flipbook.dev.socket"
	FlipbookCaster    = "flipbook.dev.caster"
	FlipbookDeathBlow = "flipbook.dev.death_blow"
	FlipbookTalk      = "flipbook.dev.talk"
	SparkleHeal       = "flipbook.dev.sparkle.heal"
	SparkleCure       = "flipbook.dev.sparkle.cure"
	SparkleSeal       = "flipbook.dev.sparkle.seal"
)

// Flipbook sizes and timing.
const (
	effectW, effectH = 64, 48
	sparkleSize      = 48
	effectFrames     = 16
	effectStepMS     = 110
	sparkleFrames    = 12
	sparkleStepMS    = 70
	sparkleStars     = 14
)

// Effect patterns.
const (
	patternRings = iota
	patternDrop
	patternSteps
	patternRise
)

func effects(p *content.Pack) {
	for _, e := range []struct {
		id      string
		pattern int
		color   byte
	}{
		{FlipbookClimbDown, patternSteps, green},
		{FlipbookFall, patternDrop, red},
		{FlipbookThrough, patternRise, blue},
		{FlipbookSocket, patternRings, gold},
		{FlipbookCaster, patternRings, purple},
		{FlipbookDeathBlow, patternDrop, purple},
		{FlipbookTalk, patternRings, blue},
	} {
		frames := make([]*canvas, effectFrames)
		for k := range frames {
			frames[k] = effectFrame(e.pattern, e.color, k)
		}
		flipbook(p, e.id, frames, effectStepMS, true)
	}
	for _, s := range []struct {
		id    string
		color byte
	}{{SparkleHeal, green}, {SparkleCure, blue}, {SparkleSeal, gold}} {
		frames := make([]*canvas, sparkleFrames)
		for k := range frames {
			frames[k] = sparkleFrame(s.color, k)
		}
		flipbook(p, s.id, frames, sparkleStepMS, false)
	}
	p.Effects = &content.EffectCues{ID: "effects", Cues: map[string]string{
		content.EffectClimbDown: FlipbookClimbDown, content.EffectFall: FlipbookFall,
		content.EffectCasterDies: FlipbookCaster, content.EffectDeathBlow: FlipbookDeathBlow,
		content.SparkleHeal: SparkleHeal, content.SparkleCure: SparkleCure,
	}}

	hall := p.Levels[Hall]
	for _, f := range hall.Features {
		switch f.ID {
		case "exit.d":
			f.Effect = FlipbookThrough
		case "socket":
			f.Effect = FlipbookSocket
		case "door.e":
			f.Sparkle = SparkleSeal
		}
	}
	heal := p.Spells[SpellHeal]
	heal.Effects = append(heal.Effects, content.Action{Sparkle: ptr(SparkleHeal)})
	for i, cl := range p.Conversations[ConversationSage].Clauses {
		if len(cl.If.Asked) == 1 && cl.If.Asked[0] == keywordID("fireball") {
			p.Conversations[ConversationSage].Clauses[i].Do = append(cl.Do, content.Action{Effect: ptr(FlipbookTalk)})
		}
	}
}

// flipbook adds a flipbook of frames shown stepMS each, its first step with a blip; a framed one
// can be cut short from its second half.
func flipbook(p *content.Pack, id string, frames []*canvas, stepMS int, framed bool) {
	sprite := "sprite." + id
	bank(p, sprite, frames, nil)
	f := &content.Flipbook{ID: id, Sprite: sprite, Framed: framed}
	for k := range frames {
		s := content.FlipbookStep{Frame: k, MS: stepMS, Cancel: framed && k >= len(frames)/2}
		if k == 0 {
			s.Sounds = []string{SFXBlip}
		}
		f.Steps = append(f.Steps, s)
	}
	p.Flipbooks[id] = f
}

// effectFrame is frame k of a view effect: a dark box with the pattern in a ramp of color.
func effectFrame(pattern int, color byte, k int) *canvas {
	c := newCanvas(effectW, effectH, black)
	c.fill(func(x, y float64) bool { return true }, func(x, y int) byte { return byte(stone + noise(x, y, 2)) })
	t := float64(k) / float64(effectFrames-1)
	cx, cy := float64(effectW)/2, float64(effectH)/2
	switch pattern {
	case patternRings:
		for r := 0; r < 3; r++ {
			rad := math.Mod(t*30+float64(r)*10, 30) + 2
			ring := and(ellipse(cx, cy, rad, rad*0.75), not(ellipse(cx, cy, rad-2, rad*0.75-2)))
			c.fill(ring, solid(color+byte(7-2*r)))
		}
	case patternDrop:
		y := 4 + t*(effectH-12)
		for i := 1; i <= 3; i++ {
			c.line(int(cx)-3, int(y)-4*i, int(cx)-3, int(y)-4*i-2, color+3)
			c.line(int(cx)+3, int(y)-4*i, int(cx)+3, int(y)-4*i-2, color+3)
		}
		c.fill(ellipse(cx, y, 5, 5), solid(color+6))
	case patternSteps, patternRise:
		for s := range 4 {
			x0, y0 := 8+s*12, 10+s*8
			c.rect(x0, y0, x0+14, y0+3, color+2)
		}
		step := int(t * 4)
		if pattern == patternRise {
			step = 3 - step
		}
		step = min(step, 3)
		c.fill(ellipse(float64(14+step*12), float64(6+step*8), 3, 4), solid(color+7))
	}
	return c
}

// sparkleFrame is frame k of a sparkle: stars over a clear area, each growing and fading on its
// own phase.
func sparkleFrame(color byte, k int) *canvas {
	c := newCanvas(sparkleSize, sparkleSize, clear)
	for j := range sparkleStars {
		phase := (k + j*5) % 8
		size := []int{0, 1, 2, 3, 2, 1, 0, 0}[phase]
		if size == 0 || k >= sparkleFrames-2 && phase < 4 {
			continue
		}
		x, y := 4+noise(j, 1, sparkleSize-8), 4+noise(j, 2, sparkleSize-8)
		v := color + byte(3+size)
		c.line(x-size, y, x+size, y, v)
		c.line(x, y-size, x, y+size, v)
	}
	return c
}
