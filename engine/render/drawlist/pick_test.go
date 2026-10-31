package drawlist

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

func TestCellAt(t *testing.T) {
	for a := -3; a <= 3; a++ {
		for b := -3; b <= 3; b++ {
			c := content.Cell{a, b}
			d := Diamond(c)
			if d[0] != TileBox(c).Add(image.Pt(16, 0)) || d[2] != TileBox(c).Add(image.Pt(16, 32)) {
				t.Errorf("diamond of %v: %v", c, d)
			}
			// Just inside each corner, and the middle, which is the tile box's center.
			for _, pt := range []image.Point{d[0].Add(image.Pt(0, 1)), d[1].Add(image.Pt(-2, 0)), d[2].Add(image.Pt(0, -1)), d[3].Add(image.Pt(2, 0)), TileBox(c).Add(image.Pt(16, 16))} {
				if got := CellAt(pt); got != c {
					t.Errorf("CellAt(%v) = %v, want %v", pt, got, c)
				}
			}
			// A thing's ground point is in its cell.
			if got := CellAt(RefPoint(CellUnits*a+10, CellUnits*b+28)); got != c {
				t.Errorf("ground point in %v picks %v", c, got)
			}
		}
	}
}

func TestThingAt(t *testing.T) {
	p, l := testPack()
	// The guard's sprite: frame 2 transparent except its middle.
	transparent := 0
	sheet := image.NewPaletted(image.Rect(0, 0, 96, 32), color.Palette{color.Black, color.White})
	for y := 8; y < 24; y++ {
		for x := 64 + 8; x < 64+24; x++ {
			sheet.SetColorIndex(x, y, 1)
		}
	}
	npc := p.Sprites["npc"]
	npc.Columns, npc.Transparent, npc.Sheet = 3, &transparent, sheet
	cmds := Build(p, LevelScene(p, l), Options{})
	// The guard is drawn at (16, 32).
	if id, _, ok := ThingAt(p, cmds, image.Pt(16+16, 32+16)); !ok || id != "guard" {
		t.Errorf("middle of the guard: %q %v", id, ok)
	}
	if _, _, ok := ThingAt(p, cmds, image.Pt(16+1, 32+1)); ok {
		t.Error("a transparent pixel picked the guard")
	}
}

func TestSimScene(t *testing.T) {
	p, l := testPack()
	l.ID = "level.00"
	l.Features[2].Armed = true // the lever, so using it turns it off
	p.Levels[l.ID] = l
	p.Manifest.Start.Level = l.ID
	w, err := sim.New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := Build(p, SimScene(w.Current()), Options{}), Build(p, LevelScene(p, l), Options{}); !reflect.DeepEqual(got, want) {
		t.Fatalf("a new world draws differently from its level:\n%v\n%v", got, want)
	}
	w.Update([]input.Command{input.Use{Cell: content.Cell{0, 0}}, input.Remove{Cell: content.Cell{0, 0}}})
	for _, c := range Build(p, SimScene(w.Current()), Options{}) {
		if c.Sprite == "tiny" {
			t.Error("a taken object is still drawn")
		}
		if c.Sprite == "lever" && c.Frame != 0 {
			t.Error("the lever is drawn in its old state")
		}
	}
}
