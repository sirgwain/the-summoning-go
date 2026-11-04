package summoning

import (
	"fmt"
	"image/color"
	"math/rand/v2"
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The end scenes (scenes.md, End scenes): the importer plays each by its rules, with random
// choices made at a fixed seed and no key or click, and writes it as a scene of 320 × 400. What
// stays put is the scene's pictures; what moves is a frame of pictures over them, one a frame.

// End scene ids, by the ending that plays each.
var endSceneIDs = map[string]string{"E": "scene.end.e", "F": "scene.end.f", "G": "scene.end.g"}

// The end scenes' palettes: palette 7, cycling and with its gold ramp green, and unmoving.
const (
	endPalette      = 7
	endCyclingID    = "palette.end"
	endGreenID      = "palette.end.green"
	endCycleTicks   = 6
	endFadeSteps    = 8
	endBattleTicks  = 360
	endSparkleTicks = 180
	endSeed         = 0xe1d
)

// The end scenes' tracks, each started with its loop (audio.md, Songs).
const (
	endTrackSummon = 6
	endTrackBorel  = 5
	endTrackWorld  = 7
)

// End-scene art, as end-scene base + n.
const (
	resEndPicture  = 0x00
	resEndFramed   = 0x01
	resEndBlast    = 0x07
	resEndGlow     = 0x0C
	resEndFrame    = 0x12
	resEndWorldTop = 0x14
	resEndWorldLow = 0x15
	resEndStar     = 0x16
	resEndBolt     = 0x25
	resEndBurst    = 0x2D
)

// endCaptionBox is where a caption shows, below the frame.
var endCaptionBox = [4]int{0, 340, 320, 50}

// The captions, in the engine's own words, by slot (scenes.md, End scenes, engine model).
var endWords = map[string]string{
	"magic":     "The portal flares, and the god of Magic steps forth.",
	"blast":     "Fire and fury break over you, and your life is spent.",
	"battle":    "Gods and powers clash across the heavens.",
	"fails":     "Abighael fights on bravely, but at last her power fails.",
	"dark":      "Darkness falls upon all who live.",
	"end":       "The end of the world is at hand.",
	"witch":     "Out of the portal steps the witch Chesschantra.",
	"held":      "The Staff is wrenched from your hands, and you cannot move.",
	"together":  "Mother and daughter join their power to call up the god of Magic.",
	"prevail":   "Against the two together, Magic cannot prevail.",
	"turns":     "Then Chesschantra turns on her own daughter.",
	"shake":     "The sky trembles as the battle rages on.",
	"borel":     "Out of the portal steps Borel.",
	"father":    "Abighael weeps to see her father once more.",
	"thanks":    "Borel turns to you and thanks you for all you have done.",
	"leave":     "He and his daughter will leave this world, to seek a place where they can live in peace.",
	"duty":      "Bringing peace back to this land, he tells you, is now your task.",
	"farewell":  "You say farewell as Borel and Abighael pass into the portal.",
	"warmonger": "Only when they are gone do you see that the WarMonger has gone with them.",
	"won":       "The day is yours... for now.",
}

// endUpdatesPerSecond is the presentation's rate: a frame with no wait is one update.
const endUpdatesPerSecond = 60

func endMS(updates int) int { return updates * 1000 / endUpdatesPerSecond }

// endStage plays an end scene: the pictures over the scene's, in layers from the bottom (the
// framed picture, the glow, what has been called up), and the clock in updates.
type endStage struct {
	im     *importer
	s      *content.Scene
	key    string
	base   int
	rng    *rand.Rand
	now    int
	framed []content.ScenePicture
	under  []content.ScenePicture
	kept   []content.ScenePicture
	shown  []content.ScenePicture
	frames int // the index of the last step that shows a frame, or -1
	cancel bool
	err    error
	sprite map[string]endArt
}

// endArt is a sprite of the end scenes and its frame size.
type endArt struct {
	id   string
	w, h int
}

func (st *endStage) fail(err error) {
	if st.err == nil {
		st.err = err
	}
}

func (st *endStage) tick() int { return st.now * int(ticksPerSecond*10) / (endUpdatesPerSecond * 10) }

// add appends a step that does not hold.
func (st *endStage) add(s content.SceneStep) { st.s.Steps = append(st.s.Steps, s) }

// show draws the layers with dyn on top for a frame of n updates.
func (st *endStage) show(n int, dyn ...content.ScenePicture) {
	over := slices.Concat(st.framed, st.under, st.kept, dyn)
	if over == nil {
		over = []content.ScenePicture{}
	}
	st.now += n
	steps := st.s.Steps
	if st.frames == len(steps)-1 && st.frames >= 0 && slices.Equal(over, st.shown) && steps[st.frames].Cancel == st.cancel {
		steps[st.frames].MS += endMS(n)
		return
	}
	if slices.Equal(over, st.shown) {
		st.s.Steps = append(st.s.Steps, content.SceneStep{MS: endMS(n), Cancel: st.cancel})
	} else {
		st.s.Steps = append(st.s.Steps, content.SceneStep{Over: &over, MS: endMS(n), Cancel: st.cancel})
		st.shown = over
	}
	st.frames = len(st.s.Steps) - 1
}

// wait holds the frame showing for ticks.
func (st *endStage) wait(ticks int) { st.show(max(1, (ms(ticks)*endUpdatesPerSecond+999)/1000)) }

// caption shows the caption of slot, held its reading time when held.
func (st *endStage) caption(slot string, held bool) {
	st.captionIn(slot, held, endCaptionBox)
}

func (st *endStage) captionIn(slot string, held bool, box [4]int) {
	key := fmt.Sprintf("end.%s.%s", st.key, slot)
	words := endWords[slot]
	st.im.strings[key] = words
	st.add(content.SceneStep{Caption: key, Box: &box})
	if held {
		st.show((readingMS(words)*endUpdatesPerSecond + 999) / 1000)
	}
}

func (st *endStage) uncaption() { st.add(content.SceneStep{NoCaption: true}) }

// fade fades the scene to color, or in from it, over the eight steps of a fade.
func (st *endStage) fade(c string, in bool) {
	st.add(content.SceneStep{Fade: c, FadeIn: in, MS: endMS(endFadeSteps)})
	st.now += endFadeSteps
}

// dim fades palette colors first–last alone to black.
func (st *endStage) dim(first, last int) {
	st.add(content.SceneStep{Fade: "#000000", FadeColors: &[2]int{first, last}, MS: endMS(endFadeSteps)})
	st.now += endFadeSteps
}

func (st *endStage) palette(id string) { st.add(content.SceneStep{Palette: id}) }

// music starts track n (fading out what plays first, with fade), when the pack has it.
func (st *endStage) music(n int, fade bool) {
	step := content.SceneStep{MusicFade: fade}
	if id := soundID(n); st.im.pack.Sounds[id] != nil {
		step.Music = id
	}
	if step.Music != "" || step.MusicFade {
		st.add(step)
	}
}

func (st *endStage) musicStop() { st.add(content.SceneStep{MusicStop: true}) }

// picture draws +n whole at (row, col) among the scene's pictures.
func (st *endStage) picture(n, row, col int) {
	a := st.whole(n)
	st.add(content.SceneStep{Picture: &content.Image{Sprite: a.id}, At: content.Offset{col, row}})
}

// clearTo starts the scene's picture afresh with a frame of rows × cols pieces at (row, col).
func (st *endStage) clearTo(rows, cols, row, col int) {
	st.framed, st.under, st.kept = nil, nil, nil
	a := st.frameArt(rows, cols)
	st.add(content.SceneStep{Picture: &content.Image{Sprite: a.id}, At: content.Offset{col, row}, Clear: true})
	st.shown = nil
}

// at is +n whole at (row, col), as a picture over the scene's.
func (st *endStage) at(n, row, col int) content.ScenePicture {
	return content.ScenePicture{Sprite: st.whole(n).id, At: content.Offset{col, row}}
}

// framedPicture draws the framed picture k.
func (st *endStage) framedPicture(k int) {
	st.framed = []content.ScenePicture{st.at(resEndFramed, 151, 212), st.at(2+k, 126, 227)}
}

// whole returns +n drawn whole as a sprite.
func (st *endStage) whole(n int) endArt {
	key := fmt.Sprintf("whole.%d", n)
	if a, ok := st.sprite[key]; ok {
		return a
	}
	b, err := st.im.bank(st.base + n)
	if err != nil {
		st.fail(err)
		return endArt{}
	}
	id, err := st.im.stripSprite(st.base+n, "sprite.strip")
	if err != nil {
		st.fail(err)
	}
	a := endArt{id, b.Width, b.Height * len(b.Frames)}
	st.sprite[key] = a
	return a
}

// pixels returns +n drawn whole, clear where it is transparent.
func (st *endStage) pixels(n int) (pix []int16, w, h int) {
	b, err := st.im.bank(st.base + n)
	if err != nil {
		st.fail(err)
		return nil, 0, 0
	}
	w, h = b.Width, b.Height*len(b.Frames)
	for _, f := range b.Frames {
		for _, c := range f {
			if int(c) == b.Transparent {
				pix = append(pix, clear)
			} else {
				pix = append(pix, int16(c))
			}
		}
	}
	return pix, w, h
}

// canvasSprite adds frames of w × h to the pack as sprite id.
func (st *endStage) canvasSprite(key, id string, frames [][]int16, w, h int) endArt {
	if err := st.im.addCanvasSprite(id, frames, w, h); err != nil {
		st.fail(fmt.Errorf("%s: %w", id, err))
	}
	a := endArt{id, w, h}
	st.sprite[key] = a
	return a
}

// frameArt returns a frame of rows × cols pieces of +12h, its inside clear.
func (st *endStage) frameArt(rows, cols int) endArt {
	key := fmt.Sprintf("frame.%dx%d", rows, cols)
	if a, ok := st.sprite[key]; ok {
		return a
	}
	b, err := st.im.bank(st.base + resEndFrame)
	if err != nil || len(b.Frames) < 22 {
		st.fail(fmt.Errorf("end scenes: the frame's pieces are missing"))
		return endArt{}
	}
	w, h := cols*b.Width, rows*b.Height
	pix := make([]int16, w*h)
	for i := range pix {
		pix[i] = clear
	}
	rng := rand.New(rand.NewPCG(endSeed, uint64(rows*100+cols)))
	for i := range rows {
		for j := range cols {
			piece := -1
			switch {
			case i == 0 || i == rows-1:
				switch {
				case j == 0 || j == cols-1:
					piece = 0
				case j == 1 || j == cols-2:
					piece = 1
				case j > cols/3 && j < cols-cols/3-1:
					piece = 3
				default:
					piece = 13 + rng.IntN(9)
				}
			case j == 0 || j == cols-1:
				switch {
				case i == 1 || i == rows-2:
					piece = 1
				case i > rows/3 && i < rows-rows/3-1:
					piece = 2
				default:
					piece = 4 + rng.IntN(9)
				}
			}
			if piece < 0 {
				continue
			}
			src := b.Frames[piece]
			for y := range b.Height {
				for x := range b.Width {
					if c := src[y*b.Width+x]; int(c) != b.Transparent {
						pix[(i*b.Height+y)*w+j*b.Width+x] = int16(c)
					}
				}
			}
		}
	}
	return st.canvasSprite(key, fmt.Sprintf("sprite.end.frame.%dx%d", rows, cols), [][]int16{pix}, w, h)
}

// glowArt returns the glow at each strength 4, 8, … 64, frame k at strength 4(k + 1).
func (st *endStage) glowArt() endArt {
	if a, ok := st.sprite["glow"]; ok {
		return a
	}
	pix, w, h := st.pixels(resEndGlow)
	frames := make([][]int16, 16)
	for k := range frames {
		m := 4 * (k + 1)
		f := make([]int16, len(pix))
		for i, c := range pix {
			f[i] = clear
			if c >= 0xB0 && int(c) < 0xB0+m {
				f[i] = int16(0xE0 + (int(c)-0xB0)%16)
			}
		}
		frames[k] = f
	}
	return st.canvasSprite("glow", "sprite.end.glow", frames, w, h)
}

// glow is the glow at strength m (4 … 64) at (row, col), or nothing at 0.
func (st *endStage) glow(m, row, col int) []content.ScenePicture {
	if m <= 0 {
		return nil
	}
	return []content.ScenePicture{{Sprite: st.glowArt().id, Frame: m/4 - 1, At: content.Offset{col, row}}}
}

// scaleCounts returns how many times each of s source pixels draws, scaled by num/den.
func scaleCounts(s, num, den int) []int {
	counts := make([]int, s)
	acc := den / 2
	for i := range counts {
		acc += num
		for acc >= den {
			acc -= den
			counts[i]++
		}
	}
	return counts
}

// scaledSize is a side of s scaled by num/den.
func scaledSize(s, num, den int) int { return (s*num + den/2) / den }

// scaledArt returns +n whole scaled by k/den, frame k − 1 for k = 1 … den, each at its top-left
// in a frame of the full size.
func (st *endStage) scaledArt(n, den int) endArt {
	key := fmt.Sprintf("scaled.%d.%d", n, den)
	if a, ok := st.sprite[key]; ok {
		return a
	}
	pix, w, h := st.pixels(n)
	frames := make([][]int16, den)
	for k := 1; k <= den; k++ {
		f := make([]int16, w*h)
		for i := range f {
			f[i] = clear
		}
		var rows, cols []int
		for i, c := range scaleCounts(h, k, den) {
			for range c {
				rows = append(rows, i)
			}
		}
		for i, c := range scaleCounts(w, k, den) {
			for range c {
				cols = append(cols, i)
			}
		}
		for y, sy := range rows {
			for x, sx := range cols {
				f[y*w+x] = pix[sy*w+sx]
			}
		}
		frames[k-1] = f
	}
	return st.canvasSprite(key, fmt.Sprintf("sprite.end.%d.scaled.%d", st.base+n, den), frames, w, h)
}

// scaled is +n scaled by k/den at (row, col).
func (st *endStage) scaled(n, k, den, row, col int) []content.ScenePicture {
	if k <= 0 {
		return nil
	}
	if k >= den {
		return []content.ScenePicture{st.at(n, row, col)}
	}
	return []content.ScenePicture{{Sprite: st.scaledArt(n, den).id, Frame: k - 1, At: content.Offset{col, row}}}
}

// callUp calls a figure up from the box at (r, c): body, extra (or −1) at (row, col), the
// footing at (pr, pc), with the caption of slot. The figure stays among the pictures kept; it
// returns the extra's index there, or −1.
func (st *endStage) callUp(r, c, body, extra, row, col, footing, pr, pc int, slot string) int {
	gr, gc := r+8, c+20
	st.cancel = false
	for k := range 16 {
		st.show(1, st.glow(4*(k+1), gr, gc)...)
	}
	st.caption(slot, false)
	b := st.whole(body)
	midR, midC := row+b.h/2, col+b.w/2
	for n := 1; n <= 10; n++ {
		at, ac := midR-scaledSize(b.h, n, 10)/2, midC-scaledSize(b.w, n, 10)/2
		dyn := slices.Concat(st.glow(64, gr, gc), st.scaled(body, n, 10, at, ac))
		if extra >= 0 {
			dyn = append(dyn, st.scaled(extra, n, 10, at, ac)...)
		}
		if n == 10 {
			dyn = append(dyn, st.at(footing, pr, pc))
		}
		st.show(1, dyn...)
	}
	figure := []content.ScenePicture{st.at(body, row, col)}
	if extra >= 0 {
		figure = append(figure, st.at(extra, row, col))
	}
	figure = append(figure, st.at(footing, pr, pc))
	for m := 64; m >= 0; m -= 4 {
		st.show(1, slices.Concat(st.glow(m, gr, gc), figure)...)
	}
	st.uncaption()
	i := -1
	if extra >= 0 {
		i = len(st.kept) + 1
	}
	st.kept = append(st.kept, figure...)
	return i
}

// endRegion is a region of a sparkles' area: its offset and size, (row, column).
type endRegion struct{ r, c, h, w int }

var (
	endMagicRegions = []endRegion{{4, 16, 16, 16}, {44, 4, 34, 18}, {48, 67, 30, 20}, {28, 56, 22, 2}, {48, 38, 6, 16}, {4, 52, 22, 24}}
	endWitchRegions = []endRegion{{20, 50, 24, 12}, {5, 60, 15, 19}, {20, 80, 30, 13}, {68, 53, 42, 12}, {58, 80, 50, 5}}
)

// sparkles twinkles stars over the area at (r, c), in its regions, with pairs at (r + pr, c +
// pc0) and (r + pr, c + pc1).
func (st *endStage) sparkles(r, c int, regions []endRegion, pr, pc0, pc1 int) {
	sid, _, err := st.im.rawSprite(st.base + resEndStar)
	if err != nil {
		st.fail(err)
		return
	}
	type slot struct {
		lit         bool
		frame, r, c int
	}
	var slots [6]slot
	pair, settled, stopping := 0, 0, false
	deadline := st.tick() + endSparkleTicks
	st.cancel = true
	for guard := 0; settled != 2 && guard < 5000; guard++ {
		var dyn []content.ScenePicture
		for i := range slots {
			if !stopping && i < 5 && !slots[i].lit && !slots[i+1].lit && pair == 0 {
				if st.rng.IntN(100) < 20 {
					slots[i] = slot{true, 0, r + pr, c + pc0}
					slots[i+1] = slot{true, 0, r + pr, c + pc1}
				}
				pair = i + 1
			}
			s := &slots[i]
			if s.lit {
				f := s.frame
				if f > 4 {
					f = 8 - f
				}
				dyn = append(dyn, content.ScenePicture{Sprite: sid, Frame: f, At: content.Offset{s.c - 8, s.r - 4}})
				s.frame++
			} else if !stopping && st.rng.IntN(100) < 25 {
				g := regions[st.rng.IntN(len(regions))]
				*s = slot{true, 0, r + g.r + st.rng.IntN(g.h), c + g.c + st.rng.IntN(g.w)}
			}
			if s.frame == 9 {
				s.lit = false
				if i+1 == pair {
					pair = 0
				}
			}
		}
		st.show(1, dyn...)
		if stopping {
			settled++
			for _, s := range slots {
				if s.lit {
					settled--
				}
			}
		}
		if st.tick() > deadline {
			stopping = true
		}
	}
	st.cancel = false
}

// blast grows +7 toward the player in copies (1 or 2), leaving a trail, with the caption of
// slot; then the scene stops cycling and fades to yellow.
func (st *endStage) blast(copies int, slot string) {
	st.caption(slot, false)
	b := st.whole(resEndBlast)
	den := 20 / copies
	var trail []content.ScenePicture
	for i := range den {
		k := i + 1
		h := max(1, scaledSize(b.h, k, den))
		w8 := ((scaledSize(b.w, k, den) + 7) / 8) * 8
		row := max(100, 152-h/2)
		cols := []int{max(8, 59+5*i-w8/2)}
		if copies == 2 {
			cols = append(cols, max(8, 139+5*i-w8/2))
		}
		for _, col := range cols {
			trail = append(trail, st.scaled(resEndBlast, k, den, row, col)...)
		}
		st.show(1, trail...)
	}
	st.uncaption()
	st.palette(paletteID(endPalette))
	st.fade("#ffff00", false)
}

// opening opens a scene: its frame and picture, the framed picture 0, track n (fading what plays
// first with fade), the colors in from black and cycling, and the framed picture 1.
func (st *endStage) opening(track int, fade bool) {
	st.palette(paletteID(endPalette))
	st.clearTo(15, 40, 80, 0)
	st.add(content.SceneStep{Fade: "#000000"})
	st.picture(resEndPicture, 100, 8)
	st.framedPicture(0)
	st.show(1)
	st.music(track, fade)
	st.fade("#000000", true)
	st.palette(endCyclingID)
	st.wait(90)
	st.framedPicture(1)
}

// endFigure is a figure of the world's end (scenes.md, End scenes).
type endFigure struct {
	homeR, homeC int
	r, c         int
	tr, tc       int
	moving       bool
	rows, width  int
	right        bool
	body, bodyDR int
	poses        [3]int
	poseDR       int
	held         [2]int
	atEnd        int
	art          endArt
}

// endFigures returns the world's end's figures: Chesschantra, Abighael and the god of Magic.
func endFigures() [3]*endFigure {
	return [3]*endFigure{
		{homeR: 54, homeC: 278, rows: 157, width: 120, body: 0x24, poses: [3]int{0x26, 0x27, 0x28}, held: [2]int{-1, -1}},
		{homeR: 72, homeC: 299, rows: 149, width: 112, body: 0x18, bodyDR: 38, poses: [3]int{0x21, 0x22, 0x23}, poseDR: 2,
			held: [2]int{0x1F, 0x20}, atEnd: 1},
		{homeR: 53, homeC: 18, rows: 177, width: 136, right: true, body: 0x17, bodyDR: 34, poses: [3]int{0x1B, 0x1C, 0x1D},
			poseDR: 22, held: [2]int{0x19, 0x1A}},
	}
}

func (f *endFigure) home() {
	f.r, f.c, f.tr, f.tc, f.moving = f.homeR, f.homeC, f.homeR, f.homeC, false
}

// wander steps the figure toward its target, or picks a new one when it stands.
func (f *endFigure) wander(rng *rand.Rand) {
	if !f.moving {
		f.tr, f.tc = f.homeR+rng.IntN(15), f.homeC+rng.IntN(15)
		f.moving = f.r != f.tr || f.c != f.tc
		return
	}
	dr, dc := absInt(f.r-f.tr), absInt(f.c-f.tc)
	if dr >= dc {
		f.r += sign(f.tr - f.r)
	}
	if dc >= dr {
		f.c += sign(f.tc - f.c)
	}
	if f.r == f.tr && f.c == f.tc {
		f.moving = false
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sign(v int) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	}
	return 0
}

// figureArt makes a figure's sprite: a frame for each pose and held picture (none, 0, 1), frame
// 3 × pose + held + 1, drawn facing right.
func (st *endStage) figureArt(f *endFigure) {
	key := fmt.Sprintf("figure.%d", f.body)
	if a, ok := st.sprite[key]; ok {
		f.art = a
		return
	}
	type part struct {
		pix     []int16
		w, h, r int
	}
	load := func(n, r int) *part {
		pix, w, h := st.pixels(n)
		return &part{pix, w, h, r}
	}
	body := load(f.body, f.bodyDR)
	var poses [3]*part
	for i, n := range f.poses {
		poses[i] = load(n, f.poseDR)
	}
	var held [2]*part
	for i, n := range f.held {
		if n >= 0 {
			held[i] = load(n, 0)
		}
	}
	w, h := 0, 0
	for _, p := range []*part{body, poses[0], poses[1], poses[2], held[0], held[1]} {
		if p != nil {
			w, h = max(w, p.w), max(h, p.r+p.h)
		}
	}
	var frames [][]int16
	for pose := range 3 {
		for k := -1; k < 2; k++ {
			fr := make([]int16, w*h)
			for i := range fr {
				fr[i] = clear
			}
			parts := []*part{body, poses[pose]}
			if k >= 0 {
				parts = append(parts, held[k])
			}
			for _, p := range parts {
				if p == nil {
					continue
				}
				for y := range p.h {
					for x := range p.w {
						if c := p.pix[y*p.w+x]; c != clear {
							fr[(p.r+y)*w+x] = c
						}
					}
				}
			}
			frames = append(frames, fr)
		}
	}
	f.art = st.canvasSprite(key, fmt.Sprintf("sprite.end.figure.%d", st.base+f.body), frames, w, h)
}

// place draws f in pose holding held (−1 for nothing).
func (f *endFigure) place(pose, held int) content.ScenePicture {
	if held >= 0 && f.held[held] < 0 {
		held = -1
	}
	p := content.ScenePicture{Sprite: f.art.id, Frame: 3*pose + held + 1, At: content.Offset{f.c, f.r}}
	if !f.right {
		p.At[0], p.Mirror = f.c-f.art.w, true
	}
	return p
}

// facing is 1 for a figure facing right and −1 facing left.
func (f *endFigure) facing() int {
	if f.right {
		return 1
	}
	return -1
}

// hit reports whether a bolt at column x meets a figure of figs facing facing.
func hit(x int, figs []*endFigure, facing int) bool {
	if facing < 0 {
		x += 24
	}
	for _, f := range figs {
		if f.facing() != facing {
			continue
		}
		if facing > 0 && f.c <= x && x <= f.c+f.width-32 || facing < 0 && f.c-(f.width-32) <= x && x <= f.c {
			return true
		}
	}
	return false
}

// cast has figs[i] cast a bolt at the figures facing the other way, the figures standing as drawn.
func (st *endStage) cast(figs []*endFigure, i int, standing []content.ScenePicture) {
	f := figs[i]
	bolt, _, err := st.im.rawSprite(st.base + resEndBolt)
	if err != nil {
		st.fail(err)
		return
	}
	pos := f.c + f.width*f.facing()
	if f.right {
		pos -= 24
	}
	lane := f.r + 44
	for guard := 0; guard < 200; guard++ {
		p := content.ScenePicture{Sprite: bolt, Frame: st.rng.IntN(2), At: content.Offset{pos, lane}, Mirror: f.right}
		st.show(1, append(slices.Clone(standing), p)...)
		if hit(pos, figs, -f.facing()) {
			break
		}
		pos += 4 * f.facing()
	}
	at := pos &^ 7
	if f.right {
		at += 8
	}
	for k := range 2 {
		st.show(1, append(slices.Clone(standing), st.at(resEndBurst+k, lane-32, at))...)
	}
	st.show(2, standing...)
}

// battle plays a battle of figs with the captions of slot and last; the first of a world's end
// starts its track and fades in.
func (st *endStage) battle(figs []*endFigure, first bool, slot, last string) {
	draw := func(pose func(i int) int, held func(f *endFigure) int) []content.ScenePicture {
		var l []content.ScenePicture
		for i, f := range figs {
			l = append(l, f.place(pose(i), held(f)))
		}
		return l
	}
	zero := func(int) int { return 0 }
	heldZero := func(*endFigure) int { return 0 }
	st.cancel = false
	st.show(1, draw(zero, heldZero)...)
	st.caption(slot, false)
	if first {
		st.music(endTrackWorld, false)
		st.fade("#000000", true)
	}
	st.cancel = true
	start := st.tick()
	next := start + (st.rng.IntN(3)+1)*18
	caster, state, casts := -1, 0, 0
	var standing []content.ScenePicture
	for guard := 0; state != 2 && guard < 20000; guard++ {
		if st.tick() > next {
			if caster >= 0 {
				st.cast(figs, caster, standing)
				if casts++; casts == 2 {
					st.uncaption()
				}
				next = st.tick() + (st.rng.IntN(3)+1)*18
				if state == 0 {
					caster = -1
				} else {
					state = 2
				}
			} else {
				caster = st.rng.IntN(len(figs))
			}
		}
		if state == 0 && st.tick() > start+endBattleTicks {
			state, next = 1, st.tick()-1
			var free []int
			for i, f := range figs {
				if f.atEnd == 0 {
					free = append(free, i)
				}
			}
			caster = free[st.rng.IntN(len(free))]
			st.caption(last, false)
		}
		if state == 2 {
			standing = draw(zero, func(f *endFigure) int { return f.atEnd })
		} else {
			for _, f := range figs {
				f.wander(st.rng)
			}
			standing = draw(func(i int) int {
				switch {
				case caster < 0:
					return 0
				case caster == i:
					return 1
				case state == 1:
					return 0
				}
				return 2
			}, heldZero)
		}
		st.show(1, standing...)
	}
	st.cancel = false
	st.uncaption()
}

// flight has f wander ten steps a frame toward (row, col) and go, with the caption of slot.
func (st *endStage) flight(f *endFigure, row, col int, slot string) {
	f.tr, f.tc, f.moving = row, col, true
	st.cancel = false
	st.show(1, f.place(0, 0))
	st.caption(slot, false)
	st.cancel = true
	for guard := 0; f.moving && guard < 1000; guard++ {
		for range 10 {
			if f.c != f.tc {
				f.wander(st.rng)
			}
		}
		st.show(1, f.place(0, 0))
	}
	st.cancel = false
}

// worldEnd opens the world's end: its frame and pictures, the music stopped, and the colors,
// yellow after the blast, to black; the palette stands still.
func (st *endStage) worldEnd() {
	st.clearTo(21, 40, 0, 0)
	st.add(content.SceneStep{Fade: "#ffff00"})
	st.picture(resEndWorldTop, 20, 8)
	st.musicStop()
	st.fade("#000000", false)
	st.picture(resEndWorldLow, 220, 8)
}

// worldClose closes the world's end with +n whole at (32, col) and the colors going in three
// stages.
func (st *endStage) worldClose(n, col int) {
	st.picture(n, 32, col)
	st.show(1)
	st.caption("end", false)
	st.wait(54)
	st.dim(0x00, 0xBF)
	st.wait(54)
	st.dim(0xD0, 0xDF)
	st.wait(54)
	st.fade("#000000", false)
}

// sceneE plays the god of Magic (ending E).
func (st *endStage) sceneE() {
	st.opening(endTrackSummon, true)
	extra := st.callUp(100, 8, 4, 5, 108, 41, 0x0D, 282, 50, "magic")
	st.sparkles(108, 41, endMagicRegions, 12, 42, 46)
	st.kept[extra] = st.at(6, 108, 41)
	st.framedPicture(0)
	st.blast(1, "blast")
	figs := st.worldFigures()
	st.worldEnd()
	god, abighael := figs[2], figs[1]
	st.battle([]*endFigure{god, abighael}, true, "battle", "fails")
	st.flight(god, god.homeR-16, 90, "dark")
	st.worldClose(0x1E, 92)
}

// sceneF plays Chesschantra's return (ending F).
func (st *endStage) sceneF() {
	st.opening(endTrackSummon, true)
	witch := st.callUp(100, 8, 8, 9, 122, 60, 0x0F, 280, 60, "witch")
	st.palette(endGreenID)
	st.caption("held", false)
	st.sparkles(100, 8, endWitchRegions, 34, 70, 75)
	st.palette(endCyclingID)
	magic := st.callUp(100, 88, 4, 5, 108, 121, 0x0D, 282, 130, "together")
	st.sparkles(108, 121, endMagicRegions, 12, 42, 46)
	st.kept[witch] = st.at(0x0A, 122, 60)
	st.kept[magic] = st.at(6, 108, 121)
	st.framedPicture(0)
	st.blast(2, "blast")
	figs := st.worldFigures()
	chess, abighael, god := figs[0], figs[1], figs[2]
	god.atEnd, chess.atEnd, abighael.atEnd = 1, 0, 0
	st.worldEnd()
	st.battle([]*endFigure{god, chess, abighael}, true, "battle", "prevail")
	// Chesschantra turns on her daughter: she walks toward (72, 18), turning to face right on
	// the way, with Abighael standing where the battle left her.
	chess.tr, chess.tc, chess.moving = 72, 18, true
	st.cancel = false
	st.show(1, chess.place(0, 0), abighael.place(0, 0))
	st.caption("turns", false)
	st.cancel = true
	for guard := 0; chess.moving && guard < 1000; guard++ {
		for range 10 {
			if chess.c != 18 {
				chess.wander(st.rng)
			}
		}
		if chess.c < 170 && !chess.right {
			chess.right, chess.c = true, 100
		}
		st.show(1, chess.place(0, 0), abighael.place(0, 0))
	}
	st.cancel = false
	chess.homeR, chess.homeC, chess.right = 72, 18, true
	chess.home()
	chess.atEnd, abighael.atEnd = 0, 1
	st.battle([]*endFigure{chess, abighael}, false, "shake", "fails")
	st.flight(chess, chess.homeR-16, 90, "dark")
	st.worldClose(0x29, 110)
}

// sceneG plays Borel's farewell (ending G).
func (st *endStage) sceneG() {
	st.opening(endTrackBorel, false)
	st.callUp(100, 8, 0x0B, -1, 114, 52, 0x0E, 282, 55, "borel")
	st.wait(54)
	for _, slot := range []string{"father", "thanks", "leave"} {
		st.caption(slot, true)
	}
	// The portal clears and Abighael steps out, then walks to Borel.
	st.framed = nil
	stays := st.at(0x11, 268, 220)
	st.show(1, st.at(0x2C, 192, 201), st.at(0x10, 126, 192), st.at(0x13, 284, 208), stays)
	var abighael []content.ScenePicture
	for i := range 4 {
		abighael = []content.ScenePicture{st.at(0x2B+i%2, 192, 181-20*i), st.at(0x2A, 126, 172-20*i), st.at(0x13, 284, 188-20*i)}
		st.show(max(1, (ms(3)*endUpdatesPerSecond+999)/1000), append(slices.Clone(abighael), stays)...)
	}
	borel := st.kept
	st.kept = slices.Concat(st.kept, abighael, []content.ScenePicture{stays})
	st.caption("duty", true)
	st.uncaption()
	for k := range 16 {
		st.under = st.glow(4*(k+1), 108, 52)
		st.show(1)
	}
	st.captionIn("farewell", false, [4]int{0, 340, 320, 36})
	// The two go into the portal: everything in the area at rows 100–299, columns 48–175 gives
	// way to the scene's picture, the glow and the two shrinking.
	st.kept = []content.ScenePicture{stays}
	for _, p := range borel {
		if p.At[1] >= 300 {
			st.kept = append(st.kept, p)
		}
	}
	body, lower := st.whole(0x0B), st.whole(0x2A)
	for n := 9; n >= 0; n-- {
		s := func(v int) int { return (v*n + 5) / 10 }
		b, a, e := s(body.w), s(lower.w), s(4)
		t := b + a - e
		bc := 110 - t/2
		lr := 208 - s(160)/2
		lc := bc + a
		dyn := slices.Concat(
			st.scaled(0x0B, n, 10, 208-((body.h*n+5)/10)/2, bc),
			st.scaled(0x2C, n, 10, lr+s(66), lc+s(9)),
			st.scaled(0x2A, n, 10, lr, lc))
		st.show(1, dyn...)
	}
	for m := 64; m >= 0; m -= 4 {
		st.under = st.glow(m, 108, 52)
		st.show(1)
	}
	st.caption("warmonger", true)
	st.caption("won", true)
	st.palette(paletteID(endPalette))
	st.wait(18)
	st.fade("#000000", false)
}

// worldFigures returns the world's end's figures at home, with their sprites.
func (st *endStage) worldFigures() [3]*endFigure {
	figs := endFigures()
	for _, f := range figs {
		f.home()
		st.figureArt(f)
	}
	return figs
}

// endPalettes adds palette 7 cycling, and cycling with its gold ramp green.
func (im *importer) endPalettes() bool {
	base := im.pack.Palettes[paletteID(endPalette)]
	if base == nil {
		return false
	}
	cycle := []content.PaletteEffect{
		{Kind: content.EffectRotate, PeriodMS: ms(endCycleTicks), First: 0xE0, Size: 16, Groups: 1},
		{Kind: content.EffectRotate, PeriodMS: ms(endCycleTicks), First: 0xF0, Size: 8, Groups: 1},
	}
	cycling := &content.Palette{ID: endCyclingID, Colors: base.Colors, Effects: cycle}
	green := &content.Palette{ID: endGreenID, Colors: base.Colors, Effects: cycle}
	for i := 0xF0; i < 0xF8; i++ {
		green.Colors[i] = color.RGBA{G: base.Colors[i].G, A: 255}
	}
	im.pack.Palettes[cycling.ID] = cycling
	im.pack.Palettes[green.ID] = green
	return true
}

// endScenes adds the end scenes and gives the ending's choices theirs.
func (im *importer) endScenes() {
	c := im.pack.Creation
	if c == nil || c.Ending == nil {
		return
	}
	if !im.endPalettes() {
		im.report.addf("the end scenes: palette %d is missing; left out", endPalette)
		return
	}
	plays := map[string]func(*endStage){"E": (*endStage).sceneE, "F": (*endStage).sceneF, "G": (*endStage).sceneG}
	for i := range c.Ending.Choices {
		ch := &c.Ending.Choices[i]
		play, ok := plays[ch.Name]
		if !ok {
			continue
		}
		id := endSceneIDs[ch.Name]
		st := &endStage{im: im, key: ch.Name, base: int(im.init.Bases.EndScene), frames: -1, sprite: map[string]endArt{},
			rng: rand.New(rand.NewPCG(endSeed, uint64(ch.Name[0]))),
			s:   &content.Scene{ID: id, Size: [2]int{content.SceneWidth, 2 * content.SceneHeight}, EscapeOnly: true}}
		play(st)
		st.add(content.SceneStep{MusicFade: true})
		if st.err != nil {
			im.report.addf("the end scene %s: %v; left out", ch.Name, st.err)
			continue
		}
		im.pack.Scenes[id] = st.s
		ch.Scene = id
	}
}
