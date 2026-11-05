// Package content defines the engine's content pack: its records, how they are stored on disk
// and how a pack is loaded and checked. The format is in docs/pack.md.
package content

import (
	"fmt"
	"image"
)

// Format is the pack format version this package reads and writes. It rises whenever a change
// means a pack written before must be written again (docs/pack.md): 2 added collision boxes, 3
// damage kinds as data and the shot of a fire action. A skin (ui.json) needs no new format: a
// pack without one looks as before.
const Format = 3

// Pack is a loaded content pack. Records are keyed by id.
type Pack struct {
	Manifest  Manifest
	Palettes  map[string]*Palette
	Sprites   map[string]*SpriteBank
	Floors    map[string]*FloorSet
	Walls     map[string]*WallSet
	Looks     map[string]*Look
	Items     map[string]*Item
	Things    map[string]*Thing
	Portraits map[string]*Portrait
	Texts     map[string]*Text
	// Messages is messages.json, or nil when the pack has none.
	Messages *Messages
	Levels   map[string]*Level
	Spells   map[string]*Spell
	// Magic is magic.json and Keywords keywords.json, or nil when the pack has none.
	Magic         *Magic
	Keywords      *Keywords
	Pointers      *Pointers
	Inventory     *Inventory
	Creation      *Creation
	Scenes        map[string]*Scene
	Flipbooks     map[string]*Flipbook
	Conversations map[string]*Conversation
	// Sounds are the music tracks and sound effects.
	Sounds map[string]*Sound
	// Cues is audio/cues.json, or nil when the pack has none.
	Cues *Cues
	// Effects is effects.json, or nil when the pack has none.
	Effects *EffectCues
	// Combat is combat.json, or nil when the pack has none (CombatRules).
	Combat *Combat
	// Skin is ui.json, or nil when the pack has none.
	Skin *Skin
}

// NewPack returns an empty pack with the given manifest.
func NewPack(m Manifest) *Pack {
	return &Pack{
		Manifest:  m,
		Palettes:  map[string]*Palette{},
		Sprites:   map[string]*SpriteBank{},
		Floors:    map[string]*FloorSet{},
		Walls:     map[string]*WallSet{},
		Looks:     map[string]*Look{},
		Items:     map[string]*Item{},
		Things:    map[string]*Thing{},
		Portraits: map[string]*Portrait{},
		Texts:     map[string]*Text{},
		Levels:    map[string]*Level{},
		Spells:    map[string]*Spell{},
		Scenes:    map[string]*Scene{},
		Flipbooks: map[string]*Flipbook{},

		Conversations: map[string]*Conversation{},
		Sounds:        map[string]*Sound{},
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
	// Player is the thing template of kind ThingPlayer the player is made from; empty for none.
	Player string `json:"player,omitempty"`
	// Rules names the combat rule set; empty means RulesSummoning.
	Rules string `json:"rules,omitempty"`
	// Counters names the counters the engine itself uses.
	Counters *Counters `json:"counters,omitempty"`
	// Menu is the thing template whose conversation the menu key opens: the game's disk menu.
	Menu string `json:"menu,omitempty"`
}

// RulesSummoning is the Summoning's combat rule set, the only one.
const RulesSummoning = "summoning"

// Start is where a new game begins.
type Start struct {
	Level string `json:"level"`
	Cell  Cell   `json:"cell"`
	// Music is the music track a new game starts, if any.
	Music string `json:"music,omitempty"`
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
	// Diagonal walls fill the half of their cell at this corner of its diamond, one of the
	// Corner constants; empty for a wall that fills its cell.
	Diagonal string `json:"diagonal,omitempty"`
	// Face walls can carry a decoration or a wall feature.
	Face bool `json:"face,omitempty"`
	// Sight, if set, gives the directions of the neighbors whose sight shows the upper and lower
	// halves of the floor under the wall, NoFrame for none.
	Sight []int `json:"sight,omitempty"`
	// Unseen are (direction, shape) pairs tried in order: at the first whose neighbor is inside
	// the level and not visible, a visible wall is taken as that shape instead.
	Unseen [][2]int `json:"unseen,omitempty"`
}

// Taken returns the shape a visible wall of shape s at cell c is taken as, visible reporting
// which cells are visible and in whether a cell is inside the level (docs/spec/world.md, Walls
// out of sight). dirs are the cell steps of the directions.
func (w *WallSet) Taken(s int, c Cell, dirs [8][2]int, in, visible func(Cell) bool) int {
	for range len(w.Shapes) + 1 {
		shape := w.Shape(s)
		if shape == nil {
			return s
		}
		next := s
		for _, u := range shape.Unseen {
			if u[0] < 0 || u[0] >= len(dirs) {
				continue
			}
			nb := Cell{c[0] + dirs[u[0]][0], c[1] + dirs[u[0]][1]}
			if in(nb) && !visible(nb) {
				next = u[1]
				break
			}
		}
		if next == s {
			return s
		}
		s = next
	}
	return s
}

// Corners of a cell's diamond, on screen, that a diagonal wall fills.
const (
	CornerTop    = "top"    // toward −u and −v
	CornerRight  = "right"  // toward −u and +v
	CornerBottom = "bottom" // toward +u and +v
	CornerLeft   = "left"   // toward +u and −v
)

// CornerSigns returns the signs along u and v of the corner a diagonal wall fills, and false for
// a name that is not a corner.
func CornerSigns(corner string) (su, sv int, ok bool) {
	switch corner {
	case CornerTop:
		return -1, -1, true
	case CornerRight:
		return -1, 1, true
	case CornerBottom:
		return 1, 1, true
	case CornerLeft:
		return 1, -1, true
	}
	return 0, 0, false
}

// NoFrame marks an empty grid cell.
const NoFrame = -1

func (p *Pack) String() string {
	return fmt.Sprintf("pack %s: %d palettes, %d sprite banks, %d floor sets, %d wall sets, %d looks, %d items, %d things, %d levels",
		p.Manifest.ID, len(p.Palettes), len(p.Sprites), len(p.Floors), len(p.Walls), len(p.Looks), len(p.Items), len(p.Things), len(p.Levels))
}
