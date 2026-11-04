package viewer

import (
	"image"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
)

// The status icons: a row in the screen's top-right corner showing the player's statuses that
// hold (docs/spec/scenes.md, The game screen).

// StatusBox is a status icon on the screen, at Scale screen pixels per icon pixel.
type StatusBox struct {
	Rect   image.Rectangle
	Scale  int
	Image  content.Image
	Status string
	// Name names the status under the pointer.
	Name string
}

// StatusIcons returns the status icons to draw, from the right, while the hotbar shows.
func (v *Viewer) StatusIcons() []StatusBox {
	h := v.Hotbar()
	if h == nil || v.Pack.Inventory == nil || v.World.Player() == nil {
		return nil
	}
	p := v.World.Player()
	right, top := v.Screen.X-Margin, Margin
	if r := v.PanelRect(); !r.Empty() {
		top = r.Max.Y + Margin
	}
	var out []StatusBox
	for _, si := range v.Pack.Inventory.StatusIcons {
		b := v.Pack.Sprites[si.Image.Sprite]
		if b == nil || !p.HasStatus(si.Status) {
			continue
		}
		size := image.Pt(b.FrameSize[0], b.FrameSize[1]).Mul(h.Scale)
		r := image.Rect(right-size.X, top, right, top+size.Y)
		name := content.StatusNames[si.Status]
		if si.Name != "" {
			name = inspect.Text(v.Pack, si.Name)
		}
		out = append(out, StatusBox{Rect: r, Scale: h.Scale, Image: si.Image, Status: si.Status, Name: name})
		right = r.Min.X - h.Scale
	}
	return out
}

// statusUnder names the status icon under cursor, "" for none.
func (v *Viewer) statusUnder(cursor image.Point) string {
	for _, b := range v.StatusIcons() {
		if cursor.In(b.Rect) {
			return b.Name
		}
	}
	return ""
}
