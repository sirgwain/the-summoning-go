package window

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/frame"
	"github.com/sirgwain/the-summoning-go/engine/sim"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// The armor gauge's color, and the colors of an attribute below and above its maximum.
var (
	armorColor    = color.RGBA{0x9a, 0xb4, 0xd0, 0xff}
	lowToneColor  = color.RGBA{0xe0, 0x6a, 0x5a, 0xff}
	highToneColor = color.RGBA{0x8c, 0xd8, 0x7a, 0xff}
)

// drawInventory draws the inventory panel: the figure with what it wears, the worn slots' and
// the pack's boxes with their objects, and the lines of details and stats. It draws the panel at
// its own pixels, then scales it onto the screen.
func (s *Window) drawInventory(screen *ebiten.Image) {
	v := s.v
	ip := v.Inventory()
	if ip == nil {
		return
	}
	size := ip.Rect.Size().Div(ip.Scale)
	if s.inv == nil || s.inv.Bounds().Size() != size {
		if s.inv != nil {
			s.inv.Deallocate()
		}
		s.inv = ebiten.NewImage(size.X, size.Y)
	}
	win := s.inv
	win.Clear()
	s.drawFrame(screen, frame.Iron, ip.Frame, ip.Scale, 1)
	for _, b := range ip.Slots {
		r := b.Rect.Inset(1)
		vector.FillRect(win, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), slotColor, false)
		edge := slotEdgeColor
		if b.Slot >= sim.Shelves && b.Slot < sim.SlotPack {
			edge = handEdgeColor
		}
		if s, ok := v.World.Player().OpenSlot(v.Pack); ok && s == b.Slot {
			edge = openEdgeColor
		}
		vector.StrokeRect(win, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, edge, false)
	}
	art := s.artImage(size)
	draw := func(img content.Image, at image.Point, mirrored bool) {
		if err := s.sprites.DrawImage(art, img, at.X, at.Y, mirrored); err != nil && s.err == nil {
			s.err = err
		}
	}
	if ip.HasDoll {
		draw(ip.Doll, ip.DollAt, false)
		for _, part := range ip.Parts {
			draw(part.Image, ip.DollAt.Add(part.At), part.Mirrored)
		}
	}
	for _, b := range ip.Slots {
		if img, ok := v.SlotImage(b.Slot); ok {
			fs := v.Pack.Sprites[img.Sprite].FrameSize
			draw(img, b.Rect.Min.Add(image.Pt((b.Rect.Dx()-fs[0])/2, (b.Rect.Dy()-fs[1])/2)), false)
		}
	}
	textAt := map[int]int{}
	for i, l := range ip.Lines {
		if l.Icon == nil {
			continue
		}
		draw(*l.Icon, l.At.Add(image.Pt(0, 4)), false)
		if b := v.Pack.Sprites[l.Icon.Sprite]; b != nil {
			textAt[i] = b.FrameSize[0] + 4
		}
	}
	s.presenter.PresentAt(win, art, 1, image.Point{})
	if g := ip.Armor; g.Max > 0 {
		r := g.Rect
		vector.FillRect(win, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), slotColor, false)
		full := r
		full.Max.X = r.Min.X + r.Dx()*g.Value/g.Max
		vector.FillRect(win, float32(full.Min.X), float32(full.Min.Y), float32(full.Dx()), float32(full.Dy()), armorColor, false)
		vector.StrokeRect(win, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, slotEdgeColor, false)
	}
	for i, l := range ip.Lines {
		switch l.Tone {
		case viewer.ToneLow:
			s.printIn(win, l.Text, l.At.X+textAt[i], l.At.Y, lowToneColor)
		case viewer.ToneHigh:
			s.printIn(win, l.Text, l.At.X+textAt[i], l.At.Y, highToneColor)
		default:
			s.print(win, l.Text, l.At.X+textAt[i], l.At.Y)
		}
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(ip.Scale), float64(ip.Scale))
	op.GeoM.Translate(float64(ip.Rect.Min.X), float64(ip.Rect.Min.Y))
	screen.DrawImage(win, op)
}
