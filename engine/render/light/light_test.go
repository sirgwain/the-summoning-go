package light

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
)

// testPalette has colors 1–4 rotating (red, black, red, black), color 5 cycling through
// green and blue, color 6 a bright white and color 7 a dim gray.
func testPalette() *content.Palette {
	p := &content.Palette{ID: "p"}
	p.Colors[1] = color.RGBA{200, 100, 0, 255}
	p.Colors[3] = color.RGBA{200, 100, 0, 255}
	p.Colors[6] = color.RGBA{250, 250, 200, 255}
	p.Colors[7] = color.RGBA{60, 60, 60, 255}
	p.Effects = []content.PaletteEffect{
		{Kind: content.EffectRotate, PeriodMS: 100, First: 1, Size: 4, Groups: 1},
		{Kind: content.EffectCycle, PeriodMS: 100, Index: 5, Colors: []color.RGBA{{0, 200, 0, 255}, {0, 0, 200, 255}}},
	}
	return p
}

func TestSteady(t *testing.T) {
	colors, animated := Steady(testPalette())
	if want := (color.RGBA{100, 50, 0, 255}); colors[2] != want || !animated[2] {
		t.Errorf("rotated color 2 is %v (animated %v), want the group's mean %v", colors[2], animated[2], want)
	}
	if want := (color.RGBA{0, 100, 100, 255}); colors[5] != want || !animated[5] {
		t.Errorf("cycled color 5 is %v, want %v", colors[5], want)
	}
	if colors[6] != testPalette().Colors[6] || animated[6] {
		t.Errorf("still color 6 is %v (animated %v), want its own", colors[6], animated[6])
	}
}

// pack has an 8×8 sprite "s" whose frames are: 0, eight animated pixels in its top-left 4×2 and
// the rest dim; 1, three animated pixels; 2, a bright 4×4 square. And a 32×32 tile "big" whose
// lower half has 128 animated pixels in a checker in its middle 16 columns, its upper half none.
func pack() *content.Pack {
	p := content.NewPack(content.Manifest{})
	sheet := image.NewPaletted(image.Rect(0, 0, 24, 8), nil)
	for y := range 8 {
		for x := range 24 {
			sheet.Pix[y*24+x] = 7
		}
	}
	for y := range 2 {
		for x := range 4 {
			sheet.Pix[y*24+x] = 1
		}
	}
	for x := range 3 {
		sheet.Pix[8+x] = 2
	}
	for y := 2; y < 6; y++ {
		for x := 18; x < 22; x++ {
			sheet.Pix[y*24+x] = 6
		}
	}
	p.Sprites["s"] = &content.SpriteBank{ID: "s", FrameSize: content.Offset{8, 8}, Columns: 3, Frames: 3, Sheet: sheet}
	big := image.NewPaletted(image.Rect(0, 0, 32, 32), nil)
	for y := range 32 {
		for x := range 32 {
			big.Pix[y*32+x] = 7
			if y >= 16 && x >= 8 && x < 24 && y < 32 && (x+y)%2 == 0 {
				big.Pix[y*32+x] = 1
			}
		}
	}
	p.Sprites["big"] = &content.SpriteBank{ID: "big", FrameSize: content.Offset{32, 32}, Columns: 1, Frames: 1, Sheet: big}
	return p
}

func TestFindDecoration(t *testing.T) {
	f := NewFinder(pack(), testPalette())
	got := f.Find([]drawlist.Cmd{
		{Sprite: "s", Frame: 0, X: 100, Y: 50, Glow: drawlist.GlowDecoration},
		{Sprite: "s", Frame: 0, X: 0, Y: 0},
		{Sprite: "s", Frame: 1, X: 0, Y: 0, Glow: drawlist.GlowDecoration},
		{Sprite: "s", Frame: 2, X: 0, Y: 0, Glow: drawlist.GlowDecoration},
	})
	if len(got) != 1 {
		t.Fatalf("%d lights, want 1: a frame not glowing, one with 3 glowing pixels and a decoration's bright pixels give none", len(got))
	}
	l := got[0]
	// The glowing pixels' mean is (2, 1) in the frame, dropped 8 below the wall's face.
	if l.X != 102 || l.Y != 59 {
		t.Errorf("light at (%v, %v), want (102, 59)", l.X, l.Y)
	}
	if l.Color != [3]float64{1, 0.5, 0} {
		t.Errorf("color %v, want the steady (100, 50, 0) at full", l.Color)
	}
	if l.Strength != 0.30*8/16 || l.Radius != 64 {
		t.Errorf("strength %v radius %v, want %v and 64", l.Strength, l.Radius, 0.30*8/16)
	}
}

func TestFindSpell(t *testing.T) {
	f := NewFinder(pack(), testPalette())
	got := f.Find([]drawlist.Cmd{{Sprite: "s", Frame: 2, X: 10, Y: 20, Glow: drawlist.GlowSpell}})
	if len(got) != 1 {
		t.Fatalf("%d lights, want the bright square's", len(got))
	}
	l := got[0]
	if l.X != 14 || l.Y != 24 || l.Strength != 0.45 || l.Radius != 48 || math.Abs(l.Color[0]-1)+math.Abs(l.Color[1]-1)+math.Abs(l.Color[2]-0.8) > 1e-9 {
		t.Errorf("light %+v, want at (14, 24), strength 0.45, radius 48, color (1, 1, 0.8)", l)
	}
}

// TestFindThingsAndFloors lights a thing and an object from their animated pixels alone, and a
// floor tile only with 32 of them, counting those of the half drawn.
func TestFindThingsAndFloors(t *testing.T) {
	f := NewFinder(pack(), testPalette())
	got := f.Find([]drawlist.Cmd{
		{Sprite: "s", Frame: 0, X: 10, Y: 20, Glow: drawlist.GlowThing},
		{Sprite: "s", Frame: 0, X: 30, Y: 40, Glow: drawlist.GlowObject},
		{Sprite: "s", Frame: 2, X: 0, Y: 0, Glow: drawlist.GlowThing},
		{Sprite: "s", Frame: 0, X: 0, Y: 0, Glow: drawlist.GlowFloor},
		{Sprite: "big", Frame: 0, X: 0, Y: 0, Glow: drawlist.GlowFloor},
		{Sprite: "big", Frame: 0, X: 0, Y: 16, Part: drawlist.LowerHalf, Glow: drawlist.GlowFloor},
		{Sprite: "big", Frame: 0, X: 0, Y: 0, Part: drawlist.UpperHalf, Glow: drawlist.GlowFloor},
	})
	want := []Light{
		{X: 12, Y: 21, Radius: 56, Color: [3]float64{1, 0.5, 0}, Strength: 0.35 * 8 / 16, Flicker: 0.10, Speed: 2},
		{X: 32, Y: 41, Radius: 48, Color: [3]float64{1, 0.5, 0}, Strength: 0.25 * 8 / 16, Flicker: 0.05, Speed: 1},
		{X: 16, Y: 24, Radius: 48, Color: [3]float64{1, 0.5, 0}, Strength: 0.25, Flicker: 0.05, Speed: 1},
		{X: 16, Y: 24, Radius: 48, Color: [3]float64{1, 0.5, 0}, Strength: 0.25, Flicker: 0.05, Speed: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("%d lights, want %d: a thing's bright pixels, a tile with 8 glowing pixels and a half with none give none\n%+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("light %d is %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestFindShields lights the player under each shield, and both, from the middle of its frame.
func TestFindShields(t *testing.T) {
	f := NewFinder(pack(), testPalette())
	for _, c := range []struct {
		glow drawlist.Glow
		want [][3]float64
	}{
		{drawlist.GlowFireShield, [][3]float64{{1, 0.55, 0.2}}},
		{drawlist.GlowLightningShield, [][3]float64{{0.65, 0.8, 1}}},
		{drawlist.GlowShields, [][3]float64{{1, 0.55, 0.2}, {0.65, 0.8, 1}}},
	} {
		got := f.Find([]drawlist.Cmd{{Sprite: "s", Frame: 0, X: 10, Y: 20, Fixed: true, Glow: c.glow}})
		if len(got) != len(c.want) {
			t.Fatalf("glow %d: %d lights, want %d", c.glow, len(got), len(c.want))
		}
		for i, l := range got {
			if l.X != 14 || l.Y != 24 || l.Color != c.want[i] || l.Strength != 0.5 || l.Radius != 72 {
				t.Errorf("glow %d light %d: %+v, want at (14, 24) in %v, strength 0.5, radius 72", c.glow, i, l, c.want[i])
			}
		}
	}
}

func TestReach(t *testing.T) {
	l := Light{X: 10, Y: 10, Radius: 40}
	for _, c := range []struct {
		x, y, want float64
	}{{10, 10, 1}, {30, 10, 0.5625}, {10, 20, 0.5625}, {50, 10, 0}, {10, 30, 0}} {
		if got := l.Reach(c.x, c.y); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("reach at (%v, %v) is %v, want %v", c.x, c.y, got, c.want)
		}
	}
}

func TestFlicker(t *testing.T) {
	l := Light{X: 3, Y: 7, Strength: 0.3, Flicker: 0.08, Speed: 1}
	lo, hi := math.Inf(1), math.Inf(-1)
	for i := range 1000 {
		s := l.At(float64(i) / 20)
		lo, hi = min(lo, s), max(hi, s)
	}
	if lo < 0.3*0.92-1e-9 || hi > 0.3*1.08+1e-9 || hi-lo < 0.02 {
		t.Errorf("strength ranged %v–%v, want within 8%% of 0.3 and varying", lo, hi)
	}
}
