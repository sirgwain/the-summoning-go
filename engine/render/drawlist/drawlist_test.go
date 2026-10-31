package drawlist

import (
	"image"
	"reflect"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

func TestTileBox(t *testing.T) {
	tests := map[content.Cell]image.Point{
		{0, 0}: {0, 0},
		{0, 1}: {32, 16},
		{1, 0}: {-32, 16},
		{1, 1}: {0, 32},
		{2, 5}: {96, 112},
	}
	for c, want := range tests {
		if got := TileBox(c); got != want {
			t.Errorf("TileBox(%v) = %v, want %v", c, got, want)
		}
	}
}

func testPack() (*content.Pack, *content.Level) {
	p := content.NewPack(content.Manifest{})
	p.Floors["f"] = &content.FloorSet{ID: "f", Sprite: "floor", Variants: 2}
	p.Walls["w"] = &content.WallSet{ID: "w", Sprite: "wall", Offset: content.Offset{0, -8},
		Shapes: []*content.WallShape{{Frame: 0}, {Frame: 1}, {Frame: 2}, {Frame: 3}}}
	p.Looks["exit"] = &content.Look{ID: "exit", Layer: content.LayerWall, States: map[string][]content.LookPart{
		"exit": {{Sprite: "exit", Frame: 1, Offset: content.Offset{0, -16}}},
	}}
	p.Looks["plate"] = &content.Look{ID: "plate", Layer: content.LayerFloor, States: map[string][]content.LookPart{
		"up": {{Sprite: "plate", Frame: 0, Offset: content.Offset{-16, 8}}}, "down": {{Sprite: "plate", Frame: 1, Offset: content.Offset{-16, 8}}},
	}}
	p.Looks["lever"] = &content.Look{ID: "lever", Layer: content.LayerStand, States: map[string][]content.LookPart{
		"off": {{Sprite: "lever", Frame: 0}}, "on": {{Sprite: "lever", Frame: 1, Offset: content.Offset{4, 2}}},
	}}
	n := content.NoFrame
	l := &content.Level{
		Size: [2]int{2, 2}, Floors: "f", Walls: "w",
		// (0, 0) floor with a lever; (0, 1) wall 1 with a decoration; (1, 0) exit in wall 3;
		// (1, 1) floor with a plate.
		FloorTypes:  []content.FloorType{{Variant: 1}, {Variant: 0}},
		Floor:       content.Grid{{0, 1}, {1, 1}},
		Wall:        content.Grid{{n, 1}, {3, n}},
		Overlays:    []content.Placed{{Cell: content.Cell{1, 1}, Sprite: "hazard", Frame: 2}},
		Decorations: []content.Placed{{Cell: content.Cell{0, 1}, Sprite: "deco", Frame: 4, Offset: content.Offset{8, 8}}, {Cell: content.Cell{1, 0}, Sprite: "deco", Frame: 5}},
		Props:       []content.Placed{{Cell: content.Cell{0, 0}, Sprite: "door", Frame: 0, Offset: content.Offset{-17, -3}}},
		Features: []*content.Feature{
			{ID: "exit", Kind: content.FeatureExit, Cell: content.Cell{1, 0}, Look: "exit"},
			{ID: "plate", Kind: content.FeaturePlate, Cell: content.Cell{1, 1}, Look: "plate", Down: true},
			{ID: "lever", Kind: content.FeatureLever, Cell: content.Cell{0, 0}, Look: "lever", On: true},
		},
	}
	return p, l
}

func TestBuild(t *testing.T) {
	p, l := testPack()
	got := Build(p, l)
	want := []Cmd{
		// Floors back to front, row by row and left to right, with the cells past the far edges
		// blanked.
		{"floor", 1, 0, 0, Whole},
		{"floor", 0, -32, 16, Whole},
		{"floor", 0, 32, 16, Whole},
		{"", 0, -64, 32, Whole},
		{"floor", 0, 0, 32, Whole},
		{"", 0, 64, 32, Whole},
		{"", 0, -32, 48, Whole},
		{"", 0, 32, 48, Whole},
		{"", 0, 0, 64, Whole},
		// Overlays, then floor looks in their state.
		{"hazard", 2, 0, 32, Whole},
		{"plate", 1, -16, 40, Whole},
		// Row 0: the lever and the door; then row 1: the exit look in place of its wall and
		// decoration, and the wall with its decoration.
		{"lever", 1, 4, 2, Whole},
		{"door", 0, -17, -3, Whole},
		{"exit", 1, -32, 0, Whole},
		{"wall", 1, 32, 8, Whole},
		{"deco", 4, 40, 24, Whole},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%v\nwant\n%v", got, want)
	}
}

func TestBuildFloorHalves(t *testing.T) {
	p, l := testPack()
	l.FloorHalves = content.Grid{{content.HalvesBoth, content.HalfLower}, {0, content.HalvesBoth}}
	got := Build(p, l)[:5]
	want := []Cmd{
		{"floor", 1, 0, 0, Whole},
		// (1, 0): both halves blanked.
		{"", 0, -32, 16, UpperHalf},
		{"", 0, -32, 32, LowerHalf},
		// (0, 1): only the lower half drawn.
		{"", 0, 32, 16, UpperHalf},
		{"floor", 0, 32, 32, LowerHalf},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%v\nwant\n%v", got, want)
	}
}

func TestBounds(t *testing.T) {
	_, l := testPack()
	if got, want := Bounds(l), image.Rect(-32, -32, 64, 64); got != want {
		t.Errorf("Bounds = %v, want %v", got, want)
	}
}
