package devpack

import (
	"fmt"
	"image"
	"image/color"
	"maps"
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Art dresses the dev pack in another pack's art: the dev pack keeps its levels, rules, records
// and ids, and draws with the art pack's palette, tiles, looks, figures, icons and flipbooks.
// Records are named by their ids in the art pack.
type Art struct {
	// Floors and Walls are the art pack's floor and wall sets.
	Floors, Walls string
	// FloorVariants gives the art floor variant for each of the dev pack's.
	FloorVariants map[int]int
	// WallShapes gives a wall's shape by which of its neighbors are walls: bit 0 the cell at
	// x−1, bit 1 at x+1, bit 2 at y−1, bit 3 at y+1.
	WallShapes [16]int
	// Doors are the door looks for a door between walls along x and along y; SealedDoor is a
	// look whose "sealed" state a sealed door along x takes. One along y takes it mirrored, each
	// part's x offset becoming DoorMirror − x − its width.
	Doors      [2]string
	SealedDoor string
	DoorMirror int
	// WallChecks are the check directions, upper then lower (−1 for none), whose cells must be
	// in the level for each half of the floor under a wall of each shape to show, and while
	// playing visible; DoorChecks those under a closed door between walls along x and along y. A
	// wall checks as another shape by ChecksBelowDoor when a door between walls along x is at
	// x+1, else by ChecksLeftOfDoor when one along y is at y−1 and then ChecksRightOfDoor when one
	// is at y+1 (docs/spec/world.md, Floor halves).
	WallChecks                                           [][2]int
	DoorChecks                                           [2][2]int
	ChecksBelowDoor, ChecksLeftOfDoor, ChecksRightOfDoor []int
	// WallExits are the looks of an exit in a wall, by the wall's shape.
	WallExits map[int]string
	// Features are the looks of the other features, by kind; FeatureExit is an exit on the floor.
	Features map[string]string
	// Things, Items, Portraits and Flipbooks give the art pack's record for a dev pack record,
	// by id. Spells take the art of the art pack's spell of the same id, and the flipbooks the
	// dev pack names by cue take the art pack's for the cue.
	Things, Items, Portraits, Flipbooks map[string]string
}

// Wall neighbor bits for Art.WallShapes.
const (
	wallUBefore = 1 << iota
	wallUAfter
	wallVBefore
	wallVAfter
)

// Dress dresses dev, a dev pack from Build, in art's pack, the art pack's records it names
// copied in. It returns what it could not dress: a record the art pack lacks leaves the dev
// pack's own, its colors moved to the nearest of the art palette's.
func Dress(dev, art *content.Pack, a Art) []string {
	d := &dresser{dev: dev, art: art, a: a, ids: map[string]string{}, copied: map[string]bool{}}
	d.palette()
	d.levels()
	d.things()
	d.items()
	d.spells()
	d.portraits()
	d.flipbooks()
	d.screens()
	d.recolor()
	return d.missing
}

type dresser struct {
	dev, art *content.Pack
	a        Art
	// ids maps an art record's id to its id in the dev pack, which differs where the dev pack
	// has one of its own by that id; copied are the sprite banks copied in.
	ids     map[string]string
	copied  map[string]bool
	missing []string
}

func (d *dresser) miss(format string, args ...any) {
	d.missing = append(d.missing, fmt.Sprintf(format, args...))
}

// place returns the id an art record of kind takes in the dev pack, and whether it is new.
func (d *dresser) place(kind, id string, taken func(string) bool) (string, bool) {
	key := kind + ":" + id
	if got, ok := d.ids[key]; ok {
		return got, false
	}
	to := id
	for taken(to) {
		to += ".art"
	}
	d.ids[key] = to
	return to, true
}

// sprite copies art sprite bank id in and returns its id in the dev pack; "" stays "".
func (d *dresser) sprite(id string) string {
	if id == "" {
		return ""
	}
	b := d.art.Sprites[id]
	if b == nil {
		d.miss("sprite %s", id)
		return id
	}
	to, fresh := d.place("sprite", id, func(s string) bool { return d.dev.Sprites[s] != nil })
	if fresh {
		c := *b
		c.ID = to
		d.dev.Sprites[to] = &c
		d.copied[to] = true
	}
	return to
}

func (d *dresser) image(im content.Image) content.Image {
	im.Sprite = d.sprite(im.Sprite)
	return im
}

func (d *dresser) imagePtr(im *content.Image) *content.Image {
	if im == nil {
		return nil
	}
	c := d.image(*im)
	return &c
}

// look copies art look id in and returns its id in the dev pack, or "" when the art pack has
// none.
func (d *dresser) look(id string) string {
	l := d.art.Looks[id]
	if l == nil {
		d.miss("look %s", id)
		return ""
	}
	to, fresh := d.place("look", id, func(s string) bool { return d.dev.Looks[s] != nil })
	if fresh {
		c := &content.Look{ID: to, Layer: l.Layer, States: map[string][]content.LookPart{}}
		for s, parts := range l.States {
			c.States[s] = d.parts(parts)
		}
		d.dev.Looks[to] = c
	}
	return to
}

func (d *dresser) parts(parts []content.LookPart) []content.LookPart {
	out := make([]content.LookPart, len(parts))
	for i, p := range parts {
		p.Sprite = d.sprite(p.Sprite)
		out[i] = p
	}
	return out
}

// palette takes the art pack's palette in place of the dev pack's.
func (d *dresser) palette() {
	pal := d.art.Palettes[d.art.Manifest.Palette]
	if pal == nil {
		d.miss("palette %s", d.art.Manifest.Palette)
		return
	}
	c := *pal
	c.ID = d.dev.Manifest.Palette
	c.Effects = slices.Clone(pal.Effects)
	d.dev.Palettes[c.ID] = &c
}

// levels dresses each level in the art pack's floors and walls, its walls shaped by their
// neighbors, and its features in the art pack's looks.
func (d *dresser) levels() {
	a := d.a
	floors, walls := d.art.Floors[a.Floors], d.art.Walls[a.Walls]
	if floors == nil || walls == nil {
		d.miss("floor set %s or wall set %s", a.Floors, a.Walls)
		return
	}
	floorID, _ := d.place("floors", a.Floors, func(s string) bool { return d.dev.Floors[s] != nil })
	fc := *floors
	fc.ID, fc.Sprite = floorID, d.sprite(floors.Sprite)
	d.dev.Floors[floorID] = &fc
	wallID, _ := d.place("walls", a.Walls, func(s string) bool { return d.dev.Walls[s] != nil })
	wc := *walls
	wc.ID, wc.Sprite = wallID, d.sprite(walls.Sprite)
	d.dev.Walls[wallID] = &wc

	for _, id := range slices.Sorted(maps.Keys(d.dev.Levels)) {
		l := d.dev.Levels[id]
		l.Floors, l.Walls = floorID, wallID
		for i, t := range l.FloorTypes {
			if v, ok := a.FloorVariants[t.Variant]; ok {
				l.FloorTypes[i].Variant = v
			} else {
				d.miss("%s: floor variant %d", id, t.Variant)
			}
		}
		isWall := func(x, y int) bool {
			return x >= 0 && y >= 0 && x < l.Size[0] && y < l.Size[1] && l.Wall[x][y] != content.NoFrame
		}
		neighbors := func(c content.Cell) int {
			n := 0
			for _, s := range []struct {
				dx, dy, bit int
			}{{-1, 0, wallUBefore}, {1, 0, wallUAfter}, {0, -1, wallVBefore}, {0, 1, wallVAfter}} {
				if isWall(c[0]+s.dx, c[1]+s.dy) {
					n |= s.bit
				}
			}
			return n
		}
		shaped := content.NewGrid(l.Size[0], l.Size[1])
		for x := range l.Size[0] {
			for y := range l.Size[1] {
				shaped[x][y] = content.NoFrame
				if isWall(x, y) {
					shaped[x][y] = a.WallShapes[neighbors(content.Cell{x, y})]
				}
			}
		}
		doors := map[content.Cell]int{}
		for _, f := range l.Features {
			if along := d.feature(l, f, neighbors(f.Cell), shaped); along >= 0 {
				doors[f.Cell] = along
			}
		}
		l.Wall = shaped
		d.halves(l, doors)
	}
}

// dirSteps are the cell steps of directions 0–7 (docs/spec/README.md).
var dirSteps = [8][2]int{{-1, 0}, {-1, 1}, {0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}}

// halves sets which halves of each cell's floor show: the floor under a wall or closed door
// shows on the sides its check directions say, toward cells in the level.
func (d *dresser) halves(l *content.Level, doors map[content.Cell]int) {
	a := d.a
	if a.WallChecks == nil {
		return
	}
	l.FloorHalves = content.NewGrid(l.Size[0], l.Size[1])
	l.UpperSight, l.LowerSight = content.NewGrid(l.Size[0], l.Size[1]), content.NewGrid(l.Size[0], l.Size[1])
	in := func(x, y, dir int) bool {
		if dir < 0 {
			return false
		}
		x, y = x+dirSteps[dir][0], y+dirSteps[dir][1]
		return x >= 0 && y >= 0 && x < l.Size[0] && y < l.Size[1] && l.Floor[x][y] != content.NoFrame
	}
	door := func(x, y, along int) bool {
		got, ok := doors[content.Cell{x, y}]
		return ok && got == along
	}
	for x := range l.Size[0] {
		for y := range l.Size[1] {
			l.UpperSight[x][y], l.LowerSight[x][y] = content.NoFrame, content.NoFrame
			var checks [2]int
			switch along, isDoor := doors[content.Cell{x, y}]; {
			case l.Floor[x][y] == content.NoFrame:
				l.FloorHalves[x][y] = 0
				continue
			case isDoor && along >= 0:
				checks = a.DoorChecks[along]
			case l.Wall[x][y] != content.NoFrame:
				s := l.Wall[x][y]
				switch {
				case door(x+1, y, 0):
					s = a.ChecksBelowDoor[s]
				default:
					if door(x, y-1, 1) {
						s = a.ChecksLeftOfDoor[s]
					}
					if door(x, y+1, 1) {
						s = a.ChecksRightOfDoor[s]
					}
				}
				checks = a.WallChecks[s]
			default:
				l.FloorHalves[x][y] = content.HalvesBoth
				continue
			}
			h := 0
			if in(x, y, checks[0]) {
				h |= content.HalfUpper
				l.UpperSight[x][y] = checks[0]
			}
			if in(x, y, checks[1]) {
				h |= content.HalfLower
				l.LowerSight[x][y] = checks[1]
			}
			l.FloorHalves[x][y] = h
		}
	}
}

// feature dresses a feature, and returns which way a door that is not sealed runs (0 between
// walls along x, 1 along y), or −1.
func (d *dresser) feature(l *content.Level, f *content.Feature, n int, shaped content.Grid) int {
	var look string
	door := -1
	switch {
	case f.Kind == content.FeatureDoor:
		along := 0
		if n&(wallVBefore|wallVAfter) != 0 && n&(wallUBefore|wallUAfter) == 0 {
			along = 1
		}
		look = d.look(d.a.Doors[along])
		if f.SealedBy != "" && look != "" {
			look = d.sealed(look, along)
		} else {
			door = along
		}
	case f.Kind == content.FeatureExit && shaped[f.Cell[0]][f.Cell[1]] != content.NoFrame:
		id, ok := d.a.WallExits[shaped[f.Cell[0]][f.Cell[1]]]
		if !ok {
			d.miss("%s: exit %s in a wall of shape %d", l.ID, f.ID, shaped[f.Cell[0]][f.Cell[1]])
			return door
		}
		look = d.look(id)
	default:
		id, ok := d.a.Features[f.Kind]
		if !ok {
			d.miss("%s: a look for %s %s", l.ID, f.Kind, f.ID)
			return door
		}
		look = d.look(id)
	}
	if look != "" {
		f.Look = look
	}
	return door
}

// sealed returns a door look with the sealed door's "sealed" state added.
func (d *dresser) sealed(door string, along int) string {
	s := d.art.Looks[d.a.SealedDoor]
	if s == nil || s.States["sealed"] == nil {
		d.miss("sealed door look %s", d.a.SealedDoor)
		return door
	}
	id := door + ".sealed"
	if d.dev.Looks[id] == nil {
		c := *d.dev.Looks[door]
		c.ID, c.States = id, maps.Clone(c.States)
		parts := d.parts(s.States["sealed"])
		if along == 1 {
			for i, p := range parts {
				b := d.dev.Sprites[p.Sprite]
				parts[i].Sprite = d.mirrored(p.Sprite)
				parts[i].Offset[0] = d.a.DoorMirror - p.Offset[0] - b.FrameSize[0]
			}
		}
		c.States["sealed"] = parts
		d.dev.Looks[id] = &c
	}
	return id
}

// mirrored returns a copy of sprite bank id with each frame flipped left to right.
func (d *dresser) mirrored(id string) string {
	to := id + ".mirrored"
	if d.dev.Sprites[to] != nil {
		return to
	}
	b := *d.dev.Sprites[id]
	src := b.Sheet
	sheet := image.NewPaletted(src.Rect, src.Palette)
	w, h := b.FrameSize[0], b.FrameSize[1]
	for f := range b.Frames {
		x0, y0 := src.Rect.Min.X+(f%b.Columns)*w, src.Rect.Min.Y+(f/b.Columns)*h
		for y := range h {
			for x := range w {
				sheet.SetColorIndex(x0+w-1-x, y0+y, src.ColorIndexAt(x0+x, y0+y))
			}
		}
	}
	b.ID, b.Sheet = to, sheet
	d.dev.Sprites[to] = &b
	d.copied[to] = true
	return to
}

// things gives each thing the art pack's figure for it.
func (d *dresser) things() {
	for _, id := range slices.Sorted(maps.Keys(d.a.Things)) {
		t, at := d.dev.Things[id], d.art.Things[d.a.Things[id]]
		if t == nil || at == nil {
			d.miss("thing %s as %s", id, d.a.Things[id])
			continue
		}
		t.Sprite = d.sprite(at.Sprite)
		t.Animations = maps.Clone(at.Animations)
		t.Animate = at.Animate
		t.Figure = nil
		for _, layer := range at.Figure {
			layer.Sprite, layer.Female = d.sprite(layer.Sprite), d.sprite(layer.Female)
			t.Figure = append(t.Figure, layer)
		}
	}
}

// items gives each item the art pack's images for it.
func (d *dresser) items() {
	for _, id := range slices.Sorted(maps.Keys(d.a.Items)) {
		it, at := d.dev.Items[id], d.art.Items[d.a.Items[id]]
		if it == nil || at == nil {
			d.miss("item %s as %s", id, d.a.Items[id])
			continue
		}
		it.Icon, it.Floor, it.Picture = d.image(at.Icon), d.image(at.Floor), d.imagePtr(at.Picture)
		it.Held = nil
		if at.Held != nil {
			it.Held = &content.Held{Right: d.held(at.Held.Right), Left: d.held(at.Held.Left)}
		}
		it.Doll = d.doll(at.Doll)
		it.Looks = nil
		for layer, l := range at.Looks {
			if it.Looks == nil {
				it.Looks = map[string]content.FigureLook{}
			}
			it.Looks[layer] = content.FigureLook{Sprite: d.sprite(l.Sprite), Female: d.sprite(l.Female)}
		}
		it.Fill = nil
		if at.Fill != nil {
			f := &content.Fill{Per: at.Fill.Per}
			for _, im := range at.Fill.Icons {
				f.Icons = append(f.Icons, d.image(im))
			}
			for _, dl := range at.Fill.Dolls {
				f.Dolls = append(f.Dolls, *d.doll(&dl))
			}
			it.Fill = f
		}
		if it.Flying != nil || at.Flying != nil {
			it.Flying = nil
			if at.Flying != nil {
				f := *at.Flying
				f.Sprite = d.sprite(f.Sprite)
				it.Flying = &f
			}
		}
	}
}

func (d *dresser) held(h *content.HeldLayers) *content.HeldLayers {
	if h == nil {
		return nil
	}
	return &content.HeldLayers{Behind: d.sprite(h.Behind), Front: d.sprite(h.Front)}
}

func (d *dresser) doll(im *content.DollImage) *content.DollImage {
	if im == nil {
		return nil
	}
	return &content.DollImage{Image: d.image(im.Image), At: im.At}
}

// spells gives each spell the icon and missile of the art pack's spell of the same id.
func (d *dresser) spells() {
	for _, id := range slices.Sorted(maps.Keys(d.dev.Spells)) {
		sp, at := d.dev.Spells[id], d.art.Spells[id]
		if at == nil {
			d.miss("spell %s", id)
			continue
		}
		sp.Icon = d.imagePtr(at.Icon)
		var missile *content.MissileSpell
		for _, e := range at.Effects {
			if e.Missile != nil {
				missile = e.Missile
			}
		}
		for _, e := range sp.Effects {
			if m := e.Missile; m != nil && missile != nil {
				m.Sprite, m.Cloud, m.Directional = d.sprite(missile.Sprite), d.sprite(missile.Cloud), missile.Directional
			}
		}
	}
}

// portraits gives each portrait the art pack's picture and figure colors.
func (d *dresser) portraits() {
	for _, id := range slices.Sorted(maps.Keys(d.a.Portraits)) {
		p, at := d.dev.Portraits[id], d.art.Portraits[d.a.Portraits[id]]
		if p == nil || at == nil {
			d.miss("portrait %s as %s", id, d.a.Portraits[id])
			continue
		}
		p.Image, p.Mirrored, p.FigureColors = d.image(at.Image), at.Mirrored, slices.Clone(at.FigureColors)
	}
}

// flipbooks plays the art pack's flipbooks in place of the dev pack's, without sounds the dev
// pack lacks.
func (d *dresser) flipbooks() {
	of := maps.Clone(d.a.Flipbooks)
	if d.dev.Effects != nil && d.art.Effects != nil {
		for cue, id := range d.dev.Effects.Cues {
			if at, ok := d.art.Effects.Cues[cue]; ok {
				of[id] = at
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(of)) {
		f, at := d.dev.Flipbooks[id], d.art.Flipbooks[of[id]]
		if f == nil || at == nil {
			d.miss("flipbook %s as %s", id, of[id])
			continue
		}
		c := &content.Flipbook{ID: id, Sprite: d.sprite(at.Sprite), Framed: at.Framed}
		for _, s := range at.Steps {
			s.Sounds = slices.DeleteFunc(slices.Clone(s.Sounds), func(snd string) bool { return d.dev.Sounds[snd] == nil })
			if d.dev.Sounds[s.Music] == nil {
				s.Music = ""
			}
			c.Steps = append(c.Steps, s)
		}
		d.dev.Flipbooks[id] = c
	}
}

// screens takes the art pack's gesture icons, pointers and inventory pictures, where they have
// as many as the dev pack's.
func (d *dresser) screens() {
	if m, am := d.dev.Magic, d.art.Magic; m != nil && am != nil && len(am.Icons) == len(m.Icons) {
		for i, im := range am.Icons {
			m.Icons[i] = d.image(im)
		}
		if am.Panel != nil {
			m.Panel = d.imagePtr(am.Panel)
		}
	}
	if p, ap := d.dev.Pointers, d.art.Pointers; p != nil && ap != nil {
		p.Arrow, p.Talk, p.Wait = d.image(ap.Arrow), d.imagePtr(ap.Talk), d.imagePtr(ap.Wait)
	}
	in, ai := d.dev.Inventory, d.art.Inventory
	if in == nil || ai == nil {
		return
	}
	if ai.Doll != nil {
		in.Doll = d.imagePtr(ai.Doll)
	}
	if ai.QuiverSlot != nil {
		in.QuiverSlot = d.doll(ai.QuiverSlot)
	}
	if ai.StrikeIcon != nil {
		in.StrikeIcon = d.imagePtr(ai.StrikeIcon)
	}
	if ai.ThrowIcon != nil {
		in.ThrowIcon = d.imagePtr(ai.ThrowIcon)
	}
	for _, l := range []struct{ to, from []content.Image }{
		{in.BareHands, ai.BareHands}, {in.WeaponIcons, ai.WeaponIcons}, {in.SchoolIcons, ai.SchoolIcons},
	} {
		if len(l.from) >= len(l.to) {
			for i := range l.to {
				l.to[i] = d.image(l.from[i])
			}
		}
	}
}

// recolor moves the colors of the dev pack's own sprites to the nearest of the new palette's,
// leaving out its animated colors.
func (d *dresser) recolor() {
	pal := d.dev.Palettes[d.dev.Manifest.Palette]
	animated := map[int]bool{}
	for _, e := range pal.Effects {
		switch e.Kind {
		case content.EffectRotate:
			for i := range e.Size * e.Groups {
				animated[e.First+i] = true
			}
		case content.EffectCycle:
			animated[e.Index] = true
		}
	}
	colors := make(color.Palette, 256)
	for i, c := range pal.Colors {
		colors[i] = c
	}
	for _, id := range slices.Sorted(maps.Keys(d.dev.Sprites)) {
		b := d.dev.Sprites[id]
		if d.copied[id] || b.Sheet == nil {
			continue
		}
		var to [256]uint8
		for i := range to {
			to[i] = uint8(nearest(pal, b.Sheet.Palette[i], animated, b.Transparent))
		}
		if b.Transparent != nil {
			to[*b.Transparent] = uint8(*b.Transparent)
		}
		s := image.NewPaletted(b.Sheet.Rect, colors)
		for i, px := range b.Sheet.Pix {
			s.Pix[i] = to[px]
		}
		c := *b
		c.Sheet = s
		d.dev.Sprites[id] = &c
	}
}

func nearest(pal *content.Palette, c color.Color, skip map[int]bool, transparent *int) int {
	r, g, b, _ := c.RGBA()
	best, bestD := 0, -1
	for i, p := range pal.Colors {
		if skip[i] || transparent != nil && i == *transparent {
			continue
		}
		dr, dg, db := int(r>>8)-int(p.R), int(g>>8)-int(p.G), int(b>>8)-int(p.B)
		if dd := dr*dr + dg*dg + db*db; bestD < 0 || dd < bestD {
			best, bestD = i, dd
		}
	}
	return best
}
