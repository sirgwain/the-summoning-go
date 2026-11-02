package devpack

import (
	"os"
	"sync"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/importers/summoning"
)

// The dev pack's play tests run on the dev pack, and again dressed in the game's art (GameArt)
// when SUMMONING_DATA names the game's data files (AGENTS.md, rule 3). The levels and rules are
// the same; the figures, tiles and flipbooks, and so their sizes, animations and timings, are
// the game's.

var gameArt struct {
	once sync.Once
	pack *content.Pack
	err  error
}

// artPack imports the game's pack from SUMMONING_DATA once, skipping the test without it.
func artPack(t *testing.T) *content.Pack {
	t.Helper()
	dir := os.Getenv("SUMMONING_DATA")
	if dir == "" {
		t.Skip("SUMMONING_DATA is not set")
	}
	g := &gameArt
	g.once.Do(func() {
		src, err := summoning.ReadSource(dir)
		if err != nil {
			g.err = err
			return
		}
		g.pack, _, g.err = summoning.Import(src, summoning.Options{NoAudio: true})
	})
	if g.err != nil {
		t.Fatal(g.err)
	}
	return g.pack
}

// dressed returns a builder of the dev pack in the game's art, failing the test on anything it
// could not dress.
func dressed(t *testing.T) func() *content.Pack {
	art := artPack(t)
	return func() *content.Pack {
		t.Helper()
		p := Build()
		if missing := Dress(p, art, GameArt); len(missing) > 0 {
			t.Fatalf("not dressed: %v", missing)
		}
		return p
	}
}

// forPacks runs f on the dev pack, and on the dev pack in the game's art when it can.
func forPacks(t *testing.T, f func(t *testing.T, build func() *content.Pack)) {
	t.Run("dev", func(t *testing.T) { f(t, Build) })
	t.Run("game art", func(t *testing.T) { f(t, dressed(t)) })
}

// The dressed pack verifies, every look and figure it uses comes from the game's pack, and it
// writes and loads back.
func TestGameArt(t *testing.T) {
	p := dressed(t)()
	if err := p.Verify(); err != nil {
		t.Fatal(err)
	}
	art := artPack(t)
	for _, l := range p.Levels {
		if l.Floors != GameArt.Floors || l.Walls != GameArt.Walls {
			t.Errorf("%s drawn with %s and %s", l.ID, l.Floors, l.Walls)
		}
		for _, f := range l.Features {
			if lk := p.Looks[f.Look]; lk != nil {
				for _, parts := range lk.States {
					for _, part := range parts {
						if !fromArt(art, part.Sprite) {
							t.Errorf("%s %s drawn with the dev pack's %s", l.ID, f.ID, part.Sprite)
						}
					}
				}
			}
		}
	}
	for id := range GameArt.Things {
		if s := p.Things[id].Sprite; !fromArt(art, s) {
			t.Errorf("%s drawn with the dev pack's %s", id, s)
		}
	}
	for id := range GameArt.Items {
		if s := p.Items[id].Icon.Sprite; !fromArt(art, s) {
			t.Errorf("%s's icon is the dev pack's %s", id, s)
		}
	}
	dir := t.TempDir()
	if err := content.Write(p, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := content.Load(os.DirFS(dir)); err != nil {
		t.Fatal(err)
	}
}

// fromArt reports whether sprite bank id, as the dressed pack names it, is one of art's.
func fromArt(art *content.Pack, id string) bool {
	for {
		if art.Sprites[id] != nil {
			return true
		}
		switch {
		case len(id) > 4 && id[len(id)-4:] == ".art":
			id = id[:len(id)-4]
		case len(id) > 9 && id[len(id)-9:] == ".mirrored":
			id = id[:len(id)-9]
		default:
			return false
		}
	}
}
