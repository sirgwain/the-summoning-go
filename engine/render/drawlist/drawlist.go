// Package drawlist turns a level into the ordered list of sprite frames that draw it, in view
// pixels. It does not depend on Ebitengine, so the draw order can be tested without a display.
package drawlist

import (
	"image"
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

// Build returns the commands that draw level l, back to front: every floor, then every overlay,
// then walls (each followed by its decorations) and props, ordered by screen row (a + b) and
// within a row by their top edge. Floors are drawn by the halves the level says show, the other
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
		v, halves := l.Floor[c[0]][c[1]], l.Halves(c)
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

	type standing struct {
		depth, top int
		cmds       []Cmd
	}
	var items []standing

	exits := map[content.Cell]bool{}
	for _, e := range l.Exits {
		exits[e] = true
	}
	decorations := map[content.Cell][]content.Placed{}
	for _, d := range l.Decorations {
		decorations[d.Cell] = append(decorations[d.Cell], d)
	}
	for _, c := range cellsBackToFront(l.Size[0], l.Size[1]) {
		s := l.Wall[c[0]][c[1]]
		if s == content.NoFrame {
			continue
		}
		set := p.Walls[l.Walls]
		if exits[c] {
			set = p.Walls[l.ExitWalls]
		}
		box := TileBox(c)
		item := standing{depth: c[0] + c[1], top: box.Y}
		item.cmds = append(item.cmds, Cmd{Sprite: set.Sprite, Frame: set.Shapes[s], X: box.X + set.Offset[0], Y: box.Y + set.Offset[1]})
		if !exits[c] {
			for _, d := range decorations[c] {
				item.cmds = append(item.cmds, placed(d))
			}
		}
		items = append(items, item)
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
