// Package drawlist turns a level into the ordered list of sprite frames that draw it, in view
// pixels. It does not depend on Ebitengine, so the draw order can be tested without a display.
package drawlist

import (
	"fmt"
	"image"
	"math"
	"slices"
	"sort"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Cmd draws one sprite frame, or the part of it Part names, with its top-left corner at (X, Y),
// in view pixels relative to the tile box of cell (0, 0). A Cmd with no Sprite blanks a tile box,
// or the part of one, at (X, Y) instead. Thing names the thing a Cmd draws, if it draws one, and
// Feature the feature.
// Fixed draws with the palette's own colors rather than its effects: the player's figure and
// what it holds (docs/pack.md, Palettes). Glow says whether, and how, the frame gives light.
// A Cmd with Shadow set draws a shadow instead: an ellipse Shadow pixels wide and half as high,
// centered at (X, Y) (docs/spec/world.md, Light and shadow).
type Cmd struct {
	Sprite  string
	Frame   int
	X, Y    int
	Part    Part
	Thing   string
	Feature string
	Fixed   bool
	Glow    Glow
	Shadow  int
}

// Glow is how a drawn frame gives light (docs/spec/world.md, Light and shadow).
type Glow int

const (
	NoGlow Glow = iota
	// GlowDecoration lights from the pixels in colors the palette animates.
	GlowDecoration
	// GlowSpell lights from those and the bright pixels.
	GlowSpell
	// GlowThing is a thing drawn with the palette's effects, lighting from its animated pixels.
	GlowThing
	// GlowObject is an overlay, a feature's look, a prop or a floor object, lighting from its
	// animated pixels.
	GlowObject
	// GlowFloor is a floor tile, or a half of one, lighting from its animated pixels when it has
	// enough of them.
	GlowFloor
	// GlowFireShield, GlowLightningShield and GlowShields are the player under a fire shield, a
	// lightning shield or both, lighting from the middle of its frame in the shields' colors.
	GlowFireShield
	GlowLightningShield
	GlowShields
)

// Scene is a level as it is to be drawn: its definition's fixed art, and the grids, features,
// objects and things as they are now.
type Scene struct {
	Level       *content.Level
	Floor, Wall content.Grid
	Features    []Feature
	Objects     []Object
	Things      []Thing
	// Visible reports whether a cell is drawn; nil draws every cell.
	Visible func(content.Cell) bool
	// Flights are objects in flight.
	Flights []Flight
	// Taken are the walls drawn as another shape than their own, by cell (docs/spec/world.md,
	// Walls out of sight).
	Taken map[content.Cell]int
}

// Flight is an object in flight over ground point (U, V), or with Sprite set one showing that
// sprite's Frame; Spell says it is a spell, which gives light.
type Flight struct {
	Item   string
	Sprite string
	Frame  int
	U, V   int
	Spell  bool
	// Back moves it on screen back toward where it was, as it glides (GlideBack).
	Back image.Point
}

// flightHeight is how high above its ground point an object in flight is drawn, in pixels.
const flightHeight = 16

// sightDirs are the cells' steps in each direction (docs/pack.md, Conventions).
var sightDirs = [8][2]int{{-1, 0}, {-1, 1}, {0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}}

// halves returns which halves of cell c's floor the scene draws (docs/spec/world.md,
// Visibility). An open or moving door's floor shows whole. Under a wall whose shape has sight,
// each half shows while the neighbor in its check direction counts: the shape is the one the wall
// is taken as, checking as the doors beside it closed now say (sights). Otherwise the level's
// halves apply, and under a closed door only those whose sight neighbor is visible.
func (s Scene) halves(c content.Cell, set *content.WallSet, doors map[content.Cell]string, sights map[content.Cell]*content.DoorSight, noSight map[content.Cell]bool) int {
	l := s.Level
	state, door := doors[c]
	if door && state != content.DoorClosed && state != "sealed" {
		return content.HalvesBoth
	}
	if s.Wall[c[0]][c[1]] != content.NoFrame && set != nil {
		shape := s.Shape(c)
		if ds := sights[c]; ds != nil {
			m := 0
			for i, d := range ds.Doors {
				if doors[d] == content.DoorClosed {
					m |= 1 << i
				}
			}
			if m < len(ds.As) && shape >= 0 && shape < len(ds.As[m]) {
				shape = ds.As[m][shape]
			}
		}
		if sh := set.Shape(shape); sh != nil && len(sh.Sight) == 2 {
			h := 0
			for i, bit := range []int{content.HalfUpper, content.HalfLower} {
				if d := sh.Sight[i]; d >= 0 && d < len(sightDirs) {
					nb := content.Cell{c[0] + sightDirs[d][0], c[1] + sightDirs[d][1]}
					if l.InBounds(nb) && s.Floor[nb[0]][nb[1]] != content.NoFrame && !noSight[nb] && s.shows(nb) {
						h |= bit
					}
				}
			}
			return h
		}
	}
	h := l.Halves(c)
	if s.Visible == nil || (!door || state != content.DoorClosed) && s.Wall[c[0]][c[1]] == content.NoFrame {
		return h
	}
	upper, lower := l.Sight(c)
	for _, half := range []struct{ bit, dir int }{{content.HalfUpper, upper}, {content.HalfLower, lower}} {
		if half.dir == content.NoFrame {
			continue
		}
		d := sightDirs[half.dir]
		if !s.Visible(content.Cell{c[0] + d[0], c[1] + d[1]}) {
			h &^= half.bit
		}
	}
	return h
}

// Shape returns the shape cell c's wall is drawn as: the shape it is taken as, if Taken says,
// else its own.
func (s Scene) Shape(c content.Cell) int {
	if t, ok := s.Taken[c]; ok {
		return t
	}
	return s.Wall[c[0]][c[1]]
}

// shows reports whether the scene draws cell c.
func (s Scene) shows(c content.Cell) bool { return s.Visible == nil || s.Visible(c) }

// Feature is a feature to draw: its look, in a state of the look.
type Feature struct {
	ID    string
	Cell  content.Cell
	Kind  string
	Look  string
	State string
	// Set are a socket's filled places.
	Set []int
}

// Object is an object lying on a cell's floor.
type Object struct {
	Cell content.Cell
	Item string
}

// Thing is a thing standing at a ground point (U, V), in world units.
type Thing struct {
	ID       string
	Template string
	U, V     int
	Facing   int
	// Anim names an animation the thing is stepping through, Phase its step; empty draws it
	// standing still.
	Anim  string
	Phase int
	// Held are the items in the player's hands, right then left, drawn with its frame.
	Held [2]string
	// Looks, if set, are sprites drawn in layers, bottom up, frame for frame in place of the
	// thing's own (a player's figure layers).
	Looks []string
	// Shown is damage shown over the thing, 0 for none.
	Shown int
	// Frame, if set, is the frame drawn in place of any animation; Sprite, if set, the sprite
	// drawn in place of the template's (a look the thing has taken).
	Frame  *int
	Sprite string
	// Back moves it on screen back toward where it was, as it glides (GlideBack).
	Back image.Point
	// FireShield and LightningShield are the player's shields, which give light.
	FireShield, LightningShield bool
}

// point returns the view pixel thing t's ground point is drawn at.
func (t Thing) point() image.Point { return RefPoint(t.U, t.V).Add(t.Back) }

// Label is text drawn over the view: damage shown over a thing, at the view pixel At, the top
// middle of the thing's sprite.
type Label struct {
	At   image.Point
	Text string
}

// LevelScene returns level l as its definition places it.
func LevelScene(p *content.Pack, l *content.Level) Scene {
	s := Scene{Level: l, Floor: l.Floor, Wall: l.Wall}
	for _, f := range l.Features {
		s.Features = append(s.Features, Feature{ID: f.ID, Cell: f.Cell, Kind: f.Kind, Look: f.Look, State: f.LookState(), Set: f.Set})
	}
	for _, o := range l.Objects {
		s.Objects = append(s.Objects, Object{Cell: o.Cell, Item: o.Item})
	}
	for _, t := range l.Things {
		u, v := GroundPoint(t)
		facing := 0
		if tmpl, ok := p.Things[t.Thing]; ok {
			facing = tmpl.Facing
		}
		if t.Facing != nil {
			facing = *t.Facing
		}
		s.Things = append(s.Things, Thing{ID: t.ID, Template: t.Thing, U: u, V: v, Facing: facing})
	}
	return s
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

// Options change how a level is drawn.
type Options struct {
	// Magnify draws floor objects with their inventory image.
	Magnify bool
	// Shadows draws a shadow under each figure (docs/spec/world.md, Light and shadow).
	Shadows bool
}

// shadowScale is a shadow's width per unit of footprint.
const shadowScale = 2

// maxFloorObjects is how many objects a cell's floor shows (docs/spec/items.md, Floor).
const maxFloorObjects = 8

// Build returns the commands that draw scene s, back to front: every floor, then every overlay,
// floor-layer feature look and floor object, then walls (each followed by its decorations),
// props, standing feature looks and things, ordered by screen row (a + b) and within a row by
// their top edge. A wall-layer feature look is drawn in place of its cell's wall and decorations.
// Floors are drawn by the halves the level says show. In a box floor set the other halves are
// blanked, and a cell with no floor, and each cell just past the level's far edges, blanks its
// tile box; a diamond set draws each diamond at its offset and blanks nothing (docs/pack.md).
func Build(p *content.Pack, s Scene, opts Options) []Cmd {
	var cmds []Cmd
	l := s.Level

	floors := p.Floors[l.Floors]
	// A diamond set draws each cell's diamond alone, offset from its tile box, and blanks
	// nothing: only box sets paint corners of their neighbors that need covering.
	var off content.Offset
	if floors.Diamond() {
		off = content.DiamondOffset
	}
	doors := map[content.Cell]string{}
	for _, f := range s.Features {
		if f.Kind == content.FeatureDoor {
			doors[f.Cell] = f.State
		}
	}
	sights := map[content.Cell]*content.DoorSight{}
	for i := range l.DoorSight {
		sights[l.DoorSight[i].Cell] = &l.DoorSight[i]
	}
	noSight := map[content.Cell]bool{}
	for _, c := range l.NoSight {
		noSight[c] = true
	}
	set := p.Walls[l.Walls]
	for _, c := range cellsBackToFront(l.Size[0]+1, l.Size[1]+1) {
		box := TileBox(c).Add(image.Pt(off[0], off[1]))
		if !l.InBounds(c) || s.Floor[c[0]][c[1]] == content.NoFrame || !s.shows(c) {
			if !floors.Diamond() {
				cmds = append(cmds, Cmd{X: box.X, Y: box.Y})
			}
			continue
		}
		v, halves := l.FloorTypes[s.Floor[c[0]][c[1]]].Variant, s.halves(c, set, doors, sights, noSight)
		if halves == content.HalvesBoth {
			cmds = append(cmds, Cmd{Sprite: floors.Sprite, Frame: v, X: box.X, Y: box.Y, Glow: GlowFloor})
			continue
		}
		for _, h := range []struct {
			bit  int
			part Part
			dy   int
		}{{content.HalfUpper, UpperHalf, 0}, {content.HalfLower, LowerHalf, HalfHeight}} {
			cmd := Cmd{X: box.X, Y: box.Y + h.dy, Part: h.part}
			if halves&h.bit != 0 {
				cmd.Sprite, cmd.Frame, cmd.Glow = floors.Sprite, v, GlowFloor
			} else if floors.Diamond() {
				continue
			}
			cmds = append(cmds, cmd)
		}
	}
	for _, o := range l.Overlays {
		if s.shows(o.Cell) {
			cmd := placed(o)
			cmd.Glow = GlowObject
			cmds = append(cmds, cmd)
		}
	}
	looks := map[string][]Feature{}
	wallLooks := map[content.Cell]Feature{}
	for _, f := range s.Features {
		look, ok := p.Looks[f.Look]
		if !ok || !s.shows(f.Cell) {
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
	sunk := map[content.Cell]bool{}
	for _, f := range s.Features {
		if f.Kind == content.FeaturePlate && f.State == "down" {
			sunk[f.Cell] = true
		}
	}
	shown := map[content.Cell]int{}
	for _, o := range s.Objects {
		if !s.shows(o.Cell) {
			continue
		}
		if shown[o.Cell]++; shown[o.Cell] > maxFloorObjects {
			continue
		}
		if cmd, ok := objectCmd(p, o, sunk[o.Cell], opts); ok {
			cmds = append(cmds, cmd)
		}
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
		own := s.Wall[c[0]][c[1]]
		if own == content.NoFrame || !s.shows(c) {
			continue
		}
		box := TileBox(c)
		item := standing{depth: c[0] + c[1], top: box.Y}
		if f, ok := wallLooks[c]; ok {
			item.cmds = featureCmds(p, f)
		} else {
			shape := s.Shape(c)
			item.cmds = append(item.cmds, Cmd{Sprite: set.Sprite, Frame: set.Shape(shape).Frame, X: box.X + set.Offset[0], Y: box.Y + set.Offset[1]})
			if shape == own {
				for _, d := range decorations[c] {
					cmd := placed(d)
					cmd.Glow = GlowDecoration
					item.cmds = append(item.cmds, cmd)
				}
			}
		}
		items = append(items, item)
	}
	for _, f := range looks[content.LayerStand] {
		items = append(items, standing{depth: f.Cell[0] + f.Cell[1], top: TileBox(f.Cell).Y + 1, cmds: featureCmds(p, f)})
	}
	var shadows []Cmd
	for _, t := range s.Things {
		if cmd, cell, ok := thingCmd(p, t); ok && s.shows(cell) {
			drawn := heldCmds(p, t, cmd)
			items = append(items, standing{depth: cell[0] + cell[1], top: cmd.Y, cmds: drawn})
			if sh, ok := shadowCmd(p, t, drawn); ok && opts.Shadows {
				shadows = append(shadows, sh)
			}
		}
	}
	for _, f := range s.Flights {
		cell := content.Cell{floorDiv(f.U, CellUnits), floorDiv(f.V, CellUnits)}
		img := content.Image{Sprite: f.Sprite, Frame: f.Frame}
		if f.Sprite == "" {
			it, ok := p.Items[f.Item]
			if !ok {
				continue
			}
			img = it.Floor
		}
		sprite, ok := p.Sprites[img.Sprite]
		if !ok || !s.shows(cell) || img.Frame < 0 || img.Frame >= sprite.Frames {
			continue
		}
		at := RefPoint(f.U, f.V).Add(f.Back).Sub(image.Pt(sprite.FrameSize[0]/sprite.Scale()/2, sprite.FrameSize[1]/sprite.Scale()/2+flightHeight))
		cmd := Cmd{Sprite: img.Sprite, Frame: img.Frame, X: at.X, Y: at.Y, Glow: GlowObject}
		if f.Spell {
			cmd.Glow = GlowSpell
		}
		items = append(items, standing{depth: cell[0] + cell[1], top: at.Y, cmds: []Cmd{cmd}})
	}
	for _, pr := range l.Props {
		if !s.shows(pr.Cell) {
			continue
		}
		cmd := placed(pr)
		cmd.Glow = GlowObject
		items = append(items, standing{depth: pr.Cell[0] + pr.Cell[1], top: TileBox(pr.Cell).Y + 1, cmds: []Cmd{cmd}})
	}
	cmds = append(cmds, shadows...)
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

// shadowCmd returns the shadow of thing t, drawn by cmds, if it casts one: the player, a living
// creature or a boulder. It lies under the thing's feet: centered on the lowest row its frames
// draw, at the middle of what they draw in their lowest three rows.
func shadowCmd(p *content.Pack, t Thing, cmds []Cmd) (Cmd, bool) {
	tmpl, ok := p.Things[t.Template]
	if !ok || t.Anim == "dead" {
		return Cmd{}, false
	}
	switch tmpl.Kind {
	case content.ThingPlayer, content.ThingCreature, content.ThingBoulder:
	default:
		return Cmd{}, false
	}
	type px struct{ x, y int }
	var drawn []px
	bottom := math.MinInt / 2
	for _, c := range cmds {
		b := p.Sprites[c.Sprite]
		if b == nil || b.Sheet == nil || c.Frame < 0 || c.Frame >= b.Frames {
			continue
		}
		r := b.FrameRect(c.Frame)
		for y := r.Max.Y - 1; y >= r.Min.Y; y-- {
			vy := c.Y + y - r.Min.Y
			if vy < bottom-2 {
				break
			}
			for x := r.Min.X; x < r.Max.X; x++ {
				if i := int(b.Sheet.ColorIndexAt(x, y)); b.Transparent == nil || i != *b.Transparent {
					drawn = append(drawn, px{c.X + x - r.Min.X, vy})
					bottom = max(bottom, vy)
				}
			}
		}
	}
	var sum, n int
	for _, d := range drawn {
		if d.y >= bottom-2 {
			sum, n = sum+d.x, n+1
		}
	}
	if n == 0 {
		return Cmd{}, false
	}
	return Cmd{X: floorDiv(2*sum+n, 2*n), Y: bottom, Shadow: tmpl.Size() * shadowScale}, true
}

// featureCmds draws feature f's look in its state; a socket adds the look of each filled place.
func featureCmds(p *content.Pack, f Feature) []Cmd {
	look := p.Looks[f.Look]
	parts := look.States[f.State]
	if f.Kind == content.FeatureSocket && f.State != "full" {
		for _, i := range f.Set {
			parts = slices.Concat(parts, look.States[content.SocketSetState(i)])
		}
	}
	box := TileBox(f.Cell)
	cmds := make([]Cmd, len(parts))
	for i, pt := range parts {
		cmds[i] = Cmd{Sprite: pt.Sprite, Frame: pt.Frame, X: box.X + pt.Offset[0], Y: box.Y + pt.Offset[1], Feature: f.ID, Glow: GlowObject}
	}
	return cmds
}

// Where a floor object's image goes from its cell's tile box corner, and how much lower on a
// plate that is down (docs/spec/items.md, Objects on the floor).
var (
	objectAt   = image.Pt(8, 13)
	objectSunk = 4
)

// objectCmd draws a floor object at its place in its cell's tile box, lower when sunk.
func objectCmd(p *content.Pack, o Object, sunk bool, opts Options) (Cmd, bool) {
	it, ok := p.Items[o.Item]
	if !ok {
		return Cmd{}, false // a random object is chosen when the level is first entered
	}
	img := it.Floor
	if opts.Magnify {
		img = it.Icon
	}
	if _, ok := p.Sprites[img.Sprite]; !ok {
		return Cmd{}, false
	}
	at := TileBox(o.Cell).Add(objectAt)
	if sunk {
		at.Y += objectSunk
	}
	return Cmd{Sprite: img.Sprite, Frame: img.Frame, X: at.X, Y: at.Y, Glow: GlowObject}, true
}

// CellUnits is the size of a cell in world units.
const CellUnits = 32

// GroundPoint returns where a placed thing stands, in world units (docs/pack.md, Things).
func GroundPoint(t *content.PlacedThing) (u, v int) {
	u, v = CellUnits*t.Cell[0]+CellUnits/2, CellUnits*t.Cell[1]+CellUnits/2
	if t.Offset != nil {
		u, v = u+t.Offset[0], v+t.Offset[1]
	}
	return u, v
}

// RefPoint returns the view pixel of world point (u, v) (docs/spec/world.md, Projection).
func RefPoint(u, v int) image.Point {
	return image.Pt(16+v-u, floorDiv(u, 2)+floorDiv(v, 2))
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// thingCmd draws a placed thing standing still, with its sprite's anchor at its ground point, and
// returns the cell it stands in.
func thingCmd(p *content.Pack, t Thing) (Cmd, content.Cell, bool) {
	tmpl, ok := p.Things[t.Template]
	if !ok {
		return Cmd{}, content.Cell{}, false
	}
	sid := tmpl.Sprite
	if t.Sprite != "" {
		sid = t.Sprite
	}
	sprite, ok := p.Sprites[sid]
	if !ok {
		return Cmd{}, content.Cell{}, false
	}
	frame := 0
	switch {
	case t.Frame != nil:
		if *t.Frame >= 0 && *t.Frame < sprite.Frames {
			frame = *t.Frame
		}
	case t.Sprite != "":
		// A look taken from a creature's sprite (Shape Change) shows a frame by the facing:
		// the game's facing (the table direction's next) halved (docs/spec/spells.md).
		frame = (t.Facing + 1) % 8 / 2 % max(1, sprite.Frames)
	default:
		if a, ok := p.Animation(tmpl, t.Anim); ok && t.Anim != "" && len(a.Frames) > 0 {
			frame = a.Frames[t.Phase%len(a.Frames)]
		} else if a, ok := p.Animation(tmpl, fmt.Sprintf("idle.%d", t.Facing), "idle"); ok && len(a.Frames) > 0 {
			frame = a.Frames[0]
		}
	}
	anchor, _ := sprite.AnchorOf(frame)
	at := t.point().Sub(image.Pt(anchor[0]/sprite.Scale(), anchor[1]/sprite.Scale()))
	cell := content.Cell{floorDiv(t.U, CellUnits), floorDiv(t.V, CellUnits)}
	cmd := Cmd{Sprite: sid, Frame: frame, X: at.X, Y: at.Y, Thing: t.ID, Fixed: tmpl.Kind == content.ThingPlayer && t.Sprite == ""}
	switch {
	case t.FireShield && t.LightningShield:
		cmd.Glow = GlowShields
	case t.FireShield:
		cmd.Glow = GlowFireShield
	case t.LightningShield:
		cmd.Glow = GlowLightningShield
	case !cmd.Fixed:
		cmd.Glow = GlowThing
	}
	return cmd, cell, true
}

// heldCmds draws thing t's frame cmd with the layers of the items in its hands: the layers behind
// first, then the thing, then those in front.
func heldCmds(p *content.Pack, t Thing, cmd Cmd) []Cmd {
	var behind, front []Cmd
	for hand, item := range t.Held {
		it, ok := p.Items[item]
		if !ok {
			continue
		}
		l := it.Held.Hand(hand)
		if l == nil {
			continue
		}
		for _, layer := range []struct {
			sprite string
			into   *[]Cmd
		}{{l.Behind, &behind}, {l.Front, &front}} {
			sprite, ok := p.Sprites[layer.sprite]
			if !ok || cmd.Frame >= sprite.Frames {
				continue
			}
			anchor, _ := sprite.AnchorOf(cmd.Frame)
			at := t.point().Sub(image.Pt(anchor[0]/sprite.Scale(), anchor[1]/sprite.Scale()))
			*layer.into = append(*layer.into, Cmd{Sprite: layer.sprite, Frame: cmd.Frame, X: at.X, Y: at.Y, Thing: t.ID, Fixed: cmd.Fixed})
		}
	}
	return append(append(behind, lookCmds(p, t, cmd)...), front...)
}

// lookCmds draws thing t's frame cmd as its look layers, bottom up, or as cmd itself without
// any. A layer without the frame is left out; the bottom one drawn gives cmd's light.
func lookCmds(p *content.Pack, t Thing, cmd Cmd) []Cmd {
	if len(t.Looks) == 0 {
		return []Cmd{cmd}
	}
	var out []Cmd
	for _, sid := range t.Looks {
		sprite, ok := p.Sprites[sid]
		if !ok || cmd.Frame >= sprite.Frames {
			continue
		}
		anchor, _ := sprite.AnchorOf(cmd.Frame)
		at := t.point().Sub(image.Pt(anchor[0]/sprite.Scale(), anchor[1]/sprite.Scale()))
		out = append(out, Cmd{Sprite: sid, Frame: cmd.Frame, X: at.X, Y: at.Y, Thing: t.ID, Fixed: cmd.Fixed})
	}
	if len(out) > 0 {
		out[0].Glow = cmd.Glow
	}
	return out
}

// Labels returns the damage shown over scene s's things, where it shows.
func Labels(p *content.Pack, s Scene) []Label {
	var out []Label
	for _, t := range s.Things {
		if t.Shown <= 0 {
			continue
		}
		cmd, cell, ok := thingCmd(p, t)
		if !ok || !s.shows(cell) {
			continue
		}
		w := p.Sprites[cmd.Sprite].FrameSize[0] / p.Sprites[cmd.Sprite].Scale()
		out = append(out, Label{At: image.Pt(cmd.X+w/2, cmd.Y), Text: fmt.Sprint(t.Shown)})
	}
	return out
}

func placed(pl content.Placed) Cmd {
	box := TileBox(pl.Cell)
	return Cmd{Sprite: pl.Sprite, Frame: pl.Frame, X: box.X + pl.Offset[0], Y: box.Y + pl.Offset[1]}
}

// cellsBackToFront returns the cells of a rows×columns grid by increasing a + b, then left to
// right (by increasing b − a).
//
// Optimization: this lists the cells straight in that order, one diagonal a + b = d at a time,
// with a falling so b − a rises, rather than sorting them. It runs at every rebuild of the scene,
// and sorting every cell of the level was most of a rebuild's time. The order is only a function
// of the grid's size; if the order ever needs to depend on anything else, or this goes wrong,
// sorting the cells with the comparison above is the plain way back (TestCellsBackToFront keeps
// it as the reference).
func cellsBackToFront(rows, columns int) []content.Cell {
	cells := make([]content.Cell, 0, rows*columns)
	for d := 0; d <= rows+columns-2; d++ {
		for a := min(d, rows-1); a >= max(0, d-columns+1); a-- {
			cells = append(cells, content.Cell{a, d - a})
		}
	}
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
