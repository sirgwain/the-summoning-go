package summoning

import (
	"image/color"
	"math"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Palette animation (scenes.md).

const (
	// paletteTicks is how often the palette animates, in ticks.
	paletteTicks   = 8
	ticksPerSecond = 18.2
)

// pulseColors are the colors FCh steps through, in 6-bit components.
var pulseColors = [][3]uint8{{15, 0, 27}, {24, 6, 38}, {34, 14, 47}, {45, 26, 57}}

// paletteEffects returns the animation every palette runs.
func paletteEffects() []content.PaletteEffect {
	period := int(math.Round(paletteTicks * 1000 / ticksPerSecond))
	pulse := make([]color.RGBA, len(pulseColors))
	for i, c := range pulseColors {
		pulse[i] = color.RGBA{scale6(c[0]), scale6(c[1]), scale6(c[2]), 0xFF}
	}
	return []content.PaletteEffect{
		{Kind: content.EffectRotate, PeriodMS: period, First: 0xE0, Size: 4, Groups: 4},
		{Kind: content.EffectRotate, PeriodMS: period, First: 0xF0, Size: 4, Groups: 3},
		{Kind: content.EffectCycle, PeriodMS: period, Index: 0xFC, Colors: pulse},
	}
}
