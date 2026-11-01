package summoning

import (
	"image/color"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// TestFigureLooks builds the figure's layers from hand-made banks: the bare figure's layers, a
// Chainmail's body look and torso shift, a helmet's head look (one more for a female player), and
// boots' leg colors.
func TestFigureLooks(t *testing.T) {
	pal := &content.Palette{ID: "p", Colors: [256]color.RGBA{}}
	bank := func(frames int, c byte) *SpriteBank {
		b := &SpriteBank{Width: 1, Height: 1, Transparent: 0xFF}
		for range frames {
			b.Frames = append(b.Frames, []byte{c})
		}
		return b
	}
	const base = 100
	banks := map[int]*SpriteBank{
		base:                  bank(72, 4),                // the torso, in color 4
		base + legsBank:       bank(72, legsColorFirst+1), // the legs, in color B2h
		base + bareHead:       bank(8, 1),
		base + bareHead + 1:   bank(8, 1),
		base + 6:              bank(8, 1),
		base + 7:              bank(8, 1),
		base + bareBodyMale:   bank(64, 1),
		base + bareBodyFemale: bank(64, 1),
		base + 0x1C:           bank(64, 1),
	}
	im := &importer{pack: content.NewPack(content.Manifest{}), init: &Init{}, palette: pal, report: &Report{}, banks: banks,
		objects: &Objects{Templates: make([]ObjectTemplate, 3)}, itemIDs: []string{"item.coif", "item.mail", "item.boots"}}
	im.init.Bases.CharacterAnimation = base
	im.objects.Templates[0][11], im.objects.Templates[0][13] = 6, categoryHead
	im.objects.Templates[1][11], im.objects.Templates[1][13] = 0x1C, categoryBody
	im.objects.Templates[2][11], im.objects.Templates[2][13] = 3, categoryFeet
	for _, id := range im.itemIDs {
		im.pack.Items[id] = &content.Item{ID: id}
	}
	im.pack.Things[playerID] = &content.Thing{ID: playerID}
	im.figure(72)
	if len(im.report.Lines()) > 0 {
		t.Fatalf("report %v", im.report.Lines())
	}

	want := []content.FigureLayer{
		{Name: layerTorso, FigureLook: content.FigureLook{Sprite: "sprite.player.torso.ce"}},
		{Name: layerLegs, FigureLook: content.FigureLook{Sprite: "sprite.player.legs.8"}},
		{Name: layerBody, FigureLook: content.FigureLook{Sprite: "sprite.player.body.25", Female: "sprite.player.body.26"}},
		{Name: layerHead, FigureLook: content.FigureLook{Sprite: "sprite.player.head.02", Female: "sprite.player.head.03"}},
	}
	got := im.pack.Things[playerID].Figure
	if len(got) != len(want) {
		t.Fatalf("figure %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("layer %d: %+v, want %+v", i, got[i], want[i])
		}
	}
	if l := im.pack.Items["item.coif"].Looks; len(l) != 1 || l[layerHead] != (content.FigureLook{Sprite: "sprite.player.head.06", Female: "sprite.player.head.07"}) {
		t.Errorf("coif looks %+v", l)
	}
	// Chainmail (1Ch) shifts the torso by 03h, its own colors.
	if l := im.pack.Items["item.mail"].Looks; len(l) != 2 || l[layerBody].Sprite != "sprite.player.body.1c" || l[layerTorso].Sprite != "sprite.player.torso.03" {
		t.Errorf("mail looks %+v", l)
	}
	if l := im.pack.Items["item.boots"].Looks; len(l) != 1 || l[layerLegs].Sprite != "sprite.player.legs.3" {
		t.Errorf("boots looks %+v", l)
	}
	// The torso's color 4 shifts to CEh + 1; the legs' B2h becomes set 3's second color, 02h, and
	// set 8's CDh.
	for id, want := range map[string]uint8{"sprite.player.torso.ce": 0xCF, "sprite.player.torso.03": 4, "sprite.player.legs.3": 2, "sprite.player.legs.8": 0xCD} {
		b := im.pack.Sprites[id]
		if b == nil || b.Frames != 72 {
			t.Errorf("%s: %+v", id, b)
			continue
		}
		if c := b.Sheet.ColorIndexAt(0, 0); c != want {
			t.Errorf("%s: color %02Xh, want %02Xh", id, c, want)
		}
	}
	// Every layer shares the figure's anchor at the player's ground point.
	if a := im.pack.Sprites["sprite.player.head.02"].Anchor; a == nil || *a != (content.Offset{creatureAnchor[0] - headAt[0], creatureAnchor[1] - headAt[1]}) {
		t.Errorf("head anchor %v", a)
	}
}
