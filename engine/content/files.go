package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
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
	looksDir     = "looks"
	itemsDir     = "items"
	thingsDir    = "things"
	portraitsDir = "portraits"
	textDir      = "text"
	messagesFile = "messages.json"
	levelsDir    = "levels"
	levelFile    = "level.json"
	spellsDir    = "spells"
	magicFile    = "magic.json"
	keywordsFile = "keywords.json"
	pointersFile = "pointers.json"
	talkDir      = "conversations"
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
	if err := loadRecords(fsys, looksDir+"/*.json", p.Looks, func(l *Look) string { return l.ID }); err != nil {
		return nil, err
	}
	if err := loadRecords(fsys, itemsDir+"/*.json", p.Items, func(i *Item) string { return i.ID }); err != nil {
		return nil, err
	}
	if err := loadRecords(fsys, thingsDir+"/*.json", p.Things, func(t *Thing) string { return t.ID }); err != nil {
		return nil, err
	}
	if err := loadRecords(fsys, portraitsDir+"/*.json", p.Portraits, func(t *Portrait) string { return t.ID }); err != nil {
		return nil, err
	}
	if err := loadRecords(fsys, textDir+"/*.json", p.Texts, func(t *Text) string { return t.ID }); err != nil {
		return nil, err
	}
	for _, f := range []struct {
		name string
		set  func() any
	}{
		{messagesFile, func() any { p.Messages = &Messages{}; return p.Messages }},
		{magicFile, func() any { p.Magic = &Magic{}; return p.Magic }},
		{keywordsFile, func() any { p.Keywords = &Keywords{}; return p.Keywords }},
		{pointersFile, func() any { p.Pointers = &Pointers{}; return p.Pointers }},
	} {
		if _, err := fs.Stat(fsys, f.name); err == nil {
			if err := readJSON(fsys, f.name, f.set()); err != nil {
				return nil, err
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if err := loadRecords(fsys, spellsDir+"/*.json", p.Spells, func(s *Spell) string { return s.ID }); err != nil {
		return nil, err
	}
	if err := loadRecords(fsys, talkDir+"/*.json", p.Conversations, func(c *Conversation) string { return c.ID }); err != nil {
		return nil, err
	}
	if err := loadRecords(fsys, levelsDir+"/*/"+levelFile, p.Levels, func(l *Level) string { return l.ID }); err != nil {
		return nil, err
	}

	if err := p.Verify(); err != nil {
		return nil, err
	}
	return p, nil
}

// loadRecords reads every file matching pattern as a record of type T into m.
func loadRecords[T any](fsys fs.FS, pattern string, m map[string]*T, id func(*T) string) error {
	return loadEach(fsys, pattern, func(name string) error {
		v := new(T)
		if err := readJSON(fsys, name, v); err != nil {
			return err
		}
		return add(m, id(v), v, name)
	})
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
	if err := writeRecords(dir, looksDir, "look.", p.Looks); err != nil {
		return err
	}
	if err := writeRecords(dir, itemsDir, "item.", p.Items); err != nil {
		return err
	}
	if err := writeRecords(dir, thingsDir, "thing.", p.Things); err != nil {
		return err
	}
	if err := writeRecords(dir, portraitsDir, "portrait.", p.Portraits); err != nil {
		return err
	}
	if err := writeRecords(dir, textDir, "text.", p.Texts); err != nil {
		return err
	}
	for _, f := range []struct {
		name string
		v    any
		ok   bool
	}{{messagesFile, p.Messages, p.Messages != nil}, {magicFile, p.Magic, p.Magic != nil}, {keywordsFile, p.Keywords, p.Keywords != nil}, {pointersFile, p.Pointers, p.Pointers != nil}} {
		if f.ok {
			if err := writeJSON(dir, f.name, f.v); err != nil {
				return err
			}
		}
	}
	if err := writeRecords(dir, spellsDir, "spell.", p.Spells); err != nil {
		return err
	}
	if err := writeRecords(dir, talkDir, "conversation.", p.Conversations); err != nil {
		return err
	}
	for _, id := range sortedKeys(p.Levels) {
		if err := writeJSON(dir, path.Join(levelsDir, fileName(id, "level."), levelFile), p.Levels[id]); err != nil {
			return err
		}
	}
	return nil
}

// writeRecords writes each record of m to dir/<id without prefix>.json.
func writeRecords[T any](root, dir, prefix string, m map[string]*T) error {
	for _, id := range sortedKeys(m) {
		if err := writeJSON(root, path.Join(dir, fileName(id, prefix)+".json"), m[id]); err != nil {
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
		return bytes.ReplaceAll(space.ReplaceAll(m, nil), []byte(","), []byte(", "))
	})
	return writeFile(dir, name, append(data, '\n'))
}

// numberArray matches an indented JSON array holding only numbers, which writeJSON puts on one
// line as [1, 2, 3].
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

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
