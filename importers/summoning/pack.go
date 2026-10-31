package summoning

import (
	"fmt"
	"image"
	"sort"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Source holds the data files the importer reads.
type Source struct {
	Colors, Init, Resource, Levels []byte
}

// ReadSource reads the data files from dir.
func ReadSource(dir string) (*Source, error) {
	s := &Source{}
	for _, f := range []struct {
		name string
		dst  *[]byte
	}{{FileColors, &s.Colors}, {FileInit, &s.Init}, {FileResource, &s.Resource}, {FileLevels, &s.Levels}} {
		data, err := ReadDataFile(dir, f.name)
		if err != nil {
			return nil, err
		}
		*f.dst = data
	}
	return s, nil
}

// Report lists what an import could not map, with how often each thing happened.
type Report struct {
	counts map[string]int
}

func (r *Report) addf(format string, args ...any) {
	if r.counts == nil {
		r.counts = map[string]int{}
	}
	r.counts[fmt.Sprintf(format, args...)]++
}

// Lines returns the report's entries, sorted, each with its count.
func (r *Report) Lines() []string {
	lines := make([]string, 0, len(r.counts))
	for msg, n := range r.counts {
		if n > 1 {
			msg = fmt.Sprintf("%s (×%d)", msg, n)
		}
		lines = append(lines, msg)
	}
	sort.Strings(lines)
	return lines
}

// Terrain values (world.md).
const (
	terrainLastWall     = 0x0E
	terrainFloorOverlay = 0x17 // 17h–1Ah
	terrainFloor        = 0x1B // 1Bh–1Eh
	terrainPlate        = 0x1F // 1Fh–28h
	terrainTeleporter   = 0x24
	terrainTrapdoorShut = 0x27
	terrainGemSocket    = 0x28
	terrainHazard       = 0x29 // 29h–35h
	terrainLastHazard   = 0x35
)

// Resource offsets from the terrain base, and fixed resources (world.md).
const (
	gemSocketBank   = 0x37
	gemBank         = 0x38
	sealedDoorBank  = 0x39
	hazardResource  = 67
	wallShapes      = 15
	exitShapeLeft   = 1 // terrain 1
	exitShapeRight  = 4 // terrain 3
	gemCount        = 6
	gemsAllSet      = 0x3F
	doorStageMask   = 0x07
	doorVertical    = 0x40
	doorOpen        = 4
	doorFirstSealed = 5
	teleporterArmed = 0x80
	teleporterExit  = 0xFF
)

var gemOffsets = [gemCount]content.Offset{{11, 1}, {3, 1}, {21, 1}, {19, 6}, {1, 6}, {9, 7}}

// diagonalShapes are the wall shapes that fill half their cell (world.md, Terrain).
var diagonalShapes = [wallShapes]bool{0: true, 2: true, 4: true, 5: true}

const sheetColumns = 16

type importer struct {
	pack      *content.Pack
	init      *Init
	resources [][]byte
	palette   *content.Palette
	banks     map[int]*SpriteBank
	report    *Report
}

// Import converts the data files into a content pack.
func Import(src *Source) (*content.Pack, *Report, error) {
	in, err := ReadInit(src.Init)
	if err != nil {
		return nil, nil, err
	}
	palettes, err := ReadColors(src.Colors)
	if err != nil {
		return nil, nil, err
	}
	if len(palettes) == 0 {
		return nil, nil, fmt.Errorf("colors: no full palette")
	}
	resources, err := ReadContainer(src.Resource)
	if err != nil {
		return nil, nil, fmt.Errorf("resource: %w", err)
	}
	levels, err := ReadLevels(src.Levels)
	if err != nil {
		return nil, nil, err
	}

	im := &importer{
		pack: content.NewPack(content.Manifest{
			ID:      "summoning",
			Name:    "The Summoning",
			Version: "0.1.0",
			Format:  content.Format,
			Depends: []string{},
			Palette: paletteID(0),
			Start:   content.Start{Level: levelID(int(in.StartLevel)), Cell: content.Cell{int(in.StartX), int(in.StartY)}},
		}),
		init:      in,
		resources: resources,
		banks:     map[int]*SpriteBank{},
		report:    &Report{},
	}
	for i, p := range palettes {
		pal := &content.Palette{ID: paletteID(i), Colors: p, Effects: paletteEffects()}
		im.pack.Palettes[pal.ID] = pal
	}
	im.palette = im.pack.Palettes[paletteID(0)]

	for i, l := range levels {
		cl, err := im.level(i, l)
		if err != nil {
			return nil, nil, fmt.Errorf("level %d: %w", i, err)
		}
		im.pack.Levels[cl.ID] = cl
	}
	if err := im.pack.Verify(); err != nil {
		return nil, nil, fmt.Errorf("imported pack: %w", err)
	}
	return im.pack, im.report, nil
}

func paletteID(i int) string { return fmt.Sprintf("palette.%d", i) }
func levelID(i int) string   { return fmt.Sprintf("level.%02d", i) }

// bank decodes resource id as a sprite bank, once.
func (im *importer) bank(id int) (*SpriteBank, error) {
	if b, ok := im.banks[id]; ok {
		return b, nil
	}
	if id < 0 || id >= len(im.resources) {
		return nil, fmt.Errorf("resource %d does not exist", id)
	}
	r, err := ParseResource(im.resources[id])
	if err != nil {
		return nil, fmt.Errorf("resource %d: %w", id, err)
	}
	b, err := DecodeSpriteBank(r)
	if err != nil {
		return nil, fmt.Errorf("resource %d: %w", id, err)
	}
	im.banks[id] = b
	return b, nil
}

func (im *importer) terrain(offset int) int { return int(im.init.Bases.Terrain) + offset }

func transparentPtr(t int) *int {
	if t == NoTransparency {
		return nil
	}
	return &t
}

// addSprite adds a sprite bank of same-sized frames to the pack.
func (im *importer) addSprite(id string, frames [][]byte, w, h int, transparent int) {
	fill := uint8(0)
	if transparent != NoTransparency {
		fill = uint8(transparent)
	}
	t := transparentPtr(transparent)
	sheet, cols := content.NewSheet(frames, w, h, sheetColumns, im.palette, t, fill)
	im.pack.Sprites[id] = &content.SpriteBank{
		ID: id, FrameSize: content.Offset{w, h}, Columns: cols, Frames: len(frames), Transparent: t, Sheet: sheet,
	}
}

// rawSprite adds resource id to the pack as it is and returns its id.
func (im *importer) rawSprite(id int) (string, *SpriteBank, error) {
	b, err := im.bank(id)
	if err != nil {
		return "", nil, err
	}
	sid := fmt.Sprintf("sprite.res.%d", id)
	if _, ok := im.pack.Sprites[sid]; !ok {
		im.addSprite(sid, b.Frames, b.Width, b.Height, b.Transparent)
	}
	return sid, b, nil
}

// part is one frame drawn into a composed image, at an offset from the tile box.
type part struct {
	bank  *SpriteBank
	frame int
	at    content.Offset
}

// compose draws parts in order into an image covering box (relative to the tile box). Pixels no
// part covers, and each part's transparent pixels, are left as transparent.
func compose(box image.Rectangle, transparent int, parts []part) []byte {
	w, h := box.Dx(), box.Dy()
	img := make([]byte, w*h)
	if transparent != NoTransparency {
		for i := range img {
			img[i] = uint8(transparent)
		}
	}
	for _, p := range parts {
		src := p.bank.Frames[p.frame]
		for y := range p.bank.Height {
			for x := range p.bank.Width {
				c := src[y*p.bank.Width+x]
				if int(c) == p.bank.Transparent {
					continue
				}
				dx, dy := p.at[0]+x-box.Min.X, p.at[1]+y-box.Min.Y
				if dx >= 0 && dy >= 0 && dx < w && dy < h {
					img[dy*w+dx] = c
				}
			}
		}
	}
	return img
}

// floorSet composes the floor bank's half-tiles into whole floor variants.
func (im *importer) floorSet(id int) (string, error) {
	fid := fmt.Sprintf("floors.%d", id)
	if _, ok := im.pack.Floors[fid]; ok {
		return fid, nil
	}
	b, err := im.bank(id)
	if err != nil {
		return "", err
	}
	box := image.Rect(0, 0, b.Width, 2*b.Height)
	var frames [][]byte
	for k := 0; 2*k+1 < len(b.Frames); k++ {
		frames = append(frames, compose(box, b.Transparent, []part{
			{b, 2 * k, content.Offset{0, 0}},
			{b, 2*k + 1, content.Offset{0, b.Height}},
		}))
	}
	sid := "sprite." + fid
	im.addSprite(sid, frames, box.Dx(), box.Dy(), b.Transparent)
	im.pack.Floors[fid] = &content.FloorSet{ID: fid, Sprite: sid, Variants: len(frames)}
	return fid, nil
}

// wallSet composes each wall shape's upper part, lower part and caps into one image.
func (im *importer) wallSet(id int) (string, error) {
	wid := fmt.Sprintf("walls.%d", id)
	if _, ok := im.pack.Walls[wid]; ok {
		return wid, nil
	}
	b, err := im.bank(id)
	if err != nil {
		return "", err
	}
	caps, err := im.bank(id + 1)
	if err != nil {
		return "", err
	}
	if len(b.Frames) < 2*wallShapes || len(caps.Frames) < 2 {
		return "", fmt.Errorf("wall bank %d: %d frames and %d caps", id, len(b.Frames), len(caps.Frames))
	}
	capY := -caps.Height
	box := image.Rect(0, capY, b.Width, 2*b.Height)
	frames := make([][]byte, wallShapes)
	shapes := make([]*content.WallShape, wallShapes)
	for s := range wallShapes {
		var parts []part
		if im.init.WallTops[s]&WallTopLeftCap != 0 {
			parts = append(parts, part{caps, 0, content.Offset{0, capY}})
		}
		if im.init.WallTops[s]&WallTopRightCap != 0 {
			parts = append(parts, part{caps, 1, content.Offset{b.Width / 2, capY}})
		}
		parts = append(parts, part{b, s, content.Offset{0, 0}}, part{b, s + wallShapes, content.Offset{0, b.Height}})
		frames[s] = compose(box, b.Transparent, parts)
		shapes[s] = &content.WallShape{Frame: s, Diagonal: diagonalShapes[s], Face: s == 1 || s == 3}
	}
	sid := "sprite." + wid
	im.addSprite(sid, frames, box.Dx(), box.Dy(), b.Transparent)
	im.pack.Walls[wid] = &content.WallSet{ID: wid, Sprite: sid, Offset: content.Offset{box.Min.X, box.Min.Y}, Shapes: shapes}
	return wid, nil
}

// exitLook returns the look of an exit in a wall of shape 1 or 3, composing the exit-wall bank's
// two exits into one sprite the first time.
func (im *importer) exitLook(id, wallShape int) (string, error) {
	lid := fmt.Sprintf("look.exit.%d.%d", id, wallShape)
	if _, ok := im.pack.Looks[lid]; ok {
		return lid, nil
	}
	b, err := im.bank(id)
	if err != nil {
		return "", err
	}
	if len(b.Frames) <= exitShapeRight+1 {
		return "", fmt.Errorf("exit-wall bank %d: %d frames", id, len(b.Frames))
	}
	box := image.Rect(0, -b.Height, b.Width, 2*b.Height)
	sid := fmt.Sprintf("sprite.exitwalls.%d", id)
	exits := []struct{ terrain, shape int }{{1, exitShapeLeft}, {3, exitShapeRight}}
	if _, ok := im.pack.Sprites[sid]; !ok {
		var frames [][]byte
		for _, e := range exits {
			frames = append(frames, compose(box, b.Transparent, []part{
				{b, e.shape - 1, content.Offset{0, -b.Height}},
				{b, e.shape, content.Offset{0, 0}},
				{b, e.shape + 1, content.Offset{0, b.Height}},
			}))
		}
		im.addSprite(sid, frames, box.Dx(), box.Dy(), b.Transparent)
	}
	for i, e := range exits {
		at := content.Offset{box.Min.X, box.Min.Y}
		im.pack.Looks[fmt.Sprintf("look.exit.%d.%d", id, e.terrain)] = &content.Look{
			ID: fmt.Sprintf("look.exit.%d.%d", id, e.terrain), Layer: content.LayerWall,
			States: map[string][]content.LookPart{content.FeatureExit: {{Sprite: sid, Frame: i, Offset: at}}},
		}
	}
	return lid, nil
}

// place adds frame of resource id at cell, if the bank has that frame.
func (im *importer) place(list *[]content.Placed, cell content.Cell, id, frame int, at content.Offset, what string) error {
	sid, b, err := im.rawSprite(id)
	if err != nil {
		return err
	}
	if frame >= len(b.Frames) {
		im.report.addf("%s: frame %d not in resource %d (%d frames); not drawn", what, frame, id, len(b.Frames))
		return nil
	}
	*list = append(*list, content.Placed{Cell: cell, Sprite: sid, Frame: frame, Offset: at})
	return nil
}

func (im *importer) level(index int, l *Level) (*content.Level, error) {
	h := &l.Header
	cl := &content.Level{
		ID:    levelID(index),
		Size:  [2]int{l.Rows, l.Columns},
		Floor: content.NewGrid(l.Rows, l.Columns),
		Wall:  content.NewGrid(l.Rows, l.Columns),
	}
	var err error
	if cl.Floors, err = im.floorSet(im.terrain(int(h.FloorBank))); err != nil {
		return nil, err
	}
	for k := range im.pack.Floors[cl.Floors].Variants {
		cl.FloorTypes = append(cl.FloorTypes, content.FloorType{Variant: k})
	}
	if cl.Walls, err = im.wallSet(im.terrain(int(h.WallBank))); err != nil {
		return nil, err
	}

	byCell := map[content.Cell]*Placement{}
	for i := range l.Placements {
		p := &l.Placements[i]
		byCell[content.Cell{int(p.X), int(p.Y)}] = p
	}
	armed := func(c content.Cell) bool {
		p := byCell[c]
		switch {
		case p == nil:
			return false
		case p.Type == PlaceExit:
			return true
		case p.Type == PlaceTeleporter:
			return p.Extra[0]&teleporterArmed != 0
		case p.Type == PlaceTrigger:
			return p.Extra[1] == 0
		}
		return false
	}

	for x := range l.Rows {
		for y := range l.Columns {
			c := content.Cell{x, y}
			t := int(l.TerrainAt(x, y))
			switch {
			case t == TerrainOutside:
				continue
			case t <= terrainLastWall:
				cl.Wall[x][y] = t
				cl.Floor[x][y] = 0
				continue
			case t > terrainLastHazard:
				im.report.addf("terrain %02Xh: unknown; not drawn", t)
				continue
			case t >= terrainFloor && t < terrainPlate:
				cl.Floor[x][y] = t - terrainFloor
			default:
				cl.Floor[x][y] = 0
			}

			switch {
			case t >= terrainFloorOverlay && t < terrainFloor:
				err = im.place(&cl.Overlays, c, im.terrain(int(h.FloorOverlayBank)), t-terrainFloorOverlay, content.Offset{4, 11}, "floor overlay")
			case t == terrainGemSocket:
				err = im.gemSocket(cl, c, 0)
			case t >= terrainPlate && t <= terrainGemSocket:
				if (t == terrainTeleporter && !armed(c)) || t == terrainTrapdoorShut {
					continue
				}
				err = im.place(&cl.Overlays, c, im.terrain(int(h.PlateBank)), t-terrainPlate, content.Offset{-16, 8}, "plate")
			case t >= terrainHazard:
				err = im.place(&cl.Overlays, c, hazardResource, t-terrainHazard, content.Offset{0, 0}, "hazard")
			}
			if err != nil {
				return nil, err
			}
		}
	}

	cl.FloorHalves = floorHalves(l, im.init.Directions)

	for i := range l.Placements {
		if err := im.feature(cl, l, &l.Placements[i]); err != nil {
			return nil, err
		}
	}
	return cl, nil
}

// feature adds what a placement draws: decorations, exits in walls and doors. Everything else
// in the entity stream belongs to later milestones.
func (im *importer) feature(cl *content.Level, l *Level, p *Placement) error {
	h := &l.Header
	c := content.Cell{int(p.X), int(p.Y)}
	t := int(l.TerrainAt(c[0], c[1]))
	wallExit := (p.Type == PlaceTeleporter || p.Type == PlaceExit) && t <= terrainLastWall

	switch {
	case wallExit:
		if t != 1 && t != 3 {
			im.report.addf("exit in a wall of shape %d; drawn as a wall", t)
			return nil
		}
		to, ok := exitDestination(p)
		if !ok {
			im.report.addf("in-level teleporter in a wall; drawn as a wall")
			return nil
		}
		look, err := im.exitLook(im.terrain(int(h.ExitWallBank)), t)
		if err != nil {
			return err
		}
		cl.Features = append(cl.Features, &content.Feature{
			ID: fmt.Sprintf("exit.%d.%d", c[0], c[1]), Kind: content.FeatureExit, Cell: c, Look: look, To: to,
		})

	case p.Type == PlaceDecoration:
		if t != 1 && t != 3 {
			return nil
		}
		ornament, decoration := p.Decoration()
		if decoration >= len(h.DecorationBanks) {
			im.report.addf("decoration %d: no such decoration bank; not drawn", decoration)
			return nil
		}
		id := int(im.init.Bases.WallDecoration) + int(h.DecorationBanks[decoration])
		return im.place(&cl.Decorations, c, id, ornament, content.Offset{8, 8}, "decoration")

	case p.Type == PlaceDoor:
		return im.door(cl, h, c, p.Extra[0])
	}
	return nil
}

// exitDestination returns where an exit placement leads: a level exit, or a teleporter record of
// kind FFh (rules.md, Exits).
func exitDestination(p *Placement) (*content.Destination, bool) {
	var level, x, y byte
	switch {
	case p.Type == PlaceExit:
		level, x, y = p.Extra[1], p.Extra[2], p.Extra[3]
	case p.Type == PlaceTeleporter && p.Extra[0] == teleporterExit:
		level, x, y = p.Extra[1], p.Extra[2], p.Extra[3]
	default:
		return nil, false
	}
	return &content.Destination{Level: levelID(int(level)), Cell: content.Cell{int(x), int(y)}}, true
}

// door adds a door's two sprites at its initial stage. They are placed from the reference point
// of the cell's corner, which is (16, 0) from the tile box. An open door is not drawn.
func (im *importer) door(cl *content.Level, h *LevelHeader, c content.Cell, state byte) error {
	stage := int(state & doorStageMask)
	if stage == doorOpen {
		return nil
	}
	id, first := im.terrain(int(h.DoorBank)), 2*stage
	at := [2]content.Offset{{16 - 33, -3}, {16 - 1, -14}}
	switch {
	case stage >= doorFirstSealed:
		id, first = im.terrain(sealedDoorBank), 4*(stage-doorFirstSealed)
	case state&doorVertical != 0:
		id++
		at = [2]content.Offset{{16 - 30, -14}, {16 + 2, -3}}
	}
	for i := range 2 {
		if err := im.place(&cl.Props, c, id, first+i, at[i], fmt.Sprintf("door stage %d", stage)); err != nil {
			return err
		}
	}
	return nil
}

// gemSocket adds a gem socket showing the gems set in mask.
func (im *importer) gemSocket(cl *content.Level, c content.Cell, mask int) error {
	at := content.Offset{-1, 16}
	if mask == gemsAllSet {
		return im.place(&cl.Overlays, c, im.terrain(gemSocketBank), 1, at, "gem socket")
	}
	if err := im.place(&cl.Overlays, c, im.terrain(gemSocketBank), 0, at, "gem socket"); err != nil {
		return err
	}
	for g := range gemCount {
		if mask&(1<<g) != 0 {
			o := content.Offset{at[0] + gemOffsets[g][0], at[1] + gemOffsets[g][1]}
			if err := im.place(&cl.Overlays, c, im.terrain(gemBank), g, o, "gem"); err != nil {
				return err
			}
		}
	}
	return nil
}
