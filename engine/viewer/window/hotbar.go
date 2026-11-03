package window

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/frame"
	"github.com/sirgwain/the-summoning-go/engine/sim"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// artImage returns a cleared index image of size, kept from frame to frame.
func (s *Window) artImage(size image.Point) *ebiten.Image {
	img, ok := s.art[size]
	if !ok {
		img = ebiten.NewImage(size.X, size.Y)
		s.art[size] = img
	}
	img.Clear()
	return img
}

// actionWait is how much of an action box shows its wait: its hand's delay still to run, or all
// of it when the box cannot act for another reason.
func actionWait(a viewer.ActionButton) float64 {
	if !a.Ready && a.Wait == 0 {
		return 1
	}
	return a.Wait
}

// fillWait shades the part of box r a wait still has to run, left of it (0..1): a sector from
// twelve o'clock round to where the wait has reached, so the box clears clockwise as it runs out.
func fillWait(dst *ebiten.Image, r image.Rectangle, left float64, c color.Color) {
	switch {
	case left <= 0 || r.Empty():
		return
	case left >= 1:
		vector.FillRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
		return
	}
	cx, cy := float32(r.Min.X)+float32(r.Dx())/2, float32(r.Min.Y)+float32(r.Dy())/2
	// The sector reaches past the corners; the box clips it.
	radius := float32(math.Hypot(float64(r.Dx()), float64(r.Dy())))/2 + 1
	var p vector.Path
	p.MoveTo(cx, cy)
	p.Arc(cx, cy, radius, float32(-math.Pi/2+2*math.Pi*(1-left)), float32(3*math.Pi/2), vector.Clockwise)
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.FillPath(dst.SubImage(r).(*ebiten.Image), &p, nil, op)
}

// drawHotbar draws the hotbar: boxes, gauges and action buttons at native resolution, then the
// portrait and the objects' icons through the palette at the hotbar's scale.
func (s *Window) drawHotbar(screen *ebiten.Image) {
	v := s.v
	h := v.Hotbar()
	if h == nil {
		return
	}
	fill := func(r image.Rectangle, c color.Color) {
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
	}
	stroke := func(r image.Rectangle, c color.Color) {
		vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, c, false)
	}
	s.drawFrame(screen, frame.Iron, h.Frame, h.Scale)
	for i, g := range h.Gauges {
		r := h.Screen(g.Rect)
		fill(r, slotColor)
		if g.Max > 0 {
			full := r
			c := gaugeColors[i]
			if g.Value >= g.Max && i == viewer.GaugeLoad {
				c = overloadColor
			}
			full.Min.Y = r.Max.Y - r.Dy()*min(g.Value, g.Max)/g.Max
			fill(full, c)
		}
		stroke(r, slotEdgeColor)
	}
	for _, b := range h.Slots {
		r := h.Screen(b.Rect).Inset(1)
		fill(r, slotColor)
		edge := slotEdgeColor
		if b.Slot >= sim.Shelves {
			edge = handEdgeColor
		}
		if s, ok := v.World.Player().OpenSlot(v.Pack); ok && s == b.Slot {
			edge = openEdgeColor
		}
		stroke(r, edge)
	}
	for _, b := range h.Spells {
		r := h.Screen(b.Rect).Inset(1)
		fill(r, slotColor)
		fillWait(screen, r, b.Wait, waitColor)
		stroke(r, spellEdgeColor)
	}
	fill(h.Screen(h.Gesture.Rect).Inset(1), slotColor)
	stroke(h.Screen(h.Gesture.Rect).Inset(1), spellEdgeColor)
	for _, b := range []viewer.PanelButton{h.Log, h.Book} {
		box := h.Screen(b.Rect).Inset(1)
		fill(box, actionColor)
		stroke(box, slotEdgeColor)
		ebitenutil.DebugPrintAt(screen, b.Label, box.Min.X+(box.Dx()-len(b.Label)*viewer.CharWidth)/2, box.Min.Y+(box.Dy()-viewer.LineHeight)/2)
	}
	for _, a := range h.Actions {
		r := h.Screen(a.Rect).Inset(1)
		fill(r, actionColor)
		fillWait(screen, r, actionWait(a), waitColor)
		stroke(r, slotEdgeColor)
		if a.Icon == nil {
			ebitenutil.DebugPrintAt(screen, a.Label, r.Min.X+(r.Dx()-len(a.Label)*viewer.CharWidth)/2, r.Min.Y+(r.Dy()-viewer.LineHeight)/2)
		}
	}

	art := s.artImage(h.Rect.Size().Div(h.Scale))
	if h.Has {
		if err := s.sprites.DrawImage(art, h.Portrait, h.PortraitRect.Min.X, h.PortraitRect.Min.Y, h.Mirrored); err != nil && s.err == nil {
			s.err = err
		}
	}
	centered := func(img content.Image, r image.Rectangle) {
		b := v.Pack.Sprites[img.Sprite]
		if b == nil {
			return
		}
		at := r.Min.Add(image.Pt((r.Dx()-b.FrameSize[0])/2, (r.Dy()-b.FrameSize[1])/2))
		if err := s.sprites.DrawImage(art, img, at.X, at.Y, false); err != nil && s.err == nil {
			s.err = err
		}
	}
	for _, b := range h.Slots {
		if img, ok := v.SlotImage(b.Slot); ok {
			centered(img, b.Rect)
		}
	}
	for _, b := range h.Spells {
		if b.Has && b.Count > 0 {
			centered(b.Icon, b.Rect)
		}
	}
	// An action box shows the action's icon alone; the hand's box shows the object.
	for _, a := range h.Actions {
		if a.Icon != nil {
			centered(*a.Icon, a.Rect)
		}
	}
	if h.Gesture.Has {
		centered(h.Gesture.Icon, h.Gesture.Rect)
	}
	s.presenter.PresentAt(screen, art, h.Scale, h.Rect.Min)
	for _, a := range h.Actions {
		fillWait(screen, h.Screen(a.Rect).Inset(1), actionWait(a), waitShade)
	}
	for _, b := range h.Spells {
		fillWait(screen, h.Screen(b.Rect).Inset(1), b.Wait, waitShade)
		if b.Count > 1 {
			r := h.Screen(b.Rect)
			ebitenutil.DebugPrintAt(screen, fmt.Sprint(min(b.Count, 9)), r.Max.X-viewer.CharWidth-2, r.Max.Y-viewer.LineHeight)
		}
	}
}
