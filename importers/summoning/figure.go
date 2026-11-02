package summoning

import (
	"fmt"
	"image"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The player's figure in layers, and the looks worn objects give them (player.md, Figure and
// Worn looks).

// Figure layers, bottom up, and where each bank's frames go on the figure.
const (
	layerTorso = "torso"
	layerLegs  = "legs"
	layerBody  = "body"
	layerHead  = "head"
)

var (
	torsoAt = content.Offset{0, 13}
	legsAt  = content.Offset{0, 18}
	bodyAt  = content.Offset{0, 5}
	headAt  = content.Offset{8, 0}
)

// Looks of the bare figure, and the range of body looks the torso shifts are given for.
const (
	bareHead        = 2
	bareBodyMale    = 0x25
	bareBodyFemale  = 0x26
	bareFeet        = 8
	firstBodyLook   = 0x18
	torsoOwnColors  = 3
	torsoShiftFirst = 3
	torsoShiftCount = 10
	legsColorFirst  = 0xB1
	legsColorSkip   = 8
	// Categories of worn objects with figure looks (items.md): head, body and feet.
	categoryHead = 0
	categoryBody = 1
	categoryFeet = 2
)

// torsoShifts are the torso's color shifts for body looks 18h..2Bh.
var torsoShifts = [...]int{
	0x80, 0xC6, 0x06, 0x03, 0x03, 0xC6, 0x03, 0x25, 0x80, 0x00,
	0xC6, 0x60, 0x03, 0xCE, 0xCE, 0x80, 0xC6, 0xC6, 0x80, 0x00,
}

// legColorSets are the legs' color sets 1..8: color B1h + i drawn in color i of the row.
var legColorSets = [...][10]byte{
	{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x08},
	{0x00, 0x01, 0x02, 0x03, 0x04, 0xCD, 0xCF, 0xD0, 0xD2, 0xD2},
	{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x90, 0x92, 0x92},
	{0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0A},
	{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x37, 0x37},
	{0xE8, 0xEA, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x58, 0x58},
	{0xE4, 0xE6, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x59},
	{0xCC, 0xCD, 0xCF, 0xD0, 0xD1, 0xD2, 0xD3, 0xD5, 0xD7, 0xD7},
}

// figure builds the player's figure layers for its n frames.
type figure struct {
	im   *importer
	base int
	n    int
}

// figure gives the player template its layers and every worn item its looks.
func (im *importer) figure(n int) {
	fg := &figure{im: im, base: int(im.init.Bases.CharacterAnimation), n: n}
	torso := func(look int) content.FigureLook {
		return content.FigureLook{Sprite: fg.torso(torsoShift(look))}
	}
	layers := []content.FigureLayer{
		{Name: layerTorso, FigureLook: torso(bareBodyMale)},
		{Name: layerLegs, FigureLook: content.FigureLook{Sprite: fg.legs(bareFeet)}},
		{Name: layerBody, FigureLook: content.FigureLook{Sprite: fg.body(bareBodyMale), Female: fg.body(bareBodyFemale)}},
		{Name: layerHead, FigureLook: content.FigureLook{Sprite: fg.head(bareHead), Female: fg.head(bareHead + 1)}},
	}
	if fem := torso(bareBodyFemale).Sprite; fem != layers[0].Sprite {
		layers[0].Female = fem
	}
	for _, l := range layers {
		if l.Sprite == "" || l.Female == "" && l.Name != layerTorso && l.Name != layerLegs {
			im.report.addf("player figure: layer %s missing; drawn whole", l.Name)
			return
		}
	}
	im.pack.Things[playerID].Figure = layers

	for i := range im.objects.Templates {
		tm := &im.objects.Templates[i]
		look := int(tm[11] & 0x7F)
		looks := map[string]content.FigureLook{}
		switch tm.Category() {
		case categoryHead:
			if look != 0 {
				if s, f := fg.head(look), fg.head(look+1); s != "" && f != "" {
					looks[layerHead] = content.FigureLook{Sprite: s, Female: f}
				}
			}
		case categoryBody:
			if look != 0 {
				if s := fg.body(look); s != "" {
					looks[layerBody] = content.FigureLook{Sprite: s}
				}
				if k := torsoShift(look); k >= 0 {
					if s := fg.torso(k); s != "" {
						looks[layerTorso] = content.FigureLook{Sprite: s}
					}
				}
			}
		case categoryFeet:
			if s := fg.legs(look); s != "" {
				looks[layerLegs] = content.FigureLook{Sprite: s}
			}
		}
		if len(looks) > 0 {
			im.pack.Items[im.itemIDs[i]].Looks = looks
		}
	}
}

// torsoShift returns body look look's torso shift, or −1 when the table has none.
func torsoShift(look int) int {
	if look < firstBodyLook || look-firstBodyLook >= len(torsoShifts) {
		return -1
	}
	return torsoShifts[look-firstBodyLook]
}

// bodyFrame returns the torso's and legs' frame for the figure's frame f: f in the walk's groups,
// the step in the strikes'.
func bodyFrame(f int) int {
	if f/8 >= playerWalkGroups {
		return f % 8
	}
	return f
}

// torso returns the sprite of the torso with color shift k (torsoOwnColors for its own), adding
// it the first time; "" when it cannot be made.
func (fg *figure) torso(k int) string {
	if k < 0 {
		k = torsoOwnColors
	}
	return fg.layer(fmt.Sprintf("sprite.player.torso.%02x", k), 0, torsoAt, bodyFrame, func(c byte) byte {
		if k != torsoOwnColors && int(c) >= torsoShiftFirst && int(c) < torsoShiftFirst+torsoShiftCount {
			return byte(k + int(c) - torsoShiftFirst)
		}
		return c
	})
}

// legs returns the sprite of the legs in color set set (0 for their own colors).
func (fg *figure) legs(set int) string {
	if set < 0 || set > len(legColorSets) {
		return ""
	}
	return fg.layer(fmt.Sprintf("sprite.player.legs.%d", set), legsBank, legsAt, bodyFrame, func(c byte) byte {
		i := int(c) - legsColorFirst
		if set == 0 || i < 0 || i >= len(legColorSets[0]) || i == legsColorSkip {
			return c
		}
		return legColorSets[set-1][i]
	})
}

// body returns the sprite of body look look.
func (fg *figure) body(look int) string {
	return fg.layer(fmt.Sprintf("sprite.player.body.%02x", look), look, bodyAt, func(f int) int {
		return headFrames[f/8] + f%8
	}, nil)
}

// head returns the sprite of head look look.
func (fg *figure) head(look int) string {
	return fg.layer(fmt.Sprintf("sprite.player.head.%02x", look), look, headAt, func(f int) int { return f % 8 }, nil)
}

// layer returns sprite id, adding it the first time: for each of the figure's frames, the frame
// frame picks of bank C + bank, drawn at at, its colors through recolor (nil keeps them). It
// returns "" when the bank is missing.
func (fg *figure) layer(id string, bank int, at content.Offset, frame func(int) int, recolor func(byte) byte) string {
	im := fg.im
	if _, ok := im.pack.Sprites[id]; ok {
		return id
	}
	b, err := im.bank(fg.base + bank)
	if err != nil {
		im.report.addf("player figure %s: %v", id, err)
		return ""
	}
	box := image.Rect(at[0], at[1], at[0]+b.Width, at[1]+b.Height)
	frames := make([][]byte, fg.n)
	for f := range fg.n {
		var parts []part
		if k := frame(f); k < len(b.Frames) {
			parts = []part{{b, k, at}}
		}
		frames[f] = compose(box, b.Transparent, parts)
		if recolor != nil {
			for i, c := range frames[f] {
				if int(c) != b.Transparent {
					frames[f][i] = recolor(c)
				}
			}
		}
	}
	im.addSprite(id, frames, box.Dx(), box.Dy(), b.Transparent)
	anchor := content.Offset{creatureAnchor[0] - box.Min.X, creatureAnchor[1] - box.Min.Y}
	im.pack.Sprites[id].Anchor = &anchor
	return id
}
