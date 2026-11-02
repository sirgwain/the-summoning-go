package summoning

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// solid is a bank of n frames w × h, frame i filled with color c + i, with transparent index t.
func solid(n, w, h, c, t int) *SpriteBank {
	b := &SpriteBank{Width: w, Height: h, Transparent: t}
	for i := range n {
		f := make([]byte, w*h)
		for j := range f {
			f[j] = byte(c + i)
		}
		b.Frames = append(b.Frames, f)
	}
	return b
}

func flipbookImporter(banks map[int]*SpriteBank) *importer {
	pal := &content.Palette{ID: "palette.0"}
	p := content.NewPack(content.Manifest{})
	p.Palettes[pal.ID] = pal
	return &importer{pack: p, init: &Init{}, palette: pal, report: &Report{}, banks: banks}
}

// pixelAt reads a pixel of a flipbook frame from its sheet.
func pixelAt(t *testing.T, p *content.Pack, fb *content.Flipbook, frame, x, y int) uint8 {
	t.Helper()
	b := p.Sprites[fb.Sprite]
	col, row := frame%b.Columns, frame/b.Columns
	return b.Sheet.ColorIndexAt(col*b.FrameSize[0]+x, row*b.FrameSize[1]+y)
}

func TestEffectFall(t *testing.T) {
	const base = 1000
	banks := map[int]*SpriteBank{}
	for i := range 3 {
		banks[base+6+i] = solid(1, 56, 96, 10+i, NoTransparency)
	}
	im := flipbookImporter(banks)
	im.pack.Sounds["sfx.21"] = &content.Sound{ID: "sfx.21"}
	b := newFlipbook(im, base, 56, 96, 2)
	effectFall(b)
	if err := b.write(effectID(2), true); err != nil {
		t.Fatal(err)
	}
	fb := im.pack.Flipbooks[effectID(2)]
	if len(fb.Steps) != 9 || im.pack.Sprites[fb.Sprite].Frames != 3 || !fb.Framed {
		t.Fatalf("%d steps, %d frames, framed %v; want 9, 3, true", len(fb.Steps), im.pack.Sprites[fb.Sprite].Frames, fb.Framed)
	}
	for i, s := range fb.Steps {
		if s.Frame != i%3 || s.MS != ms(5) || !s.Cancel {
			t.Errorf("step %d: %+v", i, s)
		}
		if got := pixelAt(t, im.pack, fb, s.Frame, 20, 50); got != uint8(10+i%3) {
			t.Errorf("step %d draws color %d, want %d", i, got, 10+i%3)
		}
	}
	if s := fb.Steps[0].Sounds; len(s) != 1 || s[0] != "sfx.21" {
		t.Errorf("first step's sounds %v, want sfx.21", s)
	}
}

func TestFlipbookCanvas(t *testing.T) {
	// A 4 × 2 bank: color 5 with transparent 0 in its right column; a strip of two 1-pixel lines.
	art := &SpriteBank{Width: 4, Height: 2, Transparent: 0, Frames: [][]byte{{5, 5, 6, 0, 5, 5, 6, 0}}}
	lines := &SpriteBank{Width: 2, Height: 1, Transparent: NoTransparency, Frames: [][]byte{{7, 7}, {8, 8}}}
	im := flipbookImporter(map[int]*SpriteBank{1: art, 2: lines})
	b := newFlipbook(im, 0, 8, 4, 1)
	b.show(3, false) // a blank frame
	b.draw(1, 0, 0)
	b.mirrored(1, 2, 4)
	b.show(0, false)
	b.hold(2, false) // joins the step before
	b.blank()
	b.clipTo(0, 0, 3, 2)
	b.lines(2, 1, 1, false, func(i int) int { return i }, nil)
	b.show(1, true)
	b.music(0x0F)
	b.stopMusic()
	if err := b.write("flipbook.t", false); err != nil {
		t.Fatal(err)
	}
	fb := im.pack.Flipbooks["flipbook.t"]
	want := []content.FlipbookStep{
		{Frame: 0, MS: ms(3)},
		{Frame: 1, MS: noWaitMS + ms(2)},
		{Frame: 2, MS: ms(1), Cancel: true},
		{Frame: 2, MS: 0, Music: "", MusicStop: true},
	}
	if len(fb.Steps) != len(want) {
		t.Fatalf("steps %+v", fb.Steps)
	}
	for i := range want {
		if fb.Steps[i].Frame != want[i].Frame || fb.Steps[i].MS != want[i].MS || fb.Steps[i].Cancel != want[i].Cancel ||
			fb.Steps[i].MusicStop != want[i].MusicStop || fb.Steps[i].Music != "" {
			t.Errorf("step %d: %+v, want %+v (music.15 is not in the pack, so it is dropped)", i, fb.Steps[i], want[i])
		}
	}
	tr := *im.pack.Sprites[fb.Sprite].Transparent
	for _, c := range []struct{ frame, x, y, want int }{
		{0, 0, 0, tr},
		{1, 0, 0, 5}, {1, 2, 0, 6}, {1, 3, 0, tr}, // drawn as is
		{1, 4, 2, tr}, {1, 5, 2, 6}, {1, 7, 3, 5}, // mirrored: its transparent column on the left
		{2, 1, 1, 7}, {2, 2, 1, 7}, // line 0 at (1, 1)
		{2, 2, 2, 8}, {2, 3, 2, tr}, // line 1 shifted right by 1, cut by the clip at column 2
	} {
		if got := pixelAt(t, im.pack, fb, c.frame, c.x, c.y); int(got) != c.want {
			t.Errorf("frame %d (%d, %d) = %d, want %d", c.frame, c.x, c.y, got, c.want)
		}
	}
}

func TestSparkle(t *testing.T) {
	// Five star frames, 1 × 1, in colors 60h..64h.
	stars := solid(5, 1, 1, 0x60, NoTransparency)
	im := flipbookImporter(map[int]*SpriteBank{0x5E: stars})
	b := newFlipbook(im, 0, sparkleSize, sparkleSize, 1)
	playSparkle(b, 0x5E, 0x30)
	if err := b.write(sparkleID("heal"), false); err != nil {
		t.Fatal(err)
	}
	fb := im.pack.Flipbooks[sparkleID("heal")]
	tr := *im.pack.Sprites[fb.Sprite].Transparent
	if pixelAt(t, im.pack, fb, fb.Steps[0].Frame, 7, 0) != uint8(tr) {
		t.Error("the first frame is not blank")
	}
	total := 0
	seen := map[uint8]bool{}
	for _, s := range fb.Steps {
		total += s.MS
		for y := range sparkleSize {
			for x := range sparkleSize {
				if c := pixelAt(t, im.pack, fb, s.Frame, x, y); c != uint8(tr) {
					seen[c] = true
					if y%2 != 0 || y >= 40 {
						t.Fatalf("a star at row %d", y)
					}
				}
			}
		}
	}
	if total != sparkleFrames*ms(sparkleTicks) {
		t.Errorf("plays %d ms, want %d", total, sparkleFrames*ms(sparkleTicks))
	}
	for c := range seen {
		if c < 0x30 || c > 0x34 {
			t.Errorf("color %#x drawn; the stars should be remapped to 30h..34h", c)
		}
	}
	if len(seen) != 5 {
		t.Errorf("colors drawn %v; want all five star frames", seen)
	}
}
