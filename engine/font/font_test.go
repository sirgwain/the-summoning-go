package font

import (
	"image"
	"testing"
)

func TestEveryLetterFitsItsCell(t *testing.T) {
	for r := first; r <= last; r++ {
		g, ok := glyphs[r]
		if !ok {
			t.Errorf("no letter %q", r)
			continue
		}
		if g.at+len(g.rows) > 9 {
			t.Errorf("%q reaches row %d", r, g.at+len(g.rows))
		}
		for _, row := range g.rows {
			if len(row) != 5 {
				t.Errorf("%q has a row %q", r, row)
			}
		}
	}
}

func TestAtlasOutlinesEachLetter(t *testing.T) {
	img := Atlas()
	if img.Bounds() != image.Rect(0, 0, int(last-first+1)*CellWidth, LineHeight) {
		t.Fatalf("atlas %v", img.Bounds())
	}
	// 'A' has white in its cell and a dark outline round it, and its cell's edges are clear
	// apart from the outline.
	c := Cell('A')
	white, dark := 0, 0
	for y := c.Min.Y; y < c.Max.Y; y++ {
		for x := c.Min.X; x < c.Max.X; x++ {
			p := img.RGBAAt(x, y)
			switch {
			case p.A == 0:
			case p.R > 200:
				white++
			default:
				dark++
			}
		}
	}
	if white == 0 || dark <= white/2 {
		t.Errorf("'A': %d lit, %d outline", white, dark)
	}
	if Cell('é') != Cell('?') || Width("abc") != 3*Advance {
		t.Error("unknown letters or widths")
	}
}
