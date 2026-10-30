package content

import (
	"image"
	"image/color"
)

// NewSheet lays frames of the given size out on an indexed sheet with up to maxColumns columns.
// frames[i] holds frame i's palette indexes, row-major. Pixels outside every frame get fill.
// The sheet's palette is pal, with the transparent index (if any) fully transparent, so the PNG
// shows correctly in an image editor.
func NewSheet(frames [][]byte, w, h, maxColumns int, pal *Palette, transparent *int, fill uint8) (sheet *image.Paletted, columns int) {
	columns = max(1, min(maxColumns, len(frames)))
	rows := max(1, (len(frames)+columns-1)/columns)

	p := make(color.Palette, len(pal.Colors))
	for i, c := range pal.Colors {
		p[i] = c
	}
	if transparent != nil {
		p[*transparent] = color.RGBA{}
	}

	sheet = image.NewPaletted(image.Rect(0, 0, columns*w, rows*h), p)
	for i := range sheet.Pix {
		sheet.Pix[i] = fill
	}
	for i, f := range frames {
		x0, y0 := i%columns*w, i/columns*h
		for y := range h {
			copy(sheet.Pix[sheet.PixOffset(x0, y0+y):][:w], f[y*w:(y+1)*w])
		}
	}
	return sheet, columns
}
