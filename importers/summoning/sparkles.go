package summoning

// The sparkles of healing, cures and the sealed doors (scenes.md, Sparkles), each rendered into a
// flipbook of its 48 × 48 area.

const (
	sparkleSize   = 48
	sparkleStars  = 96
	sparkleFrames = 25
	sparkleTicks  = 2
	// sparkleOwnColor draws a sparkle in its bank's own colors.
	sparkleOwnColor = 0x60
)

// sparkles are the kinds of sparkle: their name, bank and color.
var sparkles = []struct {
	name        string
	bank, color int
}{
	{"cure", 0x5E, 0x30},
	{"heal", 0x5E, 0x50},
	{"restore", 0x60, sparkleOwnColor},
	{"seal.5", 0x5F, sparkleOwnColor},
	{"seal.6", 0x5E, 0x80},
	{"seal.7", 0x5E, 0x30},
}

// sparkleID is the id of a sparkle's flipbook.
func sparkleID(name string) string { return "flipbook.sparkle." + name }

// sparkleFrame is the star frame each phase from 8 draws.
var sparkleFrame = [16]int{8: 1, 9: 2, 10: 3, 11: 4, 12: 3, 13: 2, 14: 1, 15: 0}

func (im *importer) sparkles() {
	for i, s := range sparkles {
		b := newFlipbook(im, 0, sparkleSize, sparkleSize, uint64(100+i))
		playSparkle(b, s.bank, s.color)
		if err := b.write(sparkleID(s.name), false); err != nil {
			im.report.addf("sparkle %s: %v; left out", s.name, err)
		}
	}
}

func playSparkle(b *flipbook, bank, color int) {
	var phase, step [sparkleStars]int
	h := 9
	for j := range sparkleStars {
		phase[j] = (5 + 5*j) % 16
		step[j] = h
		if h = (h + 7) % 16; h == 0 {
			h = 7
		}
	}
	var m remap
	if color != sparkleOwnColor {
		m = func(c byte) (byte, bool) {
			if c >= 0x60 && c <= 0x6F {
				return byte(color) + c - 0x60, true
			}
			return c, true
		}
	}
	stars := b.res(bank)
	for k := range sparkleFrames {
		b.blank()
		limit := 7 + k
		for _, pass := range [][2]int{{8, 15}, {9, 14}, {10, 13}, {11, 12}} {
			row, col := 0, 7
			for j := range sparkleStars {
				if row >= 40 {
					break
				}
				p := phase[j]
				if (p == pass[0] || p == pass[1]) && p < limit {
					b.blit(stars, sparkleFrame[p], row, col, false, false, m)
				}
				if col += step[j]; col >= 32 {
					col -= 32
					row += 2
				}
			}
		}
		b.show(sparkleTicks, false)
		for j := range phase {
			if k < sparkleFrames-10 || phase[j] >= 7 {
				phase[j] = (phase[j] + 1) % 16
			}
		}
	}
}
