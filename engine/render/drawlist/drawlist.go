// Package drawlist turns a level into the ordered list of sprite frames that draw it, in view
// pixels. It does not depend on Ebitengine, so the draw order can be tested without a display.
package drawlist

import (
	"image"
	"slices"
	"sort"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Cmd draws one sprite frame, or the part of it Part names, with its top-left corner at (X, Y),
// in view pixels relative to the tile box of cell (0, 0). A Cmd with no Sprite blanks a tile box,
// or the part of one, at (X, Y) instead.
type Cmd struct {
	Sprite string
	Frame  int
	X, Y   int
	Part   Part
}

// Part is the part of a frame or tile box a Cmd draws.
type Part int

const (
	Whole Part = iota
	// UpperHalf and LowerHalf are the top and bottom halves of a floor tile.
	UpperHalf
	LowerHalf
)

// HalfHeight is the height of a floor half.
const HalfHeight = TileSize / 2

// TileSize is the width and height of a tile box.
const TileSize = 32

// TileBox returns the top-left corner of cell c's tile box (docs/spec/world.md).
func TileBox(c content.Cell) image.Point {
	a, b := c[0], c[1]
	return image.Pt(32*(b-a), 16*(a+b))
}

// Build returns the commands that draw level l, back to front: every floor, then every overlay
// and floor-layer feature look, then walls (each followed by its decorations), props and
// standing feature looks, ordered by screen row (a + b) and within a row by their top edge. A
// wall-layer feature look is drawn in place of its cell's wall and decorations. Features are drawn
// in the states the level gives them. Floors are drawn by the halves the level says show, the other
// halves blanked; a cell with no floor, and each cell just past the level's far edges, blanks
// its tile box.
func Build(p *content.Pack, l *content.Level) []Cmd {
	var cmds []Cmd

	floors := p.Floors[l.Floors]
	for _, c := range cellsBackToFront(l.Size[0]+1, l.Size[1]+1) {
		box := TileBox(c)
		if !l.InBounds(c) || l.Floor[c[0]][c[1]] == content.NoFrame {
			cmds = append(cmds, Cmd{X: box.X, Y: box.Y})
			continue
		}
		v, halves := l.FloorTypes[l.Floor[c[0]][c[1]]].Variant, l.Halves(c)
		if halves == content.HalvesBoth {
			cmds = append(cmds, Cmd{Sprite: floors.Sprite, Frame: v, X: box.X, Y: box.Y})
			continue
		}
		for _, h := range []struct {
			bit  int
			part Part
			dy   int
		}{{content.HalfUpper, UpperHalf, 0}, {content.HalfLower, LowerHalf, HalfHeight}} {
			cmd := Cmd{X: box.X, Y: box.Y + h.dy, Part: h.part}
			if halves&h.bit != 0 {
				cmd.Sprite, cmd.Frame = floors.Sprite, v
			}
			cmds = append(cmds, cmd)
		}
	}
	for _, o := range l.Overlays {
		cmds = append(cmds, placed(o))
	}
	looks := map[string][]*content.Feature{}
	wallLooks := map[content.Cell]*content.Feature{}
	for _, f := range l.Features {
		look, ok := p.Looks[f.Look]
		if !ok {
			continue
		}
		looks[look.Layer] = append(looks[look.Layer], f)
		if look.Layer == content.LayerWall {
			wallLooks[f.Cell] = f
		}
	}
	for _, f := range looks[content.LayerFloor] {
		cmds = append(cmds, featureCmds(p, f)...)
	}

	type standing struct {
		depth, top int
		cmds       []Cmd
	}
	var items []standing

	decorations := map[content.Cell][]content.Placed{}
	for _, d := range l.Decorations {
		decorations[d.Cell] = append(decorations[d.Cell], d)
	}
	for _, c := range cellsBackToFront(l.Size[0], l.Size[1]) {
		s := l.Wall[c[0]][c[1]]
		if s == content.NoFrame {
			continue
		}
		box := TileBox(c)
		item := standing{depth: c[0] + c[1], top: box.Y}
		if f, ok := wallLooks[c]; ok {
			item.cmds = featureCmds(p, f)
		} else {
			set := p.Walls[l.Walls]
			item.cmds = append(item.cmds, Cmd{Sprite: set.Sprite, Frame: set.Shape(s).Frame, X: box.X + set.Offset[0], Y: box.Y + set.Offset[1]})
			for _, d := range decorations[c] {
				item.cmds = append(item.cmds, placed(d))
			}
		}
		items = append(items, item)
	}
	for _, f := range looks[content.LayerStand] {
		items = append(items, standing{depth: f.Cell[0] + f.Cell[1], top: TileBox(f.Cell).Y + 1, cmds: featureCmds(p, f)})
	}
	for _, pr := range l.Props {
		cmd := placed(pr)
		items = append(items, standing{depth: pr.Cell[0] + pr.Cell[1], top: TileBox(pr.Cell).Y + 1, cmds: []Cmd{cmd}})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].depth != items[j].depth {
			return items[i].depth < items[j].depth
		}
		return items[i].top < items[j].top
	})
	for _, it := range items {
		cmds = append(cmds, it.cmds...)
	}
	return cmds
}

// featureCmds draws feature f's look in its state; a socket adds the look of each filled place.
func featureCmds(p *content.Pack, f *content.Feature) []Cmd {
	look := p.Looks[f.Look]
	parts := look.States[f.LookState()]
	if f.Kind == content.FeatureSocket && f.LookState() != "full" {
		for _, i := range f.Set {
			parts = slices.Concat(parts, look.States[content.SocketSetState(i)])
		}
	}
	box := TileBox(f.Cell)
	cmds := make([]Cmd, len(parts))
	for i, pt := range parts {
		cmds[i] = Cmd{Sprite: pt.Sprite, Frame: pt.Frame, X: box.X + pt.Offset[0], Y: box.Y + pt.Offset[1]}
	}
	return cmds
}

func placed(pl content.Placed) Cmd {
	box := TileBox(pl.Cell)
	return Cmd{Sprite: pl.Sprite, Frame: pl.Frame, X: box.X + pl.Offset[0], Y: box.Y + pl.Offset[1]}
}

// cellsBackToFront returns the cells of a rows×columns grid by increasing a + b, then left to
// right.
func cellsBackToFront(rows, columns int) []content.Cell {
	cells := make([]content.Cell, 0, rows*columns)
	for a := range rows {
		for b := range columns {
			cells = append(cells, content.Cell{a, b})
		}
	}
	sort.SliceStable(cells, func(i, j int) bool {
		di, dj := cells[i][0]+cells[i][1], cells[j][0]+cells[j][1]
		if di != dj {
			return di < dj
		}
		return cells[i][1]-cells[i][0] < cells[j][1]-cells[j][0]
	})
	return cells
}

// Bounds returns the view-pixel rectangle the level's cells cover, including walls rising above
// their tile boxes.
func Bounds(l *content.Level) image.Rectangle {
	rows, cols := l.Size[0], l.Size[1]
	if rows == 0 || cols == 0 {
		return image.Rectangle{}
	}
	const rise = 32
	return image.Rect(
		TileBox(content.Cell{rows - 1, 0}).X,
		-rise,
		TileBox(content.Cell{0, cols - 1}).X+TileSize,
		TileBox(content.Cell{rows - 1, cols - 1}).Y+TileSize,
	)
}
