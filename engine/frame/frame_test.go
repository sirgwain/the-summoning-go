package frame

import (
	"image"
	"testing"
)

func TestDrawFillsTheBorderAndLeavesTheInside(t *testing.T) {
	for _, style := range []Style{Iron, Runic, Hull} {
		p := Draw(style, 120, 60)
		if got := p.Base.Bounds(); got != image.Rect(0, 0, 120, 60) {
			t.Fatalf("style %d: bounds %v", style, got)
		}
		// Every frame pixel is opaque; the inside is clear.
		for y := range 60 {
			for x := range 120 {
				a := p.Base.RGBAAt(x, y).A
				inFrame := min(x, y, 119-x, 59-y) < Width
				if inFrame && a != 255 {
					t.Fatalf("style %d: frame pixel (%d, %d) has alpha %d", style, x, y, a)
				}
				if !inFrame && a != 0 {
					t.Fatalf("style %d: inside pixel (%d, %d) has alpha %d", style, x, y, a)
				}
			}
		}
	}
}

func TestDrawIsTheSameEachTime(t *testing.T) {
	a, b := Draw(Runic, 90, 70), Draw(Runic, 90, 70)
	for i := range a.Base.Pix {
		if a.Base.Pix[i] != b.Base.Pix[i] {
			t.Fatal("two draws differ")
		}
	}
}

func TestOnlyRunicHasLightsAndOnlyOnTheFrame(t *testing.T) {
	if p := Draw(Iron, 80, 40); p.Glow[0] != nil {
		t.Error("iron has lights")
	}
	p := Draw(Runic, 120, 60)
	for g, img := range p.Glow {
		lit := 0
		for y := range 60 {
			for x := range 120 {
				if img.RGBAAt(x, y).A == 0 {
					continue
				}
				lit++
				if min(x, y, 119-x, 59-y) >= Width {
					t.Fatalf("group %d lights the inside at (%d, %d)", g, x, y)
				}
			}
		}
		if lit == 0 {
			t.Errorf("group %d has no lights", g)
		}
	}
}

func TestPulseStaysInRange(t *testing.T) {
	for i := range 100 {
		for g := range GlowGroups {
			if p := Pulse(g, float64(i)*0.13); p < 0 || p > 1 {
				t.Fatalf("pulse %v", p)
			}
		}
	}
}

func TestParchmentIsTornOnlyWhenRagged(t *testing.T) {
	p := Parchment(80, 60, true)
	if p.RGBAAt(40, 0).A != 0 || p.RGBAAt(40, 59).A != 0 {
		t.Error("a ragged sheet's top and bottom rows are not clear")
	}
	if p.RGBAAt(40, 30).A != 255 || p.RGBAAt(0, 30).A != 255 {
		t.Error("a ragged sheet is not whole in the middle and at its sides")
	}
	q := Parchment(80, 60, false)
	for _, pt := range []image.Point{{0, 0}, {79, 59}, {40, 30}} {
		if q.RGBAAt(pt.X, pt.Y).A != 255 {
			t.Errorf("a plain sheet is clear at %v", pt)
		}
	}
	r := Roll(12, 100)
	if r.RGBAAt(6, 50).A != 255 || r.RGBAAt(0, 0).A != 0 {
		t.Error("the roll is not a cylinder with rounded ends")
	}
}

// TestHullLights gives the hull frame amber lights in every group, and finds every built-in style
// by its name.
func TestHullLights(t *testing.T) {
	p := Draw(Hull, 200, 120)
	for g, img := range p.Glow {
		if img == nil {
			t.Fatalf("group %d has no lights", g)
		}
		lit := false
		for y := range 120 {
			for x := range 200 {
				if c := img.RGBAAt(x, y); c.A > 200 && c.R > c.B {
					lit = true
				}
			}
		}
		if !lit {
			t.Errorf("group %d has no amber light", g)
		}
	}
	for name, want := range map[string]Style{"iron": Iron, "runic": Runic, "hull": Hull} {
		if Names[name] != want {
			t.Errorf("style %q is %d, want %d", name, Names[name], want)
		}
	}
}
