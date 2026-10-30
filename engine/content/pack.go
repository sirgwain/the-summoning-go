// Package content defines the engine's content pack: its records, how they are stored on disk
// and how a pack is loaded and checked. The format is in docs/pack.md.
package content

import (
	"fmt"
	"image"
	"image/color"
)

// Format is the pack format version this package reads and writes.
const Format = 0

// Pack is a loaded content pack. Records are keyed by id.
type Pack struct {
	Manifest Manifest
	Palettes map[string]*Palette
	Sprites  map[string]*SpriteBank
	Floors   map[string]*FloorSet
	Walls    map[string]*WallSet
	Levels   map[string]*Level
}

// NewPack returns an empty pack with the given manifest.
func NewPack(m Manifest) *Pack {
	return &Pack{
		Manifest: m,
		Palettes: map[string]*Palette{},
		Sprites:  map[string]*SpriteBank{},
		Floors:   map[string]*FloorSet{},
		Walls:    map[string]*WallSet{},
		Levels:   map[string]*Level{},
	}
}

// Manifest is pack.json.
type Manifest struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Format  int      `json:"format"`
	Depends []string `json:"depends"`
	// Palette is the palette gameplay draws with.
	Palette string `json:"palette"`
	Start   Start  `json:"start"`
}

// Start is where a new game begins.
type Start struct {
	Level string `json:"level"`
	Cell  Cell   `json:"cell"`
}

// Cell is a cell address (x, y): row x, column y.
type Cell [2]int

// Offset is a pixel offset (dx, dy).
type Offset [2]int

// Palette is a 256-color palette.
type Palette struct {
	ID     string
	Colors [256]color.RGBA
}

// SpriteBank is a sheet of same-sized frames in an indexed image, laid out left to right, top
// to bottom.
type SpriteBank struct {
	ID        string `json:"id"`
	FrameSize Offset `json:"frame_size"`
	Columns   int    `json:"columns"`
	Frames    int    `json:"frames"`
	// Transparent is the palette index drawn as transparent, or nil for an opaque bank.
	Transparent *int `json:"transparent"`
	// Sheet holds the frames' palette indexes.
	Sheet *image.Paletted `json:"-"`
}

// FrameRect returns frame i's rectangle on the sheet.
func (b *SpriteBank) FrameRect(i int) image.Rectangle {
	w, h := b.FrameSize[0], b.FrameSize[1]
	x, y := i%b.Columns*w, i/b.Columns*h
	return image.Rect(x, y, x+w, y+h)
}

// FloorSet is a level's floor variants: variant k is frame k of its sprite bank, covering the
// cell's tile box.
type FloorSet struct {
	ID       string `json:"id"`
	Sprite   string `json:"sprite"`
	Variants int    `json:"variants"`
}

// WallSet has one composed image per wall shape.
type WallSet struct {
	ID     string `json:"id"`
	Sprite string `json:"sprite"`
	// Offset places each image relative to the cell's tile box.
	Offset Offset `json:"offset"`
	// Shapes is the frame for each shape, or NoFrame for a shape the set does not have.
	Shapes []int `json:"shapes"`
}

// NoFrame marks a missing wall shape and an empty grid cell.
const NoFrame = -1

// Floor halves, as bits of Level.FloorHalves.
const (
	HalfUpper  = 1
	HalfLower  = 2
	HalvesBoth = HalfUpper | HalfLower
)

// Halves returns which halves of cell c's floor are drawn.
func (l *Level) Halves(c Cell) int {
	if l.FloorHalves == nil {
		return HalvesBoth
	}
	return l.FloorHalves[c[0]][c[1]]
}

// Level is the drawable part of a level.
type Level struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Size is the number of rows and columns.
	Size   [2]int `json:"size"`
	Floors string `json:"floors"`
	Walls  string `json:"walls"`
	// ExitWalls draws the cells listed in Exits, by their shape in Wall.
	ExitWalls string `json:"exit_walls,omitempty"`
	// Floor is each cell's floor variant; Wall each cell's wall shape. NoFrame for none.
	Floor Grid `json:"floor"`
	Wall  Grid `json:"wall"`
	// FloorHalves says which halves of each cell's floor are drawn (HalfUpper, HalfLower); the
	// others are blanked. Nil draws every floor whole.
	FloorHalves Grid   `json:"floor_halves,omitempty"`
	Exits       []Cell `json:"exits,omitempty"`
	// Overlays are drawn after all floors and before anything standing.
	Overlays []Placed `json:"overlays,omitempty"`
	// Decorations are drawn on their cell's wall, right after it.
	Decorations []Placed `json:"decorations,omitempty"`
	// Props stand in their cell and are drawn in depth order with the walls.
	Props []Placed `json:"props,omitempty"`
}

// Placed is a sprite frame placed relative to a cell's tile box.
type Placed struct {
	Cell   Cell   `json:"cell"`
	Sprite string `json:"sprite"`
	Frame  int    `json:"frame"`
	Offset Offset `json:"offset"`
}

// InBounds reports whether c is inside the level.
func (l *Level) InBounds(c Cell) bool {
	return c[0] >= 0 && c[1] >= 0 && c[0] < l.Size[0] && c[1] < l.Size[1]
}

// NewGrid returns a rows×columns grid filled with NoFrame.
func NewGrid(rows, columns int) Grid {
	g := make(Grid, rows)
	for x := range g {
		g[x] = make([]int, columns)
		for y := range g[x] {
			g[x][y] = NoFrame
		}
	}
	return g
}

func (p *Pack) String() string {
	return fmt.Sprintf("pack %s: %d palettes, %d sprite banks, %d floor sets, %d wall sets, %d levels",
		p.Manifest.ID, len(p.Palettes), len(p.Sprites), len(p.Floors), len(p.Walls), len(p.Levels))
}
