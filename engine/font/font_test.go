package font

import (
	"image"
	"testing"
)

func TestEveryLetterFitsItsCell(t *testing.T) {
	for _, f := range []*Face{Small, Large} {
		for r := first; r <= last; r++ {
			g, ok := f.glyphs[r]
			if !ok {
				t.Errorf("face %d: no letter %q", f.Advance, r)
				continue
			}
			if g.at+len(g.rows) > f.Rows {
				t.Errorf("face %d: %q reaches row %d", f.Advance, r, g.at+len(g.rows))
			}
			for _, row := range g.rows {
				if len(row) != f.Width {
					t.Errorf("face %d: %q has a row %q", f.Advance, r, row)
				}
			}
		}
		// The outline, a row above the capitals and below the descenders, stays in the line.
		if f.top < 1 || f.top+f.Rows+1 > f.LineHeight || f.Width+2 > f.CellWidth() {
			t.Errorf("face %d does not fit its line", f.Advance)
		}
	}
}

func TestAtlasOutlinesEachLetter(t *testing.T) {
	for _, f := range []*Face{Small, Large} {
		img := f.Atlas()
		if img.Bounds() != image.Rect(0, 0, int(last-first+1)*f.CellWidth(), f.LineHeight) {
			t.Fatalf("face %d: atlas %v", f.Advance, img.Bounds())
		}
		// 'A' has white in its cell and a dark outline round it.
		c := f.Cell('A')
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
			t.Errorf("face %d: 'A' has %d lit, %d outline", f.Advance, white, dark)
		}
		if f.Cell('é') != f.Cell('?') || f.Measure("abc") != 3*f.Advance {
			t.Errorf("face %d: unknown letters or widths", f.Advance)
		}
	}
}
