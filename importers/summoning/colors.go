package summoning

import (
	"fmt"
	"image/color"
)

// Palette is a full 256-color palette.
type Palette [256]color.RGBA

const (
	colorsPerBlock   = 16
	blocksPerPalette = 256 / colorsPerBlock
)

// ReadColors reads COLORS into its full palettes: palette p is blocks 16p..16p+15. Blocks after
// the last full palette are ignored.
func ReadColors(data []byte) ([]Palette, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("colors: empty file")
	}
	blocks := int(data[0])
	if 1+blocks*colorsPerBlock*3 > len(data) {
		return nil, fmt.Errorf("colors: %d blocks need %d bytes, have %d", blocks, 1+blocks*colorsPerBlock*3, len(data))
	}
	rgb := data[1:]
	palettes := make([]Palette, blocks/blocksPerPalette)
	for p := range palettes {
		for i := range 256 {
			c := rgb[(p*256+i)*3:]
			palettes[p][i] = color.RGBA{scale6(c[0]), scale6(c[1]), scale6(c[2]), 0xFF}
		}
	}
	return palettes, nil
}

// scale6 converts a 6-bit color component to 8 bits, rounded. The two unused high bits are
// ignored.
func scale6(c byte) uint8 {
	c &= 0x3F
	return uint8((int(c)*255 + 31) / 63)
}
