// Package content defines the engine's content pack: its records, how they are stored on disk
// and how a pack is loaded and checked. The format is in docs/pack.md.
package content

import (
	"fmt"
	"image"
)

// Format is the pack format version this package reads and writes.
const Format = 1

// Pack is a loaded content pack. Records are keyed by id.
type Pack struct {
	Manifest Manifest
	Palettes map[string]*Palette
	Sprites  map[string]*SpriteBank
	Floors   map[string]*FloorSet
	Walls    map[string]*WallSet
	Looks    map[string]*Look
	Items    map[string]*Item
	Things   map[string]*Thing
	Texts    map[string]*Text
	// Messages is messages.json, or nil when the pack has none.
	Messages *Messages
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
		Looks:    map[string]*Look{},
		Items:    map[string]*Item{},
		Things:   map[string]*Thing{},
		Texts:    map[string]*Text{},
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

// SpriteBank is a sheet of same-sized frames in an indexed image, laid out left to right, top
// to bottom.
type SpriteBank struct {
	ID        string `json:"id"`
	FrameSize Offset `json:"frame_size"`
	Columns   int    `json:"columns"`
	Frames    int    `json:"frames"`
	// Transparent is the palette index drawn as transparent, or nil for an opaque bank.
	Transparent *int `json:"transparent"`
	// Anchor is the pixel of every frame drawn at a thing's ground point; Anchors gives one per
	// frame instead. Sprites drawn only as tiles and looks have neither.
	Anchor     *Offset              `json:"anchor,omitempty"`
	Anchors    []Offset             `json:"anchors,omitempty"`
	Animations map[string]Animation `json:"animations,omitempty"`
	// Sheet holds the frames' palette indexes.
	Sheet *image.Paletted `json:"-"`
}

// AnchorOf returns frame i's anchor, and false when the sprite has none.
func (b *SpriteBank) AnchorOf(i int) (Offset, bool) {
	switch {
	case b.Anchors != nil && i >= 0 && i < len(b.Anchors):
		return b.Anchors[i], true
	case b.Anchor != nil:
		return *b.Anchor, true
	}
	return Offset{}, false
}

// Animation modes.
const (
	AnimLoop    = "loop"
	AnimOnce    = "once"
	AnimRandom  = "random"
	AnimFlicker = "flicker"
)

// Animation is a named sequence of a sprite's frames.
type Animation struct {
	Frames []int `json:"frames"`
	// MS is how long each frame shows; zero for a single still frame.
	MS Duration `json:"ms,omitzero"`
	// Mode is one of the Anim constants; empty means AnimLoop.
	Mode string `json:"mode,omitempty"`
}

// FrameRect returns frame i's rectangle on the sheet.
func (b *SpriteBank) FrameRect(i int) image.Rectangle {
	w, h := b.FrameSize[0], b.FrameSize[1]
	x, y := i%b.Columns*w, i/b.Columns*h
	return image.Rect(x, y, x+w, y+h)
}

// FloorSet is a level's floor variants: variant k is frame k of its sprite bank.
type FloorSet struct {
	ID       string `json:"id"`
	Sprite   string `json:"sprite"`
	Variants int    `json:"variants"`
	// Shape is what each image covers, FloorBox or FloorDiamond; empty means FloorBox.
	Shape string `json:"shape,omitempty"`
}

// Floor set shapes (docs/pack.md, Floor sets).
const (
	// FloorBox images fill a cell's 32×32 tile box, with corners of its neighbors' diamonds.
	FloorBox = "box"
	// FloorDiamond images are the cell's 64×32 diamond alone, drawn at DiamondOffset from the
	// tile box.
	FloorDiamond = "diamond"
)

// DiamondOffset places a diamond floor image relative to its cell's tile box.
var DiamondOffset = Offset{-16, 0}

// Diamond reports whether the set's images are diamonds.
func (f *FloorSet) Diamond() bool { return f.Shape == FloorDiamond }

// FrameSize returns the size of the set's images.
func (f *FloorSet) FrameSize() Offset {
	if f.Diamond() {
		return Offset{64, 32}
	}
	return Offset{32, 32}
}

// WallSet has one composed image per wall shape.
type WallSet struct {
	ID     string `json:"id"`
	Sprite string `json:"sprite"`
	// Offset places each image relative to the cell's tile box.
	Offset Offset `json:"offset"`
	// Shapes describes each shape, nil for a shape the set does not have.
	Shapes []*WallShape `json:"shapes"`
}

// Shape returns shape s, or nil when the set has no such shape.
func (w *WallSet) Shape(s int) *WallShape {
	if s < 0 || s >= len(w.Shapes) {
		return nil
	}
	return w.Shapes[s]
}

// WallShape is one shape of a wall set.
type WallShape struct {
	Frame int `json:"frame"`
	// Diagonal walls fill half their cell.
	Diagonal bool `json:"diagonal,omitempty"`
	// Face walls can carry a decoration or a wall feature.
	Face bool `json:"face,omitempty"`
}

// NoFrame marks an empty grid cell.
const NoFrame = -1

func (p *Pack) String() string {
	return fmt.Sprintf("pack %s: %d palettes, %d sprite banks, %d floor sets, %d wall sets, %d looks, %d items, %d things, %d levels",
		p.Manifest.ID, len(p.Palettes), len(p.Sprites), len(p.Floors), len(p.Walls), len(p.Looks), len(p.Items), len(p.Things), len(p.Levels))
}
