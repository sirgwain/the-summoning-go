package summoning

import (
	"fmt"
	"image"
	"math/rand/v2"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Rendering the view effects and sparkles into flipbooks (scenes.md, View effects and Sparkles).
// The importer plays each one by its rules on a canvas of its box and keeps every frame shown.

// noWaitMS is how long a frame shown with no wait lasts (scenes.md, Engine model).
const noWaitMS = 17

// quickFadeTicks is how long the quick fade takes: the master volume from 127 down by 3 a tick
// (audio.md, Fades).
const quickFadeTicks = 43

// clear marks a canvas pixel nothing has drawn.
const clear = -1

// remap gives the color a pixel draws in, and false to leave it out.
type remap func(c byte) (byte, bool)

// flipbook plays one animation and collects its frames and steps.
type flipbook struct {
	im   *importer
	base int // the resource n of "+n" is base + n
	w, h int
	pix  []int16
	// clip bounds what line-by-line drawing touches; everything stays within the box.
	clip   image.Rectangle
	rng    *rand.Rand
	err    error
	frames [][]int16
	steps  []content.FlipbookStep
	// shown is the frame on screen; now is the time played, in milliseconds.
	shown   int
	now     int
	pending content.FlipbookStep
	banks   map[int]*SpriteBank
}

func newFlipbook(im *importer, base, w, h int, seed uint64) *flipbook {
	b := &flipbook{im: im, base: base, w: w, h: h, pix: make([]int16, w*h), rng: rand.New(rand.NewPCG(seed, 0x5eed)),
		shown: -1, banks: map[int]*SpriteBank{}}
	b.blank()
	b.unclip()
	return b
}

func (b *flipbook) blank() {
	for i := range b.pix {
		b.pix[i] = clear
	}
}

func (b *flipbook) unclip() { b.clip = image.Rect(0, 0, b.w, b.h) }

// clipTo limits line drawing to rows top..bottom and columns left..right, inclusive.
func (b *flipbook) clipTo(top, left, bottom, right int) {
	b.clip = image.Rect(left, top, right+1, bottom+1).Intersect(image.Rect(0, 0, b.w, b.h))
}

// art is resource +n.
func (b *flipbook) art(n int) *SpriteBank {
	return b.res(b.base + n)
}

// res is resource id.
func (b *flipbook) res(id int) *SpriteBank {
	if bank, ok := b.banks[id]; ok {
		return bank
	}
	bank, err := b.im.bank(id)
	if err != nil {
		if b.err == nil {
			b.err = err
		}
		bank = &SpriteBank{Transparent: NoTransparency}
	}
	b.banks[id] = bank
	return bank
}

// blit draws a frame of bank with its top-left at (row, col).
func (b *flipbook) blit(bank *SpriteBank, f, row, col int, mirror, clipped bool, m remap) {
	if f < 0 || f >= len(bank.Frames) {
		return
	}
	area := image.Rect(0, 0, b.w, b.h)
	if clipped {
		area = b.clip
	}
	src := bank.Frames[f]
	for y := range bank.Height {
		dy := row + y
		if dy < area.Min.Y || dy >= area.Max.Y {
			continue
		}
		for x := range bank.Width {
			sx := x
			if mirror {
				sx = bank.Width - 1 - x
			}
			c := src[y*bank.Width+sx]
			if int(c) == bank.Transparent {
				continue
			}
			if m != nil {
				var ok bool
				if c, ok = m(c); !ok {
					continue
				}
			}
			dx := col + x
			if dx >= area.Min.X && dx < area.Max.X {
				b.pix[dy*b.w+dx] = int16(c)
			}
		}
	}
}

// draw draws +n as a whole: its frames stacked, each its height below the last.
func (b *flipbook) draw(n, row, col int) { b.drawAs(n, row, col, false, nil) }

// mirrored draws +n as a whole, flipped left to right.
func (b *flipbook) mirrored(n, row, col int) { b.drawAs(n, row, col, true, nil) }

func (b *flipbook) drawAs(n, row, col int, mirror bool, m remap) {
	bank := b.art(n)
	for f := range bank.Frames {
		b.blit(bank, f, row+f*bank.Height, col, mirror, false, m)
	}
}

// frame draws frame f of +n.
func (b *flipbook) frame(n, f, row, col int) { b.blit(b.art(n), f, row, col, false, false, nil) }

// lines draws +n line by line: line i at (row + i, col + shift(i)), within the clip, leaving out
// the lines show refuses.
func (b *flipbook) lines(n, row, col int, mirror bool, shift func(i int) int, show func(i int) bool) {
	bank := b.art(n)
	for i := range bank.Frames {
		if show != nil && !show(i) {
			continue
		}
		dx := 0
		if shift != nil {
			dx = shift(i)
		}
		b.blit(bank, i, row+i, col+dx, mirror, true, nil)
	}
}

// pixel sets one pixel.
func (b *flipbook) pixel(row, col int, c byte) {
	if row >= 0 && col >= 0 && row < b.h && col < b.w {
		b.pix[row*b.w+col] = int16(c)
	}
}

// sound starts sound effect n with the next frame shown.
func (b *flipbook) sound(n int) { b.pending.Sounds = append(b.pending.Sounds, soundID(firstEffect+n)) }

// music starts track n with the next frame shown.
func (b *flipbook) music(n int) { b.pending.Music = soundID(n) }

// fadeOut fades the music track out, holding the frame on screen while it fades.
func (b *flipbook) fadeOut() {
	b.pending.MusicFade = true
	b.showFrame(b.shown, ms(quickFadeTicks), false)
}

// stopMusic stops the music track with the next step.
func (b *flipbook) stopMusic() { b.pending.MusicStop = true }

// show shows the canvas for ticks (no wait for 0).
func (b *flipbook) show(ticks int, cancel bool) {
	b.showFrame(b.snapshot(), tickMS(ticks), cancel)
}

func tickMS(ticks int) int {
	if ticks == 0 {
		return noWaitMS
	}
	return ms(ticks)
}

// hold keeps the frame on screen for ticks.
func (b *flipbook) hold(ticks int, cancel bool) { b.showFrame(b.shown, tickMS(ticks), cancel) }

// snapshot adds the canvas as a frame, or finds the same frame already kept, and returns it.
func (b *flipbook) snapshot() int {
	for i := len(b.frames) - 1; i >= 0; i-- {
		if equalPix(b.frames[i], b.pix) {
			return i
		}
	}
	b.frames = append(b.frames, append([]int16(nil), b.pix...))
	return len(b.frames) - 1
}

func equalPix(a, c []int16) bool {
	if len(a) != len(c) {
		return false
	}
	for i := range a {
		if a[i] != c[i] {
			return false
		}
	}
	return true
}

// showFrame shows frame f for ms. A step with nothing new joins the one before it.
func (b *flipbook) showFrame(f, ms int, cancel bool) {
	if f < 0 {
		f = b.snapshot()
	}
	p := b.pending
	p.Frame, p.MS, p.Cancel = f, ms, cancel
	b.pending = content.FlipbookStep{}
	b.now += ms
	b.shown = f
	if n := len(b.steps); n > 0 && p.Sounds == nil && p.Music == "" && !p.MusicFade && !p.MusicStop {
		last := &b.steps[n-1]
		if last.Frame == f && last.Cancel == cancel {
			last.MS += ms
			return
		}
	}
	b.steps = append(b.steps, p)
}

// overlay shows the frame on screen with draw's drawing over it, for ticks, and leaves the canvas
// as it was.
func (b *flipbook) overlay(ticks int, draw func()) {
	saved := b.pix
	b.pix = make([]int16, len(saved))
	if b.shown >= 0 {
		copy(b.pix, b.frames[b.shown])
	} else {
		b.blank()
	}
	draw()
	b.show(ticks, false)
	b.pix = saved
}

// tick is the game's clock at the time played.
func (b *flipbook) tick() int { return int(float64(b.now) * ticksPerSecond / 1000) }

// write adds the flipbook and its sprite to the pack.
func (b *flipbook) write(id string, framed bool) error {
	if b.err != nil {
		return b.err
	}
	if len(b.steps) == 0 {
		return fmt.Errorf("no frames")
	}
	if p := b.pending; p.Sounds != nil || p.Music != "" || p.MusicFade || p.MusicStop {
		b.showFrame(b.shown, 0, false)
	}
	used := make([]bool, 256)
	for _, f := range b.frames {
		for _, c := range f {
			if c != clear {
				used[c] = true
			}
		}
	}
	transparent := -1
	for c := 255; c >= 0; c-- {
		if !used[c] {
			transparent = c
			break
		}
	}
	if transparent < 0 {
		return fmt.Errorf("every color is drawn; none is left for transparency")
	}
	frames := make([][]byte, len(b.frames))
	for i, f := range b.frames {
		px := make([]byte, len(f))
		for j, c := range f {
			if c == clear {
				px[j] = byte(transparent)
			} else {
				px[j] = byte(c)
			}
		}
		frames[i] = px
	}
	for i := range b.steps {
		s := &b.steps[i]
		kept := s.Sounds[:0]
		for _, sid := range s.Sounds {
			if b.im.pack.Sounds[sid] != nil {
				kept = append(kept, sid)
			}
		}
		if s.Sounds = kept; len(kept) == 0 {
			s.Sounds = nil
		}
		if s.Music != "" && b.im.pack.Sounds[s.Music] == nil {
			s.Music = ""
		}
	}
	sid := "sprite." + id
	b.im.addSprite(sid, frames, b.w, b.h, transparent)
	b.im.pack.Flipbooks[id] = &content.Flipbook{ID: id, Sprite: sid, Framed: framed, Steps: b.steps}
	return nil
}
