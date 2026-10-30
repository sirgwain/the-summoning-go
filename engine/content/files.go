package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Pack layout.
const (
	manifestFile = "pack.json"
	palettesDir  = "palettes"
	spritesDir   = "sprites"
	floorsDir    = "tiles/floors"
	wallsDir     = "tiles/walls"
	levelsDir    = "levels"
	levelFile    = "level.json"
)

// Load reads the pack in fsys and verifies it.
func Load(fsys fs.FS) (*Pack, error) {
	var m Manifest
	if err := readJSON(fsys, manifestFile, &m); err != nil {
		return nil, err
	}
	if m.Format != Format {
		return nil, fmt.Errorf("%s: format %d, want %d", manifestFile, m.Format, Format)
	}
	p := NewPack(m)

	if err := loadEach(fsys, palettesDir+"/*.json", func(name string) error {
		pal := &Palette{}
		if err := readJSON(fsys, name, pal); err != nil {
			return err
		}
		return add(p.Palettes, pal.ID, pal, name)
	}); err != nil {
		return nil, err
	}
	if err := loadEach(fsys, spritesDir+"/*.json", func(name string) error {
		b := &SpriteBank{}
		if err := readJSON(fsys, name, b); err != nil {
			return err
		}
		sheet, err := readSheet(fsys, strings.TrimSuffix(name, ".json")+".png")
		if err != nil {
			return err
		}
		b.Sheet = sheet
		return add(p.Sprites, b.ID, b, name)
	}); err != nil {
		return nil, err
	}
	if err := loadEach(fsys, floorsDir+"/*.json", func(name string) error {
		f := &FloorSet{}
		if err := readJSON(fsys, name, f); err != nil {
			return err
		}
		return add(p.Floors, f.ID, f, name)
	}); err != nil {
		return nil, err
	}
	if err := loadEach(fsys, wallsDir+"/*.json", func(name string) error {
		w := &WallSet{}
		if err := readJSON(fsys, name, w); err != nil {
			return err
		}
		return add(p.Walls, w.ID, w, name)
	}); err != nil {
		return nil, err
	}
	if err := loadEach(fsys, levelsDir+"/*/"+levelFile, func(name string) error {
		l := &Level{}
		if err := readJSON(fsys, name, l); err != nil {
			return err
		}
		return add(p.Levels, l.ID, l, name)
	}); err != nil {
		return nil, err
	}

	if err := p.Verify(); err != nil {
		return nil, err
	}
	return p, nil
}

func loadEach(fsys fs.FS, pattern string, load func(name string) error) error {
	names, err := fs.Glob(fsys, pattern)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := load(name); err != nil {
			return err
		}
	}
	return nil
}

func add[T any](m map[string]*T, id string, v *T, name string) error {
	if id == "" {
		return fmt.Errorf("%s: no id", name)
	}
	if _, ok := m[id]; ok {
		return fmt.Errorf("%s: duplicate id %s", name, id)
	}
	m[id] = v
	return nil
}

func readJSON(fsys fs.FS, name string, v any) error {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func readSheet(fsys fs.FS, name string) (*image.Paletted, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	sheet, ok := img.(*image.Paletted)
	if !ok {
		return nil, fmt.Errorf("%s: not an indexed PNG", name)
	}
	return sheet, nil
}

// Write writes the pack into dir, creating it if needed. Each record goes in a file named after
// its id.
func Write(p *Pack, dir string) error {
	if err := writeJSON(dir, manifestFile, p.Manifest); err != nil {
		return err
	}
	for _, id := range sortedKeys(p.Palettes) {
		if err := writeJSON(dir, path.Join(palettesDir, fileName(id, "palette.")+".json"), p.Palettes[id]); err != nil {
			return err
		}
	}
	for _, id := range sortedKeys(p.Sprites) {
		b := p.Sprites[id]
		base := path.Join(spritesDir, fileName(id, "sprite."))
		if b.Sheet == nil {
			return fmt.Errorf("sprite bank %s: no sheet", id)
		}
		if err := writeJSON(dir, base+".json", b); err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, b.Sheet); err != nil {
			return fmt.Errorf("sprite bank %s: %w", id, err)
		}
		if err := writeFile(dir, base+".png", buf.Bytes()); err != nil {
			return err
		}
	}
	for _, id := range sortedKeys(p.Floors) {
		if err := writeJSON(dir, path.Join(floorsDir, fileName(id, "floors.")+".json"), p.Floors[id]); err != nil {
			return err
		}
	}
	for _, id := range sortedKeys(p.Walls) {
		if err := writeJSON(dir, path.Join(wallsDir, fileName(id, "walls.")+".json"), p.Walls[id]); err != nil {
			return err
		}
	}
	for _, id := range sortedKeys(p.Levels) {
		if err := writeJSON(dir, path.Join(levelsDir, fileName(id, "level."), levelFile), p.Levels[id]); err != nil {
			return err
		}
	}
	return nil
}

// fileName is the file name of a record: its id without the kind prefix.
func fileName(id, prefix string) string { return strings.TrimPrefix(id, prefix) }

func writeJSON(dir, name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	data = numberArray.ReplaceAllFunc(data, func(m []byte) []byte {
		return space.ReplaceAll(m, []byte(" "))
	})
	return writeFile(dir, name, append(data, '\n'))
}

// numberArray matches an indented JSON array holding only numbers, which writeJSON puts on one
// line.
var (
	numberArray = regexp.MustCompile(`\[\s*-?\d+(?:,\s*-?\d+)*\s*\]`)
	space       = regexp.MustCompile(`\s+`)
)

func writeFile(dir, name string, data []byte) error {
	full := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o644)
}

func sortedKeys[T any](m map[string]*T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type paletteJSON struct {
	ID     string   `json:"id"`
	Colors []string `json:"colors"`
}

func (p *Palette) MarshalJSON() ([]byte, error) {
	out := paletteJSON{ID: p.ID, Colors: make([]string, len(p.Colors))}
	for i, c := range p.Colors {
		out.Colors[i] = fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return json.Marshal(out)
}

func (p *Palette) UnmarshalJSON(data []byte) error {
	var in paletteJSON
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	if len(in.Colors) != len(p.Colors) {
		return fmt.Errorf("palette %s: %d colors, want %d", in.ID, len(in.Colors), len(p.Colors))
	}
	p.ID = in.ID
	for i, s := range in.Colors {
		var r, g, b uint8
		if n, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); n != 3 || err != nil || len(s) != 7 {
			return fmt.Errorf("palette %s: bad color %q", in.ID, s)
		}
		p.Colors[i] = color.RGBA{r, g, b, 0xFF}
	}
	return nil
}

// Verify checks that every reference in the pack resolves and every record is consistent.
func (p *Pack) Verify() error {
	var errs []error
	fail := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	m := p.Manifest
	if _, ok := p.Palettes[m.Palette]; !ok {
		fail("pack.json: palette %q not found", m.Palette)
	}
	if l, ok := p.Levels[m.Start.Level]; !ok {
		fail("pack.json: start level %q not found", m.Start.Level)
	} else if !l.InBounds(m.Start.Cell) {
		fail("pack.json: start cell %v outside %s", m.Start.Cell, l.ID)
	}

	for _, id := range sortedKeys(p.Sprites) {
		b := p.Sprites[id]
		if b.FrameSize[0] <= 0 || b.FrameSize[1] <= 0 || b.Columns <= 0 || b.Frames < 0 {
			fail("sprite bank %s: bad layout", id)
			continue
		}
		if b.Transparent != nil && (*b.Transparent < 0 || *b.Transparent > 255) {
			fail("sprite bank %s: transparent index %d", id, *b.Transparent)
		}
		if b.Frames > 0 && !b.FrameRect(b.Frames-1).In(b.Sheet.Bounds()) {
			fail("sprite bank %s: %d frames do not fit the %v sheet", id, b.Frames, b.Sheet.Bounds().Size())
		}
	}
	frames := func(owner, sprite string) int {
		b, ok := p.Sprites[sprite]
		if !ok {
			fail("%s: sprite bank %q not found", owner, sprite)
			return -1
		}
		return b.Frames
	}

	for _, id := range sortedKeys(p.Floors) {
		f := p.Floors[id]
		if n := frames("floor set "+id, f.Sprite); n >= 0 && f.Variants > n {
			fail("floor set %s: %d variants, sprite bank has %d frames", id, f.Variants, n)
		}
	}
	for _, id := range sortedKeys(p.Walls) {
		w := p.Walls[id]
		n := frames("wall set "+id, w.Sprite)
		for s, fr := range w.Shapes {
			if fr != NoFrame && (fr < 0 || (n >= 0 && fr >= n)) {
				fail("wall set %s: shape %d uses frame %d of %d", id, s, fr, n)
			}
		}
	}

	for _, id := range sortedKeys(p.Levels) {
		errs = append(errs, p.verifyLevel(p.Levels[id], frames)...)
	}
	return errors.Join(errs...)
}

func (p *Pack) verifyLevel(l *Level, frames func(owner, sprite string) int) []error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("level %s: "+format, append([]any{l.ID}, args...)...))
	}

	floors, ok := p.Floors[l.Floors]
	if !ok {
		fail("floor set %q not found", l.Floors)
	}
	walls, ok := p.Walls[l.Walls]
	if !ok {
		fail("wall set %q not found", l.Walls)
	}
	var exitWalls *WallSet
	if l.ExitWalls != "" {
		if exitWalls, ok = p.Walls[l.ExitWalls]; !ok {
			fail("exit wall set %q not found", l.ExitWalls)
		}
	} else if len(l.Exits) > 0 {
		fail("exits but no exit wall set")
	}

	checkGrid := func(name string, g Grid, valid func(v int) bool) {
		if len(g) != l.Size[0] {
			fail("%s has %d rows, want %d", name, len(g), l.Size[0])
			return
		}
		for x, row := range g {
			if len(row) != l.Size[1] {
				fail("%s row %d has %d cells, want %d", name, x, len(row), l.Size[1])
				continue
			}
			for y, v := range row {
				if v != NoFrame && !valid(v) {
					fail("%s (%d, %d): %d out of range", name, x, y, v)
				}
			}
		}
	}
	checkGrid("floor", l.Floor, func(v int) bool { return floors == nil || v < floors.Variants })
	checkGrid("wall", l.Wall, func(v int) bool { return walls == nil || (v < len(walls.Shapes) && walls.Shapes[v] != NoFrame) })
	if l.FloorHalves != nil {
		checkGrid("floor_halves", l.FloorHalves, func(v int) bool { return v <= HalvesBoth })
	}

	for _, c := range l.Exits {
		if !l.InBounds(c) {
			fail("exit %v outside the level", c)
			continue
		}
		if len(l.Wall) != l.Size[0] || len(l.Wall[c[0]]) != l.Size[1] {
			continue
		}
		s := l.Wall[c[0]][c[1]]
		if s == NoFrame {
			fail("exit %v is not a wall", c)
		} else if exitWalls != nil && (s >= len(exitWalls.Shapes) || exitWalls.Shapes[s] == NoFrame) {
			fail("exit %v: exit wall set has no shape %d", c, s)
		}
	}

	for _, layer := range []struct {
		kind string
		list []Placed
	}{{"overlay", l.Overlays}, {"decoration", l.Decorations}, {"prop", l.Props}} {
		kind := layer.kind
		for _, pl := range layer.list {
			if !l.InBounds(pl.Cell) {
				fail("%s at %v outside the level", kind, pl.Cell)
			}
			if n := frames("level "+l.ID, pl.Sprite); n >= 0 && (pl.Frame < 0 || pl.Frame >= n) {
				fail("%s at %v: frame %d of %d in %s", kind, pl.Cell, pl.Frame, n, pl.Sprite)
			}
		}
	}
	return errs
}
