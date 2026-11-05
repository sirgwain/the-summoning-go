package content

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

// Mods (docs/mods-plan.md, phase 2): a pack that loads over another, adding records and
// replacing them by id. This first cut takes sprites, floor and wall sets and text; patches of
// other records come later.

// ModKind is a mod's manifest kind.
const ModKind = "mod"

// LoadMod loads the mod in fsys over pack p, verifies the result and returns what the mod
// changed, a line each. A sprite replacing one of p's must keep its frames, animations and
// size at its own density, so whatever uses it still works.
func LoadMod(p *Pack, fsys fs.FS) ([]string, error) {
	var m Manifest
	if err := readJSON(fsys, manifestFile, &m); err != nil {
		return nil, err
	}
	switch {
	case m.Kind != ModKind:
		return nil, fmt.Errorf("%s: %s is not a mod (kind %q)", manifestFile, m.ID, m.Kind)
	case m.Format != Format:
		return nil, fmt.Errorf("%s: mod %s has format %d, want %d", manifestFile, m.ID, m.Format, Format)
	case !slices.Contains(m.Depends, p.Manifest.ID):
		return nil, fmt.Errorf("%s: mod %s does not depend on %s", manifestFile, m.ID, p.Manifest.ID)
	}
	var changes []string
	note := func(what, id string, replaced bool) {
		verb := "adds"
		if replaced {
			verb = "replaces"
		}
		changes = append(changes, fmt.Sprintf("%s %s %s %s", m.ID, verb, what, id))
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
		old, replaced := p.Sprites[b.ID]
		if replaced {
			if err := sameShape(old, b); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		p.Sprites[b.ID] = b
		note("sprite", b.ID, replaced)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := loadEach(fsys, floorsDir+"/*.json", func(name string) error {
		f := &FloorSet{}
		if err := readJSON(fsys, name, f); err != nil {
			return err
		}
		_, replaced := p.Floors[f.ID]
		p.Floors[f.ID] = f
		note("floor set", f.ID, replaced)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := loadEach(fsys, wallsDir+"/*.json", func(name string) error {
		w := &WallSet{}
		if err := readJSON(fsys, name, w); err != nil {
			return err
		}
		_, replaced := p.Walls[w.ID]
		p.Walls[w.ID] = w
		note("wall set", w.ID, replaced)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := loadEach(fsys, textDir+"/*.json", func(name string) error {
		t := &Text{}
		if err := readJSON(fsys, name, t); err != nil {
			return err
		}
		base, ok := p.Texts[t.ID]
		if !ok {
			p.Texts[t.ID] = t
			note("text", t.ID, false)
			return nil
		}
		for k, s := range t.Strings {
			base.Strings[k] = s
		}
		note(fmt.Sprintf("%d strings of", len(t.Strings)), t.ID, true)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := p.Verify(); err != nil {
		return changes, fmt.Errorf("with mod %s: %w", m.ID, err)
	}
	return changes, nil
}

// sameShape reports how sprite b, replacing old, differs from it in what its users need: its
// frame count, its named animations, and its frame size and anchors at its own density.
func sameShape(old, b *SpriteBank) error {
	k, ko := b.Scale(), old.Scale()
	switch {
	case b.Frames != old.Frames:
		return fmt.Errorf("sprite %s has %d frames, the one it replaces %d", b.ID, b.Frames, old.Frames)
	case b.FrameSize[0]*ko != old.FrameSize[0]*k || b.FrameSize[1]*ko != old.FrameSize[1]*k:
		return fmt.Errorf("sprite %s frames are %v at density %d, the one it replaces %v at %d", b.ID, b.FrameSize, k, old.FrameSize, ko)
	}
	for name := range old.Animations {
		if _, ok := b.Animations[name]; !ok {
			return fmt.Errorf("sprite %s lacks animation %q of the one it replaces", b.ID, name)
		}
	}
	return nil
}
