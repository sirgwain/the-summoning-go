package viewer

import (
	"image"
	"maps"
	"slices"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

func TestStatusIcons(t *testing.T) {
	v := newViewer(t)
	sprite := slices.Sorted(maps.Keys(v.Pack.Sprites))[0]
	if v.Pack.Inventory == nil {
		v.Pack.Inventory = &content.Inventory{ID: "inventory"}
	}
	img := content.Image{Sprite: sprite}
	v.Pack.Inventory.StatusIcons = []content.StatusIcon{
		{Status: content.StatusConfused, Image: img},
		{Status: content.StatusPoisoned, Image: img},
		{Status: content.StatusFireShield, Image: img},
	}
	v.SetPlay(true)
	v.HideDebug = true
	if got := v.StatusIcons(); len(got) != 0 {
		t.Fatalf("%d icons with no status", len(got))
	}
	p := v.World.Player()
	p.Confusion = 40
	p.Timed[sim.TimedFireShield] = 2
	got := v.StatusIcons()
	if len(got) != 2 || got[0].Status != content.StatusConfused || got[1].Status != content.StatusFireShield {
		t.Fatalf("icons %+v", got)
	}
	h := v.Hotbar()
	size := v.Pack.Sprites[sprite].FrameSize
	first, second := got[0].Rect, got[1].Rect
	if first.Max != image.Pt(v.Screen.X-Margin, Margin+size[1]*h.Scale) || first.Dx() != size[0]*h.Scale {
		t.Errorf("the first icon at %v", first)
	}
	if second.Max.X != first.Min.X-h.Scale || second.Min.Y != first.Min.Y {
		t.Errorf("the second icon at %v, the first at %v", second, first)
	}
	if lines := v.HoverLines(first.Min.Add(image.Pt(1, 1))); len(lines) != 1 || lines[0] != "Confused" {
		t.Errorf("hovering the first icon: %v", lines)
	}
}
