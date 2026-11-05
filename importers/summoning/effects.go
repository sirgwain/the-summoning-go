package summoning

import (
	"fmt"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The view effects, each played by the rules of scenes.md, View effects, into a flipbook.

// effectBoxes are the view effects' boxes, width and height (scenes.md, What plays them).
var effectBoxes = [16][2]int{
	{96, 64}, {96, 64}, {56, 96}, {80, 80}, {48, 112}, {72, 64}, {120, 88}, {104, 64},
	{72, 80}, {152, 32}, {96, 80}, {72, 64}, {88, 64}, {72, 80}, {72, 88}, {72, 80},
}

// effectID is the id of view effect n's flipbook; the conversations' effect actions name it.
func effectID(n int) string { return fmt.Sprintf("flipbook.effect.%d", n) }

// The view effects the importer names (scenes.md, What plays them).
const (
	viewClimbDown    = 0
	viewClimbThrough = 1
	viewFall         = 2
	viewCasterDies   = 5
	viewSocket       = 8
	viewMending      = 9
	viewDeathBlow    = 12
)

// effectCues are the flipbooks the engine's effect cues play.
var effectCues = map[string]string{
	content.EffectClimbDown: effectID(viewClimbDown), content.EffectFall: effectID(viewFall),
	content.EffectCasterDies: effectID(viewCasterDies), content.EffectDeathBlow: effectID(viewDeathBlow),
	content.SparkleCure: sparkleID("cure"), content.SparkleHeal: sparkleID("heal"),
	content.SparkleWeaponSkills: sparkleID("weapon_skills"), content.SparkleMagicSkills: sparkleID("restore"),
}

// transitions are how the level transitions and teleports show (scenes.md, Level transitions):
// the teleporter closes on the player and opens again, scroll slides in from the right, rise and
// descend squeeze in five steps, and the ripple zigzags. Loading a game, the engine's own, closes
// to black from the sides, as the teleporter does about the player, and parts again.
func transitions() map[string]*content.Transition {
	scroll := &content.Transition{Style: content.StyleSlide, From: content.SideRight, MS: 350}
	return map[string]*content.Transition{
		content.TransitionTeleporter: {Style: content.StyleIris, MS: 700, HoldMS: 150},
		content.TransitionRipple:     {Style: content.StyleRipple, MS: 750, Size: 8, Rows: 16},
		content.TransitionExitWall:   scroll,
		content.TransitionTravel:     scroll,
		content.TransitionFall:       {Style: content.StyleSqueeze, From: content.SideBottom, MS: 300, Steps: 5},
		content.TransitionExitFloor:  {Style: content.StyleSqueeze, From: content.SideTop, MS: 300, Steps: 5},
		content.TransitionLoad:       {Style: content.StyleCurtains, MS: 700, HoldMS: 150},
	}
}

// linkFlipbooks writes the effect cues and drops the references to flipbooks that could not be
// rendered, so that what would play them plays nothing. Conversations naming effects 16 and up
// lose them this way, as those effects do nothing.
func (im *importer) linkFlipbooks() {
	have := func(id string) bool { return im.pack.Flipbooks[id] != nil }
	cues := map[string]string{}
	for cue, id := range effectCues {
		// A flipbook left out is reported as it is rendered.
		if have(id) {
			cues[cue] = id
		}
	}
	combat := content.DefaultCombat
	im.pack.Combat = &combat
	im.pack.Effects = &content.EffectCues{ID: "effects", Cues: cues, Transitions: transitions(),
		ItemLost: &content.LostItem{Style: content.LostShatter, MS: 1200, Inventory: true}}
	keep := func(as []content.Action) []content.Action {
		out := as[:0]
		for _, a := range as {
			if a.Effect != nil && !have(*a.Effect) || a.Sparkle != nil && !have(*a.Sparkle) {
				continue
			}
			out = append(out, a)
		}
		return out
	}
	for _, l := range im.pack.Levels {
		for _, f := range l.Features {
			if !have(f.Effect) {
				f.Effect = ""
			}
			if !have(f.Sparkle) {
				f.Sparkle = ""
			}
		}
	}
	for _, c := range im.pack.Conversations {
		for i := range c.Clauses {
			c.Clauses[i].Do = keep(c.Clauses[i].Do)
		}
	}
	for _, s := range im.pack.Spells {
		s.Effects = keep(s.Effects)
	}
}

var effectPlays = [16]func(b *flipbook){
	func(b *flipbook) { effectClimb(b, 0) }, func(b *flipbook) { effectClimb(b, 1) },
	effectFall, effectGhost, effectDunstan, effectCasterDeath, effectGates, effectSteelSmiter,
	effectGemSocket, effectMending, effectRukn, effectHonorah, effectDeathBlow, effectKern,
	effectMurchad, effectPadrake,
}

// effects renders the sixteen view effects.
func (im *importer) effects() {
	base := int(im.init.Bases.ViewEffect)
	for n, play := range effectPlays {
		box := effectBoxes[n]
		b := newFlipbook(im, base, box[0], box[1], uint64(n))
		play(b)
		if err := b.write(effectID(n), true); err != nil {
			im.report.addf("view effect %d: %v; left out", n, err)
		}
	}
}

// effectClimb is effects 0 and 1: climbing down a pit, and through an exit marker.
func effectClimb(b *flipbook, mode int) {
	var mirrored [3]bool
	stride := 2
	if mode == 1 {
		stride = 1
		mirrored[0] = true
	}
	p := 0
	for k := range 24 {
		b.draw(p, 0, 0)
		b.drawAs(3+p, 0x26, 0, mirrored[p], nil)
		if (mode == 1 && k%3 == 2) || (mode == 0 && k%3 == 0) {
			b.sound(0x0F)
		}
		b.show(5, true)
		mirrored[p] = !mirrored[p]
		p = (p + stride) % 3
	}
}

// effectFall is effect 2.
func effectFall(b *flipbook) {
	b.sound(0x15)
	for k := range 9 {
		b.draw(6+k%3, 0, 0)
		b.show(5, true)
	}
}

// effectGhost is effect 3: Chesschantra's ghost rises.
func effectGhost(b *flipbook) {
	b.music(0x0E)
	edge, col, phase := 0x4B, 9, 0
	for range 100 {
		if phase == 0x0F {
			col ^= 1
		}
		if edge > 0 {
			edge--
		}
		b.draw(0x0B, 0, 0)
		b.lines(0x0C, 0, col, false, func(i int) int {
			s := (i + phase) & 0x0F
			if i < 0x1E {
				switch {
				case s < 3:
					return -1
				case s >= 8 && s < 0x0B:
					return 1
				}
				return 0
			}
			d := 0
			switch {
			case s < 3:
			case s < 8:
				d = s - 2
			case s < 0x0B:
				d = 6
			default:
				d = 0x10 - s
			}
			return d - 3
		}, func(i int) bool { return i >= edge })
		b.draw(0x0A, 0x32, 0)
		b.show(0, edge == 0)
		phase = (phase + 1) & 0x0F
	}
	b.fadeOut()
}

// effectDunstan is effect 4.
func effectDunstan(b *flipbook) {
	b.draw(0x0D, 0, 0)
	b.show(0x0F, false)
	b.clipTo(0x31, 0, 0x4C, 0x18)
	slide := func(step int) {
		b.draw(0x0D, 0, 0)
		b.draw(0x0E, 0x31, 0x14)
		b.lines(0x0F, 0x31, -(0x17 - step), false, nil, nil)
		b.show(0, false)
	}
	for step := range 0x18 {
		slide(step)
	}
	bob := 0
	for range 0x32 {
		b.draw(0x0D, 0, 0)
		b.draw(0x0E, 0x31, 0x14)
		b.draw(0x0F, 0x31+bob, 0)
		b.show(0, false)
		if b.rng.IntN(100) < 0x19 {
			b.sound(0x11)
			bob ^= 1
		}
	}
	for step := 0x17; step >= 0; step-- {
		slide(step)
	}
	b.draw(0x0D, 0, 0)
	b.show(0, false)
	b.draw(0x10, 0x2C, 0)
	b.show(5, false)
	blink := false
	for step := range 0x96 {
		b.draw(0x0D, 0, 0)
		b.draw(0x10, 0x2C, 0)
		b.draw(0x11, 5, 0x0F)
		if step != 0 && step%5 == 0 {
			blink = !blink
		}
		if blink && step <= 0x32 {
			b.draw(0x12, 0x5A, 8)
		}
		b.show(0, true)
	}
}

// casterSparks are the pixels falling in effect 5: row, column and color.
var casterSparks = [4][3]int{{0x19, 0x0F, 0x38}, {0x14, 0x25, 0xA4}, {0x1C, 0x19, 0xA4}, {0x1E, 0x1D, 0x38}}

// effectCasterDeath is effect 5.
func effectCasterDeath(b *flipbook) {
	b.draw(0x13, 0, 0)
	b.show(0x0A, true)
	for n := 0x14; n < 0x17; n++ {
		b.draw(0x13, 0, 0)
		b.draw(n, 8, 0x10)
		b.show(4, true)
	}
	b.draw(0x13, 0, 0)
	b.draw(0x17, 3, 0)
	b.sound(0x12)
	b.show(4, true)
	b.draw(0x79, 0, 0)
	b.show(4, true)
	for f := 0x1E; f < 0x3A; f += 2 {
		b.draw(0x18, 0, 0)
		b.draw(0x19, 0x1E, 0x10)
		b.draw(0x7A, f, 0x0A)
		b.draw(0x7A, f+3, 0x33)
		b.mirrored(0x7A, f-2, 0x1E)
		for i, s := range casterSparks {
			if i == 1 && f >= 0x2C {
				continue
			}
			b.pixel(s[0]+f-0x1E, s[1], byte(s[2]))
		}
		b.show(1, true)
	}
	for n := 0x1A; n < 0x1C; n++ {
		b.draw(0x18, 0, 0)
		b.draw(n, 0x1E, 0x10)
		b.show(0x0A, true)
	}
	b.draw(0x18, 0, 0)
	b.show(0x0F, true)
}

// effectGates is effect 6: the citadel guards open the gates.
func effectGates(b *flipbook) {
	b.clipTo(0, 0x28, b.h, 0x50)
	for k := range 0x19 {
		a, c := max(0, k-4), min(k, 0x14)
		b.draw(0x1C, 0, 0x28)
		b.lines(0x1D, 0, 0x28-a, false, nil, nil)
		b.lines(0x1D, 0, 0x38+c, true, nil, nil)
		b.draw(0x1E, 0, 0)
		b.mirrored(0x1E, 0, 0x50)
		b.draw(0x1F+a%3, 8, 0)
		b.mirrored(0x1F+c%3, 8, 0x58)
		b.sound(0x17)
		b.show(5, false)
	}
	b.hold(0x24, true)
}

// effectSteelSmiter is effect 7.
func effectSteelSmiter(b *flipbook) {
	b.sound(0x13)
	pose := 0
	frame := func(stage, t int) {
		pose = (pose + 1) % 3
		b.draw(0x25, 0, 0x30)
		b.draw(0x28+stage, 5, 0x44)
		b.draw(0x22+pose, 0, 0)
		if stage == 0 {
			b.draw(0x26, 0x23, 0x27)
			b.lines(0x2A+t/10, 0x1F, 0x0E, false, func(i int) int {
				return [4]int{-1, 0, 1, 0}[(i+t)%4]
			}, nil)
		} else {
			b.draw(0x27, 0x15, 0x30)
			q := t / 10
			if t >= 0x0F {
				q = 2
			}
			b.draw(0x2D+q, 4, 0x33)
			if t >= 0x14 && t <= 0x16 {
				b.draw(0x1C+t, -1, 0x3E)
			}
		}
		b.show(5, false)
	}
	start := 0x14
	for range 2 {
		for t := start; t < 0x1E; t++ {
			frame(0, t)
		}
		start = 0
		for t := range 0x1E {
			frame(1, t)
		}
	}
	b.stopMusic()
}

// effectGemSocket is effect 8: the sixth gem opens the wall.
func effectGemSocket(b *flipbook) {
	b.music(0x0F)
	flash := func() {
		right := b.rng.IntN(100)/50 == 1
		b.overlay(1, func() {
			if right {
				b.mirrored(0x37, 3, 0x3E)
			} else {
				b.draw(0x37, 3, 2)
			}
		})
	}
	flashesFor := func(ticks int) {
		end := b.now + ms(ticks)
		for b.now < end {
			if b.tick()%9 == 0 {
				flash()
				continue
			}
			next := int(float64(b.tick()+1)*1000/ticksPerSecond) + 1
			b.showFrame(b.shown, min(next, end)-b.now, false)
		}
	}
	only := func(first, last byte) remap {
		return func(c byte) (byte, bool) { return c, c >= first && c <= last }
	}
	for k := range 0x13 {
		b.draw(0x33, 0, -4)
		b.mirrored(0x33, 0, 0x24)
		b.draw(0x34, 4, 0x0C)
		flash()
		switch {
		case k <= 6:
			b.drawAs(0x35, 0x3C, 8, false, func(c byte) (byte, bool) { return 0, c >= 0xE8 && c <= 0xEB })
		case k <= 0x0E:
			b.draw(0x35, 0x3C, 8)
		case k < 0x12:
			b.drawAs(0x35, 0x3C, 8, false, only(0xE8, byte(0xEA-(k-0x0F))))
		default:
			b.draw(0x36, 0x3C, 8)
		}
		b.show(5, false)
	}
	flashesFor(0x36)
	b.clipTo(0, 0x0C, b.h, 0x3B)
	for k := range 0x31 {
		b.draw(0x33, 0, -4)
		b.mirrored(0x33, 0, 0x24)
		b.draw(0x36, 0x3C, 8)
		if k&1 == 1 {
			flash()
		}
		b.lines(0x34, 4, 0x0C, false, func(i int) int {
			if i&1 == 1 {
				return k
			}
			return -k
		}, nil)
		b.show(0, false)
	}
	flashesFor(0x6C)
	b.fadeOut()
}

// effectMending is effect 9: the staff halves join.
func effectMending(b *flipbook) {
	for i := range 6 {
		b.lines(0x38, 0, i*0x1D, false, nil, nil)
	}
	background := append([]int16(nil), b.pix...)
	restore := func() { copy(b.pix, background) }
	mirror43 := false
	for s := 0x60; s >= 0; s-- {
		restore()
		b.lines(0x43, 2, 0x4E+s, mirror43, nil, nil)
		if s%3 == 0 {
			mirror43 = !mirror43
		}
		b.lines(0x39, 8, -s, false, nil, nil)
		b.lines(0x44, 0x0B, 0x57-s, false, nil, nil)
		b.lines(0x3A, 5, 0x5C+s, false, nil, nil)
		b.lines(0x45, 9, 0x5C+s, false, nil, nil)
		b.show(0, false)
	}
	b.sound(0x0E)
	for f := range 0x5A {
		restore()
		if f != 0x59 {
			b.drawAs(0x43, 2, 0x4E, mirror43, nil)
			if f%3 == 0 {
				mirror43 = !mirror43
			}
		}
		b.draw(0x39, 8, 0)
		b.draw(0x3A, 5, 0x5C)
		if f < 0x1E {
			b.draw(0x44, 0x0B, 0x57)
			b.draw(0x45, 9, 0x5C)
		} else {
			b.draw(0x39+f/0x0F, 0x0A, 0x55)
		}
		b.show(0, false)
	}
	b.hold(0x12, false)
	last := -1
	for f := range 0x1E {
		restore()
		b.draw(0x39, 8, 0)
		b.draw(0x3A, 5, 0x5C)
		b.draw(0x3E, 0x0A, 0x55)
		if f != 0x1D {
			pick := b.rng.IntN(3)
			if pick != last {
				b.sound(0x0D)
				last = pick
			}
			b.draw(0x40+pick, 0, 0)
		} else {
			b.draw(0x3F, 5, 0)
		}
		b.show(5, false)
	}
	b.sound(0x10)
	b.hold(0x6C, true)
}

// effectRukn is effect 10: a figure grows.
func effectRukn(b *flipbook) {
	b.show(0x36, false)
	b.music(0x0F)
	wobble := []int{0, 0, 1, 2, 3, 3, 2, 1}
	var phase [3]int
	grown := func(n, row, col, p int, special bool) {
		b.lines(n, row, col, false, func(i int) int {
			s := (i + p) & 7
			if special {
				switch {
				case i < 0x17:
					s = 2
				case i < 0x28:
					if s = s/2 + 2; s >= 4 {
						s = 7 - s
					}
				}
			}
			return wobble[s]
		}, nil)
	}
	growth := 0
	for range 0x50 {
		b.draw(0x5A, 0, 0)
		switch {
		case growth < 0x0F:
			grown(0x5F, 8, 0x0D, phase[0], false)
		case growth < 0x1E:
			grown(0x60, 2, 0x0B, phase[0], false)
		case growth < 0x2D:
			grown(0x61, 6, 0x11, phase[0], false)
			grown(0x62, 6, 0x1B, phase[1], false)
		default:
			grown(0x5D, 6, 0x12, phase[0], false)
			grown(0x5E, 6, 0x25, phase[1], false)
			grown(0x5C, 2, 0x35, phase[2], true)
		}
		phase[0] = (phase[0] + 1) & 7
		phase[1] = (phase[1] - 1) & 7
		phase[2] = (phase[2] + 1) & 7
		b.draw(0x5B, 0x18, 0)
		b.show(0, growth == 0x2D)
		if growth < 0x2D {
			growth++
		}
	}
	b.fadeOut()
}

// effectHonorah is effect 11.
func effectHonorah(b *flipbook) {
	b.music(0x0F)
	back := func() { b.draw(0x46, 0, 0) }
	back()
	b.show(0x12, false)
	for col := 0x48; col >= 0x28; col-- {
		back()
		b.lines(0x47, 0x13, col, false, nil, nil)
		b.show(0, false)
	}
	for _, n := range []int{0x47, 0x48, 0x49, 0x48, 0x47} {
		back()
		b.draw(n, 0x13, 0x28)
		b.show(0x12, false)
	}
	for col := 0x28; col < 0x49; col++ {
		back()
		b.lines(0x47, 0x13, col, false, nil, nil)
		b.show(0, false)
	}
	for step := range 3 {
		back()
		b.draw(0x4A, 0x1E, 0x26)
		if step == 1 {
			b.draw(0x4B, 0x28, 0x20)
		}
		b.show(0x12, false)
	}
	for gap := 1; gap < 0x21; gap <<= 1 {
		back()
		b.show(3, true)
		back()
		b.draw(0x4C, 0x11, 0x1B)
		b.show(gap, false)
	}
	back()
	b.draw(0x7B, 2, 4)
	b.show(0x48, true)
	b.fadeOut()
}

// deathBlowColumns are effect 12's falling columns: start row, column and range.
var deathBlowColumns = [3][3]int{{0x31, 0x28, 0x23}, {0x30, 0x2E, 0x1B}, {0x1E, 0x45, 0x3B}}

// deathBlowPieces are effect 12's falling pieces: start row, column and art.
var deathBlowPieces = [6][3]int{{5, 0x44, 0x52}, {4, 0x30, 0x53}, {0x16, 0x50, 0x54}, {0x15, 4, 0x56}, {0x23, 0x0D, 0x55}, {0x27, 0, 0x57}}

// effectDeathBlow is effect 12.
func effectDeathBlow(b *flipbook) {
	var rows [3]int
	for i, c := range deathBlowColumns {
		rows[i] = c[0]
	}
	columns := func() {
		for i, c := range deathBlowColumns {
			b.lines(0x7C, rows[i], c[1], false, nil, nil)
			if rows[i] += 4; rows[i] >= c[0]+c[2] {
				rows[i] = c[0]
			}
		}
	}
	frame := func(draw func()) {
		b.draw(0x4D, 0, 0)
		columns()
		draw()
		b.show(0, true)
	}
	b.music(0x10)
	for end := b.now + ms(0x36); b.now < end; {
		frame(func() { b.draw(0x4E, 4, 0) })
	}
	for stage := range 5 {
		for end := b.now + ms(9); b.now < end; {
			frame(func() {
				b.draw(0x4E, 4, 0)
				if stage >= 2 {
					b.draw(0x4D+stage, 4, 0)
				}
				if stage == 0 {
					b.draw(0x58, 0x23, 0x24)
				} else {
					b.draw(0x59, 0x23, 0x24)
				}
			})
		}
	}
	const never, gone = -1, -2
	var deadline [6]int
	var row [6]int
	for i, p := range deathBlowPieces {
		deadline[i], row[i] = never, p[0]
	}
	deadline[0] = b.now + ms(0x36+b.rng.IntN(0x19))
	deadline[4] = b.now + ms(0x36+b.rng.IntN(0x19))
	for settled := false; !settled; {
		frame(func() {
			settled = true
			for i, p := range deathBlowPieces {
				if deadline[i] == gone {
					continue
				}
				if deadline[i] != never && b.now >= deadline[i] {
					row[i] += 5
				}
				if row[i] < b.h {
					settled = false
				} else {
					deadline[i] = gone
					next := map[int]int{0: 1, 1: 2, 2: 3, 4: 3, 3: 5}
					if n, ok := next[i]; ok && deadline[n] != gone {
						deadline[n] = b.now
					}
				}
				b.lines(p[2], row[i], p[1], false, nil, nil)
			}
			b.draw(0x59, 0x23, 0x24)
		})
	}
	for _, n := range []int{0x59, 0x58, 0} {
		for end := b.now + ms(0x12); b.now < end; {
			b.draw(0x4D, 0, 0)
			columns()
			if n != 0 {
				b.draw(n, 0x23, 0x24)
			}
			b.show(1, true)
		}
	}
	for end := b.now + ms(0x12); b.now < end; {
		frame(func() {})
	}
}

// effectKern is effect 13.
func effectKern(b *flipbook) {
	stage, n := 0, 0x64
	for f := 0; ; f++ {
		b.draw(0x63, 0, 0)
		done := false
		if stage == 0 {
			b.draw(n, 1, 0)
			if n == 0x65 {
				b.sound(0x18)
			}
			if n++; n == 0x66 {
				if f == 0x19 || f == 0x1A || f > 0x32 {
					stage = 1
				} else {
					n = 0x64
				}
			}
		} else {
			b.draw(0x66, 0x0C, 0)
			if n >= 0x67 && n <= 0x69 {
				b.draw(0x67, 0x0A, 0x18)
			}
			if n++; n == 0x6A {
				done = f > 0x32
				stage, n = 0, 0x64
			}
		}
		b.show(5, false)
		if done {
			return
		}
	}
}

// effectMurchad is effect 14.
func effectMurchad(b *flipbook) {
	b.music(0x0F)
	for rise := 0x17; rise >= 0x0F; rise-- {
		b.draw(0x68, 0, 0)
		b.draw(0x69, rise, 0x18)
		b.draw(0x6C, 0x23, 0x18)
		b.show(0, false)
	}
	strip, stage, toggle, bob := 0x6A, 0, false, 0
	for f, done := 0, false; !done; f++ {
		b.draw(0x68, 0, 0)
		wait := 0
		if stage == 0 {
			b.draw(strip, 0x21, 0x18)
			strip ^= 1
			if b.rng.IntN(100) < 0x0A {
				done = f > 0x4B
				stage, bob = 1, 0x19
			}
			wait = 6 - b.rng.IntN(3)
		} else {
			b.draw(0x69, bob, 0x18)
			b.draw(0x6C, 0x23, 0x18)
			if bob < 0x14 {
				if !toggle {
					bob -= 2
				} else {
					bob++
				}
				toggle = !toggle
			} else {
				bob--
			}
			if bob == 0x0E {
				done = f > 0x4B
				stage = 0
			}
		}
		b.show(wait, false)
	}
	b.hold(0x12, true)
	b.fadeOut()
}

// effectPadrake is effect 15.
func effectPadrake(b *flipbook) {
	back := func() { b.draw(0x6D, 0, 0) }
	back()
	b.show(0x36, true)
	b.clipTo(0, 0, b.h-1, b.w-1)
	for range 3 {
		for k := range 0x12 {
			back()
			ticks := 1
			switch {
			case k < 3:
				b.draw(0x6E, 0x28, 0x24)
				ticks = 3
			case k < 9:
				b.draw(0x6E, 0x28, 0x24)
				b.draw(0x6F, 0x33+3*(k-3), 0x25)
				ticks = 0
			case k < 0x0E:
				if k > 0x0A {
					b.draw(0x6B+k, 0x16, 0x0B)
				}
				for f := 0; f <= min(k-9, 3); f++ {
					b.frame(0x75, f, 0x3F, 0x0F)
				}
				b.lines(0x67+k, 0x3A, 0x1A, false, nil, nil)
			default:
				if k < 0x11 {
					b.draw(0x10-k+0x76, 0x16, 0x0B)
				}
				for f := 0x11 - k; f >= 0; f-- {
					if f < 4 {
						b.frame(0x75, f, 0x3F, 0x0F)
					}
				}
				b.lines(0x11-k+0x70, 0x3A, 0x1A, true, nil, nil)
			}
			if k == 9 {
				b.sound(0x14)
			}
			b.show(ticks, true)
		}
		back()
		b.show(0x36, true)
	}
}
