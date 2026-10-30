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
	p.Walls["w"] = &content.WallSet{ID: "w", Sprite: "wall", Offset: content.Offset{0, -8}, Shapes: []int{0, 1, 2, 3}}
	p.Walls["x"] = &content.WallSet{ID: "x", Sprite: "exit", Offset: content.Offset{0, -16}, Shapes: []int{-1, 0, -1, 1}}
	n := content.NoFrame
	l := &content.Level{
		Size: [2]int{2, 2}, Floors: "f", Walls: "w", ExitWalls: "x",
		// (0, 0) floor; (0, 1) wall 1 with a decoration; (1, 0) exit in wall 3; (1, 1) floor.
		Floor:       content.Grid{{1, 0}, {0, 0}},
		Wall:        content.Grid{{n, 1}, {3, n}},
		Exits:       []content.Cell{{1, 0}},
		Overlays:    []content.Placed{{Cell: content.Cell{1, 1}, Sprite: "plate", Frame: 2, Offset: content.Offset{-16, 8}}},
		Decorations: []content.Placed{{Cell: content.Cell{0, 1}, Sprite: "deco", Frame: 4, Offset: content.Offset{8, 8}}, {Cell: content.Cell{1, 0}, Sprite: "deco", Frame: 5}},
		Props:       []content.Placed{{Cell: content.Cell{0, 0}, Sprite: "door", Frame: 0, Offset: content.Offset{-17, -3}}},
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
		{"plate", 2, -16, 40, Whole},
		// The door in row 0; then row 1: the exit wall (no decoration) and the wall with its
		// decoration.
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
